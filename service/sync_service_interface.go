package service

import (
	"context"

	"armario-mascota-me/models"
)

// SyncServiceInterface defines the contract for synchronization operations
type SyncServiceInterface interface {
	SyncDesignAssets(ctx context.Context, folderID string) ([]models.DesignAsset, error)
	// SyncDesignAssetsWithStats synchronizes assets and returns insertion stats:
	// inserted = new rows created, skipped = already existed (by drive_file_id), total = total assets seen in Drive.
	// status parameter determines the status to set for newly inserted assets (defaults to "pending" if empty)
	SyncDesignAssetsWithStats(ctx context.Context, folderID string, status string) (assets []models.DesignAsset, inserted int, skipped int, total int, err error)
}
