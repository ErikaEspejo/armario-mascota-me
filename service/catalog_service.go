package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"armario-mascota-me/models"
	"armario-mascota-me/repository"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// CatalogService handles catalog generation operations
type CatalogService struct {
	repository      repository.CatalogRepositoryInterface
	designAssetRepo repository.DesignAssetRepositoryInterface
	driveService    DriveServiceInterface
	baseURL         string // Base URL for image endpoints (e.g., "http://localhost:8080")
}

// detectChromePath detects the path to Chrome/Chromium executable
// Checks CHROME_PATH env var first, then common installation paths
func detectChromePath() string {
	// Check environment variable first
	if chromePath := os.Getenv("CHROME_PATH"); chromePath != "" {
		if _, err := os.Stat(chromePath); err == nil {
			log.Printf("🔍 Using Chrome from CHROME_PATH: %s", chromePath)
			return chromePath
		}
		log.Printf("⚠️  CHROME_PATH set but file not found: %s", chromePath)
	}

	// Common paths to check
	paths := []string{
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"/usr/bin/google-chrome",
		"/usr/bin/google-chrome-stable",
		"/snap/bin/chromium",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			log.Printf("🔍 Auto-detected Chrome: %s", path)
			return path
		}
	}

	log.Printf("⚠️  Chrome/Chromium not found in common paths. chromedp will attempt to find it automatically.")
	return ""
}

// NewCatalogService creates a new CatalogService
func NewCatalogService(
	repo repository.CatalogRepositoryInterface,
	designAssetRepo repository.DesignAssetRepositoryInterface,
	driveService DriveServiceInterface,
	baseURL string,
) *CatalogService {
	return &CatalogService{
		repository:      repo,
		designAssetRepo: designAssetRepo,
		driveService:    driveService,
		baseURL:         baseURL,
	}
}

// fetchImageAsBase64 fetches an image from the image endpoint and converts it to base64
func (s *CatalogService) fetchImageAsBase64(imageURL string) (string, error) {
	// If imageURL is already a full URL, use it; otherwise prepend baseURL
	var fullURL string
	if imageURL[0] == '/' {
		fullURL = s.baseURL + imageURL
	} else {
		fullURL = imageURL
	}

	log.Printf("📥 Fetching image: %s", fullURL)

	// Make HTTP request
	resp, err := http.Get(fullURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("image endpoint returned status %d", resp.StatusCode)
	}

	// Read image data
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read image data: %w", err)
	}

	// Convert to base64
	base64Str := base64.StdEncoding.EncodeToString(imageData)
	return base64Str, nil
}

// convertItemsToBase64 converts image URLs to base64 for all items
func (s *CatalogService) convertItemsToBase64(ctx context.Context, items []models.CatalogItem) {
	for i := range items {
		if items[i].ImageURL != "" {
			base64, err := s.fetchImageAsBase64(items[i].ImageURL)
			if err != nil {
				log.Printf("⚠️  Warning: Failed to fetch image for item %d: %v", items[i].ID, err)
				// Continue without image
				continue
			}
			items[i].ImageBase64 = base64
		}
	}
}

// loadStaticAsset loads a static asset file and converts it to base64 if needed
func (s *CatalogService) loadStaticAsset(filename string, useBase64 bool) (string, string, error) {
	// Try different extensions
	extensions := []string{".png", ".jpg", ".jpeg"}
	var filePath string
	var found bool

	for _, ext := range extensions {
		path := filepath.Join("static", "catalog", filename+ext)
		if _, err := os.Stat(path); err == nil {
			filePath = path
			found = true
			break
		}
	}

	if !found {
		return "", "", fmt.Errorf("static asset not found: %s", filename)
	}

	if useBase64 {
		// Read file and convert to base64
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", "", fmt.Errorf("failed to read file: %w", err)
		}
		log.Printf("📁 Loaded static asset %s: %d bytes", filePath, len(data))
		base64Str := base64.StdEncoding.EncodeToString(data)

		// Determine MIME type
		mimeType := "image/png"
		if filepath.Ext(filePath) == ".jpg" || filepath.Ext(filePath) == ".jpeg" {
			mimeType = "image/jpeg"
		}

		dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Str)
		log.Printf("✅ Created data URI for %s: %s (length: %d chars)", filename, mimeType, len(dataURI))
		return dataURI, "", nil
	}

	// Return URL path
	urlPath := fmt.Sprintf("/static/catalog/%s%s", filename, filepath.Ext(filePath))
	return urlPath, "", nil
}

