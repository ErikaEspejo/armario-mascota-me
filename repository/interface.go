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
	GetByID(ctx context.Context, id int) (*models.DesignAssetDetail, error)
	UpdateDescriptionAndHighlights(ctx context.Context, code string, description string, hasHighlights bool) error
	GetPending(ctx context.Context) ([]models.DesignAssetDetail, error)
	UpdateFullDesignAsset(ctx context.Context, id int, code, description, colorPrimary, colorSecondary, hoodieType, imageType, decoID, decoBase string, hasHighlights bool, status string) error
	FilterDesignAssets(ctx context.Context, filters FilterParams) ([]models.DesignAssetDetail, error)
}

// ItemRepositoryInterface defines the contract for item repository operations
type ItemRepositoryInterface interface {
	UpsertStock(ctx context.Context, designAssetID int, size string, quantity int) (*models.AddStockResponse, error)
	FilterItems(ctx context.Context, filters ItemFilterParams) ([]models.ItemCard, error)
}

// ReservedOrderRepositoryInterface defines the contract for reserved order repository operations
type ReservedOrderRepositoryInterface interface {
	Create(ctx context.Context, req *models.CreateReservedOrderRequest) (*models.ReservedOrder, error)
	AddItem(ctx context.Context, orderID int64, itemID int64, qty int) (*models.ReservedOrderLine, error)
	RemoveItem(ctx context.Context, orderID int64, itemID int64) error
	UpdateItemQuantity(ctx context.Context, orderID int64, itemID int64, newQty int) (*models.ReservedOrderLine, error)
	UpdateOrder(ctx context.Context, req *models.UpdateReservedOrderRequest) (*models.ReservedOrderResponse, error)
	GetByID(ctx context.Context, id int64) (*models.ReservedOrderResponse, error)
	List(ctx context.Context, status *string) ([]models.ReservedOrderListItem, error)
	Cancel(ctx context.Context, id int64) (*models.ReservedOrder, error)
	Complete(ctx context.Context, id int64) (*models.ReservedOrder, error)
	GetAllWithFullItems(ctx context.Context, status *string) ([]models.ReservedOrderWithFullItems, error)
}

// SaleRepositoryInterface defines the contract for sale repository operations
type SaleRepositoryInterface interface {
	Sell(ctx context.Context, reservedOrderID int64, req *models.SellRequest) (*models.Sale, error)
	GetByID(ctx context.Context, saleID int64) (*models.SaleDetailResponse, error)
	List(ctx context.Context, from, to *string) ([]models.SaleListItem, error)
}
