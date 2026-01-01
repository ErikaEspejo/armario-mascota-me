package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"armario-mascota-me/db"
	"armario-mascota-me/models"
	"armario-mascota-me/utils"
)

// ItemFilterParams represents optional filter parameters for items
type ItemFilterParams struct {
	Size           *string
	ColorPrimary   *string
	ColorSecondary *string
	HoodieType     *string
}

// NormalizeSize normalizes size values: Mini -> MN, Intermedio -> IT
func NormalizeSize(size string) string {
	sizeUpper := strings.ToUpper(strings.TrimSpace(size))
	
	// Normalize Mini variations to MN
	if sizeUpper == "MINI" || sizeUpper == "MN" {
		return "MN"
	}
	
	// Normalize Intermedio variations to IT
	if sizeUpper == "INTERMEDIO" || sizeUpper == "IT" {
		return "IT"
	}
	
	// Return original size (already trimmed and uppercase)
	return sizeUpper
}

// ItemRepository handles database operations for items
type ItemRepository struct{}

// NewItemRepository creates a new ItemRepository
func NewItemRepository() *ItemRepository {
	return &ItemRepository{}
}

// Ensure ItemRepository implements ItemRepositoryInterface
var _ ItemRepositoryInterface = (*ItemRepository)(nil)

// UpsertStock adds stock to an item, creating it if it doesn't exist
func (r *ItemRepository) UpsertStock(ctx context.Context, designAssetID int, size string, quantity int) (*models.AddStockResponse, error) {
	log.Printf("📦 UpsertStock: design_asset_id=%d, size=%s, quantity=%d", designAssetID, size, quantity)

	// Start transaction
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("❌ Error starting transaction: %v", err)
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// First, verify that design_asset exists and get code and hoodie_type
	var code string
	var hoodieType string
	queryDesignAsset := `
		SELECT code, COALESCE(hoodie_type, '') as hoodie_type
		FROM design_assets
		WHERE id = $1
	`
	err = tx.QueryRowContext(ctx, queryDesignAsset, designAssetID).Scan(&code, &hoodieType)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("❌ Design asset with id %d does not exist", designAssetID)
			return nil, fmt.Errorf("design asset with id %d does not exist", designAssetID)
		}
		log.Printf("❌ Error fetching design asset: %v", err)
		return nil, fmt.Errorf("failed to get design asset: %w", err)
	}

	log.Printf("✓ Design asset found: code=%s, hoodie_type=%s", code, hoodieType)

	// Normalize size: Mini -> MN, Intermedio -> IT
	sizeNormalized := NormalizeSize(size)
	log.Printf("📏 Size normalized: %s -> %s", size, sizeNormalized)

	// Calculate price based on hoodie_type (buso_type) and size
	price := utils.CalculatePrice(hoodieType, sizeNormalized)
	log.Printf("💰 Calculated price: %d cents for hoodie_type=%s, size=%s", price, hoodieType, sizeNormalized)

	// Generate SKU: size + "_" + code (using normalized size)
	sku := fmt.Sprintf("%s_%s", sizeNormalized, code)
	log.Printf("🏷️  Generated SKU: %s", sku)

	// Insert or update item using ON CONFLICT
	// If item exists, only update stock_total (add quantity)
	// If item doesn't exist, create it with stock_total = quantity
	queryUpsert := `
		INSERT INTO items (design_asset_id, size, sku, price, stock_total, stock_reserved, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, 0, true, NOW())
		ON CONFLICT (design_asset_id, size) 
		DO UPDATE SET 
			stock_total = items.stock_total + EXCLUDED.stock_total
		RETURNING id, sku, size, price, stock_total, stock_reserved
	`

	var response models.AddStockResponse
	err = tx.QueryRowContext(ctx, queryUpsert, designAssetID, sizeNormalized, sku, price, quantity).Scan(
		&response.ID,
		&response.SKU,
		&response.Size,
		&response.Price,
		&response.StockTotal,
		&response.StockReserved,
	)
	if err != nil {
		log.Printf("❌ Error upserting item: %v", err)
		return nil, fmt.Errorf("failed to upsert item: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("❌ Error committing transaction: %v", err)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("✓ Successfully upserted item: id=%d, sku=%s, stock_total=%d", response.ID, response.SKU, response.StockTotal)
	return &response, nil
}

// FilterItems retrieves items matching the provided filters
// Always filters by items.is_active=true, design_assets.is_active=true, and design_assets.status='ready'
func (r *ItemRepository) FilterItems(ctx context.Context, filters ItemFilterParams) ([]models.ItemCard, error) {
	log.Printf("🔍 Filtering items with filters: size=%v, colorPrimary=%v, colorSecondary=%v, hoodieType=%v",
		filters.Size, filters.ColorPrimary, filters.ColorSecondary, filters.HoodieType)

	// Build base query with JOIN
	baseQuery := `
		SELECT i.id, i.sku, i.size, i.price, i.stock_total, i.stock_reserved, i.design_asset_id,
		       COALESCE(da.description, '') as description
		FROM items i
		INNER JOIN design_assets da ON i.design_asset_id = da.id
		WHERE i.is_active = true 
		  AND da.is_active = true 
		  AND da.status = 'ready'
	`

	// Build WHERE conditions dynamically
	var conditions []string
	var args []interface{}
	argIndex := 1

	if filters.Size != nil && *filters.Size != "" {
		conditions = append(conditions, fmt.Sprintf("i.size = $%d", argIndex))
		args = append(args, *filters.Size)
		argIndex++
	}

	if filters.ColorPrimary != nil && *filters.ColorPrimary != "" {
		conditions = append(conditions, fmt.Sprintf("da.color_primary = $%d", argIndex))
		args = append(args, *filters.ColorPrimary)
		argIndex++
	}

	if filters.ColorSecondary != nil && *filters.ColorSecondary != "" {
		conditions = append(conditions, fmt.Sprintf("da.color_secondary = $%d", argIndex))
		args = append(args, *filters.ColorSecondary)
		argIndex++
	}

	if filters.HoodieType != nil && *filters.HoodieType != "" {
		conditions = append(conditions, fmt.Sprintf("da.hoodie_type = $%d", argIndex))
		args = append(args, *filters.HoodieType)
		argIndex++
	}

	// Append conditions to query
	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	// Add ORDER BY
	baseQuery += " ORDER BY i.created_at DESC"

	log.Printf("🔍 Executing filter query with %d conditions", len(conditions))

	rows, err := db.DB.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		log.Printf("❌ Error filtering items: %v", err)
		return nil, fmt.Errorf("failed to filter items: %w", err)
	}
	defer rows.Close()

	var items []models.ItemCard
	for rows.Next() {
		var item models.ItemCard
		err := rows.Scan(
			&item.ID,
			&item.SKU,
			&item.Size,
			&item.Price,
			&item.StockTotal,
			&item.StockReserved,
			&item.DesignAssetID,
			&item.Description,
		)
		if err != nil {
			log.Printf("❌ Error scanning filtered item: %v", err)
			continue
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		log.Printf("❌ Error iterating filtered items: %v", err)
		return nil, fmt.Errorf("failed to iterate filtered items: %w", err)
	}

	log.Printf("✓ Successfully filtered %d items", len(items))
	return items, nil
}

