package repository

import (
	"context"
	"fmt"
	"log"
	"strings"

	"armario-mascota-me/db"
	"armario-mascota-me/models"
	"armario-mascota-me/utils"
)

// capitalizeWords capitalizes the first letter of each word
func capitalizeWords(s string) string {
	if s == "" {
		return s
	}
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

// CatalogRepository handles database operations for catalog generation
type CatalogRepository struct{}

// NewCatalogRepository creates a new CatalogRepository
func NewCatalogRepository() *CatalogRepository {
	return &CatalogRepository{}
}

// Ensure CatalogRepository implements CatalogRepositoryInterface
var _ CatalogRepositoryInterface = (*CatalogRepository)(nil)

// GetItemsBySizeForCatalog retrieves all active items for a specific size with design asset information
func (r *CatalogRepository) GetItemsBySizeForCatalog(ctx context.Context, size string) ([]models.CatalogItem, error) {
	log.Printf("🔍 GetItemsBySizeForCatalog: Fetching items for size=%s", size)

	// Normalize size
	normalizedSize := utils.NormalizeSize(size)
	log.Printf("📏 Size normalized: %s -> %s", size, normalizedSize)

	query := `
		SELECT 
			i.id, 
			i.stock_total, 
			i.stock_reserved,
			i.sku,
			da.id as design_asset_id, 
			da.code, 
			COALESCE(da.deco_id, '') as deco_id, 
			COALESCE(da.color_primary, '') as color_primary, 
			COALESCE(da.color_secondary, '') as color_secondary, 
			COALESCE(da.hoodie_type, '') as hoodie_type,
			da.drive_file_id
		FROM items i
		INNER JOIN design_assets da ON i.design_asset_id = da.id
		WHERE i.size = $1 
		  AND i.is_active = true
		  AND da.is_active = true
		  AND da.status IN ('ready', 'custom-ready')
		  AND (i.stock_total - i.stock_reserved) > 0
		ORDER BY da.code ASC
	`

	rows, err := db.DB.QueryContext(ctx, query, normalizedSize)
	if err != nil {
		log.Printf("❌ Error querying items for catalog: %v", err)
		return nil, fmt.Errorf("failed to query items: %w", err)
	}
	defer rows.Close()

	var items []models.CatalogItem
	for rows.Next() {
		var item models.CatalogItem
		var stockTotal, stockReserved int
		var sku, code, decoID, colorPrimary, colorSecondary, hoodieType, driveFileID string

		err := rows.Scan(
			&item.ID,
			&stockTotal,
			&stockReserved,
			&sku,
			&item.DesignAssetID,
			&code,
			&decoID,
			&colorPrimary,
			&colorSecondary,
			&hoodieType,
			&driveFileID,
		)
		if err != nil {
			log.Printf("❌ Error scanning catalog item: %v", err)
			continue
		}

		// Calculate available quantity
		availableQty := stockTotal - stockReserved
		if availableQty < 0 {
			availableQty = 0
		}

		// Map color primary code to readable name and capitalize each word
		colorPrimaryName := utils.MapCodeToColor(colorPrimary)
		item.ColorPrimaryName = capitalizeWords(colorPrimaryName)

		// Map hoodie type code to readable name and capitalize each word
		hoodieTypeName := utils.MapCodeToHoodieType(hoodieType)
		item.HoodieTypeName = capitalizeWords(hoodieTypeName)

		// Set SKU in uppercase
		item.SKU = strings.ToUpper(sku)

		// Set fields
		item.Code = code
		item.ColorPrimary = colorPrimary
		item.ColorSecondary = colorSecondary
		item.HoodieType = hoodieType
		item.AvailableQty = availableQty

		// Construct image URL (will be converted to base64 in service if needed)
		item.ImageURL = fmt.Sprintf("/admin/design-assets/pending/%d/image?size=medium", item.DesignAssetID)

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		log.Printf("❌ Error iterating catalog items: %v", err)
		return nil, fmt.Errorf("failed to iterate items: %w", err)
	}

	log.Printf("✓ Successfully fetched %d items for catalog (size=%s)", len(items), normalizedSize)
	return items, nil
}

