package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"armario-mascota-me/models"
	"armario-mascota-me/repository"
)

// FinanceTransactionController handles HTTP requests for finance transactions
type FinanceTransactionController struct {
	repository repository.FinanceTransactionRepositoryInterface
}

// NewFinanceTransactionController creates a new FinanceTransactionController
func NewFinanceTransactionController(repo repository.FinanceTransactionRepositoryInterface) *FinanceTransactionController {
	return &FinanceTransactionController{
		repository: repo,
	}
}

// Create handles POST /admin/finance-transactions
// Example request:
// POST /admin/finance-transactions
// {
//   "type": "expense",
//   "source": "manual",
//   "sourceId": 0,
//   "occurredAt": "2026-01-04T10:30:00Z",
//   "amount": 50000,
//   "destination": "Nequi",
//   "category": "materiales",
//   "notes": "Compra de materiales"
// }
// Example response:
// {
//   "id": 1,
//   "type": "expense",
//   "source": "manual",
//   "sourceId": 0,
//   "occurredAt": "2026-01-04T10:30:00Z",
//   "amount": 50000,
//   "destination": "Nequi",
//   "category": "materiales",
//   "notes": "Compra de materiales",
//   "createdAt": "2026-01-04T10:30:00Z"
// }
func (c *FinanceTransactionController) Create(w http.ResponseWriter, r *http.Request) {
	log.Printf("📥 CreateFinanceTransaction: Received %s request to %s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		log.Printf("❌ CreateFinanceTransaction: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.CreateFinanceTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ CreateFinanceTransaction: Failed to decode request body: %v", err)
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Type != "income" && req.Type != "expense" {
		log.Printf("❌ CreateFinanceTransaction: Invalid type: %s", req.Type)
		http.Error(w, "type must be 'income' or 'expense'", http.StatusBadRequest)
		return
	}

	if req.Amount <= 0 {
		log.Printf("❌ CreateFinanceTransaction: amount must be greater than 0: %d", req.Amount)
		http.Error(w, "amount must be greater than 0", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Source) == "" {
		log.Printf("❌ CreateFinanceTransaction: source is required")
		http.Error(w, "source is required", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Destination) == "" {
		log.Printf("❌ CreateFinanceTransaction: destination is required")
		http.Error(w, "destination is required", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	transaction, err := c.repository.Create(ctx, &req)
	if err != nil {
		log.Printf("❌ CreateFinanceTransaction: Error creating transaction: %v", err)
		errMsg := err.Error()
		if strings.Contains(errMsg, "Invalid") || strings.Contains(errMsg, "required") {
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to create finance transaction: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ CreateFinanceTransaction: Successfully created transaction id=%d", transaction.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(transaction); err != nil {
		log.Printf("❌ CreateFinanceTransaction: Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

