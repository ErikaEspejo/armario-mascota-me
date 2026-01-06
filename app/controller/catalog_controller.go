package controller

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

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

	log.Printf("📋 GenerateCatalog: size=%s (normalized=%s), format=%s", size, normalizedSize, format)

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

	log.Printf("✓ GenerateCatalog: Found %d items for size=%s", len(items), normalizedSize)
	
	// Log pagination info
	pagesCount := (len(items) + 8) / 9 // Ceiling division
	log.Printf("📄 GenerateCatalog: Will generate %d pages (9 items per page)", pagesCount)

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

		if len(pngs) == 1 {
			// Single page: return PNG directly
			filename := fmt.Sprintf("catalog_%s.png", normalizedSize)
			w.Header().Set("Content-Type", "image/png")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write(pngs[1]); err != nil {
				log.Printf("❌ GenerateCatalog: Error writing PNG response: %v", err)
			}
		} else {
			// Multiple pages: return ZIP
			var zipBuf bytes.Buffer
			zipWriter := zip.NewWriter(&zipBuf)

			for pageNum, pngData := range pngs {
				filename := fmt.Sprintf("catalog_%s_page_%d.png", normalizedSize, pageNum)
				fileWriter, err := zipWriter.Create(filename)
				if err != nil {
					log.Printf("⚠️  Warning: Failed to create zip entry for page %d: %v", pageNum, err)
					continue
				}
				if _, err := fileWriter.Write(pngData); err != nil {
					log.Printf("⚠️  Warning: Failed to write zip entry for page %d: %v", pageNum, err)
					continue
				}
			}

			if err := zipWriter.Close(); err != nil {
				log.Printf("❌ GenerateCatalog: Error closing zip writer: %v", err)
				http.Error(w, "Failed to create ZIP file", http.StatusInternalServerError)
				return
			}

			filename := fmt.Sprintf("catalog_%s.zip", normalizedSize)
			w.Header().Set("Content-Type", "application/zip")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write(zipBuf.Bytes()); err != nil {
				log.Printf("❌ GenerateCatalog: Error writing ZIP response: %v", err)
			}
		}
	}

	log.Printf("✅ GenerateCatalog: Successfully generated %s catalog for size=%s", format, normalizedSize)
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

	log.Printf("📋 RenderCatalog: size=%s (normalized=%s)", size, normalizedSize)

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

	log.Printf("✓ RenderCatalog: Found %d items for size=%s", len(items), normalizedSize)

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

