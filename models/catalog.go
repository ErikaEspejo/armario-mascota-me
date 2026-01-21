package models

// CatalogItem represents a single item in the catalog
type CatalogItem struct {
	ID              int    `json:"id"`
	DesignAssetID   int    `json:"designAssetId"`
	ImageURL        string `json:"imageUrl"`
	ImageBase64     string `json:"imageBase64"` // For PDF/PNG generation
	ColorPrimary    string `json:"colorPrimary"`  // Code (e.g., "AC")
	ColorPrimaryName string `json:"colorPrimaryName"` // Human-readable name (e.g., "azul cielo")
	ColorSecondary  string `json:"colorSecondary"`
	HoodieType      string `json:"hoodieType"`
	HoodieTypeName  string `json:"hoodieTypeName"` // Human-readable name (capitalized)
	SKU             string `json:"sku"`            // SKU in uppercase
	Code            string `json:"code"`            // Full code
	AvailableQty    int    `json:"availableQty"`
	IsCustom        bool   `json:"isCustom"`       // True when any component code is CSM (custom)
}

// CatalogData represents the data structure passed to the catalog template
type CatalogData struct {
	Size      string        `json:"size"`
	Items     []CatalogItem  `json:"items"`
	PageCount int           `json:"pageCount"`
}

