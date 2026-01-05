package models

// FinanceTransaction represents a financial transaction in the database
type FinanceTransaction struct {
	ID         int64  `json:"id"`
	Type       string `json:"type"` // 'income' or 'expense'
	Source     string `json:"source"`
	SourceID   int64  `json:"sourceId"`
	OccurredAt string `json:"occurredAt"`
	Amount     int64  `json:"amount"`
	Destination string `json:"destination"`
	Category   string `json:"category,omitempty"`
	Notes      string `json:"notes,omitempty"`
	CreatedAt  string `json:"createdAt"`
}

// CreateFinanceTransactionRequest represents the request body for creating a finance transaction
// Example: {
//   "type": "expense",
//   "source": "manual",
//   "sourceId": 0,
//   "occurredAt": "2026-01-04T10:30:00Z",
//   "amount": 50000,
//   "destination": "Nequi",
//   "category": "materiales",
//   "notes": "Compra de materiales"
// }
type CreateFinanceTransactionRequest struct {
	Type       string `json:"type"`       // 'income' or 'expense'
	Source     string `json:"source"`     // e.g., "manual", "sale", "refund", etc.
	SourceID   int64  `json:"sourceId"`   // ID of the source record (0 if not applicable)
	OccurredAt string `json:"occurredAt,omitempty"` // ISO 8601 timestamp (optional, defaults to now)
	Amount     int64  `json:"amount"`
	Destination string `json:"destination"`
	Category   string `json:"category,omitempty"`
	Notes      string `json:"notes,omitempty"`
}

