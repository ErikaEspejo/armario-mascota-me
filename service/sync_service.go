package service

import (
	"context"
	"fmt"
	"log"

	"armario-mascota-me/models"
	"armario-mascota-me/repository"
)

// SyncService handles synchronization between Google Drive and PostgreSQL
// Implements SyncServiceInterface
type SyncService struct {
	driveService DriveServiceInterface
	repository   repository.DesignAssetRepositoryInterface
}

// NewSyncService creates a new SyncService
func NewSyncService(driveService DriveServiceInterface, repo repository.DesignAssetRepositoryInterface) *SyncService {
	return &SyncService{
		driveService: driveService,
		repository:   repo,
	}
}

// Ensure SyncService implements SyncServiceInterface
var _ SyncServiceInterface = (*SyncService)(nil)

// SyncDesignAssets synchronizes design assets from Google Drive to PostgreSQL
// Returns the list of design assets from Google Drive
// Uses "pending" as default status for backward compatibility
func (s *SyncService) SyncDesignAssets(ctx context.Context, folderID string) ([]models.DesignAsset, error) {
	assets, _, _, _, err := s.SyncDesignAssetsWithStats(ctx, folderID, "pending")
	return assets, err
}

// SyncDesignAssetsWithStats synchronizes design assets from Google Drive to PostgreSQL and returns stats.
// inserted = new rows created, skipped = already existed (by drive_file_id), total = total assets seen in Drive.
// status parameter determines the status to set for newly inserted assets (defaults to "pending" if empty)
func (s *SyncService) SyncDesignAssetsWithStats(ctx context.Context, folderID string, status string) (assets []models.DesignAsset, inserted int, skipped int, total int, err error) {
	log.Printf("🔄 Starting synchronization process for folder: %s, status: %s", folderID, status)

	// Default to "pending" if status is empty (backward compatibility)
	if status == "" {
		status = "pending"
	}

	// Get all design assets from Google Drive
	driveAssets, err := s.driveService.ListDesignAssets(folderID)
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("failed to list design assets from Drive: %w", err)
	}

	log.Printf("📦 Processing %d design assets from Google Drive", len(driveAssets))
	total = len(driveAssets)

	// Process each asset
	for _, asset := range driveAssets {
		// Check if asset already exists
		exists, err := s.repository.ExistsByDriveFileID(ctx, asset.DriveFileID)
		if err != nil {
			log.Printf("❌ Error checking existence for drive_file_id: %s: %v", asset.DriveFileID, err)
			continue
		}

		if exists {
			log.Printf("⏭️  Skipping drive_file_id: %s (already exists in database)", asset.DriveFileID)
			skipped++
			continue
		}

		log.Printf("🆕 New file detected (drive_file_id: %s)", asset.DriveFileID)

		// Convert to database model - only drive_file_id and image_url
		dbAsset := &models.DesignAssetDB{
			DriveFileID: asset.DriveFileID,
			ImageURL:    asset.ImageURL,
			// All other fields will be set from the frontend interface
		}

		// Insert into database with the specified status
		log.Printf("💾 Attempting to insert into database (drive_file_id: %s, status: %s)", asset.DriveFileID, status)
		if err := s.repository.Insert(ctx, dbAsset, status); err != nil {
			log.Printf("❌ Error inserting drive_file_id %s into database: %v", asset.DriveFileID, err)
			continue
		}

		log.Printf("✅ Successfully processed (drive_file_id: %s)", asset.DriveFileID)
		inserted++
	}

	log.Printf("🎉 Synchronization completed successfully: %d inserted, %d skipped, %d total processed", inserted, skipped, total)
	return driveAssets, inserted, skipped, total, nil
}
