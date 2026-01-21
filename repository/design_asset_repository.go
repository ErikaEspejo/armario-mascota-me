package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
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

// FilterParams represents optional filter parameters for design assets
type FilterParams struct {
	ColorPrimary   *string
	ColorSecondary *string
	HoodieType     *string
	ImageType      *string
	DecoBase       *string
}

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
// If status is empty, defaults to "pending" for backward compatibility
func (r *DesignAssetRepository) Insert(ctx context.Context, asset *models.DesignAssetDB, status string) error {
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

	// Default to "pending" if status is empty (backward compatibility)
	if status == "" {
		status = "pending"
	}

	result, err := db.DB.ExecContext(ctx, query,
		code, // Use drive_file_id as code
		asset.DriveFileID,
		asset.ImageURL,
		nextDecoIDStr, // Convert to string since deco_id is text in database
		status,        // Use provided status or default to "pending"
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

// getByStatus is a generic helper method that retrieves design assets by status
// This method contains the common SQL query logic used by GetPending and GetCustomPending
func (r *DesignAssetRepository) getByStatus(ctx context.Context, status string, limit int) ([]models.DesignAssetDetail, error) {
	log.Printf("🔍 Fetching design assets with status = '%s' (limit: %d)", status, limit)

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
		WHERE status = $1
		ORDER BY created_at ASC
		LIMIT $2
	`

	rows, err := db.DB.QueryContext(ctx, query, status, limit)
	if err != nil {
		log.Printf("❌ Error fetching design assets with status '%s': %v", status, err)
		return nil, fmt.Errorf("failed to get design assets with status '%s': %w", status, err)
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
			log.Printf("❌ Error scanning design asset with status '%s': %v", status, err)
			continue
		}
		assets = append(assets, asset)
	}

	if err := rows.Err(); err != nil {
		log.Printf("❌ Error iterating design assets with status '%s': %v", status, err)
		return nil, fmt.Errorf("failed to iterate design assets with status '%s': %w", status, err)
	}

	log.Printf("✓ Successfully fetched %d design assets with status '%s'", len(assets), status)
	return assets, nil
}

// GetPending retrieves all design assets with status = 'pending' (limited to 10 rows)
func (r *DesignAssetRepository) GetPending(ctx context.Context) ([]models.DesignAssetDetail, error) {
	return r.getByStatus(ctx, "pending", 10)
}

// GetCustomPending retrieves all design assets with status = 'custom-pending' (limited to 10 rows)
func (r *DesignAssetRepository) GetCustomPending(ctx context.Context) ([]models.DesignAssetDetail, error) {
	return r.getByStatus(ctx, "custom-pending", 10)
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

// UpdateFullDesignAsset updates all fields of a design asset by ID
func (r *DesignAssetRepository) UpdateFullDesignAsset(ctx context.Context, id int, code, description, colorPrimary, colorSecondary, hoodieType, imageType, decoID, decoBase string, hasHighlights bool, status string) error {
	log.Printf("🔄 Updating full design asset: id=%d, code=%s, description=%s, colorPrimary=%s, colorSecondary=%s, hoodieType=%s, imageType=%s, decoID=%s, decoBase=%s, hasHighlights=%v, status=%s",
		id, code, description, colorPrimary, colorSecondary, hoodieType, imageType, decoID, decoBase, hasHighlights, status)

	query := `
		UPDATE design_assets
		SET code = $1, 
		    description = $2, 
		    color_primary = $3, 
		    color_secondary = $4, 
		    hoodie_type = $5, 
		    image_type = $6, 
		    deco_id = $7, 
		    deco_base = $8, 
		    has_highlights = $9, 
		    status = $10
		WHERE id = $11
	`

	result, err := db.DB.ExecContext(ctx, query,
		code,
		description,
		colorPrimary,
		colorSecondary,
		hoodieType,
		imageType,
		decoID,
		decoBase,
		hasHighlights,
		status,
		id)
	if err != nil {
		log.Printf("❌ Error updating full design asset %d: %v", id, err)
		return fmt.Errorf("failed to update design asset: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("⚠️  Warning: Could not get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		log.Printf("⚠️  No rows updated for id: %d (record may not exist)", id)
		return fmt.Errorf("design asset with id %d not found", id)
	}

	log.Printf("✅ Successfully updated full design asset: id=%d (rows affected: %d)", id, rowsAffected)
	return nil
}

// FilterDesignAssets retrieves design assets matching the provided filters
// Always filters by status='ready' and is_active=true
func (r *DesignAssetRepository) FilterDesignAssets(ctx context.Context, filters FilterParams) ([]models.DesignAssetDetail, error) {
	log.Printf("🔍 Filtering design assets with filters: colorPrimary=%v, colorSecondary=%v, hoodieType=%v, imageType=%v, decoBase=%v",
		filters.ColorPrimary, filters.ColorSecondary, filters.HoodieType, filters.ImageType, filters.DecoBase)

	// Build base query
	baseQuery := `
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
		WHERE status = 'ready' AND is_active = true
	`

	// Build WHERE conditions dynamically
	var conditions []string
	var args []interface{}
	argIndex := 1

	if filters.ColorPrimary != nil && *filters.ColorPrimary != "" {
		conditions = append(conditions, fmt.Sprintf("color_primary = $%d", argIndex))
		args = append(args, *filters.ColorPrimary)
		argIndex++
	}

	if filters.ColorSecondary != nil && *filters.ColorSecondary != "" {
		conditions = append(conditions, fmt.Sprintf("color_secondary = $%d", argIndex))
		args = append(args, *filters.ColorSecondary)
		argIndex++
	}

	if filters.HoodieType != nil && *filters.HoodieType != "" {
		conditions = append(conditions, fmt.Sprintf("hoodie_type = $%d", argIndex))
		args = append(args, *filters.HoodieType)
		argIndex++
	}

	if filters.ImageType != nil && *filters.ImageType != "" {
		conditions = append(conditions, fmt.Sprintf("image_type = $%d", argIndex))
		args = append(args, *filters.ImageType)
		argIndex++
	}

	if filters.DecoBase != nil && *filters.DecoBase != "" {
		conditions = append(conditions, fmt.Sprintf("deco_base = $%d", argIndex))
		args = append(args, *filters.DecoBase)
		argIndex++
	}

	// Append conditions to query
	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	// Add ORDER BY
	baseQuery += " ORDER BY created_at DESC"

	log.Printf("🔍 Executing filter query with %d conditions", len(conditions))

	rows, err := db.DB.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		log.Printf("❌ Error filtering design assets: %v", err)
		return nil, fmt.Errorf("failed to filter design assets: %w", err)
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
			log.Printf("❌ Error scanning filtered design asset: %v", err)
			continue
		}
		assets = append(assets, asset)
	}

	if err := rows.Err(); err != nil {
		log.Printf("❌ Error iterating filtered design assets: %v", err)
		return nil, fmt.Errorf("failed to iterate filtered design assets: %w", err)
	}

	log.Printf("✓ Successfully filtered %d design assets", len(assets))
	return assets, nil
}
