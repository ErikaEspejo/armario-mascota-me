package models

// DesignAssetDB represents a design asset for database operations
type DesignAssetDB struct {
	Code           string
	Description    string
	DriveFileID    string
	ImageURL       string
	ColorPrimary   string
	ColorSecondary string
	HoodieType     string
	ImageType      string
	DecoID         string
	DecoBase       string
	CreatedAt      string // RFC3339 format from Google Drive
	IsActive       bool
	HasHiglights   bool
}