// paginateItems splits items into pages of 9 items each
func paginateItems(items []models.CatalogItem) [][]models.CatalogItem {
	const itemsPerPage = 9
	var pages [][]models.CatalogItem

	for i := 0; i < len(items); i += itemsPerPage {
		end := i + itemsPerPage
		if end > len(items) {
			end = len(items)
		}
		pages = append(pages, items[i:end])
	}

	return pages
}

// RenderCatalogHTML renders the catalog HTML template
func (s *CatalogService) RenderCatalogHTML(ctx context.Context, size string, items []models.CatalogItem, useBase64 bool) (string, error) {
	// Convert images to base64 if needed for HTML direct view (not for PDF/PNG)
	if useBase64 {
		s.convertItemsToBase64(ctx, items)
	}

	// Paginate items
	pages := paginateItems(items)
	log.Printf("📄 RenderCatalogHTML: Paginated %d items into %d pages", len(items), len(pages))
	for i, page := range pages {
		log.Printf("  Page %d: %d items", i+1, len(page))
	}

	// Always use absolute URLs for logo and background
	// Determine file extension
	var logoExt, bgExt string
	extensions := []string{".png", ".jpg", ".jpeg"}
	for _, ext := range extensions {
		if _, err := os.Stat(filepath.Join("static", "catalog", "logo"+ext)); err == nil {
			logoExt = ext
			break
		}
	}
	for _, ext := range extensions {
		if _, err := os.Stat(filepath.Join("static", "catalog", "background"+ext)); err == nil {
			bgExt = ext
			break
		}
	}

	logoURL := ""
	backgroundURL := ""

	if logoExt != "" {
		logoURL = fmt.Sprintf("%s/static/catalog/logo%s", s.baseURL, logoExt)
		log.Printf("📸 Logo URL: %s", logoURL)
	} else {
		log.Printf("⚠️  Warning: Logo file not found")
	}

	if bgExt != "" {
		backgroundURL = fmt.Sprintf("%s/static/catalog/background%s", s.baseURL, bgExt)
		log.Printf("📸 Background URL: %s", backgroundURL)
	} else {
		log.Printf("⚠️  Warning: Background file not found")
	}

	// Prepare template data
	templateData := struct {
		Size          string
		Pages         [][]models.CatalogItem
		LogoURL       string
		BackgroundURL string
	}{
		Size:          size,
		Pages:         pages,
		LogoURL:       logoURL,
		BackgroundURL: backgroundURL,
	}

	// Load template
	templatePath := filepath.Join("templates", "catalog.html")
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Render template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	htmlContent := buf.String()
	return htmlContent, nil
}

