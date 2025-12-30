package router

import (
	"net/http"

	"armario-mascota-me/app/controller"
)

type Controllers struct {
	DesignAsset *controller.DesignAssetController
}

// pingHandler handles GET /ping
func pingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func SetupRoutes(controllers *Controllers) {
	// Ping endpoint
	http.HandleFunc("/ping", pingHandler)

	// Design assets routes
	http.HandleFunc("/admin/design-assets/load", controllers.DesignAsset.LoadImages)

	// Get pending design assets
	http.HandleFunc("/admin/design-assets/pending", controllers.DesignAsset.GetPendingDesignAssets)

	// Design asset by code - handles both GET (get) and PUT (update)
	http.HandleFunc("/admin/design-assets/", func(w http.ResponseWriter, r *http.Request) {
		// Route to appropriate handler based on HTTP method
		if r.Method == http.MethodGet {
			controllers.DesignAsset.GetDesignAssetByCode(w, r)
		} else if r.Method == http.MethodPut {
			controllers.DesignAsset.UpdateDesignAsset(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}
