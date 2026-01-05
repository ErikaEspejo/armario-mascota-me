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

// Create handles POST /admin/finance/transactions
// Example request:
// POST /admin/finance/transactions
// {
//   "type": "expense",
//   "amount": 45000,
//   "destination": "Caja",
//   "category": "materiales",
//   "counterparty": "Proveedor telas",
//   "notes": "Franela 10m"
// }
// Example response:
// {
//   "id": 1,
//   "type": "expense",
//   "source": "manual",
//   "occurredAt": "2026-01-04T15:20:00Z",
//   "amount": 45000,
//   "destination": "Caja",
//   "category": "materiales",
//   "counterparty": "Proveedor telas",
//   "notes": "Franela 10m",
//   "createdAt": "2026-01-04T15:20:00Z"
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

	if strings.TrimSpace(req.Destination) == "" {
		log.Printf("❌ CreateFinanceTransaction: destination is required")
		http.Error(w, "destination is required", http.StatusBadRequest)
		return
	}

	// Note: source and sourceId are automatically set to 'manual' and NULL in the repository
	// The request body doesn't need to include them

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

// List handles GET /admin/finance/transactions
// Query params: from, to, type, source, destination, category, q, limit, cursor
// Example response:
// {
//   "transactions": [
//     {
//       "id": 101,
//       "occurredAt": "2026-01-04T15:20:00Z",
//       "type": "income",
//       "amount": 100000,
//       "destination": "Nequi",
//       "category": "venta",
//       "source": "sale",
//       "sourceId": 10,
//       "counterparty": "Juan Pérez",
//       "notes": "Pedido #3"
//     }
//   ],
//   "pagination": { "limit": 50, "nextCursor": "..." }
// }
func (c *FinanceTransactionController) List(w http.ResponseWriter, r *http.Request) {
	log.Printf("📥 ListFinanceTransactions: Received %s request to %s", r.Method, r.URL.Path)

	if r.Method != http.MethodGet {
		log.Printf("❌ ListFinanceTransactions: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	req := &models.FinanceTransactionListRequest{}

	// Parse query parameters
	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		// Validate date format
		_, err := time.Parse("2006-01-02", fromStr)
		if err != nil {
			log.Printf("❌ ListFinanceTransactions: Invalid from date format: %s", fromStr)
			http.Error(w, "Invalid from date format. Use YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		req.From = &fromStr
	}

	if toStr := r.URL.Query().Get("to"); toStr != "" {
		// Validate date format
		_, err := time.Parse("2006-01-02", toStr)
		if err != nil {
			log.Printf("❌ ListFinanceTransactions: Invalid to date format: %s", toStr)
			http.Error(w, "Invalid to date format. Use YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		req.To = &toStr
	}

	if typeStr := r.URL.Query().Get("type"); typeStr != "" {
		if typeStr != "income" && typeStr != "expense" {
			log.Printf("❌ ListFinanceTransactions: Invalid type: %s", typeStr)
			http.Error(w, "type must be 'income' or 'expense'", http.StatusBadRequest)
			return
		}
		req.Type = &typeStr
	}

	if sourceStr := r.URL.Query().Get("source"); sourceStr != "" {
		req.Source = &sourceStr
	}

	if destinationStr := r.URL.Query().Get("destination"); destinationStr != "" {
		req.Destination = &destinationStr
	}

	if categoryStr := r.URL.Query().Get("category"); categoryStr != "" {
		req.Category = &categoryStr
	}

	if qStr := r.URL.Query().Get("q"); qStr != "" {
		req.Q = &qStr
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			log.Printf("❌ ListFinanceTransactions: Invalid limit: %s", limitStr)
			http.Error(w, "limit must be a positive integer", http.StatusBadRequest)
			return
		}
		if limit > 200 {
			limit = 200
		}
		req.Limit = limit
	}

	if cursorStr := r.URL.Query().Get("cursor"); cursorStr != "" {
		req.Cursor = &cursorStr
	}

	ctx := context.Background()
	response, err := c.repository.List(ctx, req)
	if err != nil {
		log.Printf("❌ ListFinanceTransactions: Error fetching transactions: %v", err)
		errMsg := err.Error()
		if strings.Contains(errMsg, "Invalid") || strings.Contains(errMsg, "invalid") {
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to fetch transactions: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ ListFinanceTransactions: Successfully fetched %d transactions", len(response.Transactions))

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("❌ ListFinanceTransactions: Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// Summary handles GET /admin/finance/summary
// Query params: from (optional YYYY-MM-DD), to (optional YYYY-MM-DD)
// Example response:
// {
//   "currency": "COP",
//   "balanceAllTime": 350000,
//   "byDestinationAllTime": [
//     { "destination": "Nequi", "balance": 200000 },
//     { "destination": "Caja", "balance": 150000 }
//   ],
//   "range": {
//     "from": "2026-01-01",
//     "to": "2026-01-31",
//     "openingBalance": 120000,
//     "income": 500000,
//     "expense": 270000,
//     "net": 230000,
//     "closingBalance": 350000
//   },
//   "byDestinationRange": [
//     { "destination": "Nequi", "income": 300000, "expense": 100000, "net": 200000 },
//     { "destination": "Caja", "income": 200000, "expense": 170000, "net": 30000 }
//   ]
// }
func (c *FinanceTransactionController) Summary(w http.ResponseWriter, r *http.Request) {
	log.Printf("📥 SummaryFinanceTransactions: Received %s request to %s", r.Method, r.URL.Path)

	if r.Method != http.MethodGet {
		log.Printf("❌ SummaryFinanceTransactions: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var from, to *string

	// Parse query parameters
	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		// Validate date format
		_, err := time.Parse("2006-01-02", fromStr)
		if err != nil {
			log.Printf("❌ SummaryFinanceTransactions: Invalid from date format: %s", fromStr)
			http.Error(w, "Invalid from date format. Use YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		from = &fromStr
	}

	if toStr := r.URL.Query().Get("to"); toStr != "" {
		// Validate date format
		_, err := time.Parse("2006-01-02", toStr)
		if err != nil {
			log.Printf("❌ SummaryFinanceTransactions: Invalid to date format: %s", toStr)
			http.Error(w, "Invalid to date format. Use YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		to = &toStr
	}

	// Both from and to must be provided together for range calculations
	if (from != nil && to == nil) || (from == nil && to != nil) {
		log.Printf("❌ SummaryFinanceTransactions: Both from and to must be provided together")
		http.Error(w, "Both from and to must be provided together for range calculations", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	response, err := c.repository.Summary(ctx, from, to)
	if err != nil {
		log.Printf("❌ SummaryFinanceTransactions: Error calculating summary: %v", err)
		errMsg := err.Error()
		if strings.Contains(errMsg, "Invalid") || strings.Contains(errMsg, "invalid") {
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to calculate summary: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ SummaryFinanceTransactions: Successfully calculated summary")

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("❌ SummaryFinanceTransactions: Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// Dashboard handles GET /admin/finance/dashboard
// Query params: period (month|quarter|year), from (YYYY-MM-DD), to (YYYY-MM-DD), compareWith (previous|last_year)
// Example response: See FinanceDashboardResponse structure
func (c *FinanceTransactionController) Dashboard(w http.ResponseWriter, r *http.Request) {
	log.Printf("📥 DashboardFinanceTransactions: Received %s request to %s", r.Method, r.URL.Path)

	if r.Method != http.MethodGet {
		log.Printf("❌ DashboardFinanceTransactions: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	req := &models.FinanceDashboardRequest{}

	// Parse query parameters
	if periodStr := r.URL.Query().Get("period"); periodStr != "" {
		if periodStr != "month" && periodStr != "quarter" && periodStr != "year" {
			log.Printf("❌ DashboardFinanceTransactions: Invalid period: %s", periodStr)
			http.Error(w, "period must be 'month', 'quarter', or 'year'", http.StatusBadRequest)
			return
		}
		req.Period = &periodStr
	}

	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		// Validate date format
		_, err := time.Parse("2006-01-02", fromStr)
		if err != nil {
			log.Printf("❌ DashboardFinanceTransactions: Invalid from date format: %s", fromStr)
			http.Error(w, "Invalid from date format. Use YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		req.From = &fromStr
	}

	if toStr := r.URL.Query().Get("to"); toStr != "" {
		// Validate date format
		_, err := time.Parse("2006-01-02", toStr)
		if err != nil {
			log.Printf("❌ DashboardFinanceTransactions: Invalid to date format: %s", toStr)
			http.Error(w, "Invalid to date format. Use YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		req.To = &toStr
	}

	// Validate that if from/to are provided, both must be provided
	if (req.From != nil && req.To == nil) || (req.From == nil && req.To != nil) {
		log.Printf("❌ DashboardFinanceTransactions: Both from and to must be provided together")
		http.Error(w, "Both from and to must be provided together", http.StatusBadRequest)
		return
	}

	if compareWithStr := r.URL.Query().Get("compareWith"); compareWithStr != "" {
		if compareWithStr != "previous" && compareWithStr != "last_year" {
			log.Printf("❌ DashboardFinanceTransactions: Invalid compareWith: %s", compareWithStr)
			http.Error(w, "compareWith must be 'previous' or 'last_year'", http.StatusBadRequest)
			return
		}
		req.CompareWith = &compareWithStr
	}

	ctx := context.Background()
	response, err := c.repository.Dashboard(ctx, req)
	if err != nil {
		log.Printf("❌ DashboardFinanceTransactions: Error calculating dashboard: %v", err)
		errMsg := err.Error()
		if strings.Contains(errMsg, "Invalid") || strings.Contains(errMsg, "invalid") {
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to calculate dashboard: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ DashboardFinanceTransactions: Successfully calculated dashboard")

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("❌ DashboardFinanceTransactions: Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