// GeneratePDF generates a PDF from HTML using chromedp
// size parameter is used to construct the render URL
func (s *CatalogService) GeneratePDF(ctx context.Context, size string) ([]byte, error) {
	log.Printf("📄 Generating PDF from URL: size=%s", size)

	// Create context with timeout (30 seconds)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Detect Chrome/Chromium path and configure chromedp
	chromePath := detectChromePath()
	var allocCtx context.Context
	var allocCancel context.CancelFunc

	if chromePath != "" {
		// Use detected Chrome path
		opts := append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.ExecPath(chromePath),
			chromedp.NoSandbox, // Required for running in Docker/containers
		)
		allocCtx, allocCancel = chromedp.NewExecAllocator(ctx, opts...)
		defer allocCancel()
	} else {
		// Let chromedp auto-detect (may fail in containers)
		allocCtx, allocCancel = chromedp.NewExecAllocator(ctx, chromedp.NoSandbox)
		defer allocCancel()
	}

	chromedpCtx, chromedpCancel := chromedp.NewContext(allocCtx)
	defer chromedpCancel()

	// Construct render URL
	renderURL := fmt.Sprintf("%s/admin/catalog/render?size=%s", s.baseURL, size)
	log.Printf("🌐 Navigating to: %s", renderURL)

	var pdfBuf []byte

	// Run chromedp with proper viewport and wait for network/idle
	// 210mm = 794px at 96 DPI, 350mm = 1323px at 96 DPI
	err := chromedp.Run(chromedpCtx,
		chromedp.EmulateViewport(794, 1323),
		chromedp.Navigate(renderURL),
		chromedp.WaitReady("body"),
		chromedp.Sleep(2000), // Wait for initial page load
		// Wait for fonts and images to load
		chromedp.Evaluate(`
			(function() {
				return Promise.all([
					document.fonts.ready,
					Promise.all(Array.from(document.querySelectorAll('img')).map(img => {
						return new Promise((resolve) => {
							if (img.complete && img.naturalWidth > 0 && img.naturalHeight > 0) {
								resolve();
								return;
							}
							const timeout = setTimeout(() => resolve(), 5000);
							img.onload = () => { clearTimeout(timeout); resolve(); };
							img.onerror = () => { clearTimeout(timeout); resolve(); };
						});
					}))
				]);
			})();
		`, nil),
		// Set body and html to exact size
		chromedp.Evaluate(`
			document.documentElement.style.width = '210mm';
			document.documentElement.style.height = '350mm';
			document.body.style.width = '210mm';
			document.body.style.height = '350mm';
		`, nil),
		chromedp.Sleep(1000), // Final wait for layout
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			// 210mm x 350mm = 8.27" x 13.78" (1mm = 0.03937 inches)
			pdfBuf, _, err = page.PrintToPDF().
				WithPrintBackground(true).
				WithPaperWidth(8.27).   // 210mm in inches
				WithPaperHeight(13.78). // 350mm in inches
				WithMarginTop(0).       // No margins, padding is in CSS
				WithMarginBottom(0).
				WithMarginLeft(0).
				WithMarginRight(0).
				Do(ctx)
			return err
		}),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	log.Printf("✓ PDF generated: %d bytes", len(pdfBuf))
	return pdfBuf, nil
}

