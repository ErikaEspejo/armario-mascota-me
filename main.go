package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"armario-mascota-me/app"
	"armario-mascota-me/db"
)

func main() {
	// Load .env file in development (ignores error if file doesn't exist)
	// In production, variables should be set directly
	if os.Getenv("ENV") != "production" {
		// Get current working directory
		wd, err := os.Getwd()
		if err != nil {
			log.Printf("Warning: Could not get working directory: %v", err)
		} else {
			log.Printf("Current working directory: %s", wd)
		}
		
		// Try to load .env from current directory
		envPath := ".env"
		if err := godotenv.Load(envPath); err != nil {
			log.Printf("Warning: .env file not found at %s, using system environment variables", envPath)
			log.Printf("Error details: %v", err)
		} else {
			log.Printf("Successfully loaded environment variables from %s", envPath)
		}
	}

	// Initialize application
	if err := app.Initialize(); err != nil {
		log.Fatal(err)
	}
	defer db.CloseDB()

	// Start server
	port := ":8080"
	log.Printf("Server starting on port %s", port)
	log.Printf("Load images endpoint: GET http://localhost%s/admin/design-assets/load?folderId=YOUR_FOLDER_ID", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

