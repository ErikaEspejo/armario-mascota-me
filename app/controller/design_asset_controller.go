package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"armario-mascota-me/models"
	"armario-mascota-me/repository"
	"armario-mascota-me/service"
	"armario-mascota-me/utils"
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

// UpdateFullDesignAsset handles POST /admin/design-assets/update
// Updates all fields of a design asset including code generation
func (c *DesignAssetController) UpdateFullDesignAsset(w http.ResponseWriter, r *http.Request) {
	log.Printf("📥 UpdateFullDesignAsset: Received %s request to %s", r.Method, r.URL.Path)
	
	if r.Method != http.MethodPost {
		log.Printf("❌ UpdateFullDesignAsset: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var updateReq models.DesignAssetFullUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		log.Printf("❌ UpdateFullDesignAsset: Failed to decode request body: %v", err)
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	log.Printf("📋 UpdateFullDesignAsset: Request body decoded - ID: %s, Description: %s, ColorPrimary: %s, ColorSecondary: %s, HoodieType: %s, ImageType: %s, DecoBase: %s, HasHighlights: %v",
		updateReq.ID, updateReq.Description, updateReq.ColorPrimary, updateReq.ColorSecondary, updateReq.HoodieType, updateReq.ImageType, updateReq.DecoBase, updateReq.HasHighlights)

	// Convert ID from string to int
	id, err := strconv.Atoi(updateReq.ID)
	if err != nil {
		log.Printf("❌ UpdateFullDesignAsset: Invalid ID format: %s, error: %v", updateReq.ID, err)
		http.Error(w, fmt.Sprintf("Invalid id format: %v", err), http.StatusBadRequest)
		return
	}

	log.Printf("🔄 UpdateFullDesignAsset: Converting ID %s to int: %d", updateReq.ID, id)

	// Normalize all strings to lowercase
	description := strings.ToLower(strings.TrimSpace(updateReq.Description))
	colorPrimary := strings.ToLower(strings.TrimSpace(updateReq.ColorPrimary))
	colorSecondary := strings.ToLower(strings.TrimSpace(updateReq.ColorSecondary))
	hoodieType := strings.ToLower(strings.TrimSpace(updateReq.HoodieType))
	imageType := strings.ToLower(strings.TrimSpace(updateReq.ImageType))
	decoBase := strings.ToLower(strings.TrimSpace(updateReq.DecoBase))

	log.Printf("🔤 UpdateFullDesignAsset: Normalized values - Description: %s, ColorPrimary: %s, ColorSecondary: %s, HoodieType: %s, ImageType: %s, DecoBase: %s",
		description, colorPrimary, colorSecondary, hoodieType, imageType, decoBase)

	// Map values using utility functions (returns uppercase codes)
	colorPrimaryCode := utils.MapColorToCode(colorPrimary)
	colorSecondaryCode := utils.MapColorToCode(colorSecondary)
	hoodieTypeCode := utils.MapHoodieTypeToCode(hoodieType)
	imageTypeCode := utils.ParseImageTypeSizes(imageType)
	
	// Map decoBase values: N/A -> 0, Círculo -> C, Nube -> N
	decoBaseMapped := decoBase
	if decoBase == "n/a" {
		decoBaseMapped = "0"
	} else if decoBase == "círculo" || decoBase == "circulo" {
		decoBaseMapped = "C"
	} else if decoBase == "nube" {
		decoBaseMapped = "N"
	}
	decoBaseUpper := strings.ToUpper(decoBaseMapped)

	log.Printf("🗺️  UpdateFullDesignAsset: Mapped codes - ColorPrimary: %s -> %s, ColorSecondary: %s -> %s, HoodieType: %s -> %s, ImageType: %s -> %s, DecoBase: %s -> %s",
		colorPrimary, colorPrimaryCode, colorSecondary, colorSecondaryCode, hoodieType, hoodieTypeCode, imageType, imageTypeCode, decoBase, decoBaseUpper)

	// Build code: colorPrimary_colorSecondary-hoodieType-imageType{ID}-decoBase
	code := fmt.Sprintf("%s_%s-%s-%s%d-%s", colorPrimaryCode, colorSecondaryCode, hoodieTypeCode, imageTypeCode, id, decoBaseUpper)

	log.Printf("🏷️  UpdateFullDesignAsset: Generated code: %s", code)

	// Use ID (converted to string) as decoID
	decoID := strconv.Itoa(id)

	// Store values in uppercase for database
	descriptionUpper := strings.ToUpper(description)
	colorPrimaryUpper := colorPrimaryCode
	colorSecondaryUpper := colorSecondaryCode
	hoodieTypeUpper := hoodieTypeCode
	imageTypeUpper := imageTypeCode
	decoBaseUpperDB := decoBaseUpper

	log.Printf("💾 UpdateFullDesignAsset: Preparing to update database - ID: %d, Code: %s, DecoID: %s, Status: ready", id, code, decoID)

	ctx := context.Background()

	// Update design asset with status="ready"
	if err := c.repository.UpdateFullDesignAsset(ctx, id, code, descriptionUpper, colorPrimaryUpper, colorSecondaryUpper, hoodieTypeUpper, imageTypeUpper, decoID, decoBaseUpperDB, updateReq.HasHighlights, "ready"); err != nil {
		log.Printf("❌ UpdateFullDesignAsset: Error updating full design asset: %v", err)
		http.Error(w, fmt.Sprintf("Failed to update design asset: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ UpdateFullDesignAsset: Successfully updated design asset - ID: %d, Code: %s", id, code)

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Design asset updated successfully",
		"id":      id,
		"code":    code,
	})
}

// FilterDesignAssets handles GET /admin/design-assets/filter
// Filters design assets by query parameters: colorPrimary, colorSecondary, hoodieType, imageType, decoBase
func (c *DesignAssetController) FilterDesignAssets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()

	// Parse query parameters
	queryParams := r.URL.Query()
	
	// Helper function to decode and normalize query param
	decodeAndNormalize := func(param string) string {
		if param == "" {
			return ""
		}
		decoded, err := url.QueryUnescape(param)
		if err != nil {
			log.Printf("⚠️  Warning: Failed to decode param %s: %v", param, err)
			decoded = param
		}
		return strings.ToLower(strings.TrimSpace(decoded))
	}

	// Extract and decode query parameters
	colorPrimaryRaw := queryParams.Get("colorPrimary")
	colorSecondaryRaw := queryParams.Get("colorSecondary")
	hoodieTypeRaw := queryParams.Get("hoodieType")
	imageTypeRaw := queryParams.Get("imageType")
	decoBaseRaw := queryParams.Get("decoBase")

	// Build FilterParams with mapped codes
	var filters repository.FilterParams

	// Map colorPrimary
	if colorPrimaryRaw != "" {
		colorPrimaryNormalized := decodeAndNormalize(colorPrimaryRaw)
		colorPrimaryCode := utils.MapColorToCode(colorPrimaryNormalized)
		filters.ColorPrimary = &colorPrimaryCode
		log.Printf("🔍 Filter: colorPrimary=%s -> %s", colorPrimaryRaw, colorPrimaryCode)
	}

	// Map colorSecondary
	if colorSecondaryRaw != "" {
		colorSecondaryNormalized := decodeAndNormalize(colorSecondaryRaw)
		colorSecondaryCode := utils.MapColorToCode(colorSecondaryNormalized)
		filters.ColorSecondary = &colorSecondaryCode
		log.Printf("🔍 Filter: colorSecondary=%s -> %s", colorSecondaryRaw, colorSecondaryCode)
	}

	// Map hoodieType
	if hoodieTypeRaw != "" {
		hoodieTypeNormalized := decodeAndNormalize(hoodieTypeRaw)
		hoodieTypeCode := utils.MapHoodieTypeToCode(hoodieTypeNormalized)
		filters.HoodieType = &hoodieTypeCode
		log.Printf("🔍 Filter: hoodieType=%s -> %s", hoodieTypeRaw, hoodieTypeCode)
	}

	// Map imageType
	if imageTypeRaw != "" {
		imageTypeNormalized := decodeAndNormalize(imageTypeRaw)
		imageTypeCode := utils.MapImageTypeToCode(imageTypeNormalized)
		filters.ImageType = &imageTypeCode
		log.Printf("🔍 Filter: imageType=%s -> %s", imageTypeRaw, imageTypeCode)
	}

	// Map decoBase
	if decoBaseRaw != "" {
		decoBaseNormalized := decodeAndNormalize(decoBaseRaw)
		// Map decoBase values: N/A -> 0, Círculo -> C, Nube -> N
		decoBaseMapped := decoBaseNormalized
		if decoBaseNormalized == "n/a" {
			decoBaseMapped = "0"
		} else if decoBaseNormalized == "círculo" || decoBaseNormalized == "circulo" {
			decoBaseMapped = "C"
		} else if decoBaseNormalized == "nube" {
			decoBaseMapped = "N"
		}
		decoBaseUpper := strings.ToUpper(decoBaseMapped)
		filters.DecoBase = &decoBaseUpper
		log.Printf("🔍 Filter: decoBase=%s -> %s", decoBaseRaw, decoBaseUpper)
	}

	// Get filtered design assets from database
	assets, err := c.repository.FilterDesignAssets(ctx, filters)
	if err != nil {
		log.Printf("❌ Error filtering design assets: %v", err)
		http.Error(w, fmt.Sprintf("Failed to filter design assets: %v", err), http.StatusInternalServerError)
		return
	}

	// Build response with optimized image URLs (similar to GetPendingDesignAssets)
	// Convert codes back to readable values before sending response
	response := make([]models.DesignAssetDetailWithOptimizedURL, len(assets))
	for i, asset := range assets {
		// Convert codes to readable values
		asset.ColorPrimary = utils.MapCodeToColor(asset.ColorPrimary)
		asset.ColorSecondary = utils.MapCodeToColor(asset.ColorSecondary)
		asset.HoodieType = utils.MapCodeToHoodieType(asset.HoodieType)
		// imageType is kept as-is from database (no conversion)
		asset.DecoBase = utils.MapCodeToDecoBase(asset.DecoBase)

		// Construct URL to optimized image endpoint
		optimizedURL := fmt.Sprintf("/admin/design-assets/pending/%d/image?size=thumb", asset.ID)
		response[i] = models.DesignAssetDetailWithOptimizedURL{
			DesignAssetDetail:  asset,
			OptimizedImageUrl: optimizedURL,
		}
	}

	log.Printf("✅ FilterDesignAssets: Returning %d filtered design assets", len(response))

	// Set content type and return JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("❌ Error encoding filter response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
