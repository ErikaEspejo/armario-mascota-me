package service

import "armario-mascota-me/models"

// DriveServiceInterface defines the contract for Google Drive operations
type DriveServiceInterface interface {
	ListDesignAssets(folderID string) ([]models.DesignAsset, error)
	DownloadImage(fileID string) ([]byte, error)
}


