package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"armario-mascota-me/repository"
	"armario-mascota-me/service"
	"armario-mascota-me/utils"
)

// CatalogController handles HTTP requests for catalog generation
type CatalogController struct {
	repository      repository.CatalogRepositoryInterface
	catalogService  *service.CatalogService
	designAssetRepo repository.DesignAssetRepositoryInterface
	driveService    service.DriveServiceInterface
	baseURL         string
	// Temporary storage for PNG pages (key: sessionID, value: map of page number to PNG data)
	pngStorage      map[string]map[int][]byte
	pngStorageMutex sync.RWMutex
}

// NewCatalogController creates a new CatalogController
func NewCatalogController(
	repo repository.CatalogRepositoryInterface,
	designAssetRepo repository.DesignAssetRepositoryInterface,
	driveService service.DriveServiceInterface,
	baseURL string,
) *CatalogController {
	catalogService := service.NewCatalogService(repo, designAssetRepo, driveService, baseURL)
	return &CatalogController{
		repository:      repo,
		catalogService:  catalogService,
		designAssetRepo: designAssetRepo,
		driveService:    driveService,
		baseURL:         baseURL,
		pngStorage:      make(map[string]map[int][]byte),
	}
}

// validSizes is a map of valid size values
var validSizes = map[string]bool{
	"XS": true,
	"S":  true,
	"M":  true,
	"L":  true,
	"XL": true,
	"MN": true, // Mini
	"IT": true, // Intermedio
}

// validFormats is a map of valid format values
var validFormats = map[string]bool{
	"html": true,
	"pdf":  true,
	"png":  true,
}

