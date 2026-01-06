package router

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"armario-mascota-me/app/controller"
)

type Controllers struct {
	DesignAsset        *controller.DesignAssetController
	Item               *controller.ItemController
	ReservedOrder      *controller.ReservedOrderController
	Sale               *controller.SaleController
	FinanceTransaction *controller.FinanceTransactionController
	Catalog            *controller.CatalogController
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

// serveStaticFiles serves static files from the static directory
func serveStaticFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the file path from URL (remove /static prefix)
	path := strings.TrimPrefix(r.URL.Path, "/static/")
	if path == "" || strings.Contains(path, "..") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Build full file path
	filePath := filepath.Join("static", path)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Determine content type based on file extension
	contentType := "application/octet-stream"
	if strings.HasSuffix(path, ".png") {
		contentType = "image/png"
	} else if strings.HasSuffix(path, ".jpg") || strings.HasSuffix(path, ".jpeg") {
		contentType = "image/jpeg"
	} else if strings.HasSuffix(path, ".gif") {
		contentType = "image/gif"
	} else if strings.HasSuffix(path, ".svg") {
		contentType = "image/svg+xml"
	}

	// Serve the file
	w.Header().Set("Content-Type", contentType)
	http.ServeFile(w, r, filePath)
}

func SetupRoutes(controllers *Controllers) {
	// Ping endpoint
	http.HandleFunc("/ping", pingHandler)

	// Static files
	http.HandleFunc("/static/", serveStaticFiles)

	// Design assets routes
	http.HandleFunc("/admin/design-assets/load", controllers.DesignAsset.LoadImages)

	// Get pending design assets
	http.HandleFunc("/admin/design-assets/pending", controllers.DesignAsset.GetPendingDesignAssets)

	// Update full design asset
	http.HandleFunc("/admin/design-assets/update", controllers.DesignAsset.UpdateFullDesignAsset)

	// Filter design assets
	http.HandleFunc("/admin/design-assets/filter", controllers.DesignAsset.FilterDesignAssets)

	// Get optimized image for pending asset
	http.HandleFunc("/admin/design-assets/pending/", func(w http.ResponseWriter, r *http.Request) {
		// Check if this is the image endpoint
		if strings.HasSuffix(r.URL.Path, "/image") {
			controllers.DesignAsset.GetOptimizedImage(w, r)
			return
		}
		// Otherwise, return 404
		http.Error(w, "Not found", http.StatusNotFound)
	})

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

	// Items routes
	// Add stock to item
	http.HandleFunc("/admin/items/stock", controllers.Item.AddStock)
	
	// Filter items
	http.HandleFunc("/admin/items/filter", controllers.Item.FilterItems)

	// Catalog routes
	http.HandleFunc("/admin/catalog", controllers.Catalog.GenerateCatalog)
	http.HandleFunc("/admin/catalog/render", controllers.Catalog.RenderCatalog)

	// Reserved orders routes
	// Create reserved order
	http.HandleFunc("/admin/reserved-orders", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			controllers.ReservedOrder.CreateOrder(w, r)
		} else if r.Method == http.MethodGet {
			controllers.ReservedOrder.ListOrders(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Get separated carts with full item information
	http.HandleFunc("/admin/reserved-orders/separated", controllers.ReservedOrder.GetSeparatedCarts)

	// Reserved order actions (must be before the generic /:id route)
	http.HandleFunc("/admin/reserved-orders/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/admin/reserved-orders/")
		
		// Route to specific actions first
		if strings.HasSuffix(path, "/cancel") {
			controllers.ReservedOrder.CancelOrder(w, r)
			return
		}
		if strings.HasSuffix(path, "/complete") {
			controllers.ReservedOrder.CompleteOrder(w, r)
			return
		}
		if strings.HasSuffix(path, "/sell") {
			controllers.Sale.Sell(w, r)
			return
		}
		// Handle DELETE /admin/reserved-orders/:orderId/items/:itemId
		if strings.Contains(path, "/items/") && r.Method == http.MethodDelete {
			controllers.ReservedOrder.RemoveItem(w, r)
			return
		}
		// Handle PUT/PATCH /admin/reserved-orders/:orderId/items/:itemId
		if strings.Contains(path, "/items/") && (r.Method == http.MethodPut || r.Method == http.MethodPatch) {
			controllers.ReservedOrder.UpdateItemQuantity(w, r)
			return
		}
		// Handle POST /admin/reserved-orders/:id/items
		if strings.HasSuffix(path, "/items") && r.Method == http.MethodPost {
			controllers.ReservedOrder.AddItem(w, r)
			return
		}
		
		// Handle PUT /admin/reserved-orders/:id (update entire order)
		if r.Method == http.MethodPut && !strings.Contains(path, "/") {
			controllers.ReservedOrder.UpdateOrder(w, r)
			return
		}
		
		// Otherwise, treat as GET /admin/reserved-orders/:id
		if r.Method == http.MethodGet {
			controllers.ReservedOrder.GetOrder(w, r)
			return
		}
		
		// Method not allowed
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	// Sales routes
	// List sales
	http.HandleFunc("/admin/sales", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			controllers.Sale.ListSales(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Get sale by ID
	http.HandleFunc("/admin/sales/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			controllers.Sale.GetSale(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Finance routes
	// Finance transactions - handles both POST (create) and GET (list)
	http.HandleFunc("/admin/finance/transactions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			controllers.FinanceTransaction.Create(w, r)
		} else if r.Method == http.MethodGet {
			controllers.FinanceTransaction.List(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Finance summary
	http.HandleFunc("/admin/finance/summary", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			controllers.FinanceTransaction.Summary(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Finance dashboard
	http.HandleFunc("/admin/finance/dashboard", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			controllers.FinanceTransaction.Dashboard(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}
