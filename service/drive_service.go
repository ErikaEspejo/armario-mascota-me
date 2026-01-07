package service

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"armario-mascota-me/models"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// DriveService handles Google Drive API operations
// Implements DriveServiceInterface
type DriveService struct {
	client *drive.Service
}

// Ensure DriveService implements DriveServiceInterface
var _ DriveServiceInterface = (*DriveService)(nil)

// NewDriveService creates a new DriveService instance
// Accepts either credentials JSON bytes or a path to credentials file
// If credentialsJSON is provided, it will be used; otherwise credentialsPath will be used
func NewDriveService(credentialsJSON []byte, credentialsPath string) (*DriveService, error) {
	ctx := context.Background()

	var opts []option.ClientOption

	// Prefer JSON credentials from environment variable if provided
	if len(credentialsJSON) > 0 {
		log.Printf("Connecting to Google Drive API with credentials from environment variable")
		opts = append(opts, option.WithCredentialsJSON(credentialsJSON))
	} else if credentialsPath != "" {
		log.Printf("Connecting to Google Drive API with credentials file: %s", credentialsPath)
		opts = append(opts, option.WithCredentialsFile(credentialsPath))
	} else {
		return nil, fmt.Errorf("either GOOGLE_APPLICATION_CREDENTIALS_JSON or GOOGLE_APPLICATION_CREDENTIALS must be set")
	}

	// Create Drive service using credentials
	driveService, err := drive.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create drive service: %w", err)
	}

	log.Printf("✓ Google Drive API connection established successfully")
	return &DriveService{
		client: driveService,
	}, nil
}

// ListDesignAssets lists all image files in a Google Drive folder and parses them
func (ds *DriveService) ListDesignAssets(folderID string) ([]models.DesignAsset, error) {
	log.Printf("Fetching files from Google Drive folder: %s", folderID)

	// Build query to list files in the folder
	query := fmt.Sprintf("'%s' in parents and trashed=false", folderID)

	// List files
	var allFiles []*drive.File
	pageToken := ""
	pageCount := 0
	for {
		call := ds.client.Files.List().
			Q(query).
			Fields("nextPageToken, files(id, name, mimeType, createdTime, modifiedTime)")

		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		r, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("failed to list files: %w", err)
		}

		allFiles = append(allFiles, r.Files...)
		pageCount++
		pageToken = r.NextPageToken

		if pageToken == "" {
			break
		}
	}

	log.Printf("✓ Retrieved %d total files from Google Drive (fetched in %d pages)", len(allFiles))

	// Filter images and build simple assets
	var designAssets []models.DesignAsset
	imageMimeTypes := map[string]bool{
		"image/png":  true,
		"image/jpeg": true,
		"image/jpg":  true,
	}

	for _, file := range allFiles {
		// Check if it's an image
		if !imageMimeTypes[strings.ToLower(file.MimeType)] {
			continue
		}

		// Build public URL
		imageURL := fmt.Sprintf("https://drive.google.com/uc?id=%s", file.Id)

		// Create simple asset with only drive_file_id and image_url
		asset := models.DesignAsset{
			DriveFileID: file.Id,
			ImageURL:    imageURL,
		}

		designAssets = append(designAssets, asset)
	}

	log.Printf("✓ Successfully processed %d image files from Google Drive", len(designAssets))
	return designAssets, nil
}

// DownloadImage downloads an image file from Google Drive by file ID
func (ds *DriveService) DownloadImage(fileID string) ([]byte, error) {
	log.Printf("📥 Downloading image from Google Drive: fileID=%s", fileID)

	resp, err := ds.client.Files.Get(fileID).Download()
	if err != nil {
		log.Printf("❌ Error downloading image from Drive (fileID: %s): %v", fileID, err)
		return nil, fmt.Errorf("failed to download image from Drive: %w", err)
	}
	defer resp.Body.Close()

	// Read all bytes from response body
	imageData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ Error reading image data (fileID: %s): %v", fileID, err)
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	log.Printf("✓ Successfully downloaded image from Drive: fileID=%s, size=%d bytes", fileID, len(imageData))
	return imageData, nil
}
