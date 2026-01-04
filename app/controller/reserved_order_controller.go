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
	"armario-mascota-me/utils"
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

// RemoveItem handles DELETE /admin/reserved-orders/:id/items/:itemId
// Removes an item from a reserved order and releases stock reservation
// Example request:
// DELETE /admin/reserved-orders/1/items/123
// Example response:
// {
//   "message": "Item removed successfully"
// }
func (c *ReservedOrderController) RemoveItem(w http.ResponseWriter, r *http.Request) {
	log.Printf("📥 RemoveItem: Received %s request to %s", r.Method, r.URL.Path)

	if r.Method != http.MethodDelete {
		log.Printf("❌ RemoveItem: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract order ID and item ID from URL path
	// Path format: /admin/reserved-orders/{orderId}/items/{itemId}
	path := strings.TrimPrefix(r.URL.Path, "/admin/reserved-orders/")
	if path == "" {
		http.Error(w, "order id parameter is required", http.StatusBadRequest)
		return
	}

	// Split path to get orderId and itemId
	// Expected format: {orderId}/items/{itemId}
	parts := strings.Split(path, "/")
	if len(parts) != 3 || parts[1] != "items" {
		http.Error(w, "invalid path format. Expected: /admin/reserved-orders/{orderId}/items/{itemId}", http.StatusBadRequest)
		return
	}

	orderID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		log.Printf("❌ RemoveItem: Invalid order id: %s", parts[0])
		http.Error(w, "invalid order id parameter", http.StatusBadRequest)
		return
	}

	itemID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		log.Printf("❌ RemoveItem: Invalid item id: %s", parts[2])
		http.Error(w, "invalid item id parameter", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	err = c.repository.RemoveItem(ctx, orderID, itemID)
	if err != nil {
		log.Printf("❌ RemoveItem: Error removing item: %v", err)
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") {
			http.Error(w, errMsg, http.StatusNotFound)
			return
		}
		if strings.Contains(errMsg, "not in reserved status") {
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to remove item: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ RemoveItem: Successfully removed item_id=%d from order_id=%d", itemID, orderID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]string{"message": "Item removed successfully"}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("❌ RemoveItem: Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// UpdateOrder handles PUT /admin/reserved-orders/:id
// Updates a reserved order with its lines
// If qty = 0 in a line, that line will be deleted and stock will be released
// Example request:
// PUT /admin/reserved-orders/1
// {
//   "id": 1,
//   "status": "reserved",
//   "assignedTo": "Erika",
//   "orderType": "retail",
//   "customerName": "Pepito",
//   "customerPhone": "3152956953",
//   "notes": "Mayorista",
//   "lines": [
//     {
//       "id": 1,
//       "reservedOrderId": 1,
//       "itemId": 27,
//       "qty": 1
//     },
//     {
//       "id": 2,
//       "reservedOrderId": 1,
//       "itemId": 28,
//       "qty": 0  // This will delete the line and release stock
//     }
//   ]
// }
// Example response:
// {
//   "id": 1,
//   "status": "reserved",
//   "assignedTo": "Erika",
//   "orderType": "retail",
//   "customerName": "Pepito",
//   "customerPhone": "3152956953",
//   "notes": "Mayorista",
//   "createdAt": "2024-01-15T10:30:00Z",
//   "updatedAt": "2024-01-15T10:30:00Z",
//   "lines": [
//     {
//       "id": 1,
//       "reservedOrderId": 1,
//       "itemId": 27,
//       "qty": 1,
//       "unitPrice": 50000,
//       "createdAt": "2024-01-15T10:30:00Z"
//     }
//   ],
//   "total": 50000
// }
func (c *ReservedOrderController) UpdateOrder(w http.ResponseWriter, r *http.Request) {
	log.Printf("📥 UpdateOrder: Received %s request to %s", r.Method, r.URL.Path)

	if r.Method != http.MethodPut {
		log.Printf("❌ UpdateOrder: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract order ID from URL path
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
		log.Printf("❌ UpdateOrder: Invalid order id: %s", path)
		http.Error(w, "invalid order id parameter", http.StatusBadRequest)
		return
	}

	var req models.UpdateReservedOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ UpdateOrder: Failed to decode request body: %v", err)
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate that the order ID in the body matches the URL
	if req.ID != orderID {
		log.Printf("❌ UpdateOrder: Order ID mismatch: URL=%d, body=%d", orderID, req.ID)
		http.Error(w, "order id in URL does not match order id in body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if strings.TrimSpace(req.AssignedTo) == "" {
		log.Printf("❌ UpdateOrder: assignedTo is required")
		http.Error(w, "assignedTo is required", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.OrderType) == "" {
		log.Printf("❌ UpdateOrder: orderType is required")
		http.Error(w, "orderType is required", http.StatusBadRequest)
		return
	}

	// Validate lines - qty = 0 means delete, qty > 0 means update/add
	for i, line := range req.Lines {
		if line.Qty < 0 {
			log.Printf("❌ UpdateOrder: Line %d has invalid qty: %d (qty must be >= 0)", i, line.Qty)
			http.Error(w, fmt.Sprintf("line %d: qty must be >= 0 (0 to delete, >0 to update/add)", i), http.StatusBadRequest)
			return
		}
		if line.ReservedOrderID != orderID {
			log.Printf("❌ UpdateOrder: Line %d reservedOrderId mismatch: %d != %d", i, line.ReservedOrderID, orderID)
			http.Error(w, fmt.Sprintf("line %d: reservedOrderId must match order id", i), http.StatusBadRequest)
			return
		}
	}

	ctx := context.Background()
	order, err := c.repository.UpdateOrder(ctx, &req)
	if err != nil {
		log.Printf("❌ UpdateOrder: Error updating order: %v", err)
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") {
			http.Error(w, errMsg, http.StatusNotFound)
			return
		}
		if strings.Contains(errMsg, "not in reserved status") {
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		if strings.Contains(errMsg, "insufficient stock") {
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to update order: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ UpdateOrder: Successfully updated order_id=%d", orderID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(order); err != nil {
		log.Printf("❌ UpdateOrder: Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// UpdateItemQuantity handles PUT /admin/reserved-orders/:orderId/items/:itemId
// Updates the quantity of an item in a reserved order
// If qty = 0, the item will be deleted from the order and stock will be released
// Example request:
// PUT /admin/reserved-orders/1/items/123
// {
//   "qty": 3
// }
// Or to delete:
// {
//   "qty": 0
// }
// Example response:
// {
//   "id": 1,
//   "reservedOrderId": 1,
//   "itemId": 123,
//   "qty": 3,
//   "unitPrice": 50000,
//   "createdAt": "2024-01-15T10:30:00Z"
// }
func (c *ReservedOrderController) UpdateItemQuantity(w http.ResponseWriter, r *http.Request) {
	log.Printf("📥 UpdateItemQuantity: Received %s request to %s", r.Method, r.URL.Path)

	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		log.Printf("❌ UpdateItemQuantity: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract order ID and item ID from URL path
	// Path format: /admin/reserved-orders/{orderId}/items/{itemId}
	path := strings.TrimPrefix(r.URL.Path, "/admin/reserved-orders/")
	if path == "" {
		http.Error(w, "order id parameter is required", http.StatusBadRequest)
		return
	}

	// Split path to get orderId and itemId
	// Expected format: {orderId}/items/{itemId}
	parts := strings.Split(path, "/")
	if len(parts) != 3 || parts[1] != "items" {
		http.Error(w, "invalid path format. Expected: /admin/reserved-orders/{orderId}/items/{itemId}", http.StatusBadRequest)
		return
	}

	orderID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		log.Printf("❌ UpdateItemQuantity: Invalid order id: %s", parts[0])
		http.Error(w, "invalid order id parameter", http.StatusBadRequest)
		return
	}

	itemID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		log.Printf("❌ UpdateItemQuantity: Invalid item id: %s", parts[2])
		http.Error(w, "invalid item id parameter", http.StatusBadRequest)
		return
	}

	var req models.UpdateItemQuantityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ UpdateItemQuantity: Failed to decode request body: %v", err)
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.Qty < 0 {
		log.Printf("❌ UpdateItemQuantity: Invalid qty: %d", req.Qty)
		http.Error(w, "qty must be >= 0 (0 to delete, >0 to update)", http.StatusBadRequest)
		return
	}

	// If qty is 0, treat as deletion
	if req.Qty == 0 {
		ctx := context.Background()
		err = c.repository.RemoveItem(ctx, orderID, itemID)
		if err != nil {
			log.Printf("❌ UpdateItemQuantity: Error removing item: %v", err)
			errMsg := err.Error()
			if strings.Contains(errMsg, "not found") {
				http.Error(w, errMsg, http.StatusNotFound)
				return
			}
			if strings.Contains(errMsg, "not in reserved status") {
				http.Error(w, errMsg, http.StatusBadRequest)
				return
			}
			http.Error(w, fmt.Sprintf("Failed to remove item: %v", err), http.StatusInternalServerError)
			return
		}

		log.Printf("✅ UpdateItemQuantity: Successfully removed item_id=%d from order_id=%d (qty=0)", itemID, orderID)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]string{"message": "Item removed successfully (qty=0)"}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("❌ UpdateItemQuantity: Error encoding response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
		return
	}

	ctx := context.Background()
	line, err := c.repository.UpdateItemQuantity(ctx, orderID, itemID, req.Qty)
	if err != nil {
		log.Printf("❌ UpdateItemQuantity: Error updating item quantity: %v", err)
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") {
			http.Error(w, errMsg, http.StatusNotFound)
			return
		}
		if strings.Contains(errMsg, "not in reserved status") {
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		if strings.Contains(errMsg, "insufficient stock") {
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to update item quantity: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ UpdateItemQuantity: Successfully updated item_id=%d quantity to %d in order_id=%d", itemID, req.Qty, orderID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(line); err != nil {
		log.Printf("❌ UpdateItemQuantity: Error encoding response: %v", err)
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

// GetSeparatedCarts handles GET /admin/reserved-orders/separated
// Returns all reserved orders with complete item information including design asset details and image endpoints
// Example response:
// {
//   "carts": [
//     {
//       "id": 1,
//       "status": "reserved",
//       "assignedTo": "Erika",
//       "orderType": "detal",
//       "customerName": "Juan Pérez",
//       "customerPhone": "+1234567890",
//       "notes": "Cliente VIP",
//       "createdAt": "2024-01-15T10:30:00Z",
//       "updatedAt": "2024-01-15T10:30:00Z",
//       "lines": [
//         {
//           "id": 1,
//           "reservedOrderId": 1,
//           "itemId": 123,
//           "qty": 2,
//           "unitPrice": 50000,
//           "createdAt": "2024-01-15T10:30:00Z",
//           "item": {
//             "id": 123,
//             "sku": "MN_ABC123",
//             "size": "MN",
//             "price": 50000,
//             "stockTotal": 10,
//             "stockReserved": 2,
//             "designAssetId": 45,
//             "description": "Hoodie con diseño especial",
//             "colorPrimary": "BL",
//             "colorSecondary": "NG",
//             "hoodieType": "BE",
//             "imageType": "IT",
//             "decoId": "123",
//             "decoBase": "C",
//             "colorPrimaryLabel": "negro",
//             "colorSecondaryLabel": "azul cielo",
//             "hoodieTypeLabel": "buso tipo esqueleto",
//             "imageTypeLabel": "buso pequeño (tallas mini - intermedio)",
//             "decoBaseLabel": "Círculo",
//             "imageUrlThumb": "/admin/design-assets/pending/45/image?size=thumb",
//             "imageUrlMedium": "/admin/design-assets/pending/45/image?size=medium"
//           }
//         }
//       ],
//       "total": 100000
//     }
//   ]
// }
func (c *ReservedOrderController) GetSeparatedCarts(w http.ResponseWriter, r *http.Request) {
	log.Printf("📥 GetSeparatedCarts: Received %s request to %s", r.Method, r.URL.Path)

	if r.Method != http.MethodGet {
		log.Printf("❌ GetSeparatedCarts: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()
	carts, err := c.repository.GetAllWithFullItems(ctx)
	if err != nil {
		log.Printf("❌ GetSeparatedCarts: Error fetching carts: %v", err)
		http.Error(w, fmt.Sprintf("Failed to fetch carts: %v", err), http.StatusInternalServerError)
		return
	}

	// Build image endpoints and apply mappings for readable labels
	for i := range carts {
		for j := range carts[i].Lines {
			item := &carts[i].Lines[j].Item
			designAssetID := item.DesignAssetID
			
			// Build image endpoints
			item.ImageUrlThumb = fmt.Sprintf("/admin/design-assets/pending/%d/image?size=thumb", designAssetID)
			item.ImageUrlMedium = fmt.Sprintf("/admin/design-assets/pending/%d/image?size=medium", designAssetID)
			
			// Apply mappings for readable labels
			item.ColorPrimaryLabel = utils.MapCodeToColor(item.ColorPrimary)
			item.ColorSecondaryLabel = utils.MapCodeToColor(item.ColorSecondary)
			item.HoodieTypeLabel = utils.MapCodeToHoodieType(item.HoodieType)
			item.ImageTypeLabel = utils.MapCodeToImageType(item.ImageType)
			item.DecoBaseLabel = utils.MapCodeToDecoBase(item.DecoBase)
		}
	}

	log.Printf("✅ GetSeparatedCarts: Successfully fetched %d carts", len(carts))

	response := models.SeparatedCartsResponse{
		Carts: carts,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("❌ GetSeparatedCarts: Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

