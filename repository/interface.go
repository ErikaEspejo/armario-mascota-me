package repository

import (
	"context"

	"armario-mascota-me/models"
)

// DesignAssetRepositoryInterface defines the contract for design asset repository operations
type DesignAssetRepositoryInterface interface {
	ExistsByDriveFileID(ctx context.Context, driveFileID string) (bool, error)
	Insert(ctx context.Context, asset *models.DesignAssetDB) error
	GetByCode(ctx context.Context, code string) (*models.DesignAssetDetail, error)
	UpdateDescriptionAndHighlights(ctx context.Context, code string, description string, hasHighlights bool) error
	GetPending(ctx context.Context) ([]models.DesignAssetDetail, error)
}
