package models

// Item represents an item in the database
type Item struct {
	ID            int    `json:"id"`
	DesignAssetID int    `json:"designAssetId"`
	Size          string `json:"size"`
	SKU           string `json:"sku"`
	Price         int    `json:"price"`
	StockTotal    int    `json:"stockTotal"`
	StockReserved int    `json:"stockReserved"`
	IsActive      bool   `json:"isActive"`
	CreatedAt     string `json:"createdAt"`
}

// AddStockRequest represents the request body for adding stock
type AddStockRequest struct {
	DesignAssetID int `json:"design_asset_id"`
	Size          string `json:"size"`
	Quantity      int    `json:"quantity"`
}

// AddStockResponse represents the response after adding stock
type AddStockResponse struct {
	ID            int    `json:"id"`
	SKU           string `json:"sku"`
	Size          string `json:"size"`
	Price         int    `json:"price"`
	StockTotal    int    `json:"stock_total"`
	StockReserved int    `json:"stock_reserved"`
}

