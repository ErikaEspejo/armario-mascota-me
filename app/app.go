package app

import (
	"fmt"
	"os"

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
	if credentialsPath == "" {
		return fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS environment variable is not set")
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
		DesignAsset: controller.NewDesignAssetController(syncService, designAssetRepo),
	}

	// Setup routes using standard http router
	router.SetupRoutes(controllers)

	return nil
}

