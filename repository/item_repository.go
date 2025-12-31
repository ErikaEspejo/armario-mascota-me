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

// normalizeSize normalizes size values: Mini -> MN, Intermedio -> IT
func normalizeSize(size string) string {
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
	sizeNormalized := normalizeSize(size)
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

