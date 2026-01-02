package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"armario-mascota-me/models"
	"armario-mascota-me/repository"
)

// ReservedOrderController handles HTTP requests for reserved orders
type ReservedOrderController struct {
	repository repository.ReservedOrderRepositoryInterface
}

// NewReservedOrderController creates a new ReservedOrderController
func NewReservedOrderController(repo repository.ReservedOrderRepositoryInterface) *ReservedOrderController {
	return &ReservedOrderController{
		repository: repo,
	}
}

// CreateOrder handles POST /admin/reserved-orders
// Example request:
// POST /admin/reserved-orders
// {
//   "assignedTo": "Erika",
//   "orderType": "detal",
//   "customerName": "Juan Pérez",
//   "customerPhone": "+1234567890",
//   "notes": "Cliente VIP"
// }
// Example response:
// {
//   "id": 1,
//   "status": "reserved",
//   "assignedTo": "Erika",
//   "orderType": "detal",
//   "customerName": "Juan Pérez",
//   "customerPhone": "+1234567890",
//   "notes": "Cliente VIP",
//   "createdAt": "2024-01-15T10:30:00Z",
//   "updatedAt": "2024-01-15T10:30:00Z"
// }
func (c *ReservedOrderController) CreateOrder(w http.ResponseWriter, r *http.Request) {
	log.Printf("📥 CreateOrder: Received %s request to %s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		log.Printf("❌ CreateOrder: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read body for logging
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("❌ CreateOrder: Failed to read request body: %v", err)
		http.Error(w, fmt.Sprintf("Failed to read request body: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Log the raw body
	log.Printf("📋 CreateOrder: Request body: %s", string(bodyBytes))

	// Create a new reader from the body bytes for decoding
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var req models.CreateReservedOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ CreateOrder: Failed to decode request body: %v", err)
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.AssignedTo) == "" {
		log.Printf("❌ CreateOrder: assigned_to is required")
		http.Error(w, "assigned_to is required", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.OrderType) == "" {
		log.Printf("❌ CreateOrder: order_type is required")
		http.Error(w, "order_type is required", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	order, err := c.repository.Create(ctx, &req)
	if err != nil {
		log.Printf("❌ CreateOrder: Error creating order: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create order: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ CreateOrder: Successfully created order id=%d", order.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(order); err != nil {
		log.Printf("❌ CreateOrder: Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// AddItem handles POST /admin/reserved-orders/:id/items
// Example request:
// POST /admin/reserved-orders/1/items
// {
//   "itemId": 123,
//   "qty": 2
// }
// Example response:
// {
//   "id": 1,
//   "reservedOrderId": 1,
//   "itemId": 123,
//   "qty": 2,
//   "unitPrice": 50000,
//   "createdAt": "2024-01-15T10:30:00Z"
// }
func (c *ReservedOrderController) AddItem(w http.ResponseWriter, r *http.Request) {
	log.Printf("📥 AddItem: Received %s request to %s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		log.Printf("❌ AddItem: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract order ID from URL path
	// Path format: /admin/reserved-orders/{id}/items
	path := strings.TrimPrefix(r.URL.Path, "/admin/reserved-orders/")
	if path == "" {
		http.Error(w, "order id parameter is required", http.StatusBadRequest)
		return
	}

	// Extract ID (remove /items suffix)
	idStr := strings.TrimSuffix(path, "/items")
	if idStr == path {
		http.Error(w, "invalid path format", http.StatusBadRequest)
		return
	}

	orderID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Printf("❌ AddItem: Invalid order id: %s", idStr)
		http.Error(w, "invalid order id parameter", http.StatusBadRequest)
		return
	}

	var req models.AddItemToOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ AddItem: Failed to decode request body: %v", err)
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.ItemID <= 0 {
		log.Printf("❌ AddItem: Invalid item_id: %d", req.ItemID)
		http.Error(w, "item_id must be greater than 0", http.StatusBadRequest)
		return
	}

	if req.Qty <= 0 {
		log.Printf("❌ AddItem: Invalid qty: %d", req.Qty)
		http.Error(w, "qty must be greater than 0", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	line, err := c.repository.AddItem(ctx, orderID, req.ItemID, req.Qty)
	if err != nil {
		log.Printf("❌ AddItem: Error adding item: %v", err)
		errMsg := err.Error()
		if strings.Contains(errMsg, "insufficient stock") {
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "not in reserved status") {
			http.Error(w, errMsg, http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to add item: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ AddItem: Successfully added item to order: line_id=%d", line.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(line); err != nil {
		log.Printf("❌ AddItem: Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetOrder handles GET /admin/reserved-orders/:id
// Example response:
// {
//   "id": 1,
//   "status": "reserved",
//   "assignedTo": "Erika",
//   "customerName": "Juan Pérez",
//   "customerPhone": "+1234567890",
//   "notes": "Cliente VIP",
//   "createdAt": "2024-01-15T10:30:00Z",
//   "updatedAt": "2024-01-15T10:30:00Z",
//   "lines": [
//     {
//       "id": 1,
//       "reservedOrderId": 1,
//       "itemId": 123,
//       "itemSku": "MN_ABC123",
//       "itemSize": "MN",
//       "qty": 2,
//       "unitPrice": 50000,
//       "itemPrice": 50000,
//       "createdAt": "2024-01-15T10:30:00Z"
//     }
//   ],
//   "total": 100000
// }
func (c *ReservedOrderController) GetOrder(w http.ResponseWriter, r *http.Request) {
	log.Printf("📥 GetOrder: Received %s request to %s", r.Method, r.URL.Path)

	if r.Method != http.MethodGet {
		log.Printf("❌ GetOrder: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract order ID from URL path
	// Path format: /admin/reserved-orders/{id}
	path := strings.TrimPrefix(r.URL.Path, "/admin/reserved-orders/")
	if path == "" {
		http.Error(w, "order id parameter is required", http.StatusBadRequest)
		return
	}

	// Check if path contains sub-paths (like /items, /cancel, /complete)
	if strings.Contains(path, "/") {
		http.Error(w, "invalid path format", http.StatusBadRequest)
		return
	}

	orderID, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		log.Printf("❌ GetOrder: Invalid order id: %s", path)
		http.Error(w, "invalid order id parameter", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	order, err := c.repository.GetByID(ctx, orderID)
	if err != nil {
		log.Printf("❌ GetOrder: Error fetching order: %v", err)
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to fetch order: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ GetOrder: Successfully fetched order id=%d", orderID)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(order); err != nil {
		log.Printf("❌ GetOrder: Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// ListOrders handles GET /admin/reserved-orders?status=reserved
// Example response:
// {
//   "orders": [
//     {
//       "id": 1,
//       "status": "reserved",
//       "assignedTo": "Erika",
//       "customerName": "Juan Pérez",
//       "createdAt": "2024-01-15T10:30:00Z",
//       "updatedAt": "2024-01-15T10:30:00Z",
//       "lineCount": 2,
//       "total": 100000
//     }
//   ]
// }
func (c *ReservedOrderController) ListOrders(w http.ResponseWriter, r *http.Request) {
	log.Printf("📥 ListOrders: Received %s request to %s", r.Method, r.URL.Path)

	if r.Method != http.MethodGet {
		log.Printf("❌ ListOrders: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse status query parameter
	status := r.URL.Query().Get("status")
	var statusPtr *string
	if status != "" {
		statusPtr = &status
	}

	ctx := context.Background()
	orders, err := c.repository.List(ctx, statusPtr)
	if err != nil {
		log.Printf("❌ ListOrders: Error fetching orders: %v", err)
		http.Error(w, fmt.Sprintf("Failed to fetch orders: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ ListOrders: Successfully fetched %d orders", len(orders))

	response := models.ReservedOrderListResponse{
		Orders: orders,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("❌ ListOrders: Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// CancelOrder handles POST /admin/reserved-orders/:id/cancel
// Example response:
// {
//   "id": 1,
//   "status": "canceled",
//   "assignedTo": "Erika",
//   "createdAt": "2024-01-15T10:30:00Z",
//   "updatedAt": "2024-01-15T11:00:00Z"
// }
func (c *ReservedOrderController) CancelOrder(w http.ResponseWriter, r *http.Request) {
	log.Printf("📥 CancelOrder: Received %s request to %s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		log.Printf("❌ CancelOrder: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract order ID from URL path
	// Path format: /admin/reserved-orders/{id}/cancel
	path := strings.TrimPrefix(r.URL.Path, "/admin/reserved-orders/")
	if path == "" {
		http.Error(w, "order id parameter is required", http.StatusBadRequest)
		return
	}

	// Extract ID (remove /cancel suffix)
	idStr := strings.TrimSuffix(path, "/cancel")
	if idStr == path {
		http.Error(w, "invalid path format", http.StatusBadRequest)
		return
	}

	orderID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Printf("❌ CancelOrder: Invalid order id: %s", idStr)
		http.Error(w, "invalid order id parameter", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	order, err := c.repository.Cancel(ctx, orderID)
	if err != nil {
		log.Printf("❌ CancelOrder: Error canceling order: %v", err)
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "not in reserved status") {
			http.Error(w, errMsg, http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to cancel order: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ CancelOrder: Successfully canceled order id=%d", orderID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(order); err != nil {
		log.Printf("❌ CancelOrder: Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// CompleteOrder handles POST /admin/reserved-orders/:id/complete
// Example response:
// {
//   "id": 1,
//   "status": "completed",
//   "assignedTo": "Erika",
//   "createdAt": "2024-01-15T10:30:00Z",
//   "updatedAt": "2024-01-15T11:00:00Z"
// }
func (c *ReservedOrderController) CompleteOrder(w http.ResponseWriter, r *http.Request) {
	log.Printf("📥 CompleteOrder: Received %s request to %s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		log.Printf("❌ CompleteOrder: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract order ID from URL path
	// Path format: /admin/reserved-orders/{id}/complete
	path := strings.TrimPrefix(r.URL.Path, "/admin/reserved-orders/")
	if path == "" {
		http.Error(w, "order id parameter is required", http.StatusBadRequest)
		return
	}

	// Extract ID (remove /complete suffix)
	idStr := strings.TrimSuffix(path, "/complete")
	if idStr == path {
		http.Error(w, "invalid path format", http.StatusBadRequest)
		return
	}

	orderID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Printf("❌ CompleteOrder: Invalid order id: %s", idStr)
		http.Error(w, "invalid order id parameter", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	order, err := c.repository.Complete(ctx, orderID)
	if err != nil {
		log.Printf("❌ CompleteOrder: Error completing order: %v", err)
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "not in reserved status") {
			http.Error(w, errMsg, http.StatusNotFound)
			return
		}
		if strings.Contains(errMsg, "insufficient reserved stock") {
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to complete order: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ CompleteOrder: Successfully completed order id=%d", orderID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(order); err != nil {
		log.Printf("❌ CompleteOrder: Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

