package models

// DesignAssetFullUpdateRequest represents the request body for full update of a design asset
type DesignAssetFullUpdateRequest struct {
	ID             string `json:"id"`
	Description    string `json:"description"`
	ColorPrimary   string `json:"colorPrimary"`
	ColorSecondary string `json:"colorSecondary"`
	HoodieType     string `json:"hoodieType"`
	ImageType      string `json:"imageType"`
	DecoBase       string `json:"decoBase"`
	HasHighlights  bool   `json:"hasHighlights"`
}

