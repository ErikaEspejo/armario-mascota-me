package service

// DownloadServiceInterface defines the contract for image download operations
type DownloadServiceInterface interface {
	DownloadAllImages(folderID string) (int, int, []string, error)
}

