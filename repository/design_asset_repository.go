package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"armario-mascota-me/db"
	"armario-mascota-me/models"
)

// DesignAssetRepository handles database operations for design assets
// Implements DesignAssetRepositoryInterface
type DesignAssetRepository struct{}

// NewDesignAssetRepository creates a new DesignAssetRepository
func NewDesignAssetRepository() *DesignAssetRepository {
	return &DesignAssetRepository{}
}

// Ensure DesignAssetRepository implements DesignAssetRepositoryInterface
var _ DesignAssetRepositoryInterface = (*DesignAssetRepository)(nil)

// ExistsByDriveFileID checks if a design asset exists by drive_file_id
func (r *DesignAssetRepository) ExistsByDriveFileID(ctx context.Context, driveFileID string) (bool, error) {
	log.Printf("🔍 Checking if drive_file_id exists in database: %s", driveFileID)

	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM design_assets WHERE drive_file_id = $1)`
	err := db.DB.QueryRowContext(ctx, query, driveFileID).Scan(&exists)
	if err != nil {
		log.Printf("❌ Error checking existence for drive_file_id %s: %v", driveFileID, err)
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	log.Printf("🔍 Existence check result for drive_file_id %s: exists=%v", driveFileID, exists)
	return exists, nil
}

// GetMaxDecoID returns the maximum deco_id value in the database
// deco_id is stored as text, so we need to cast it to integer for MAX comparison
func (r *DesignAssetRepository) GetMaxDecoID(ctx context.Context) (int, error) {
	var maxDecoID sql.NullInt64
	// Cast deco_id to integer for MAX comparison, then convert back
	query := `SELECT MAX(CAST(deco_id AS INTEGER)) FROM design_assets WHERE deco_id IS NOT NULL AND deco_id ~ '^[0-9]+$'`
	
	err := db.DB.QueryRowContext(ctx, query).Scan(&maxDecoID)
	if err != nil {
		return 0, fmt.Errorf("failed to get max deco_id: %w", err)
	}

	if !maxDecoID.Valid {
		// No records exist, start from 1
		return 0, nil
	}

	return int(maxDecoID.Int64), nil
}

// Insert inserts a new design asset into the database
// Only inserts drive_file_id, image_url, and deco_id (ascending number), other fields will be set from the frontend
func (r *DesignAssetRepository) Insert(ctx context.Context, asset *models.DesignAssetDB) error {
	log.Printf("💾 Repository.Insert called for drive_file_id: %s", asset.DriveFileID)

	// Get the next deco_id (max + 1)
	maxDecoID, err := r.GetMaxDecoID(ctx)
	if err != nil {
		log.Printf("❌ Error getting max deco_id: %v", err)
		return fmt.Errorf("failed to get max deco_id: %w", err)
	}

	nextDecoID := maxDecoID + 1
	nextDecoIDStr := fmt.Sprintf("%d", nextDecoID)
	log.Printf("🔢 Next deco_id will be: %s", nextDecoIDStr)

	query := `
		INSERT INTO design_assets (
			code, drive_file_id, image_url, deco_id, status, created_at, is_active
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (drive_file_id) DO NOTHING
	`

	log.Printf("💾 Executing INSERT query for drive_file_id: %s", asset.DriveFileID)

	// Use drive_file_id as code (since we're not parsing filename anymore)
	code := asset.DriveFileID

	// Use current time for created_at
	createdAt := time.Now()

	// Status is always 'pending' when loading images
	status := "pending"

	result, err := db.DB.ExecContext(ctx, query,
		code,                    // Use drive_file_id as code
		asset.DriveFileID,
		asset.ImageURL,
		nextDecoIDStr, // Convert to string since deco_id is text in database
		status,        // Always 'pending' when loading images
		createdAt,
		true, // is_active defaults to true
	)

	if err != nil {
		log.Printf("❌ Database INSERT error for drive_file_id %s: %v", asset.DriveFileID, err)
		return fmt.Errorf("failed to insert design asset: %w", err)
	}

	log.Printf("💾 INSERT query executed successfully for drive_file_id: %s", asset.DriveFileID)

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("⚠️  Warning: Could not get rows affected: %v", err)
	}

	if rowsAffected > 0 {
		log.Printf("💾 Database: Successfully inserted design asset (drive_file_id: %s, deco_id: %s)", asset.DriveFileID, nextDecoIDStr)
	} else {
		log.Printf("⚠️  Database: No rows inserted (likely due to ON CONFLICT) for drive_file_id: %s", asset.DriveFileID)
	}

	return nil
}

// GetByCode retrieves a design asset by its code
func (r *DesignAssetRepository) GetByCode(ctx context.Context, code string) (*models.DesignAssetDetail, error) {
	log.Printf("🔍 Fetching design asset by code: %s", code)

	query := `
		SELECT id, code, 
		       COALESCE(description, '') as description, 
		       drive_file_id, 
		       image_url,
		       COALESCE(color_primary, '') as color_primary, 
		       COALESCE(color_secondary, '') as color_secondary, 
		       COALESCE(hoodie_type, '') as hoodie_type, 
		       COALESCE(image_type, '') as image_type,
		       COALESCE(deco_id, '') as deco_id, 
		       COALESCE(deco_base, '') as deco_base, 
		       is_active, 
		       has_highlights
		FROM design_assets
		WHERE code = $1
	`

	var asset models.DesignAssetDetail
	err := db.DB.QueryRowContext(ctx, query, code).Scan(
		&asset.ID,
		&asset.Code,
		&asset.Description,
		&asset.DriveFileID,
		&asset.ImageURL,
		&asset.ColorPrimary,
		&asset.ColorSecondary,
		&asset.HoodieType,
		&asset.ImageType,
		&asset.DecoID,
		&asset.DecoBase,
		&asset.IsActive,
		&asset.HasHighlights,
	)

	if err != nil {
		log.Printf("❌ Error fetching design asset by code %s: %v", code, err)
		return nil, fmt.Errorf("failed to get design asset: %w", err)
	}

	log.Printf("✓ Successfully fetched design asset: %s", code)
	return &asset, nil
}

// UpdateDescriptionAndHighlights updates the description and has_highlights fields of a design asset
func (r *DesignAssetRepository) UpdateDescriptionAndHighlights(ctx context.Context, code string, description string, hasHighlights bool) error {
	log.Printf("🔄 Updating design asset: code=%s, description=%s, hasHighlights=%v", code, description, hasHighlights)

	query := `
		UPDATE design_assets
		SET description = $1, has_highlights = $2
		WHERE code = $3
	`

	result, err := db.DB.ExecContext(ctx, query, description, hasHighlights, code)
	if err != nil {
		log.Printf("❌ Error updating design asset %s: %v", code, err)
		return fmt.Errorf("failed to update design asset: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("⚠️  Warning: Could not get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		log.Printf("⚠️  No rows updated for code: %s (record may not exist)", code)
		return fmt.Errorf("design asset with code %s not found", code)
	}

	log.Printf("✅ Successfully updated design asset: code=%s (rows affected: %d)", code, rowsAffected)
	return nil
}

// GetPending retrieves all design assets with status = 'pending'
func (r *DesignAssetRepository) GetPending(ctx context.Context) ([]models.DesignAssetDetail, error) {
	log.Printf("🔍 Fetching all design assets with status = 'pending'")

	query := `
		SELECT id, code, 
		       COALESCE(description, '') as description, 
		       drive_file_id, 
		       image_url,
		       COALESCE(color_primary, '') as color_primary, 
		       COALESCE(color_secondary, '') as color_secondary, 
		       COALESCE(hoodie_type, '') as hoodie_type, 
		       COALESCE(image_type, '') as image_type,
		       COALESCE(deco_id, '') as deco_id, 
		       COALESCE(deco_base, '') as deco_base, 
		       is_active, 
		       has_highlights
		FROM design_assets
		WHERE status = 'pending'
		ORDER BY created_at ASC
	`

	rows, err := db.DB.QueryContext(ctx, query)
	if err != nil {
		log.Printf("❌ Error fetching pending design assets: %v", err)
		return nil, fmt.Errorf("failed to get pending design assets: %w", err)
	}
	defer rows.Close()

	var assets []models.DesignAssetDetail
	for rows.Next() {
		var asset models.DesignAssetDetail
		err := rows.Scan(
			&asset.ID,
			&asset.Code,
			&asset.Description,
			&asset.DriveFileID,
			&asset.ImageURL,
			&asset.ColorPrimary,
			&asset.ColorSecondary,
			&asset.HoodieType,
			&asset.ImageType,
			&asset.DecoID,
			&asset.DecoBase,
			&asset.IsActive,
			&asset.HasHighlights,
		)
		if err != nil {
			log.Printf("❌ Error scanning pending design asset: %v", err)
			continue
		}
		assets = append(assets, asset)
	}

	if err := rows.Err(); err != nil {
		log.Printf("❌ Error iterating pending design assets: %v", err)
		return nil, fmt.Errorf("failed to iterate pending design assets: %w", err)
	}

	log.Printf("✓ Successfully fetched %d pending design assets", len(assets))
	return assets, nil
}

// GetByID retrieves a design asset by its ID
func (r *DesignAssetRepository) GetByID(ctx context.Context, id int) (*models.DesignAssetDetail, error) {
	log.Printf("🔍 Fetching design asset by ID: %d", id)

	query := `
		SELECT id, code, 
		       COALESCE(description, '') as description, 
		       drive_file_id, 
		       image_url,
		       COALESCE(color_primary, '') as color_primary, 
		       COALESCE(color_secondary, '') as color_secondary, 
		       COALESCE(hoodie_type, '') as hoodie_type, 
		       COALESCE(image_type, '') as image_type,
		       COALESCE(deco_id, '') as deco_id, 
		       COALESCE(deco_base, '') as deco_base, 
		       is_active, 
		       has_highlights
		FROM design_assets
		WHERE id = $1
	`

	var asset models.DesignAssetDetail
	err := db.DB.QueryRowContext(ctx, query, id).Scan(
		&asset.ID,
		&asset.Code,
		&asset.Description,
		&asset.DriveFileID,
		&asset.ImageURL,
		&asset.ColorPrimary,
		&asset.ColorSecondary,
		&asset.HoodieType,
		&asset.ImageType,
		&asset.DecoID,
		&asset.DecoBase,
		&asset.IsActive,
		&asset.HasHighlights,
	)

	if err != nil {
		log.Printf("❌ Error fetching design asset by ID %d: %v", id, err)
		return nil, fmt.Errorf("failed to get design asset: %w", err)
	}

	log.Printf("✓ Successfully fetched design asset: ID=%d", id)
	return &asset, nil
}
