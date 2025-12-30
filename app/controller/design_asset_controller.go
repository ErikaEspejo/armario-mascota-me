package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
}

// NewDesignAssetController creates a new DesignAssetController
func NewDesignAssetController(syncService service.SyncServiceInterface, repo repository.DesignAssetRepositoryInterface) *DesignAssetController {
	return &DesignAssetController{
		syncService: syncService,
		repository:  repo,
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
// Returns all design assets with status = 'pending'
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

	// Set content type and return JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(assets); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
