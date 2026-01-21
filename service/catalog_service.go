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
	"armario-mascota-me/pricing"
	"armario-mascota-me/repository"
	"armario-mascota-me/utils"

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
			return chromePath
		}
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
			return path
		}
	}

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
		base64Str := base64.StdEncoding.EncodeToString(data)

		// Determine MIME type
		mimeType := "image/png"
		if filepath.Ext(filePath) == ".jpg" || filepath.Ext(filePath) == ".jpeg" {
			mimeType = "image/jpeg"
		}

		dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Str)
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

	// Always use absolute URLs for logo and background
	// Determine file extension
	var logoExt, bgExt, introExt string
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
	for _, ext := range extensions {
		if _, err := os.Stat(filepath.Join("static", "catalog", "intro"+ext)); err == nil {
			introExt = ext
			break
		}
	}

	logoURL := ""
	backgroundURL := ""
	introURL := ""

	if logoExt != "" {
		logoURL = fmt.Sprintf("%s/static/catalog/logo%s", s.baseURL, logoExt)
	}

	if bgExt != "" {
		backgroundURL = fmt.Sprintf("%s/static/catalog/background%s", s.baseURL, bgExt)
	}
	if introExt != "" {
		introURL = fmt.Sprintf("%s/static/catalog/intro%s", s.baseURL, introExt)
	}

	// Pricing for intro page (BUSOS pricebook by size bucket)
	retailPrice := ""
	wholesalePrice := ""
	if engine := pricing.GetEngine(); engine != nil {
		if r, w, ok := engine.GetCatalogBusoPrices(size); ok {
			retailPrice = utils.FormatCOP(r)
			wholesalePrice = utils.FormatCOP(w)
		}
	}

	// Prepare template data
	templateData := struct {
		Size           string
		Pages          [][]models.CatalogItem
		LogoURL        string
		BackgroundURL  string
		IntroURL       string
		RetailPrice    string
		WholesalePrice string
	}{
		Size:           size,
		Pages:          pages,
		LogoURL:        logoURL,
		BackgroundURL:  backgroundURL,
		IntroURL:       introURL,
		RetailPrice:    retailPrice,
		WholesalePrice: wholesalePrice,
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
			chromedp.NoSandbox,                          // Required for running in Docker/containers
			chromedp.Flag("enable-print-preview", true), // Enable print preview
		)
		allocCtx, allocCancel = chromedp.NewExecAllocator(ctx, opts...)
		defer allocCancel()
	} else {
		// Let chromedp auto-detect (may fail in containers)
		opts := append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.NoSandbox,
			chromedp.Flag("enable-print-preview", true), // Enable print preview
		)
		allocCtx, allocCancel = chromedp.NewExecAllocator(ctx, opts...)
		defer allocCancel()
	}

	chromedpCtx, chromedpCancel := chromedp.NewContext(allocCtx)
	defer chromedpCancel()

	// Enable Page domain for printing
	if err := chromedp.Run(chromedpCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		return page.Enable().Do(ctx)
	})); err != nil {
		// Log warning but continue
	}

	// Construct render URL
	renderURL := fmt.Sprintf("%s/admin/catalog/render?size=%s", s.baseURL, size)

	var pdfBuf []byte

	// Run chromedp with proper viewport and wait for network/idle
	// 210mm = 794px at 96 DPI, 350mm = 1323px at 96 DPI
	// Use a larger viewport height to accommodate multiple pages
	err := chromedp.Run(chromedpCtx,
		chromedp.EmulateViewport(794, 5000), // Large height to show all pages
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
		// Set html and body width, but let height be auto to accommodate all pages
		chromedp.Evaluate(`
			document.documentElement.style.width = '210mm';
			document.documentElement.style.height = 'auto';
			document.documentElement.style.minHeight = '350mm';
			document.body.style.width = '210mm';
			document.body.style.height = 'auto';
			document.body.style.minHeight = '350mm';
		`, nil),
		chromedp.Sleep(1000), // Final wait for layout
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			// 210mm x 350mm = 8.27" x 13.78" (1mm = 0.03937 inches)
			// PrintToPDF will automatically handle page breaks via CSS page-break-after
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

	return pdfBuf, nil
}

