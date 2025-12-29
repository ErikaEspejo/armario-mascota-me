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
	http.HandleFunc("/admin/design-assets/sync", controllers.DesignAsset.SyncDesignAssets)
}
