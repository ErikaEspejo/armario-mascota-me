package controller

import (
	"encoding/json"
	"fmt"
	"net/http"

	"armario-mascota-me/service"
)

const folderID = "1TtK0fnadxl3r1-8iYlv2GFf5LgdKxmID"

// DesignAssetController handles HTTP requests for design assets
type DesignAssetController struct {
	driveService *service.DriveService
}

// NewDesignAssetController creates a new DesignAssetController
func NewDesignAssetController(driveService *service.DriveService) *DesignAssetController {
	return &DesignAssetController{
		driveService: driveService,
	}
}

// SyncDesignAssets handles GET /admin/design-assets/sync
func (c *DesignAssetController) SyncDesignAssets(w http.ResponseWriter, r *http.Request) {
	// Only allow GET method
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Call service to list design assets
	designAssets, err := c.driveService.ListDesignAssets(folderID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list design assets: %v", err), http.StatusInternalServerError)
		return
	}

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Encode and send JSON response
	if err := json.NewEncoder(w).Encode(designAssets); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
