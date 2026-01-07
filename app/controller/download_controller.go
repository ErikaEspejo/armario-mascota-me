package controller

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"armario-mascota-me/service"
)

// DownloadController handles HTTP requests for image downloads
type DownloadController struct {
	downloadService service.DownloadServiceInterface
}

// NewDownloadController creates a new DownloadController
func NewDownloadController(downloadService service.DownloadServiceInterface) *DownloadController {
	return &DownloadController{
		downloadService: downloadService,
	}
}

// DownloadImages handles POST /admin/images/download
// Downloads all images from BASE_GOOGLE_DRIVE_FOLDER_ID, optimizes them, and saves them locally
func (c *DownloadController) DownloadImages(w http.ResponseWriter, r *http.Request) {
	// Only allow POST method
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get folder ID from environment variable
	folderID := os.Getenv("BASE_GOOGLE_DRIVE_FOLDER_ID")
	if folderID == "" {
		http.Error(w, "BASE_GOOGLE_DRIVE_FOLDER_ID environment variable is not set", http.StatusInternalServerError)
		return
	}

	log.Printf("📥 Download request received for folder: %s", folderID)

	// Execute download process
	totalImages, downloaded, errors, err := c.downloadService.DownloadAllImages(folderID)
	if err != nil {
		log.Printf("❌ Download failed: %v", err)
		http.Error(w, fmt.Sprintf("Failed to download images: %v", err), http.StatusInternalServerError)
		return
	}

	// Build response
	response := map[string]interface{}{
		"status":       "success",
		"total_images": totalImages,
		"downloaded":   downloaded,
		"failed":       len(errors),
		"errors":       errors,
	}

	// Set content type and return JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("❌ Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Download request completed: %d/%d images downloaded", downloaded, totalImages)
}