// GenerateCatalog handles GET /admin/catalog?size=XS&format=pdf|png|html
func (c *CatalogController) GenerateCatalog(w http.ResponseWriter, r *http.Request) {
	// Check if this is actually a png-page request that got routed here
	if strings.HasPrefix(r.URL.Path, "/admin/catalog/png-page") {
		c.DownloadPNGPage(w, r)
		return
	}
	
	if r.Method != http.MethodGet {
		log.Printf("❌ GenerateCatalog: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()

	// Parse query parameters
	size := strings.TrimSpace(r.URL.Query().Get("size"))
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))

	// Validate size parameter
	if size == "" {
		log.Printf("❌ GenerateCatalog: size parameter is required")
		http.Error(w, "size parameter is required", http.StatusBadRequest)
		return
	}

	// Normalize size
	normalizedSize := utils.NormalizeSize(size)
	if !validSizes[normalizedSize] {
		log.Printf("❌ GenerateCatalog: Invalid size: %s", size)
		http.Error(w, fmt.Sprintf("Invalid size. Valid sizes: XS, S, M, L, XL, MN (Mini), IT (Intermedio)"), http.StatusBadRequest)
		return
	}

	// Validate format parameter
	if format == "" {
		log.Printf("❌ GenerateCatalog: format parameter is required")
		http.Error(w, "format parameter is required. Valid formats: html, pdf, png", http.StatusBadRequest)
		return
	}

	if !validFormats[format] {
		log.Printf("❌ GenerateCatalog: Invalid format: %s", format)
		http.Error(w, "Invalid format. Valid formats: html, pdf, png", http.StatusBadRequest)
		return
	}

	// Get items from repository
	items, err := c.repository.GetItemsBySizeForCatalog(ctx, normalizedSize)
	if err != nil {
		log.Printf("❌ GenerateCatalog: Error fetching items: %v", err)
		http.Error(w, fmt.Sprintf("Failed to fetch items: %v", err), http.StatusInternalServerError)
		return
	}

	// Check if there are any items
	if len(items) == 0 {
		log.Printf("⚠️  GenerateCatalog: No items found for size=%s", normalizedSize)
		http.Error(w, fmt.Sprintf("No active items found for size %s", normalizedSize), http.StatusNotFound)
		return
	}

	// Render HTML (with base64 images for PDF/PNG)
	useBase64 := format == "pdf" || format == "png"
	htmlContent, err := c.catalogService.RenderCatalogHTML(ctx, normalizedSize, items, useBase64)
	if err != nil {
		log.Printf("❌ GenerateCatalog: Error rendering HTML: %v", err)
		http.Error(w, fmt.Sprintf("Failed to render catalog: %v", err), http.StatusInternalServerError)
		return
	}

	// Handle different formats
	switch format {
	case "html":
		// Return HTML directly
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(htmlContent)); err != nil {
			log.Printf("❌ GenerateCatalog: Error writing HTML response: %v", err)
		}

	case "pdf":
		// Generate PDF using render endpoint
		pdfData, err := c.catalogService.GeneratePDF(ctx, normalizedSize)
		if err != nil {
			log.Printf("❌ GenerateCatalog: Error generating PDF: %v", err)
			http.Error(w, fmt.Sprintf("Failed to generate PDF: %v", err), http.StatusInternalServerError)
			return
		}

		// Set headers and return PDF
		filename := fmt.Sprintf("catalog_%s.pdf", normalizedSize)
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(pdfData); err != nil {
			log.Printf("❌ GenerateCatalog: Error writing PDF response: %v", err)
		}

	case "png":
		// Generate PNG using render endpoint
		pngs, err := c.catalogService.GeneratePNG(ctx, normalizedSize)
		if err != nil {
			log.Printf("❌ GenerateCatalog: Error generating PNG: %v", err)
			http.Error(w, fmt.Sprintf("Failed to generate PNG: %v", err), http.StatusInternalServerError)
			return
		}

		// Generate a unique session ID
		sessionID := fmt.Sprintf("%s_%d", normalizedSize, time.Now().UnixNano())
		
		// Store PNGs temporarily
		c.pngStorageMutex.Lock()
		c.pngStorage[sessionID] = pngs
		c.pngStorageMutex.Unlock()
		
		// Schedule cleanup after 10 minutes
		go func() {
			time.Sleep(10 * time.Minute)
			c.pngStorageMutex.Lock()
			delete(c.pngStorage, sessionID)
			c.pngStorageMutex.Unlock()
		}()
		
		// Generate download links for each page
		type PageLink struct {
			Page     int    `json:"page"`
			URL      string `json:"url"`
			Filename string `json:"filename"`
		}
		
		var pages []PageLink
		for i := 1; i <= len(pngs); i++ {
			if _, exists := pngs[i]; exists {
				// Only return the path, not the full URL
				downloadPath := fmt.Sprintf("/admin/catalog/png-page?session=%s&page=%d", sessionID, i)
				// For single page, use simpler filename without page number
				var filename string
				if len(pngs) == 1 {
					filename = fmt.Sprintf("catalog_%s.png", normalizedSize)
				} else {
					filename = fmt.Sprintf("catalog_%s_page_%d.png", normalizedSize, i)
				}
				pages = append(pages, PageLink{
					Page:     i,
					URL:      downloadPath,
					Filename: filename,
				})
			}
		}
		
		response := map[string]interface{}{
			"sessionId": sessionID,
			"totalPages": len(pngs),
			"size": normalizedSize,
			"pages": pages,
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("❌ GenerateCatalog: Error encoding JSON response: %v", err)
		}
	}
}

// RenderCatalog handles GET /admin/catalog/render?size=XS
// Returns the HTML template for the catalog (used by chromedp for PDF/PNG generation)
func (c *CatalogController) RenderCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Printf("❌ RenderCatalog: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()

	// Parse query parameters
	size := strings.TrimSpace(r.URL.Query().Get("size"))

	// Validate size parameter
	if size == "" {
		log.Printf("❌ RenderCatalog: size parameter is required")
		http.Error(w, "size parameter is required", http.StatusBadRequest)
		return
	}

	// Normalize size
	normalizedSize := utils.NormalizeSize(size)
	if !validSizes[normalizedSize] {
		log.Printf("❌ RenderCatalog: Invalid size: %s", size)
		http.Error(w, fmt.Sprintf("Invalid size. Valid sizes: XS, S, M, L, XL, MN (Mini), IT (Intermedio)"), http.StatusBadRequest)
		return
	}

	// Get items from repository
	items, err := c.repository.GetItemsBySizeForCatalog(ctx, normalizedSize)
	if err != nil {
		log.Printf("❌ RenderCatalog: Error fetching items: %v", err)
		http.Error(w, fmt.Sprintf("Failed to fetch items: %v", err), http.StatusInternalServerError)
		return
	}

	// Check if there are any items
	if len(items) == 0 {
		log.Printf("⚠️  RenderCatalog: No items found for size=%s", normalizedSize)
		http.Error(w, fmt.Sprintf("No active items found for size %s", normalizedSize), http.StatusNotFound)
		return
	}

	// Render HTML with absolute URLs (no base64)
	htmlContent, err := c.catalogService.RenderCatalogHTML(ctx, normalizedSize, items, false)
	if err != nil {
		log.Printf("❌ RenderCatalog: Error rendering HTML: %v", err)
		http.Error(w, fmt.Sprintf("Failed to render catalog: %v", err), http.StatusInternalServerError)
		return
	}

	// Return HTML directly
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(htmlContent)); err != nil {
		log.Printf("❌ RenderCatalog: Error writing HTML response: %v", err)
	}
}

