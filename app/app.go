package app

import (
	"fmt"
	"os"

	"armario-mascota-me/app/controller"
	"armario-mascota-me/app/router"
	"armario-mascota-me/service"
)

// Initialize initializes the application
func Initialize() error {
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

	// Create controllers
	controllers := &router.Controllers{
		DesignAsset: controller.NewDesignAssetController(driveService),
		// Add more controllers here as needed
	}

	// Setup routes using standard http router
	router.SetupRoutes(controllers)

	return nil
}

