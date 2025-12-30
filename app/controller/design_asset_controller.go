package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"armario-mascota-me/models"
	"armario-mascota-me/repository"
	"armario-mascota-me/service"
)

const folderID = "1TtK0fnadxl3r1-8iYlv2GFf5LgdKxmID"

// DesignAssetController handles HTTP requests for design assets
type DesignAssetController struct {
	syncService service.SyncServiceInterface
	repository  repository.DesignAssetRepositoryInterface
	driveService service.DriveServiceInterface
}

// NewDesignAssetController creates a new DesignAssetController
func NewDesignAssetController(syncService service.SyncServiceInterface, repo repository.DesignAssetRepositoryInterface, driveService service.DriveServiceInterface) *DesignAssetController {
	return &DesignAssetController{
		syncService: syncService,
		repository:  repo,
		driveService: driveService,
	}
}

// LoadImages handles GET /admin/design-assets/load
// This endpoint fetches images from Google Drive, syncs them to the database, and returns them
func (c *DesignAssetController) LoadImages(w http.ResponseWriter, r *http.Request) {
	// Only allow GET method
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Execute synchronization (fetches from Drive and syncs to DB)
	ctx := context.Background()
	designAssets, err := c.syncService.SyncDesignAssets(ctx, folderID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load and sync design assets: %v", err), http.StatusInternalServerError)
		return
	}

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Encode and send JSON response with the design assets
	if err := json.NewEncoder(w).Encode(designAssets); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetDesignAssetByCode handles GET /admin/design-assets/:code
// Returns a design asset with all details including image for editing
func (c *DesignAssetController) GetDesignAssetByCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract code from URL path
	// Path format: /admin/design-assets/{code}
	path := strings.TrimPrefix(r.URL.Path, "/admin/design-assets/")
	if path == "" || path == "load" || path == "pending" {
		http.Error(w, "code parameter is required", http.StatusBadRequest)
		return
	}

	code := path
	ctx := context.Background()

	// Get design asset from database
	asset, err := c.repository.GetByCode(ctx, code)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get design asset: %v", err), http.StatusNotFound)
		return
	}

	// Set content type and return JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(asset); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// UpdateDesignAsset handles PUT /admin/design-assets/:code
// Updates description and has_highlights fields
func (c *DesignAssetController) UpdateDesignAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract code from URL path
	path := strings.TrimPrefix(r.URL.Path, "/admin/design-assets/")
	if path == "" || path == "load" || path == "pending" {
		http.Error(w, "code parameter is required", http.StatusBadRequest)
		return
	}

	code := path

	// Parse request body
	var updateReq models.DesignAssetUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	// Update design asset
	if err := c.repository.UpdateDescriptionAndHighlights(ctx, code, updateReq.Description, updateReq.HasHighlights); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update design asset: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Design asset updated successfully",
		"code":    code,
	})
}

// GetPendingDesignAssets handles GET /admin/design-assets/pending
// Returns all design assets with status = 'pending' (metadata only, no image processing)
func (c *DesignAssetController) GetPendingDesignAssets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()

	// Get pending design assets from database
	assets, err := c.repository.GetPending(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get pending design assets: %v", err), http.StatusInternalServerError)
		return
	}

	// Build response with optimized image URLs (lazy processing - URLs only, no actual processing)
	response := make([]models.DesignAssetDetailWithOptimizedURL, len(assets))
	for i, asset := range assets {
		// Construct URL to optimized image endpoint
		optimizedURL := fmt.Sprintf("/admin/design-assets/pending/%d/image?size=thumb", asset.ID)
		response[i] = models.DesignAssetDetailWithOptimizedURL{
			DesignAssetDetail:  asset,
			OptimizedImageUrl: optimizedURL,
		}
	}

	// Set content type and return JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetOptimizedImage handles GET /admin/design-assets/pending/:id/image?size=thumb|medium
// Returns optimized image with lazy processing and cache
func (c *DesignAssetController) GetOptimizedImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL path
	// Path format: /admin/design-assets/pending/{id}/image
	path := strings.TrimPrefix(r.URL.Path, "/admin/design-assets/pending/")
	if path == "" {
		http.Error(w, "id parameter is required", http.StatusBadRequest)
		return
	}

	// Extract ID from path (remove /image suffix)
	idStr := strings.TrimSuffix(path, "/image")
	if idStr == path {
		http.Error(w, "invalid path format", http.StatusBadRequest)
		return
	}

	var id int
	var err error
	if id, err = strconv.Atoi(idStr); err != nil {
		http.Error(w, "invalid id parameter", http.StatusBadRequest)
		return
	}

	// Get size parameter (default: medium)
	size := r.URL.Query().Get("size")
	if size == "" {
		size = "medium"
	}
	if size != "thumb" && size != "medium" {
		size = "medium"
	}

	ctx := context.Background()

	// Get design asset from database
	asset, err := c.repository.GetByID(ctx, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get design asset: %v", err), http.StatusNotFound)
		return
	}

	// Ensure cache directory exists
	if err := service.EnsureCacheDir(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to ensure cache directory: %v", err), http.StatusInternalServerError)
		return
	}

	// Get cache path
	cachePath := service.GetCachePath(id, size)

	// Check if cached image exists
	var imageData []byte
	if service.CacheExists(cachePath) {
		// Read from cache
		imageData, err = service.ReadFromCache(cachePath)
		if err != nil {
			log.Printf("⚠️  Error reading from cache, will reprocess: %v", err)
			// Fall through to processing
			imageData = nil
		}
	}

	// If not in cache or failed to read, process the image
	if imageData == nil {
		// Download image from Drive
		originalData, err := c.driveService.DownloadImage(asset.DriveFileID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to download image from Drive: %v", err), http.StatusInternalServerError)
			return
		}

		// Optimize image
		imageData, err = service.OptimizeImage(originalData, size)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to optimize image: %v", err), http.StatusInternalServerError)
			return
		}

		// Save to cache
		if err := service.SaveToCache(cachePath, imageData); err != nil {
			log.Printf("⚠️  Warning: Failed to save to cache: %v", err)
			// Continue anyway, we still have the image data
		}
	}

	// Return image
	w.Header().Set("Content-Type", "image/jpeg")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(imageData); err != nil {
		log.Printf("❌ Error writing image response: %v", err)
	}
}