// DownloadPNGPage handles GET /admin/catalog/png-page?session=XXX&page=N
// Returns a specific PNG page from temporary storage
func (c *CatalogController) DownloadPNGPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Printf("❌ DownloadPNGPage: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := strings.TrimSpace(r.URL.Query().Get("session"))
	pageStr := strings.TrimSpace(r.URL.Query().Get("page"))

	if sessionID == "" {
		log.Printf("❌ DownloadPNGPage: session parameter is required")
		http.Error(w, "session parameter is required", http.StatusBadRequest)
		return
	}

	pageNum, err := strconv.Atoi(pageStr)
	if err != nil || pageNum < 1 {
		log.Printf("❌ DownloadPNGPage: Invalid page number: %s", pageStr)
		http.Error(w, "Invalid page number", http.StatusBadRequest)
		return
	}

	// Retrieve PNG from temporary storage
	c.pngStorageMutex.RLock()
	pngs, exists := c.pngStorage[sessionID]
	c.pngStorageMutex.RUnlock()

	if !exists {
		log.Printf("❌ DownloadPNGPage: Session not found: %s", sessionID)
		http.Error(w, "Session expired or not found", http.StatusNotFound)
		return
	}

	pngData, exists := pngs[pageNum]
	if !exists {
		log.Printf("❌ DownloadPNGPage: Page %d not found in session %s", pageNum, sessionID)
		http.Error(w, fmt.Sprintf("Page %d not found", pageNum), http.StatusNotFound)
		return
	}

	// Validate PNG data (PNG files start with PNG signature)
	if len(pngData) < 8 {
		log.Printf("❌ DownloadPNGPage: PNG data too short for page %d (%d bytes)", pageNum, len(pngData))
		http.Error(w, "Invalid PNG data", http.StatusInternalServerError)
		return
	}
	pngSignature := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if len(pngData) < 8 || !equalBytes(pngData[:8], pngSignature) {
		log.Printf("❌ DownloadPNGPage: Invalid PNG signature for page %d (first 8 bytes: %x)", pageNum, pngData[:8])
		http.Error(w, "Invalid PNG data", http.StatusInternalServerError)
		return
	}

	// Extract size from session ID (format: SIZE_TIMESTAMP)
	parts := strings.Split(sessionID, "_")
	size := "L" // Default
	if len(parts) > 0 {
		size = parts[0]
	}

	filename := fmt.Sprintf("catalog_%s_page_%d.png", size, pageNum)
	
	// Set headers for PNG download - IMPORTANT: Set headers BEFORE WriteHeader
	// Use Content-Disposition: attachment to force download instead of opening in browser
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pngData)))
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	
	w.WriteHeader(http.StatusOK)
	
	// Write PNG data directly
	n, err := w.Write(pngData)
	if err != nil {
		log.Printf("❌ DownloadPNGPage: Error writing PNG response: %v", err)
		return
	}
	if n != len(pngData) {
		log.Printf("⚠️ DownloadPNGPage: Partial write: wrote %d of %d bytes", n, len(pngData))
	}
}

// equalBytes compares two byte slices
func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// getPageNumbers returns a slice of page numbers from a PNG map
func getPageNumbers(pngs map[int][]byte) []int {
	pages := make([]int, 0, len(pngs))
	for pageNum := range pngs {
		pages = append(pages, pageNum)
	}
	return pages
}

