package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"armario-mascota-me/models"
	"armario-mascota-me/repository"
)

// SaleController handles HTTP requests for sales
type SaleController struct {
	repository repository.SaleRepositoryInterface
}

// NewSaleController creates a new SaleController
func NewSaleController(repo repository.SaleRepositoryInterface) *SaleController {
	return &SaleController{
		repository: repo,
	}
}

// Sell handles POST /admin/reserved-orders/:id/sell
// Example request:
// POST /admin/reserved-orders/3/sell
// {
//   "amountPaid": 100000,
//   "paymentMethod": "transfer",
//   "paymentDestination": "Nequi",
//   "notes": "Pago completo"
// }
// Example response:
// {
//   "id": 10,
//   "reservedOrderId": 3,
//   "soldAt": "2026-01-04T10:30:00Z",
//   "customerName": "Juan Pérez",
//   "amountPaid": 100000,
//   "paymentMethod": "transfer",
//   "paymentDestination": "Nequi",
//   "status": "paid",
//   "notes": "Pago completo",
//   "createdAt": "2026-01-04T10:30:00Z"
// }
func (c *SaleController) Sell(w http.ResponseWriter, r *http.Request) {
	log.Printf("📥 Sell: Received %s request to %s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		log.Printf("❌ Sell: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract order ID from URL path
	// Path format: /admin/reserved-orders/{id}/sell
	path := strings.TrimPrefix(r.URL.Path, "/admin/reserved-orders/")
	if path == "" {
		http.Error(w, "order id parameter is required", http.StatusBadRequest)
		return
	}

	// Extract ID (remove /sell suffix)
	idStr := strings.TrimSuffix(path, "/sell")
	if idStr == path {
		http.Error(w, "invalid path format", http.StatusBadRequest)
		return
	}

	orderID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Printf("❌ Sell: Invalid order id: %s", idStr)
		http.Error(w, "invalid order id parameter", http.StatusBadRequest)
		return
	}

	var req models.SellRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Sell: Failed to decode request body: %v", err)
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.AmountPaid <= 0 {
		log.Printf("❌ Sell: amountPaid must be greater than 0: %d", req.AmountPaid)
		http.Error(w, "amountPaid must be greater than 0", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.PaymentMethod) == "" {
		log.Printf("❌ Sell: paymentMethod is required")
		http.Error(w, "paymentMethod is required", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.PaymentDestination) == "" {
		log.Printf("❌ Sell: paymentDestination is required")
		http.Error(w, "paymentDestination is required", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	sale, err := c.repository.Sell(ctx, orderID, &req)
	if err != nil {
		log.Printf("❌ Sell: Error selling order: %v", err)
		errMsg := err.Error()
		if strings.Contains(errMsg, "order not found") {
			http.Error(w, errMsg, http.StatusNotFound)
			return
		}
		if strings.Contains(errMsg, "not in reserved status") {
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		if strings.Contains(errMsg, "already has a sale") {
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		if strings.Contains(errMsg, "insufficient reserved stock") {
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to sell order: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Sell: Successfully sold order id=%d, sale id=%d", orderID, sale.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(sale); err != nil {
		log.Printf("❌ Sell: Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// ListSales handles GET /admin/sales?from=YYYY-MM-DD&to=YYYY-MM-DD
// Example response:
// {
//   "sales": [
//     {
//       "id": 10,
//       "soldAt": "2026-01-04T10:30:00Z",
//       "reservedOrderId": 3,
//       "customerName": "Juan Pérez",
//       "amountPaid": 100000,
//       "paymentDestination": "Nequi",
//       "paymentMethod": "transfer"
//     }
//   ]
// }
func (c *SaleController) ListSales(w http.ResponseWriter, r *http.Request) {
	log.Printf("📥 ListSales: Received %s request to %s", r.Method, r.URL.Path)

	if r.Method != http.MethodGet {
		log.Printf("❌ ListSales: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	var from, to *string
	if fromStr != "" {
		// Validate date format
		_, err := time.Parse("2006-01-02", fromStr)
		if err != nil {
			log.Printf("❌ ListSales: Invalid from date format: %s", fromStr)
			http.Error(w, "Invalid from date format. Use YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		from = &fromStr
	}

	if toStr != "" {
		// Validate date format
		_, err := time.Parse("2006-01-02", toStr)
		if err != nil {
			log.Printf("❌ ListSales: Invalid to date format: %s", toStr)
			http.Error(w, "Invalid to date format. Use YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		to = &toStr
	}

	ctx := context.Background()
	sales, err := c.repository.List(ctx, from, to)
	if err != nil {
		log.Printf("❌ ListSales: Error fetching sales: %v", err)
		http.Error(w, fmt.Sprintf("Failed to fetch sales: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ ListSales: Successfully fetched %d sales", len(sales))

	response := models.SaleListResponse{
		Sales: sales,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("❌ ListSales: Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetSale handles GET /admin/sales/:id
// Example response:
// {
//   "id": 10,
//   "reservedOrderId": 3,
//   "soldAt": "2026-01-04T10:30:00Z",
//   "customerName": "Juan Pérez",
//   "amountPaid": 100000,
//   "paymentMethod": "transfer",
//   "paymentDestination": "Nequi",
//   "status": "paid",
//   "notes": "Pago completo",
//   "createdAt": "2026-01-04T10:30:00Z",
//   "order": {
//     "id": 3,
//     "status": "completed",
//     ...
//   }
// }
func (c *SaleController) GetSale(w http.ResponseWriter, r *http.Request) {
	log.Printf("📥 GetSale: Received %s request to %s", r.Method, r.URL.Path)

	if r.Method != http.MethodGet {
		log.Printf("❌ GetSale: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract sale ID from URL path
	// Path format: /admin/sales/{id}
	path := strings.TrimPrefix(r.URL.Path, "/admin/sales/")
	if path == "" {
		http.Error(w, "sale id parameter is required", http.StatusBadRequest)
		return
	}

	// Check if path contains sub-paths
	if strings.Contains(path, "/") {
		http.Error(w, "invalid path format", http.StatusBadRequest)
		return
	}

	saleID, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		log.Printf("❌ GetSale: Invalid sale id: %s", path)
		http.Error(w, "invalid sale id parameter", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	sale, err := c.repository.GetByID(ctx, saleID)
	if err != nil {
		log.Printf("❌ GetSale: Error fetching sale: %v", err)
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to fetch sale: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ GetSale: Successfully fetched sale id=%d", saleID)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(sale); err != nil {
		log.Printf("❌ GetSale: Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}


