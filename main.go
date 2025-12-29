package main

import (
	"log"
	"net/http"

	"armario-mascota-me/app"
)

func main() {
	// Initialize application
	if err := app.Initialize(); err != nil {
		log.Fatal(err)
	}

	// Start server
	port := ":8080"
	log.Printf("Server starting on port %s", port)
	log.Printf("Endpoint available at: http://localhost%s/admin/design-assets/sync?folderId=YOUR_FOLDER_ID", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

