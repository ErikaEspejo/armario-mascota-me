package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"armario-mascota-me/models"
	"armario-mascota-me/repository"
	"armario-mascota-me/utils"
)

// ItemController handles HTTP requests for items
type ItemController struct {
	repository repository.ItemRepositoryInterface
}

// NewItemController creates a new ItemController
func NewItemController(repo repository.ItemRepositoryInterface) *ItemController {
	return &ItemController{
		repository: repo,
	}
}

// AddStock handles POST /admin/items/stock
// Adds stock to an item, creating it if it doesn't exist
func (c *ItemController) AddStock(w http.ResponseWriter, r *http.Request) {
	log.Printf("📥 AddStock: Received %s request to %s", r.Method, r.URL.Path)

	// Only allow POST method
	if r.Method != http.MethodPost {
		log.Printf("❌ AddStock: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req models.AddStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ AddStock: Failed to decode request body: %v", err)
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	log.Printf("📋 AddStock: Request decoded - design_asset_id=%d, size=%s, quantity=%d", req.DesignAssetID, req.Size, req.Quantity)

	// Validate input
	if req.DesignAssetID <= 0 {
		log.Printf("❌ AddStock: Invalid design_asset_id: %d", req.DesignAssetID)
		http.Error(w, "design_asset_id must be greater than 0", http.StatusBadRequest)
		return
	}

	if req.Quantity <= 0 {
		log.Printf("❌ AddStock: Invalid quantity: %d", req.Quantity)
		http.Error(w, "quantity must be greater than 0", http.StatusBadRequest)
		return
	}

	sizeTrimmed := strings.TrimSpace(req.Size)
	if sizeTrimmed == "" {
		log.Printf("❌ AddStock: size cannot be empty")
		http.Error(w, "size cannot be empty", http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	// Call repository to upsert stock
	response, err := c.repository.UpsertStock(ctx, req.DesignAssetID, sizeTrimmed, req.Quantity)
	if err != nil {
		log.Printf("❌ AddStock: Error upserting stock: %v", err)
		// Check if it's a "not found" error
		if strings.Contains(err.Error(), "does not exist") {
			http.Error(w, fmt.Sprintf("Design asset not found: %v", err), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to add stock: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ AddStock: Successfully added stock - id=%d, sku=%s, stock_total=%d", response.ID, response.SKU, response.StockTotal)

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("❌ AddStock: Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// FilterItems handles GET /admin/items/filter
// Filters items by query parameters: size, primaryColor, secondaryColor, hoodieType
func (c *ItemController) FilterItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Printf("❌ FilterItems: Method not allowed: %s", r.Method)
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
	sizeRaw := queryParams.Get("size")
	primaryColorRaw := queryParams.Get("primaryColor")
	secondaryColorRaw := queryParams.Get("secondaryColor")
	hoodieTypeRaw := queryParams.Get("hoodieType")

	// Build ItemFilterParams with mapped codes
	var filters repository.ItemFilterParams

	// Map size (normalize: Mini -> MN, Intermedio -> IT)
	if sizeRaw != "" {
		sizeNormalized := decodeAndNormalize(sizeRaw)
		sizeCode := repository.NormalizeSize(sizeNormalized)
		filters.Size = &sizeCode
		log.Printf("🔍 Filter: size=%s -> %s", sizeRaw, sizeCode)
	}

	// Map primaryColor
	if primaryColorRaw != "" {
		primaryColorNormalized := decodeAndNormalize(primaryColorRaw)
		primaryColorCode := utils.MapColorToCode(primaryColorNormalized)
		filters.ColorPrimary = &primaryColorCode
		log.Printf("🔍 Filter: primaryColor=%s -> %s", primaryColorRaw, primaryColorCode)
	}

	// Map secondaryColor
	if secondaryColorRaw != "" {
		secondaryColorNormalized := decodeAndNormalize(secondaryColorRaw)
		secondaryColorCode := utils.MapColorToCode(secondaryColorNormalized)
		filters.ColorSecondary = &secondaryColorCode
		log.Printf("🔍 Filter: secondaryColor=%s -> %s", secondaryColorRaw, secondaryColorCode)
	}

	// Map hoodieType
	if hoodieTypeRaw != "" {
		hoodieTypeNormalized := decodeAndNormalize(hoodieTypeRaw)
		hoodieTypeCode := utils.MapHoodieTypeToCode(hoodieTypeNormalized)
		filters.HoodieType = &hoodieTypeCode
		log.Printf("🔍 Filter: hoodieType=%s -> %s", hoodieTypeRaw, hoodieTypeCode)
	}

	// Get filtered items from database
	items, err := c.repository.FilterItems(ctx, filters)
	if err != nil {
		log.Printf("❌ Error filtering items: %v", err)
		http.Error(w, fmt.Sprintf("Failed to filter items: %v", err), http.StatusInternalServerError)
		return
	}

	// Build response with image URLs
	response := make([]models.ItemCard, len(items))
	for i, item := range items {
		// Construct URL to optimized image endpoint
		imageUrl := fmt.Sprintf("/admin/design-assets/pending/%d/image?size=thumb", item.DesignAssetID)
		item.ImageUrl = imageUrl
		response[i] = item
	}

	log.Printf("✅ FilterItems: Returning %d filtered items", len(response))

	// Set content type and return JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("❌ Error encoding filter response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
