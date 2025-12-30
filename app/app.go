package app

import (
	"fmt"
	"os"
	"path/filepath"

	"armario-mascota-me/app/controller"
	"armario-mascota-me/app/router"
	"armario-mascota-me/db"
	"armario-mascota-me/repository"
	"armario-mascota-me/service"
)

// Initialize initializes the application
func Initialize() error {
	// Initialize database connection
	if err := db.InitDB(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Get credentials path from environment variable
	credentialsPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	fmt.Printf("DEBUG: GOOGLE_APPLICATION_CREDENTIALS from env: %s\n", credentialsPath)
	
	if credentialsPath == "" {
		return fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS environment variable is not set")
	}

	// Resolve relative paths to absolute paths
	// If the path is relative, resolve it from the current working directory
	if !filepath.IsAbs(credentialsPath) {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		fmt.Printf("DEBUG: Current working directory: %s\n", wd)
		credentialsPath = filepath.Join(wd, credentialsPath)
		// Normalize path separators for Windows
		credentialsPath = filepath.Clean(credentialsPath)
		fmt.Printf("DEBUG: Resolved credentials path: %s\n", credentialsPath)
	}

	// Initialize Drive service
	driveService, err := service.NewDriveService(credentialsPath)
	if err != nil {
		return err
	}

	// Initialize repository
	designAssetRepo := repository.NewDesignAssetRepository()

	// Initialize sync service
	syncService := service.NewSyncService(driveService, designAssetRepo)

	// Create controllers
	controllers := &router.Controllers{
		DesignAsset: controller.NewDesignAssetController(syncService, designAssetRepo, driveService),
	}

	// Setup routes using standard http router
	router.SetupRoutes(controllers)

	return nil
}

