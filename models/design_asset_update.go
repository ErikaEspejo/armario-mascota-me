package models

// DesignAssetUpdateRequest represents the request body for updating a design asset
type DesignAssetUpdateRequest struct {
	Description  string `json:"description"`
	HasHighlights bool  `json:"hasHighlights"`
}

// DesignAssetDetail represents a design asset with all details for editing
type DesignAssetDetail struct {
	Code           string `json:"code"`
	Description    string `json:"description"`
	DriveFileID    string `json:"driveFileId"`
	ImageURL       string `json:"imageUrl"`
	ColorPrimary   string `json:"colorPrimary"`
	ColorSecondary string `json:"colorSecondary"`
	HoodieType     string `json:"hoodieType"`
	ImageType      string `json:"imageType"`
	DecoID         string `json:"decoId"`
	DecoBase       string `json:"decoBase"`
	IsActive       bool   `json:"isActive"`
	HasHighlights  bool   `json:"hasHighlights"`
}

