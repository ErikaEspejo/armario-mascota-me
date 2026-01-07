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
		// Use Overload to ensure .env values override system environment variables
		envPath := ".env"
		if err := godotenv.Overload(envPath); err != nil {
			log.Printf("Warning: .env file not found at %s, using system environment variables", envPath)
			log.Printf("Error details: %v", err)
		} else {
			log.Printf("Successfully loaded environment variables from %s (overriding system variables)", envPath)
			// Debug: Show what was loaded
			credsJSON := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_JSON")
			credsPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
			if len(credsJSON) > 0 {
				log.Printf("DEBUG: GOOGLE_APPLICATION_CREDENTIALS_JSON is set (using JSON from environment)")
			} else if credsPath != "" {
				log.Printf("DEBUG: GOOGLE_APPLICATION_CREDENTIALS after loading .env: %s", credsPath)
			} else {
				log.Printf("DEBUG: Neither GOOGLE_APPLICATION_CREDENTIALS_JSON nor GOOGLE_APPLICATION_CREDENTIALS is set")
			}
		}
	}

	// Initialize application
	if err := app.Initialize(); err != nil {
		log.Fatal(err)
	}
	defer db.CloseDB()

	// Start server
	// Listen on 0.0.0.0 to accept connections from all interfaces (required for Docker/Render)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	// Remove leading colon if present (PORT from Render doesn't include it)
	if len(port) > 0 && port[0] == ':' {
		port = port[1:]
	}
	addr := "0.0.0.0:" + port
	log.Printf("Server starting on %s", addr)
	log.Printf("Load images endpoint: GET http://localhost:%s/admin/design-assets/load?folderId=YOUR_FOLDER_ID", port)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

