package models

// DesignAsset represents a parsed design asset from Google Drive
type DesignAsset struct {
	DriveFileID    string `json:"driveFileId"`
	FileName       string `json:"fileName"`
	ImageURL       string `json:"imageUrl"`
	CreatedTime    string `json:"createdTime"`
	ModifiedTime   string `json:"modifiedTime"`
	ColorPrimary   string `json:"colorPrimary"`
	ColorSecondary string `json:"colorSecondary"`
	BusoType       string `json:"busoType"`
	ImageType      string `json:"imageType"`
	DecoID         string `json:"decoId"`
	DecoBase       string `json:"decoBase"`
}



