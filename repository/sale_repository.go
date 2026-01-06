package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"armario-mascota-me/db"
	"armario-mascota-me/models"
	"armario-mascota-me/pricing"
)

// SaleRepository handles database operations for sales
type SaleRepository struct{}

// NewSaleRepository creates a new SaleRepository
func NewSaleRepository() *SaleRepository {
	return &SaleRepository{}
}

// Ensure SaleRepository implements SaleRepositoryInterface
var _ SaleRepositoryInterface = (*SaleRepository)(nil)

// Sell sells a reserved order by completing it, creating a sale record, and recording a financial transaction
// All operations are performed atomically in a single transaction
func (r *SaleRepository) Sell(ctx context.Context, reservedOrderID int64, req *models.SellRequest) (*models.Sale, error) {
	log.Printf("📦 Sell: Selling reserved order id=%d", reservedOrderID)

	// Start transaction
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("❌ Sell: Error starting transaction: %v", err)
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Lock order and validate it exists and is in 'reserved' status
	var orderStatus, customerName string
	var customerNameNull sql.NullString
	queryOrder := `
		SELECT status, customer_name 
		FROM reserved_orders 
		WHERE id = $1 
		FOR UPDATE
	`
	err = tx.QueryRowContext(ctx, queryOrder, reservedOrderID).Scan(&orderStatus, &customerNameNull)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("❌ Sell: Order not found: id=%d", reservedOrderID)
			return nil, fmt.Errorf("order not found")
		}
		log.Printf("❌ Sell: Error fetching order: %v", err)
		return nil, fmt.Errorf("failed to fetch order: %w", err)
	}

	if customerNameNull.Valid {
		customerName = customerNameNull.String
	}

	if orderStatus != "reserved" {
		log.Printf("❌ Sell: Order not in reserved status: status=%s", orderStatus)
		return nil, fmt.Errorf("order not in reserved status")
	}

	// Check if sale already exists for this reserved_order_id
	var existingSaleID int64
	queryExistingSale := `SELECT id FROM sales WHERE reserved_order_id = $1`
	err = tx.QueryRowContext(ctx, queryExistingSale, reservedOrderID).Scan(&existingSaleID)
	if err != sql.ErrNoRows {
		if err == nil {
			log.Printf("❌ Sell: Sale already exists for reserved_order_id=%d, sale_id=%d", reservedOrderID, existingSaleID)
			return nil, fmt.Errorf("order already has a sale associated")
		}
		log.Printf("❌ Sell: Error checking existing sale: %v", err)
		return nil, fmt.Errorf("failed to check existing sale: %w", err)
	}

	// Get all lines for this order
	queryLines := `SELECT item_id, qty FROM reserved_order_lines WHERE reserved_order_id = $1`
	rows, err := tx.QueryContext(ctx, queryLines, reservedOrderID)
	if err != nil {
		log.Printf("❌ Sell: Error fetching lines: %v", err)
		return nil, fmt.Errorf("failed to fetch order lines: %w", err)
	}
	defer rows.Close()

	type lineInfo struct {
		itemID int64
		qty    int
	}
	var lines []lineInfo

	for rows.Next() {
		var l lineInfo
		if err := rows.Scan(&l.itemID, &l.qty); err != nil {
			log.Printf("❌ Sell: Error scanning line: %v", err)
			continue
		}
		lines = append(lines, l)
	}

	if err := rows.Err(); err != nil {
		log.Printf("❌ Sell: Error iterating lines: %v", err)
		return nil, fmt.Errorf("failed to iterate order lines: %w", err)
	}

	// Calculate final pricing using pricing engine BEFORE completing the sale
	// This will freeze the snapshot by updating unit_price in reserved_order_lines
	pricingEngine := pricing.GetEngine()
	var calculatedTotal int64
	var calculatedOrderType string

	if pricingEngine != nil {
		log.Printf("💰 Sell: Calculating final pricing for order %d", reservedOrderID)
		
		// Note: We need to use a context that can work with the transaction
		// Since pricing engine uses db.DB directly, we'll calculate outside transaction first
		// then update within transaction
		breakdown, err := pricingEngine.CalculateOrderPricing(ctx, reservedOrderID)
		if err != nil {
			log.Printf("❌ Sell: Error calculating pricing: %v", err)
			return nil, fmt.Errorf("failed to calculate pricing: %w", err)
		}

		calculatedTotal = breakdown.Total
		calculatedOrderType = breakdown.OrderType
		log.Printf("💰 Sell: Calculated total=%d, orderType=%s", calculatedTotal, calculatedOrderType)

		// Freeze snapshot: Update unit_price in reserved_order_lines with calculated prices
		// Use effective unit price (lineTotal / qty) to include bundle contributions
		for _, pricingLine := range breakdown.Lines {
			// Calculate effective unit price (includes bundle contributions)
			effectiveUnitPrice := pricingLine.UnitPrice
			if pricingLine.Qty > 0 {
				effectiveUnitPrice = pricingLine.LineTotal / int64(pricingLine.Qty)
			}
			
			queryUpdatePrice := `
				UPDATE reserved_order_lines
				SET unit_price = $1
				WHERE id = $2
			`
			_, err = tx.ExecContext(ctx, queryUpdatePrice, effectiveUnitPrice, pricingLine.LineID)
			if err != nil {
				log.Printf("❌ Sell: Error freezing price for line %d: %v", pricingLine.LineID, err)
				return nil, fmt.Errorf("failed to freeze pricing snapshot: %w", err)
			}
			log.Printf("💰 Sell: Frozen line %d: qty=%d, lineTotal=%d, effectiveUnitPrice=%d", 
				pricingLine.LineID, pricingLine.Qty, pricingLine.LineTotal, effectiveUnitPrice)
		}
		log.Printf("✅ Sell: Frozen pricing snapshot for all lines")

		// Update order_type in reserved_orders
		queryUpdateOrderType := `
			UPDATE reserved_orders
			SET order_type = $1
			WHERE id = $2
		`
		_, err = tx.ExecContext(ctx, queryUpdateOrderType, strings.ToLower(calculatedOrderType), reservedOrderID)
		if err != nil {
			log.Printf("⚠️ Sell: Failed to update order_type: %v", err)
			// Continue anyway - pricing is more important
		} else {
			log.Printf("✅ Sell: Updated order_type to %s", calculatedOrderType)
		}
	} else {
		log.Printf("⚠️ Sell: Pricing engine not initialized, using request amount_paid")
		calculatedTotal = req.AmountPaid
		calculatedOrderType = "detal" // Default
	}

	// Process each line: validate stock_reserved and deduct stock_total and stock_reserved
	for _, line := range lines {
		// Lock item for update and validate stock_reserved
		var stockReserved int
		queryItem := `SELECT stock_reserved FROM items WHERE id = $1 FOR UPDATE`
		err = tx.QueryRowContext(ctx, queryItem, line.itemID).Scan(&stockReserved)
		if err != nil {
			log.Printf("❌ Sell: Error fetching item stock: %v", err)
			return nil, fmt.Errorf("failed to fetch item stock: %w", err)
		}

		if stockReserved < line.qty {
			log.Printf("❌ Sell: Insufficient reserved stock: reserved=%d, required=%d", stockReserved, line.qty)
			return nil, fmt.Errorf("insufficient reserved stock: reserved %d, required %d", stockReserved, line.qty)
		}

		// Deduct stock_total and stock_reserved
		queryUpdateStock := `
			UPDATE items
			SET stock_total = stock_total - $1,
			    stock_reserved = stock_reserved - $1
			WHERE id = $2
		`
		_, err = tx.ExecContext(ctx, queryUpdateStock, line.qty, line.itemID)
		if err != nil {
			log.Printf("❌ Sell: Error updating stock for item_id=%d: %v", line.itemID, err)
			return nil, fmt.Errorf("failed to deduct stock: %w", err)
		}
	}

	// Update order status to 'completed'
	queryUpdateOrder := `
		UPDATE reserved_orders
		SET status = 'completed', updated_at = NOW()
		WHERE id = $1
	`
	_, err = tx.ExecContext(ctx, queryUpdateOrder, reservedOrderID)
	if err != nil {
		log.Printf("❌ Sell: Error updating order: %v", err)
		return nil, fmt.Errorf("failed to update order: %w", err)
	}

	// Insert into sales
	soldAt := time.Now()
	queryInsertSale := `
		INSERT INTO sales (reserved_order_id, sold_at, customer_name, amount_paid, payment_method, payment_destination, status, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, reserved_order_id, sold_at, customer_name, amount_paid, payment_method, payment_destination, status, notes, created_at
	`

	var sale models.Sale
	var saleCustomerName, saleNotes sql.NullString

	// Use calculated total if pricing engine was used, otherwise use request amount_paid
	amountPaid := req.AmountPaid
	if pricingEngine != nil && calculatedTotal > 0 {
		amountPaid = calculatedTotal
		log.Printf("💰 Sell: Using calculated total %d for amount_paid (request had %d)", calculatedTotal, req.AmountPaid)
	}

	err = tx.QueryRowContext(ctx, queryInsertSale,
		reservedOrderID,
		soldAt,
		sql.NullString{String: customerName, Valid: customerName != ""},
		amountPaid,
		req.PaymentMethod,
		req.PaymentDestination,
		"paid",
		sql.NullString{String: req.Notes, Valid: req.Notes != ""},
	).Scan(
		&sale.ID,
		&sale.ReservedOrderID,
		&sale.SoldAt,
		&saleCustomerName,
		&sale.AmountPaid,
		&sale.PaymentMethod,
		&sale.PaymentDestination,
		&sale.Status,
		&saleNotes,
		&sale.CreatedAt,
	)
	if err != nil {
		log.Printf("❌ Sell: Error inserting sale: %v", err)
		return nil, fmt.Errorf("failed to insert sale: %w", err)
	}

	if saleCustomerName.Valid {
		sale.CustomerName = saleCustomerName.String
	}
	if saleNotes.Valid {
		sale.Notes = saleNotes.String
	}

	// Insert into finance_transactions
	queryInsertTransaction := `
		INSERT INTO finance_transactions (type, source, source_id, occurred_at, amount, destination, category, counterparty, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err = tx.ExecContext(ctx, queryInsertTransaction,
		"income",
		"sale",
		sale.ID,
		soldAt,
		amountPaid, // Use calculated amount_paid
		req.PaymentDestination,
		"venta",
		sql.NullString{}, // counterparty is NULL for sale transactions
		sql.NullString{String: req.Notes, Valid: req.Notes != ""},
	)
	if err != nil {
		log.Printf("❌ Sell: Error inserting finance transaction: %v", err)
		return nil, fmt.Errorf("failed to insert finance transaction: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("❌ Sell: Error committing transaction: %v", err)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("✅ Sell: Successfully sold order id=%d, sale id=%d", reservedOrderID, sale.ID)
	return &sale, nil
}

// GetByID retrieves a sale by ID with its associated order details
func (r *SaleRepository) GetByID(ctx context.Context, saleID int64) (*models.SaleDetailResponse, error) {
	log.Printf("📦 GetByID: Fetching sale id=%d", saleID)

	// Get sale
	querySale := `
		SELECT id, reserved_order_id, sold_at, customer_name, amount_paid, payment_method, payment_destination, status, notes, created_at
		FROM sales
		WHERE id = $1
	`

	var sale models.Sale
	var customerName, notes sql.NullString

	err := db.DB.QueryRowContext(ctx, querySale, saleID).Scan(
		&sale.ID,
		&sale.ReservedOrderID,
		&sale.SoldAt,
		&customerName,
		&sale.AmountPaid,
		&sale.PaymentMethod,
		&sale.PaymentDestination,
		&sale.Status,
		&notes,
		&sale.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("❌ GetByID: Sale not found: id=%d", saleID)
			return nil, fmt.Errorf("sale not found")
		}
		log.Printf("❌ GetByID: Error fetching sale: %v", err)
		return nil, fmt.Errorf("failed to fetch sale: %w", err)
	}

	if customerName.Valid {
		sale.CustomerName = customerName.String
	}
	if notes.Valid {
		sale.Notes = notes.String
	}

	// Get associated order using ReservedOrderRepository
	// We need to get the repository, but we can't import it circularly
	// Instead, we'll fetch the order directly here
	orderRepo := NewReservedOrderRepository()
	order, err := orderRepo.GetByID(ctx, sale.ReservedOrderID)
	if err != nil {
		log.Printf("❌ GetByID: Error fetching order: %v", err)
		return nil, fmt.Errorf("failed to fetch order: %w", err)
	}

	response := &models.SaleDetailResponse{
		Sale:  sale,
		Order: order,
	}

	log.Printf("✅ GetByID: Successfully fetched sale id=%d", saleID)
	return response, nil
}

// List retrieves sales filtered by date range
func (r *SaleRepository) List(ctx context.Context, from, to *string) ([]models.SaleListItem, error) {
	log.Printf("📦 List: Fetching sales (from=%v, to=%v)", from, to)

	query := `
		SELECT id, sold_at, reserved_order_id, customer_name, amount_paid, payment_destination, payment_method
		FROM sales
	`
	var args []interface{}
	argIndex := 1

	if from != nil && *from != "" {
		// Parse date and use start of day (00:00:00)
		fromDate, err := time.Parse("2006-01-02", *from)
		if err != nil {
			return nil, fmt.Errorf("invalid from date format: %w", err)
		}
		query += fmt.Sprintf(" WHERE sold_at >= $%d", argIndex)
		args = append(args, fromDate)
		argIndex++
	}

	if to != nil && *to != "" {
		// Parse date and use end of day (23:59:59.999999)
		toDate, err := time.Parse("2006-01-02", *to)
		if err != nil {
			return nil, fmt.Errorf("invalid to date format: %w", err)
		}
		// Set to end of day
		toDate = time.Date(toDate.Year(), toDate.Month(), toDate.Day(), 23, 59, 59, 999999999, toDate.Location())
		if argIndex == 1 {
			query += " WHERE"
		} else {
			query += " AND"
		}
		query += fmt.Sprintf(" sold_at <= $%d", argIndex)
		args = append(args, toDate)
		argIndex++
	}

	query += " ORDER BY sold_at DESC"

	rows, err := db.DB.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("❌ List: Error fetching sales: %v", err)
		return nil, fmt.Errorf("failed to fetch sales: %w", err)
	}
	defer rows.Close()

	var sales []models.SaleListItem

	for rows.Next() {
		var sale models.SaleListItem
		var customerName sql.NullString

		err := rows.Scan(
			&sale.ID,
			&sale.SoldAt,
			&sale.ReservedOrderID,
			&customerName,
			&sale.AmountPaid,
			&sale.PaymentDestination,
			&sale.PaymentMethod,
		)
		if err != nil {
			log.Printf("❌ List: Error scanning sale: %v", err)
			continue
		}

		if customerName.Valid {
			sale.CustomerName = customerName.String
		}

		sales = append(sales, sale)
	}

	if err := rows.Err(); err != nil {
		log.Printf("❌ List: Error iterating sales: %v", err)
		return nil, fmt.Errorf("failed to iterate sales: %w", err)
	}

	log.Printf("✅ List: Successfully fetched %d sales", len(sales))
	return sales, nil
}

