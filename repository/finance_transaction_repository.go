package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"armario-mascota-me/db"
	"armario-mascota-me/models"
)

// FinanceTransactionRepository handles database operations for finance transactions
type FinanceTransactionRepository struct{}

// NewFinanceTransactionRepository creates a new FinanceTransactionRepository
func NewFinanceTransactionRepository() *FinanceTransactionRepository {
	return &FinanceTransactionRepository{}
}

// Ensure FinanceTransactionRepository implements FinanceTransactionRepositoryInterface
var _ FinanceTransactionRepositoryInterface = (*FinanceTransactionRepository)(nil)

// Create creates a new finance transaction
func (r *FinanceTransactionRepository) Create(ctx context.Context, req *models.CreateFinanceTransactionRequest) (*models.FinanceTransaction, error) {
	log.Printf("💰 CreateFinanceTransaction: type=%s, source=%s, amount=%d", req.Type, req.Source, req.Amount)

	// Validate type
	if req.Type != "income" && req.Type != "expense" {
		log.Printf("❌ CreateFinanceTransaction: Invalid type: %s", req.Type)
		return nil, fmt.Errorf("type must be 'income' or 'expense'")
	}

	// Validate amount
	if req.Amount <= 0 {
		log.Printf("❌ CreateFinanceTransaction: Invalid amount: %d", req.Amount)
		return nil, fmt.Errorf("amount must be greater than 0")
	}

	// Validate source
	if req.Source == "" {
		log.Printf("❌ CreateFinanceTransaction: Source is required")
		return nil, fmt.Errorf("source is required")
	}

	// Validate destination
	if req.Destination == "" {
		log.Printf("❌ CreateFinanceTransaction: Destination is required")
		return nil, fmt.Errorf("destination is required")
	}

	// Parse occurredAt or use current time
	var occurredAt time.Time
	if req.OccurredAt != "" {
		var err error
		occurredAt, err = time.Parse(time.RFC3339, req.OccurredAt)
		if err != nil {
			log.Printf("❌ CreateFinanceTransaction: Invalid occurredAt format: %s", req.OccurredAt)
			return nil, fmt.Errorf("invalid occurredAt format, use RFC3339 (e.g., 2006-01-02T15:04:05Z07:00): %w", err)
		}
	} else {
		occurredAt = time.Now()
	}

	// Insert into finance_transactions
	queryInsert := `
		INSERT INTO finance_transactions (type, source, source_id, occurred_at, amount, destination, category, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, type, source, source_id, occurred_at, amount, destination, category, notes, created_at
	`

	var transaction models.FinanceTransaction
	var category, notes sql.NullString

	err := db.DB.QueryRowContext(ctx, queryInsert,
		req.Type,
		req.Source,
		req.SourceID,
		occurredAt,
		req.Amount,
		req.Destination,
		sql.NullString{String: req.Category, Valid: req.Category != ""},
		sql.NullString{String: req.Notes, Valid: req.Notes != ""},
	).Scan(
		&transaction.ID,
		&transaction.Type,
		&transaction.Source,
		&transaction.SourceID,
		&transaction.OccurredAt,
		&transaction.Amount,
		&transaction.Destination,
		&category,
		&notes,
		&transaction.CreatedAt,
	)

	if err != nil {
		log.Printf("❌ CreateFinanceTransaction: Error inserting transaction: %v", err)
		return nil, fmt.Errorf("failed to insert finance transaction: %w", err)
	}

	if category.Valid {
		transaction.Category = category.String
	}
	if notes.Valid {
		transaction.Notes = notes.String
	}

	log.Printf("✅ CreateFinanceTransaction: Successfully created transaction id=%d", transaction.ID)
	return &transaction, nil
}

