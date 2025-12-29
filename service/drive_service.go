package service

import (
	"context"
	"fmt"
	"log"
	"strings"

	"armario-mascota-me/models"
	"armario-mascota-me/utils"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// DriveService handles Google Drive API operations
type DriveService struct {
	client *drive.Service
}

// NewDriveService creates a new DriveService instance
// credentialsPath should be the path to the Service Account JSON file
func NewDriveService(credentialsPath string) (*DriveService, error) {
	ctx := context.Background()

	// Create Drive service using credentials file
	// option.WithCredentialsFile automatically handles Service Account authentication
	driveService, err := drive.NewService(ctx, option.WithCredentialsFile(credentialsPath))
	if err != nil {
		return nil, fmt.Errorf("failed to create drive service: %w", err)
	}

	return &DriveService{
		client: driveService,
	}, nil
}

// ListDesignAssets lists all image files in a Google Drive folder and parses them
func (ds *DriveService) ListDesignAssets(folderID string) ([]models.DesignAsset, error) {
	// Build query to list files in the folder
	query := fmt.Sprintf("'%s' in parents and trashed=false", folderID)

	// List files
	var allFiles []*drive.File
	pageToken := ""
	for {
		call := ds.client.Files.List().
			Q(query).
			Fields("nextPageToken, files(id, name, mimeType)")

		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		r, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("failed to list files: %w", err)
		}

		allFiles = append(allFiles, r.Files...)
		pageToken = r.NextPageToken

		if pageToken == "" {
			break
		}
	}

	// Filter images and parse
	var designAssets []models.DesignAsset
	imageMimeTypes := map[string]bool{
		"image/png":  true,
		"image/jpeg": true,
		"image/jpg": true,
	}

	for _, file := range allFiles {
		// Check if it's an image
		if !imageMimeTypes[strings.ToLower(file.MimeType)] {
			continue
		}

		// Parse filename
		parsed, err := utils.ParseFileName(file.Name)
		if err != nil {
			log.Printf("warning: failed to parse filename %s: %v", file.Name, err)
			continue // Skip files that don't match the pattern
		}

		// Build public URL
		imageURL := fmt.Sprintf("https://drive.google.com/uc?id=%s", file.Id)

		// Set Drive-specific fields
		parsed.DriveFileID = file.Id
		parsed.FileName = file.Name
		parsed.ImageURL = imageURL

		designAssets = append(designAssets, *parsed)
	}

	return designAssets, nil
}

