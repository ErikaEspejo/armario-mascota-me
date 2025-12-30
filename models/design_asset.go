package models

// DesignAsset represents a design asset from Google Drive
type DesignAsset struct {
	DriveFileID string `json:"driveFileId"`
	ImageURL    string `json:"imageUrl"`
}



