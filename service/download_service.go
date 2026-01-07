package service

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// DownloadService handles downloading and optimizing images from Google Drive
// Implements DownloadServiceInterface
type DownloadService struct {
	driveService DriveServiceInterface
}

// NewDownloadService creates a new DownloadService instance
func NewDownloadService(driveService DriveServiceInterface) *DownloadService {
	return &DownloadService{
		driveService: driveService,
	}
}

// Ensure DownloadService implements DownloadServiceInterface
var _ DownloadServiceInterface = (*DownloadService)(nil)

// getDownloadDir returns the download directory path outside the project
func getDownloadDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	// Use Downloads folder in user's home directory
	downloadDir := filepath.Join(homeDir, "Downloads", "armario-images")
	return downloadDir, nil
}

// DownloadAllImages downloads all images from a Google Drive folder, optimizes them, and saves them locally
// Returns: total images found, successfully downloaded count, skipped count, list of errors, and error if fatal
func (ds *DownloadService) DownloadAllImages(folderID string) (int, int, int, []string, error) {
	log.Printf("📥 Starting download process for folder: %s", folderID)

	// Get download directory path
	downloadDir, err := getDownloadDir()
	if err != nil {
		return 0, 0, 0, nil, err
	}

	log.Printf("📁 Download directory: %s", downloadDir)

	// Ensure download directory exists
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return 0, 0, 0, nil, fmt.Errorf("failed to create download directory: %w", err)
	}

	// Get all design assets from Google Drive (this gives us file IDs)
	driveAssets, err := ds.driveService.ListDesignAssets(folderID)
	if err != nil {
		return 0, 0, 0, nil, fmt.Errorf("failed to list design assets from Drive: %w", err)
	}

	// Get file names mapping
	fileNames, err := ds.driveService.GetImageFileNames(folderID)
	if err != nil {
		return 0, 0, 0, nil, fmt.Errorf("failed to get file names from Drive: %w", err)
	}

	log.Printf("📦 Found %d images to download", len(driveAssets))

	totalImages := len(driveAssets)
	downloaded := 0
	skipped := 0
	var errors []string

	// Track used file names to avoid duplicates
	usedFileNames := make(map[string]bool)

	// For each asset, download and save
	for _, asset := range driveAssets {
		// Get file name, fallback to file ID if not found
		fileName, exists := fileNames[asset.DriveFileID]
		if !exists {
			fileName = asset.DriveFileID
		}

		// Convert extension to .jpg (since OptimizeImage returns JPEG)
		fileName = strings.TrimSuffix(fileName, ".png")
		fileName = strings.TrimSuffix(fileName, ".PNG")
		fileName = strings.TrimSuffix(fileName, ".jpg")
		fileName = strings.TrimSuffix(fileName, ".JPG")
		fileName = strings.TrimSuffix(fileName, ".jpeg")
		fileName = strings.TrimSuffix(fileName, ".JPEG")
		fileName = fileName + ".jpg"

		// Build full file path
		filePath := filepath.Join(downloadDir, fileName)

		// Check if file already exists on disk (from previous downloads)
		if _, err := os.Stat(filePath); err == nil {
			log.Printf("⏭️  Skipping %s (already exists on disk)", fileName)
			skipped++
			continue
		}

		// Check if this filename was already used in this download session (duplicate)
		if usedFileNames[fileName] {
			log.Printf("⏭️  Skipping %s (duplicate filename in this session)", fileName)
			skipped++
			continue
		}
		usedFileNames[fileName] = true

		// Download image
		imageData, err := ds.driveService.DownloadImage(asset.DriveFileID)
		if err != nil {
			errorMsg := fmt.Sprintf("Failed to download image %s (%s): %v", fileName, asset.DriveFileID, err)
			log.Printf("❌ %s", errorMsg)
			errors = append(errors, errorMsg)
			continue
		}

		// Optimize image
		optimizedData, err := OptimizeImage(imageData, "medium")
		if err != nil {
			errorMsg := fmt.Sprintf("Failed to optimize image %s (%s): %v", fileName, asset.DriveFileID, err)
			log.Printf("❌ %s", errorMsg)
			errors = append(errors, errorMsg)
			continue
		}

		// Save to downloads directory
		if err := ioutil.WriteFile(filePath, optimizedData, 0644); err != nil {
			errorMsg := fmt.Sprintf("Failed to save image %s: %v", fileName, err)
			log.Printf("❌ %s", errorMsg)
			errors = append(errors, errorMsg)
			continue
		}

		log.Printf("✓ Successfully downloaded and saved: %s", filePath)
		downloaded++
	}

	log.Printf("🎉 Download completed: %d downloaded, %d skipped, %d failed out of %d total images", downloaded, skipped, len(errors), totalImages)
	return totalImages, downloaded, skipped, errors, nil
}
