package service

import (
	"context"

	"armario-mascota-me/models"
)

// SyncServiceInterface defines the contract for synchronization operations
type SyncServiceInterface interface {
	SyncDesignAssets(ctx context.Context, folderID string) ([]models.DesignAsset, error)
}
