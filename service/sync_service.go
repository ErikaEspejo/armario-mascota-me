package service

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"

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
func (s *SyncService) SyncDesignAssets(ctx context.Context, folderID string) ([]models.DesignAsset, error) {
	log.Printf("🔄 Starting synchronization process for folder: %s", folderID)

	// Get all design assets from Google Drive
	driveAssets, err := s.driveService.ListDesignAssets(folderID)
	if err != nil {
		return nil, fmt.Errorf("failed to list design assets from Drive: %w", err)
	}

	log.Printf("📦 Processing %d design assets from Google Drive", len(driveAssets))

	inserted := 0
	skipped := 0

	// Process each asset
	for _, asset := range driveAssets {
		// Check if asset already exists
		exists, err := s.repository.ExistsByDriveFileID(ctx, asset.DriveFileID)
		if err != nil {
			log.Printf("❌ Error checking existence for %s (drive_file_id: %s): %v", asset.FileName, asset.DriveFileID, err)
			continue
		}

		if exists {
			log.Printf("⏭️  Skipping %s (drive_file_id: %s already exists in database)", asset.FileName, asset.DriveFileID)
			skipped++
			continue
		}

		log.Printf("🆕 New file detected: %s (drive_file_id: %s)", asset.FileName, asset.DriveFileID)

		// Build code_base from filename (without extension)
		codeBase := s.buildCodeBase(asset.FileName)

		// Convert to database model
		dbAsset := &models.DesignAssetDB{
			Code:           codeBase,
			Description:    "", // Empty by default, can be set later
			DriveFileID:    asset.DriveFileID,
			ImageURL:       asset.ImageURL,
			ColorPrimary:   asset.ColorPrimary,
			ColorSecondary: asset.ColorSecondary,
			HoodieType:     asset.BusoType,
			ImageType:      asset.ImageType, // ImageType maps to deco_type
			DecoID:         asset.DecoID,
			DecoBase:       asset.DecoBase,
			CreatedAt:      asset.CreatedTime,
			IsActive:       true,
			HasHiglights:   false,
		}

		// Insert into database
		log.Printf("💾 Attempting to insert into database: %s (code_base: %s)", asset.FileName, codeBase)
		if err := s.repository.Insert(ctx, dbAsset); err != nil {
			log.Printf("❌ Error inserting %s into database: %v", asset.FileName, err)
			continue
		}

		log.Printf("✅ Successfully processed: %s (code_base: %s, drive_file_id: %s)", asset.FileName, codeBase, asset.DriveFileID)
		inserted++
	}

	log.Printf("🎉 Synchronization completed successfully: %d inserted, %d skipped, %d total processed", inserted, skipped, len(driveAssets))
	return driveAssets, nil
}

// buildCodeBase constructs the code_base from filename (without extension)
func (s *SyncService) buildCodeBase(filename string) string {
	// Remove extension
	ext := filepath.Ext(filename)
	codeBase := strings.TrimSuffix(filename, ext)
	return codeBase
}