// GeneratePNG generates PNG images from HTML using chromedp
// Returns a map of page number to PNG data, or error
// size parameter is used to construct the render URL
func (s *CatalogService) GeneratePNG(ctx context.Context, size string) (map[int][]byte, error) {
	log.Printf("📸 Generating PNG from URL: size=%s", size)

	// Create context with timeout (30 seconds)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Detect Chrome/Chromium path and configure chromedp
	chromePath := detectChromePath()
	var allocCtx context.Context
	var allocCancel context.CancelFunc

	if chromePath != "" {
		// Use detected Chrome path
		opts := append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.ExecPath(chromePath),
			chromedp.NoSandbox, // Required for running in Docker/containers
		)
		allocCtx, allocCancel = chromedp.NewExecAllocator(ctx, opts...)
		defer allocCancel()
	} else {
		// Let chromedp auto-detect (may fail in containers)
		allocCtx, allocCancel = chromedp.NewExecAllocator(ctx, chromedp.NoSandbox)
		defer allocCancel()
	}

	chromedpCtx, chromedpCancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer chromedpCancel()

	// Construct render URL
	renderURL := fmt.Sprintf("%s/admin/catalog/render?size=%s", s.baseURL, size)
	log.Printf("🌐 Navigating to: %s", renderURL)

	// Get page count using JavaScript evaluation
	var pageCountVal float64
	err := chromedp.Run(chromedpCtx,
		chromedp.EmulateViewport(794, 1323),
		chromedp.Navigate(renderURL),
		chromedp.WaitReady("body"),
		chromedp.Sleep(2000), // Wait for initial page load
		// Wait for fonts and images to load
		chromedp.Evaluate(`
			(function() {
				return Promise.all([
					document.fonts.ready,
					Promise.all(Array.from(document.querySelectorAll('img')).map(img => {
						return new Promise((resolve) => {
							if (img.complete && img.naturalWidth > 0 && img.naturalHeight > 0) {
								resolve();
								return;
							}
							const timeout = setTimeout(() => resolve(), 5000);
							img.onload = () => { clearTimeout(timeout); resolve(); };
							img.onerror = () => { clearTimeout(timeout); resolve(); };
						});
					}))
				]);
			})();
		`, nil),
		// Set body and html to exact size
		chromedp.Evaluate(`
			document.documentElement.style.width = '210mm';
			document.documentElement.style.height = '350mm';
			document.body.style.width = '210mm';
			document.body.style.height = '350mm';
		`, nil),
		chromedp.Sleep(1000), // Final wait
		chromedp.Evaluate(`document.querySelectorAll('.page').length`, &pageCountVal),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get page count: %w", err)
	}

	// Convert to int
	pageCount := int(pageCountVal)

	log.Printf("📄 Found %d pages", pageCount)

	if pageCount == 0 {
		return nil, fmt.Errorf("no pages found in HTML")
	}

	// For single page, return just that screenshot
	if pageCount == 1 {
		var buf []byte
		err = chromedp.Run(chromedpCtx,
			chromedp.EmulateViewport(794, 1323),
			chromedp.Navigate(renderURL),
			chromedp.WaitReady("body"),
			chromedp.Sleep(2000),
			// Wait for fonts and images to load
			chromedp.Evaluate(`
				(function() {
					return Promise.all([
						document.fonts.ready,
						Promise.all(Array.from(document.querySelectorAll('img')).map(img => {
							return new Promise((resolve) => {
								if (img.complete && img.naturalWidth > 0 && img.naturalHeight > 0) {
									resolve();
									return;
								}
								const timeout = setTimeout(() => resolve(), 5000);
								img.onload = () => { clearTimeout(timeout); resolve(); };
								img.onerror = () => { clearTimeout(timeout); resolve(); };
							});
						}))
					]);
				})();
			`, nil),
			// Set body and html to exact size
			chromedp.Evaluate(`
				document.documentElement.style.width = '210mm';
				document.documentElement.style.height = '350mm';
				document.body.style.width = '210mm';
				document.body.style.height = '350mm';
			`, nil),
			chromedp.Sleep(1000),
			chromedp.CaptureScreenshot(&buf),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to capture screenshot: %w", err)
		}
		return map[int][]byte{1: buf}, nil
	}

	// For multiple pages, return first page only for now
	log.Printf("⚠️  Multi-page PNG: Returning first page only. Full multi-page support requires page-by-page capture.")
	var buf []byte
	err = chromedp.Run(chromedpCtx,
		chromedp.EmulateViewport(794, 1323),
		chromedp.Navigate(renderURL),
		chromedp.WaitReady("body"),
		chromedp.Sleep(2000),
		chromedp.Evaluate(`
			(function() {
				return Promise.all([
					document.fonts.ready,
					Promise.all(Array.from(document.querySelectorAll('img')).map(img => {
						return new Promise((resolve) => {
							if (img.complete && img.naturalWidth > 0 && img.naturalHeight > 0) {
								resolve();
								return;
							}
							const timeout = setTimeout(() => resolve(), 5000);
							img.onload = () => { clearTimeout(timeout); resolve(); };
							img.onerror = () => { clearTimeout(timeout); resolve(); };
						});
					}))
				]);
			})();
		`, nil),
		chromedp.Sleep(1000),
		chromedp.CaptureScreenshot(&buf),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to capture screenshot: %w", err)
	}
	return map[int][]byte{1: buf}, nil
}