// GeneratePNG generates PNG images from HTML using chromedp
// Returns a map of page number to PNG data, or error
// size parameter is used to construct the render URL
func (s *CatalogService) GeneratePNG(ctx context.Context, size string) (map[int][]byte, error) {
	// Get items to calculate expected page count
	items, err := s.repository.GetItemsBySizeForCatalog(ctx, size)
	var expectedPages int
	if err != nil {
		expectedPages = 0
	} else {
		// Ceiling division for product pages (9 items per page) + 1 intro page
		expectedPages = (len(items)+8)/9 + 1
	}

	// PNG generation can be slower than PDF because we screenshot each page.
	// Use a dynamic timeout based on expected pages to avoid truncating large catalogs.
	timeout := 30 * time.Second
	if expectedPages > 1 {
		// Base + per-page budget; capped to keep requests bounded.
		timeout = time.Duration(20+expectedPages*10) * time.Second
		if timeout > 3*time.Minute {
			timeout = 3 * time.Minute
		}
	}
	log.Printf("📸 GeneratePNG: size=%s expectedPages=%d timeout=%s", size, expectedPages, timeout)

	ctxTimeout, cancel := context.WithTimeout(ctx, timeout)
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
		allocCtx, allocCancel = chromedp.NewExecAllocator(ctxTimeout, opts...)
		defer allocCancel()
	} else {
		// Let chromedp auto-detect (may fail in containers)
		allocCtx, allocCancel = chromedp.NewExecAllocator(ctxTimeout, chromedp.NoSandbox)
		defer allocCancel()
	}

	chromedpCtx, chromedpCancel := chromedp.NewContext(allocCtx)
	defer chromedpCancel()

	// Construct render URL
	renderURL := fmt.Sprintf("%s/admin/catalog/render?size=%s", s.baseURL, size)

	// Get page count using JavaScript evaluation
	// Use a larger viewport to see all pages
	var pageCountVal float64
	err = chromedp.Run(chromedpCtx,
		chromedp.EmulateViewport(794, 5000), // Large height to see all pages
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
		// Set width but let height be auto to show all pages
		chromedp.Evaluate(`
			document.documentElement.style.width = '210mm';
			document.documentElement.style.height = 'auto';
			document.documentElement.style.minHeight = '350mm';
			document.body.style.width = '210mm';
			document.body.style.height = 'auto';
			document.body.style.minHeight = '350mm';
		`, nil),
		chromedp.Sleep(2000), // Wait for initial layout
		// Scroll to bottom to ensure all pages are rendered
		chromedp.Evaluate(`
			window.scrollTo(0, document.body.scrollHeight);
		`, nil),
		chromedp.Sleep(1000), // Wait after scroll
		chromedp.Evaluate(`
			window.scrollTo(0, 0);
		`, nil),
		chromedp.Sleep(500), // Wait after scroll back
		chromedp.Evaluate(`document.querySelectorAll('.page').length`, &pageCountVal),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get page count: %w", err)
	}

	// Convert to int
	pageCount := int(pageCountVal)

	if pageCount == 0 {
		return nil, fmt.Errorf("no pages found in HTML")
	}

	// Double-check page count with a different method and get more info
	var pageInfo struct {
		Count    float64 `json:"count"`
		HTML     string  `json:"html"`
		BodyHTML string  `json:"bodyHTML"`
	}
	err = chromedp.Run(chromedpCtx,
		chromedp.Evaluate(`
			(function() {
				const pages = document.querySelectorAll('.page');
				return {
					count: pages.length,
					html: document.documentElement.outerHTML.substring(0, 500),
					bodyHTML: document.body.innerHTML.substring(0, 500)
				};
			})();
		`, &pageInfo),
	)
	if err == nil {
		if int(pageInfo.Count) != pageCount {
			pageCount = int(pageInfo.Count)
		}
		// If expected pages is set and doesn't match detected count, use expected
		if expectedPages > 0 && pageCount != expectedPages {
			pageCount = expectedPages
		}
		if pageCount == 1 && expectedPages > 1 {
			pageCount = expectedPages
		}
	} else if expectedPages > 0 && pageCount != expectedPages {
		// If verification failed but we have expected pages, use that
		pageCount = expectedPages
	}
	log.Printf("📄 GeneratePNG: size=%s detectedPages=%d (expected=%d)", size, pageCount, expectedPages)

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

	// For multiple pages, capture each page individually
	// We already navigated and loaded the page above, so we can reuse the same context
	pngs := make(map[int][]byte)
	missingPages := make([]int, 0)
	const maxAttemptsPerPage = 2

	restoreAllPages := func() {
		_ = chromedp.Run(chromedpCtx,
			chromedp.Evaluate(`
				(function() {
					const pages = document.querySelectorAll('.page');
					pages.forEach(page => {
						page.style.display = 'flex';
						page.style.visibility = 'visible';
					});
					document.documentElement.style.height = 'auto';
					document.documentElement.style.overflow = '';
					document.body.style.height = 'auto';
					document.body.style.overflow = '';
				})();
			`, nil),
		)
	}

	// Capture each page individually
	for pageNum := 1; pageNum <= pageCount; pageNum++ {
		var buf []byte
		var lastErr error

		for attempt := 1; attempt <= maxAttemptsPerPage; attempt++ {
			buf = nil
			lastErr = chromedp.Run(chromedpCtx,
				// Set viewport to match page size
				chromedp.EmulateViewport(794, 1323), // 210mm x 350mm
				// Hide all pages except the current one and adjust body height
				chromedp.Evaluate(fmt.Sprintf(`
					(function() {
						const pages = document.querySelectorAll('.page');
						if (pages.length === 0) {
							return 0;
						}
						pages.forEach((page, index) => {
							if (index === %d - 1) {
								page.style.display = 'flex';
								page.style.visibility = 'visible';
								page.style.position = 'relative';
							} else {
								page.style.display = 'none';
								page.style.visibility = 'hidden';
							}
						});
						// Adjust body and html height to match single page
						document.documentElement.style.width = '210mm';
						document.documentElement.style.height = '350mm';
						document.documentElement.style.overflow = 'hidden';
						document.body.style.width = '210mm';
						document.body.style.height = '350mm';
						document.body.style.overflow = 'hidden';
						return pages.length;
					})();
				`, pageNum), nil),
				chromedp.Sleep(900), // Wait for display change and layout
				chromedp.CaptureScreenshot(&buf),
			)

			if lastErr == nil && len(buf) > 0 {
				break
			}

			log.Printf("⚠️ GeneratePNG: failed page=%d attempt=%d/%d err=%v buf=%d", pageNum, attempt, maxAttemptsPerPage, lastErr, len(buf))
			restoreAllPages()
			time.Sleep(400 * time.Millisecond)
		}

		if lastErr != nil || len(buf) == 0 {
			missingPages = append(missingPages, pageNum)
			// Restore for subsequent pages before continuing
			restoreAllPages()
			continue
		}

		pngs[pageNum] = buf

		// Restore all pages visibility for next iteration
		if pageNum < pageCount {
			restoreAllPages()
		}
	}

	if len(pngs) == 0 {
		return nil, fmt.Errorf("failed to capture any pages")
	}
	if len(missingPages) > 0 {
		return nil, fmt.Errorf("failed to capture all pages: missing=%v captured=%d/%d", missingPages, len(pngs), pageCount)
	}

	return pngs, nil
}
