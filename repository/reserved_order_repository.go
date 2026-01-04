package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"armario-mascota-me/db"
	"armario-mascota-me/models"
	"armario-mascota-me/utils"
)

// ReservedOrderRepository handles database operations for reserved orders
type ReservedOrderRepository struct{}

// NewReservedOrderRepository creates a new ReservedOrderRepository
func NewReservedOrderRepository() *ReservedOrderRepository {
	return &ReservedOrderRepository{}
}

// Ensure ReservedOrderRepository implements ReservedOrderRepositoryInterface
var _ ReservedOrderRepositoryInterface = (*ReservedOrderRepository)(nil)

// Create creates a new reserved order
func (r *ReservedOrderRepository) Create(ctx context.Context, req *models.CreateReservedOrderRequest) (*models.ReservedOrder, error) {
	log.Printf("📦 Create: Creating reserved order for assigned_to=%s, order_type=%s", req.AssignedTo, req.OrderType)

	if strings.TrimSpace(req.AssignedTo) == "" {
		return nil, fmt.Errorf("assigned_to cannot be empty")
	}

	if strings.TrimSpace(req.OrderType) == "" {
		return nil, fmt.Errorf("order_type cannot be empty")
	}

	// Normalize orderType to lowercase
	normalizedOrderType := strings.ToLower(strings.TrimSpace(req.OrderType))

	query := `
		INSERT INTO reserved_orders (status, assigned_to, order_type, customer_name, customer_phone, notes)
		VALUES ('reserved', $1, $2, $3, $4, $5)
		RETURNING id, status, assigned_to, order_type, customer_name, customer_phone, notes, created_at, updated_at
	`

	var order models.ReservedOrder
	var customerName, customerPhone, notes sql.NullString

	err := db.DB.QueryRowContext(ctx, query,
		req.AssignedTo,
		normalizedOrderType,
		sql.NullString{String: req.CustomerName, Valid: req.CustomerName != ""},
		sql.NullString{String: req.CustomerPhone, Valid: req.CustomerPhone != ""},
		sql.NullString{String: req.Notes, Valid: req.Notes != ""},
	).Scan(
		&order.ID,
		&order.Status,
		&order.AssignedTo,
		&order.OrderType,
		&customerName,
		&customerPhone,
		&notes,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if err != nil {
		log.Printf("❌ Create: Error creating reserved order: %v", err)
		return nil, fmt.Errorf("failed to create reserved order: %w", err)
	}

	if customerName.Valid {
		order.CustomerName = customerName.String
	}
	if customerPhone.Valid {
		order.CustomerPhone = customerPhone.String
	}
	if notes.Valid {
		order.Notes = notes.String
	}

	log.Printf("✅ Create: Successfully created reserved order id=%d", order.ID)
	return &order, nil
}

// AddItem adds an item to a reserved order with stock reservation
func (r *ReservedOrderRepository) AddItem(ctx context.Context, orderID int64, itemID int64, qty int) (*models.ReservedOrderLine, error) {
	log.Printf("📦 AddItem: Adding item_id=%d, qty=%d to order_id=%d", itemID, qty, orderID)

	if qty <= 0 {
		return nil, fmt.Errorf("qty must be greater than 0")
	}

	// Start transaction
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("❌ AddItem: Error starting transaction: %v", err)
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Validate order exists and is in 'reserved' status, get order_type
	var orderStatus, orderType string
	queryOrder := `SELECT status, order_type FROM reserved_orders WHERE id = $1`
	err = tx.QueryRowContext(ctx, queryOrder, orderID).Scan(&orderStatus, &orderType)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("❌ AddItem: Order not found: id=%d", orderID)
			return nil, fmt.Errorf("order not found")
		}
		log.Printf("❌ AddItem: Error fetching order: %v", err)
		return nil, fmt.Errorf("failed to fetch order: %w", err)
	}

	if orderStatus != "reserved" {
		log.Printf("❌ AddItem: Order not in reserved status: status=%s", orderStatus)
		return nil, fmt.Errorf("order not in reserved status")
	}

	// Validate item exists and is active, lock it for update
	// Also get hoodie_type and size to calculate correct price
	var stockTotal, stockReserved int
	var itemPrice int64
	var isActive bool
	var itemSize string
	var hoodieType string
	queryItem := `
		SELECT i.stock_total, i.stock_reserved, i.price, i.is_active, i.size,
		       COALESCE(da.hoodie_type, '') as hoodie_type
		FROM items i
		INNER JOIN design_assets da ON i.design_asset_id = da.id
		WHERE i.id = $1
		FOR UPDATE
	`
	err = tx.QueryRowContext(ctx, queryItem, itemID).Scan(&stockTotal, &stockReserved, &itemPrice, &isActive, &itemSize, &hoodieType)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("❌ AddItem: Item not found: id=%d", itemID)
			return nil, fmt.Errorf("item not found")
		}
		log.Printf("❌ AddItem: Error fetching item: %v", err)
		return nil, fmt.Errorf("failed to fetch item: %w", err)
	}

	if !isActive {
		log.Printf("❌ AddItem: Item is not active: id=%d", itemID)
		return nil, fmt.Errorf("item not found or inactive")
	}

	// Validate stock availability
	available := stockTotal - stockReserved
	if available < qty {
		log.Printf("❌ AddItem: Insufficient stock: available=%d, requested=%d", available, qty)
		return nil, fmt.Errorf("insufficient stock: available %d, requested %d", available, qty)
	}

	// Calculate price based on order type (retail or wholesale)
	// Normalize size for price calculation
	normalizedSize := utils.NormalizeSize(itemSize)
	calculatedPrice := utils.CalculatePrice(hoodieType, normalizedSize, orderType)
	log.Printf("💰 AddItem: Calculated price for order_type=%s, hoodie_type=%s, size=%s: %d cents (item base price: %d)", 
		orderType, hoodieType, normalizedSize, calculatedPrice, itemPrice)

	// Upsert reserved_order_lines (if exists, add to qty; if not, create new)
	// Use calculated price instead of item base price
	queryUpsertLine := `
		INSERT INTO reserved_order_lines (reserved_order_id, item_id, qty, unit_price)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (reserved_order_id, item_id)
		DO UPDATE SET qty = reserved_order_lines.qty + EXCLUDED.qty
		RETURNING id, reserved_order_id, item_id, qty, unit_price, created_at
	`

	var line models.ReservedOrderLine
	err = tx.QueryRowContext(ctx, queryUpsertLine, orderID, itemID, qty, calculatedPrice).Scan(
		&line.ID,
		&line.ReservedOrderID,
		&line.ItemID,
		&line.Qty,
		&line.UnitPrice,
		&line.CreatedAt,
	)
	if err != nil {
		log.Printf("❌ AddItem: Error upserting line: %v", err)
		return nil, fmt.Errorf("failed to upsert order line: %w", err)
	}

	// Update item stock_reserved
	queryUpdateStock := `
		UPDATE items
		SET stock_reserved = stock_reserved + $1
		WHERE id = $2
	`
	_, err = tx.ExecContext(ctx, queryUpdateStock, qty, itemID)
	if err != nil {
		log.Printf("❌ AddItem: Error updating stock_reserved: %v", err)
		return nil, fmt.Errorf("failed to update stock_reserved: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("❌ AddItem: Error committing transaction: %v", err)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("✅ AddItem: Successfully added item to order: line_id=%d", line.ID)
	return &line, nil
}

// GetByID retrieves a reserved order by ID with its lines
func (r *ReservedOrderRepository) GetByID(ctx context.Context, id int64) (*models.ReservedOrderResponse, error) {
	log.Printf("📦 GetByID: Fetching order id=%d", id)

	// Get order
	queryOrder := `
		SELECT id, status, assigned_to, order_type, customer_name, customer_phone, notes, created_at, updated_at
		FROM reserved_orders
		WHERE id = $1
	`

	var order models.ReservedOrder
	var customerName, customerPhone, notes sql.NullString

	err := db.DB.QueryRowContext(ctx, queryOrder, id).Scan(
		&order.ID,
		&order.Status,
		&order.AssignedTo,
		&order.OrderType,
		&customerName,
		&customerPhone,
		&notes,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("❌ GetByID: Order not found: id=%d", id)
			return nil, fmt.Errorf("order not found")
		}
		log.Printf("❌ GetByID: Error fetching order: %v", err)
		return nil, fmt.Errorf("failed to fetch order: %w", err)
	}

	if customerName.Valid {
		order.CustomerName = customerName.String
	}
	if customerPhone.Valid {
		order.CustomerPhone = customerPhone.String
	}
	if notes.Valid {
		order.Notes = notes.String
	}

	// Get lines with item details
	queryLines := `
		SELECT rol.id, rol.reserved_order_id, rol.item_id, rol.qty, rol.unit_price, rol.created_at,
		       i.sku, i.size, i.price
		FROM reserved_order_lines rol
		INNER JOIN items i ON rol.item_id = i.id
		WHERE rol.reserved_order_id = $1
		ORDER BY rol.created_at ASC
	`

	rows, err := db.DB.QueryContext(ctx, queryLines, id)
	if err != nil {
		log.Printf("❌ GetByID: Error fetching lines: %v", err)
		return nil, fmt.Errorf("failed to fetch order lines: %w", err)
	}
	defer rows.Close()

	var lines []models.ReservedOrderLine
	var total int64

	for rows.Next() {
		var line models.ReservedOrderLine
		err := rows.Scan(
			&line.ID,
			&line.ReservedOrderID,
			&line.ItemID,
			&line.Qty,
			&line.UnitPrice,
			&line.CreatedAt,
			&line.ItemSKU,
			&line.ItemSize,
			&line.ItemPrice,
		)
		if err != nil {
			log.Printf("❌ GetByID: Error scanning line: %v", err)
			continue
		}
		lines = append(lines, line)
		total += int64(line.Qty) * line.UnitPrice
	}

	if err := rows.Err(); err != nil {
		log.Printf("❌ GetByID: Error iterating lines: %v", err)
		return nil, fmt.Errorf("failed to iterate order lines: %w", err)
	}

	response := &models.ReservedOrderResponse{
		ReservedOrder: order,
		Lines:         lines,
		Total:         total,
	}

	log.Printf("✅ GetByID: Successfully fetched order id=%d with %d lines", id, len(lines))
	return response, nil
}

// List retrieves reserved orders filtered by status
func (r *ReservedOrderRepository) List(ctx context.Context, status *string) ([]models.ReservedOrderListItem, error) {
	log.Printf("📦 List: Fetching orders with status=%v", status)

	query := `
		SELECT ro.id, ro.status, ro.assigned_to, ro.order_type, ro.customer_name, ro.customer_phone, ro.notes,
		       ro.created_at, ro.updated_at,
		       COUNT(rol.id) as line_count,
		       COALESCE(SUM(rol.qty * rol.unit_price), 0) as total
		FROM reserved_orders ro
		LEFT JOIN reserved_order_lines rol ON ro.id = rol.reserved_order_id
	`
	var args []interface{}
	argIndex := 1

	if status != nil && *status != "" {
		query += fmt.Sprintf(" WHERE ro.status = $%d", argIndex)
		args = append(args, *status)
		argIndex++
	}

	query += `
		GROUP BY ro.id, ro.status, ro.assigned_to, ro.order_type, ro.customer_name, ro.customer_phone, ro.notes,
		         ro.created_at, ro.updated_at
		ORDER BY ro.created_at DESC
	`

	rows, err := db.DB.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("❌ List: Error fetching orders: %v", err)
		return nil, fmt.Errorf("failed to fetch orders: %w", err)
	}
	defer rows.Close()

	var orders []models.ReservedOrderListItem

	for rows.Next() {
		var order models.ReservedOrderListItem
		var customerName, customerPhone, notes sql.NullString

		err := rows.Scan(
			&order.ID,
			&order.Status,
			&order.AssignedTo,
			&order.OrderType,
			&customerName,
			&customerPhone,
			&notes,
			&order.CreatedAt,
			&order.UpdatedAt,
			&order.LineCount,
			&order.Total,
		)
		if err != nil {
			log.Printf("❌ List: Error scanning order: %v", err)
			continue
		}

		if customerName.Valid {
			order.CustomerName = customerName.String
		}
		if customerPhone.Valid {
			order.CustomerPhone = customerPhone.String
		}
		if notes.Valid {
			order.Notes = notes.String
		}

		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		log.Printf("❌ List: Error iterating orders: %v", err)
		return nil, fmt.Errorf("failed to iterate orders: %w", err)
	}

	log.Printf("✅ List: Successfully fetched %d orders", len(orders))
	return orders, nil
}

// Cancel cancels a reserved order and releases stock reservations
func (r *ReservedOrderRepository) Cancel(ctx context.Context, id int64) (*models.ReservedOrder, error) {
	log.Printf("📦 Cancel: Canceling order id=%d", id)

	// Start transaction
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("❌ Cancel: Error starting transaction: %v", err)
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Validate order exists and is in 'reserved' status
	var orderStatus string
	queryOrder := `SELECT status FROM reserved_orders WHERE id = $1 FOR UPDATE`
	err = tx.QueryRowContext(ctx, queryOrder, id).Scan(&orderStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("❌ Cancel: Order not found: id=%d", id)
			return nil, fmt.Errorf("order not found")
		}
		log.Printf("❌ Cancel: Error fetching order: %v", err)
		return nil, fmt.Errorf("failed to fetch order: %w", err)
	}

	if orderStatus != "reserved" {
		log.Printf("❌ Cancel: Order not in reserved status: status=%s", orderStatus)
		return nil, fmt.Errorf("order not in reserved status")
	}

	// Get all lines for this order
	queryLines := `SELECT item_id, qty FROM reserved_order_lines WHERE reserved_order_id = $1`
	rows, err := tx.QueryContext(ctx, queryLines, id)
	if err != nil {
		log.Printf("❌ Cancel: Error fetching lines: %v", err)
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
			log.Printf("❌ Cancel: Error scanning line: %v", err)
			continue
		}
		lines = append(lines, l)
	}

	if err := rows.Err(); err != nil {
		log.Printf("❌ Cancel: Error iterating lines: %v", err)
		return nil, fmt.Errorf("failed to iterate order lines: %w", err)
	}

	// Release stock reservations for each line
	for _, line := range lines {
		queryUpdateStock := `
			UPDATE items
			SET stock_reserved = GREATEST(0, stock_reserved - $1)
			WHERE id = $2
		`
		_, err = tx.ExecContext(ctx, queryUpdateStock, line.qty, line.itemID)
		if err != nil {
			log.Printf("❌ Cancel: Error updating stock for item_id=%d: %v", line.itemID, err)
			return nil, fmt.Errorf("failed to release stock reservation: %w", err)
		}
	}

	// Update order status to 'canceled'
	queryUpdateOrder := `
		UPDATE reserved_orders
		SET status = 'canceled', updated_at = NOW()
		WHERE id = $1
		RETURNING id, status, assigned_to, order_type, customer_name, customer_phone, notes, created_at, updated_at
	`

	var order models.ReservedOrder
	var customerName, customerPhone, notes sql.NullString

	err = tx.QueryRowContext(ctx, queryUpdateOrder, id).Scan(
		&order.ID,
		&order.Status,
		&order.AssignedTo,
		&order.OrderType,
		&customerName,
		&customerPhone,
		&notes,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		log.Printf("❌ Cancel: Error updating order: %v", err)
		return nil, fmt.Errorf("failed to update order: %w", err)
	}

	if customerName.Valid {
		order.CustomerName = customerName.String
	}
	if customerPhone.Valid {
		order.CustomerPhone = customerPhone.String
	}
	if notes.Valid {
		order.Notes = notes.String
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("❌ Cancel: Error committing transaction: %v", err)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("✅ Cancel: Successfully canceled order id=%d", id)
	return &order, nil
}

// Complete completes a reserved order and deducts stock
func (r *ReservedOrderRepository) Complete(ctx context.Context, id int64) (*models.ReservedOrder, error) {
	log.Printf("📦 Complete: Completing order id=%d", id)

	// Start transaction
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("❌ Complete: Error starting transaction: %v", err)
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Validate order exists and is in 'reserved' status
	var orderStatus string
	queryOrder := `SELECT status FROM reserved_orders WHERE id = $1 FOR UPDATE`
	err = tx.QueryRowContext(ctx, queryOrder, id).Scan(&orderStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("❌ Complete: Order not found: id=%d", id)
			return nil, fmt.Errorf("order not found")
		}
		log.Printf("❌ Complete: Error fetching order: %v", err)
		return nil, fmt.Errorf("failed to fetch order: %w", err)
	}

	if orderStatus != "reserved" {
		log.Printf("❌ Complete: Order not in reserved status: status=%s", orderStatus)
		return nil, fmt.Errorf("order not in reserved status")
	}

	// Get all lines for this order
	queryLines := `SELECT item_id, qty FROM reserved_order_lines WHERE reserved_order_id = $1`
	rows, err := tx.QueryContext(ctx, queryLines, id)
	if err != nil {
		log.Printf("❌ Complete: Error fetching lines: %v", err)
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
			log.Printf("❌ Complete: Error scanning line: %v", err)
			continue
		}
		lines = append(lines, l)
	}

	if err := rows.Err(); err != nil {
		log.Printf("❌ Complete: Error iterating lines: %v", err)
		return nil, fmt.Errorf("failed to iterate order lines: %w", err)
	}

	// Process each line: validate stock_reserved and deduct stock_total and stock_reserved
	for _, line := range lines {
		// Lock item for update and validate stock_reserved
		var stockReserved int
		queryItem := `SELECT stock_reserved FROM items WHERE id = $1 FOR UPDATE`
		err = tx.QueryRowContext(ctx, queryItem, line.itemID).Scan(&stockReserved)
		if err != nil {
			log.Printf("❌ Complete: Error fetching item stock: %v", err)
			return nil, fmt.Errorf("failed to fetch item stock: %w", err)
		}

		if stockReserved < line.qty {
			log.Printf("❌ Complete: Insufficient reserved stock: reserved=%d, required=%d", stockReserved, line.qty)
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
			log.Printf("❌ Complete: Error updating stock for item_id=%d: %v", line.itemID, err)
			return nil, fmt.Errorf("failed to deduct stock: %w", err)
		}
	}

	// Update order status to 'completed'
	queryUpdateOrder := `
		UPDATE reserved_orders
		SET status = 'completed', updated_at = NOW()
		WHERE id = $1
		RETURNING id, status, assigned_to, order_type, customer_name, customer_phone, notes, created_at, updated_at
	`

	var order models.ReservedOrder
	var customerName, customerPhone, notes sql.NullString

	err = tx.QueryRowContext(ctx, queryUpdateOrder, id).Scan(
		&order.ID,
		&order.Status,
		&order.AssignedTo,
		&order.OrderType,
		&customerName,
		&customerPhone,
		&notes,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		log.Printf("❌ Complete: Error updating order: %v", err)
		return nil, fmt.Errorf("failed to update order: %w", err)
	}

	if customerName.Valid {
		order.CustomerName = customerName.String
	}
	if customerPhone.Valid {
		order.CustomerPhone = customerPhone.String
	}
	if notes.Valid {
		order.Notes = notes.String
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("❌ Complete: Error committing transaction: %v", err)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("✅ Complete: Successfully completed order id=%d", id)
	return &order, nil
}

// GetAllWithFullItems retrieves all reserved orders with complete item and design asset information
func (r *ReservedOrderRepository) GetAllWithFullItems(ctx context.Context) ([]models.ReservedOrderWithFullItems, error) {
	log.Printf("📦 GetAllWithFullItems: Fetching all orders with full item information")

	// Get all orders
	queryOrders := `
		SELECT id, status, assigned_to, order_type, customer_name, customer_phone, notes, created_at, updated_at
		FROM reserved_orders
		ORDER BY created_at DESC
	`

	rows, err := db.DB.QueryContext(ctx, queryOrders)
	if err != nil {
		log.Printf("❌ GetAllWithFullItems: Error fetching orders: %v", err)
		return nil, fmt.Errorf("failed to fetch orders: %w", err)
	}
	defer rows.Close()

	var orders []models.ReservedOrder
	var customerName, customerPhone, notes sql.NullString

	for rows.Next() {
		var order models.ReservedOrder
		err := rows.Scan(
			&order.ID,
			&order.Status,
			&order.AssignedTo,
			&order.OrderType,
			&customerName,
			&customerPhone,
			&notes,
			&order.CreatedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			log.Printf("❌ GetAllWithFullItems: Error scanning order: %v", err)
			continue
		}

		if customerName.Valid {
			order.CustomerName = customerName.String
		}
		if customerPhone.Valid {
			order.CustomerPhone = customerPhone.String
		}
		if notes.Valid {
			order.Notes = notes.String
		}

		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		log.Printf("❌ GetAllWithFullItems: Error iterating orders: %v", err)
		return nil, fmt.Errorf("failed to iterate orders: %w", err)
	}

	// Build result with lines for each order
	result := make([]models.ReservedOrderWithFullItems, 0, len(orders))

	for _, order := range orders {
		// Get lines with complete item and design asset information
		queryLines := `
			SELECT rol.id, rol.reserved_order_id, rol.item_id, rol.qty, rol.unit_price, rol.created_at,
			       i.id, i.sku, i.size, i.price, i.stock_total, i.stock_reserved, i.design_asset_id,
			       COALESCE(da.description, '') as description,
			       COALESCE(da.color_primary, '') as color_primary,
			       COALESCE(da.color_secondary, '') as color_secondary,
			       COALESCE(da.hoodie_type, '') as hoodie_type,
			       COALESCE(da.image_type, '') as image_type,
			       COALESCE(da.deco_id, '') as deco_id,
			       COALESCE(da.deco_base, '') as deco_base
			FROM reserved_order_lines rol
			INNER JOIN items i ON rol.item_id = i.id
			LEFT JOIN design_assets da ON i.design_asset_id = da.id
			WHERE rol.reserved_order_id = $1
			ORDER BY rol.created_at ASC
		`

		lineRows, err := db.DB.QueryContext(ctx, queryLines, order.ID)
		if err != nil {
			log.Printf("❌ GetAllWithFullItems: Error fetching lines for order %d: %v", order.ID, err)
			continue
		}

		var lines []models.ReservedOrderLineWithItem
		var total int64

		for lineRows.Next() {
			var line models.ReservedOrderLineWithItem
			var item models.ItemFullInfo

			err := lineRows.Scan(
				&line.ID,
				&line.ReservedOrderID,
				&line.ItemID,
				&line.Qty,
				&line.UnitPrice,
				&line.CreatedAt,
				&item.ID,
				&item.SKU,
				&item.Size,
				&item.Price,
				&item.StockTotal,
				&item.StockReserved,
				&item.DesignAssetID,
				&item.Description,
				&item.ColorPrimary,
				&item.ColorSecondary,
				&item.HoodieType,
				&item.ImageType,
				&item.DecoID,
				&item.DecoBase,
			)
			if err != nil {
				log.Printf("❌ GetAllWithFullItems: Error scanning line: %v", err)
				continue
			}

			line.Item = item
			lines = append(lines, line)
			total += int64(line.Qty) * line.UnitPrice
		}
		lineRows.Close()

		if err := lineRows.Err(); err != nil {
			log.Printf("❌ GetAllWithFullItems: Error iterating lines: %v", err)
			continue
		}

		result = append(result, models.ReservedOrderWithFullItems{
			ReservedOrder: order,
			Lines:         lines,
			Total:         total,
		})
	}

	log.Printf("✅ GetAllWithFullItems: Successfully fetched %d orders with full item information", len(result))
	return result, nil
}

// RemoveItem removes an item from a reserved order and releases stock reservation
func (r *ReservedOrderRepository) RemoveItem(ctx context.Context, orderID int64, itemID int64) error {
	log.Printf("📦 RemoveItem: Removing item_id=%d from order_id=%d", itemID, orderID)

	// Start transaction
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("❌ RemoveItem: Error starting transaction: %v", err)
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Validate order exists and is in 'reserved' status
	var orderStatus string
	queryOrder := `SELECT status FROM reserved_orders WHERE id = $1 FOR UPDATE`
	err = tx.QueryRowContext(ctx, queryOrder, orderID).Scan(&orderStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("❌ RemoveItem: Order not found: id=%d", orderID)
			return fmt.Errorf("order not found")
		}
		log.Printf("❌ RemoveItem: Error fetching order: %v", err)
		return fmt.Errorf("failed to fetch order: %w", err)
	}

	if orderStatus != "reserved" {
		log.Printf("❌ RemoveItem: Order not in reserved status: status=%s", orderStatus)
		return fmt.Errorf("order not in reserved status")
	}

	// Get the line item to get the quantity
	var qty int
	queryLine := `SELECT qty FROM reserved_order_lines WHERE reserved_order_id = $1 AND item_id = $2`
	err = tx.QueryRowContext(ctx, queryLine, orderID, itemID).Scan(&qty)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("❌ RemoveItem: Item not found in order: order_id=%d, item_id=%d", orderID, itemID)
			return fmt.Errorf("item not found in order")
		}
		log.Printf("❌ RemoveItem: Error fetching line: %v", err)
		return fmt.Errorf("failed to fetch order line: %w", err)
	}

	// Delete the line item
	queryDeleteLine := `DELETE FROM reserved_order_lines WHERE reserved_order_id = $1 AND item_id = $2`
	result, err := tx.ExecContext(ctx, queryDeleteLine, orderID, itemID)
	if err != nil {
		log.Printf("❌ RemoveItem: Error deleting line: %v", err)
		return fmt.Errorf("failed to delete order line: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("❌ RemoveItem: Error getting rows affected: %v", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		log.Printf("❌ RemoveItem: No line deleted: order_id=%d, item_id=%d", orderID, itemID)
		return fmt.Errorf("item not found in order")
	}

	// Release stock reservation
	queryUpdateStock := `
		UPDATE items
		SET stock_reserved = GREATEST(0, stock_reserved - $1)
		WHERE id = $2
	`
	_, err = tx.ExecContext(ctx, queryUpdateStock, qty, itemID)
	if err != nil {
		log.Printf("❌ RemoveItem: Error updating stock_reserved: %v", err)
		return fmt.Errorf("failed to release stock reservation: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("❌ RemoveItem: Error committing transaction: %v", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("✅ RemoveItem: Successfully removed item_id=%d (qty=%d) from order_id=%d", itemID, qty, orderID)
	return nil
}

// UpdateItemQuantity updates the quantity of an item in a reserved order and adjusts stock reservation
func (r *ReservedOrderRepository) UpdateItemQuantity(ctx context.Context, orderID int64, itemID int64, newQty int) (*models.ReservedOrderLine, error) {
	log.Printf("📦 UpdateItemQuantity: Updating item_id=%d quantity to %d in order_id=%d", itemID, newQty, orderID)

	if newQty <= 0 {
		return nil, fmt.Errorf("qty must be greater than 0")
	}

	// Start transaction
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("❌ UpdateItemQuantity: Error starting transaction: %v", err)
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Validate order exists and is in 'reserved' status
	var orderStatus string
	queryOrder := `SELECT status FROM reserved_orders WHERE id = $1 FOR UPDATE`
	err = tx.QueryRowContext(ctx, queryOrder, orderID).Scan(&orderStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("❌ UpdateItemQuantity: Order not found: id=%d", orderID)
			return nil, fmt.Errorf("order not found")
		}
		log.Printf("❌ UpdateItemQuantity: Error fetching order: %v", err)
		return nil, fmt.Errorf("failed to fetch order: %w", err)
	}

	if orderStatus != "reserved" {
		log.Printf("❌ UpdateItemQuantity: Order not in reserved status: status=%s", orderStatus)
		return nil, fmt.Errorf("order not in reserved status")
	}

	// Get current quantity from the line
	var currentQty int
	var unitPrice int64
	queryLine := `SELECT qty, unit_price FROM reserved_order_lines WHERE reserved_order_id = $1 AND item_id = $2`
	err = tx.QueryRowContext(ctx, queryLine, orderID, itemID).Scan(&currentQty, &unitPrice)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("❌ UpdateItemQuantity: Item not found in order: order_id=%d, item_id=%d", orderID, itemID)
			return nil, fmt.Errorf("item not found in order")
		}
		log.Printf("❌ UpdateItemQuantity: Error fetching line: %v", err)
		return nil, fmt.Errorf("failed to fetch order line: %w", err)
	}

	// Calculate quantity difference
	qtyDiff := newQty - currentQty
	log.Printf("📊 UpdateItemQuantity: Current qty=%d, New qty=%d, Difference=%d", currentQty, newQty, qtyDiff)

	if qtyDiff == 0 {
		log.Printf("⚠️  UpdateItemQuantity: No change in quantity, returning current line")
		// Return current line without changes
		return &models.ReservedOrderLine{
			ReservedOrderID: orderID,
			ItemID:          itemID,
			Qty:             currentQty,
			UnitPrice:       unitPrice,
		}, nil
	}

	// If increasing quantity, validate stock availability
	if qtyDiff > 0 {
		var stockTotal, stockReserved int
		queryItem := `SELECT stock_total, stock_reserved FROM items WHERE id = $1 FOR UPDATE`
		err = tx.QueryRowContext(ctx, queryItem, itemID).Scan(&stockTotal, &stockReserved)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Printf("❌ UpdateItemQuantity: Item not found: id=%d", itemID)
				return nil, fmt.Errorf("item not found")
			}
			log.Printf("❌ UpdateItemQuantity: Error fetching item: %v", err)
			return nil, fmt.Errorf("failed to fetch item: %w", err)
		}

		// Validate stock availability
		available := stockTotal - stockReserved
		if available < qtyDiff {
			log.Printf("❌ UpdateItemQuantity: Insufficient stock: available=%d, requested=%d", available, qtyDiff)
			return nil, fmt.Errorf("insufficient stock: available %d, requested %d", available, qtyDiff)
		}

		// Reserve additional stock
		queryUpdateStock := `
			UPDATE items
			SET stock_reserved = stock_reserved + $1
			WHERE id = $2
		`
		_, err = tx.ExecContext(ctx, queryUpdateStock, qtyDiff, itemID)
		if err != nil {
			log.Printf("❌ UpdateItemQuantity: Error updating stock_reserved: %v", err)
			return nil, fmt.Errorf("failed to update stock_reserved: %w", err)
		}
		log.Printf("✅ UpdateItemQuantity: Reserved additional %d units of stock", qtyDiff)
	} else {
		// Decreasing quantity, release stock reservation
		queryUpdateStock := `
			UPDATE items
			SET stock_reserved = GREATEST(0, stock_reserved - $1)
			WHERE id = $2
		`
		_, err = tx.ExecContext(ctx, queryUpdateStock, -qtyDiff, itemID)
		if err != nil {
			log.Printf("❌ UpdateItemQuantity: Error updating stock_reserved: %v", err)
			return nil, fmt.Errorf("failed to update stock_reserved: %w", err)
		}
		log.Printf("✅ UpdateItemQuantity: Released %d units of stock reservation", -qtyDiff)
	}

	// Update the line quantity
	queryUpdateLine := `
		UPDATE reserved_order_lines
		SET qty = $1
		WHERE reserved_order_id = $2 AND item_id = $3
		RETURNING id, reserved_order_id, item_id, qty, unit_price, created_at
	`
	var line models.ReservedOrderLine
	err = tx.QueryRowContext(ctx, queryUpdateLine, newQty, orderID, itemID).Scan(
		&line.ID,
		&line.ReservedOrderID,
		&line.ItemID,
		&line.Qty,
		&line.UnitPrice,
		&line.CreatedAt,
	)
	if err != nil {
		log.Printf("❌ UpdateItemQuantity: Error updating line: %v", err)
		return nil, fmt.Errorf("failed to update order line: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("❌ UpdateItemQuantity: Error committing transaction: %v", err)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("✅ UpdateItemQuantity: Successfully updated item_id=%d quantity from %d to %d in order_id=%d", itemID, currentQty, newQty, orderID)
	return &line, nil
}

// UpdateOrder updates a reserved order with its lines and adjusts stock reservations
func (r *ReservedOrderRepository) UpdateOrder(ctx context.Context, req *models.UpdateReservedOrderRequest) (*models.ReservedOrderResponse, error) {
	log.Printf("📦 UpdateOrder: Updating order_id=%d", req.ID)

	// Start transaction
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("❌ UpdateOrder: Error starting transaction: %v", err)
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Validate order exists and is in 'reserved' status
	var currentStatus string
	var orderType string
	queryOrder := `SELECT status, order_type FROM reserved_orders WHERE id = $1 FOR UPDATE`
	err = tx.QueryRowContext(ctx, queryOrder, req.ID).Scan(&currentStatus, &orderType)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("❌ UpdateOrder: Order not found: id=%d", req.ID)
			return nil, fmt.Errorf("order not found")
		}
		log.Printf("❌ UpdateOrder: Error fetching order: %v", err)
		return nil, fmt.Errorf("failed to fetch order: %w", err)
	}

	if currentStatus != "reserved" {
		log.Printf("❌ UpdateOrder: Order not in reserved status: status=%s", currentStatus)
		return nil, fmt.Errorf("order not in reserved status")
	}

	// Update order fields (status should remain "reserved" unless explicitly changed)
	updateStatus := req.Status
	if updateStatus == "" {
		updateStatus = "reserved"
	}

	queryUpdateOrder := `
		UPDATE reserved_orders
		SET assigned_to = $1,
		    order_type = $2,
		    customer_name = $3,
		    customer_phone = $4,
		    notes = $5,
		    status = $6,
		    updated_at = NOW()
		WHERE id = $7
	`
	_, err = tx.ExecContext(ctx, queryUpdateOrder,
		req.AssignedTo,
		req.OrderType,
		sql.NullString{String: req.CustomerName, Valid: req.CustomerName != ""},
		sql.NullString{String: req.CustomerPhone, Valid: req.CustomerPhone != ""},
		sql.NullString{String: req.Notes, Valid: req.Notes != ""},
		updateStatus,
		req.ID,
	)
	if err != nil {
		log.Printf("❌ UpdateOrder: Error updating order: %v", err)
		return nil, fmt.Errorf("failed to update order: %w", err)
	}

	// Get current lines
	queryCurrentLines := `
		SELECT id, item_id, qty
		FROM reserved_order_lines
		WHERE reserved_order_id = $1
	`
	rows, err := tx.QueryContext(ctx, queryCurrentLines, req.ID)
	if err != nil {
		log.Printf("❌ UpdateOrder: Error fetching current lines: %v", err)
		return nil, fmt.Errorf("failed to fetch current lines: %w", err)
	}
	defer rows.Close()

	type currentLine struct {
		id     int64
		itemID int64
		qty    int
	}
	currentLinesMap := make(map[int64]currentLine) // key: item_id
	for rows.Next() {
		var cl currentLine
		if err := rows.Scan(&cl.id, &cl.itemID, &cl.qty); err != nil {
			log.Printf("❌ UpdateOrder: Error scanning current line: %v", err)
			continue
		}
		currentLinesMap[cl.itemID] = cl
	}
	if err := rows.Err(); err != nil {
		log.Printf("❌ UpdateOrder: Error iterating current lines: %v", err)
		return nil, fmt.Errorf("failed to iterate current lines: %w", err)
	}

	// Build map of requested lines (key: item_id)
	requestedLinesMap := make(map[int64]models.UpdateReservedOrderLineRequest)
	for _, line := range req.Lines {
		if line.Qty > 0 {
			requestedLinesMap[line.ItemID] = line
		}
	}

	// Process deletions: lines in current but not in requested or qty=0
	for itemID, cl := range currentLinesMap {
		if _, exists := requestedLinesMap[itemID]; !exists {
			// Delete line and release stock
			log.Printf("🗑️  UpdateOrder: Deleting line for item_id=%d (qty=%d)", itemID, cl.qty)
			queryDeleteLine := `DELETE FROM reserved_order_lines WHERE id = $1`
			_, err = tx.ExecContext(ctx, queryDeleteLine, cl.id)
			if err != nil {
				log.Printf("❌ UpdateOrder: Error deleting line: %v", err)
				return nil, fmt.Errorf("failed to delete line: %w", err)
			}

			// Release stock reservation
			queryUpdateStock := `
				UPDATE items
				SET stock_reserved = GREATEST(0, stock_reserved - $1)
				WHERE id = $2
			`
			_, err = tx.ExecContext(ctx, queryUpdateStock, cl.qty, itemID)
			if err != nil {
				log.Printf("❌ UpdateOrder: Error releasing stock: %v", err)
				return nil, fmt.Errorf("failed to release stock: %w", err)
			}
		}
	}

	// Process updates and additions
	for itemID, reqLine := range requestedLinesMap {
		if cl, exists := currentLinesMap[itemID]; exists {
			// Update existing line
			if cl.qty != reqLine.Qty {
				qtyDiff := reqLine.Qty - cl.qty
				log.Printf("🔄 UpdateOrder: Updating item_id=%d from qty=%d to qty=%d (diff=%d)", itemID, cl.qty, reqLine.Qty, qtyDiff)

				if qtyDiff > 0 {
					// Increase quantity - validate and reserve stock
					var stockTotal, stockReserved int
					queryItem := `SELECT stock_total, stock_reserved FROM items WHERE id = $1 FOR UPDATE`
					err = tx.QueryRowContext(ctx, queryItem, itemID).Scan(&stockTotal, &stockReserved)
					if err != nil {
						log.Printf("❌ UpdateOrder: Error fetching item: %v", err)
						return nil, fmt.Errorf("failed to fetch item: %w", err)
					}

					available := stockTotal - stockReserved
					if available < qtyDiff {
						log.Printf("❌ UpdateOrder: Insufficient stock: available=%d, requested=%d", available, qtyDiff)
						return nil, fmt.Errorf("insufficient stock: available %d, requested %d", available, qtyDiff)
					}

					// Reserve additional stock
					queryUpdateStock := `
						UPDATE items
						SET stock_reserved = stock_reserved + $1
						WHERE id = $2
					`
					_, err = tx.ExecContext(ctx, queryUpdateStock, qtyDiff, itemID)
					if err != nil {
						log.Printf("❌ UpdateOrder: Error reserving stock: %v", err)
						return nil, fmt.Errorf("failed to reserve stock: %w", err)
					}
				} else {
					// Decrease quantity - release stock
					queryUpdateStock := `
						UPDATE items
						SET stock_reserved = GREATEST(0, stock_reserved - $1)
						WHERE id = $2
					`
					_, err = tx.ExecContext(ctx, queryUpdateStock, -qtyDiff, itemID)
					if err != nil {
						log.Printf("❌ UpdateOrder: Error releasing stock: %v", err)
						return nil, fmt.Errorf("failed to release stock: %w", err)
					}
				}

				// Update line quantity
				queryUpdateLine := `UPDATE reserved_order_lines SET qty = $1 WHERE id = $2`
				_, err = tx.ExecContext(ctx, queryUpdateLine, reqLine.Qty, cl.id)
				if err != nil {
					log.Printf("❌ UpdateOrder: Error updating line: %v", err)
					return nil, fmt.Errorf("failed to update line: %w", err)
				}
			}
		} else {
			// Add new line
			log.Printf("➕ UpdateOrder: Adding new line for item_id=%d (qty=%d)", itemID, reqLine.Qty)

			// Validate item exists and get price
			var stockTotal, stockReserved int
			var itemPrice int64
			var isActive bool
			var itemSize string
			var hoodieType string
			queryItem := `
				SELECT i.stock_total, i.stock_reserved, i.price, i.is_active, i.size,
				       COALESCE(da.hoodie_type, '') as hoodie_type
				FROM items i
				INNER JOIN design_assets da ON i.design_asset_id = da.id
				WHERE i.id = $1
				FOR UPDATE
			`
			err = tx.QueryRowContext(ctx, queryItem, itemID).Scan(&stockTotal, &stockReserved, &itemPrice, &isActive, &itemSize, &hoodieType)
			if err != nil {
				if err == sql.ErrNoRows {
					log.Printf("❌ UpdateOrder: Item not found: id=%d", itemID)
					return nil, fmt.Errorf("item not found: id=%d", itemID)
				}
				log.Printf("❌ UpdateOrder: Error fetching item: %v", err)
				return nil, fmt.Errorf("failed to fetch item: %w", err)
			}

			if !isActive {
				log.Printf("❌ UpdateOrder: Item is not active: id=%d", itemID)
				return nil, fmt.Errorf("item not found or inactive: id=%d", itemID)
			}

			// Validate stock availability
			available := stockTotal - stockReserved
			if available < reqLine.Qty {
				log.Printf("❌ UpdateOrder: Insufficient stock: available=%d, requested=%d", available, reqLine.Qty)
				return nil, fmt.Errorf("insufficient stock: available %d, requested %d", available, reqLine.Qty)
			}

			// Calculate price
			normalizedSize := utils.NormalizeSize(itemSize)
			calculatedPrice := utils.CalculatePrice(hoodieType, normalizedSize, req.OrderType)

			// Insert line
			queryInsertLine := `
				INSERT INTO reserved_order_lines (reserved_order_id, item_id, qty, unit_price)
				VALUES ($1, $2, $3, $4)
			`
			_, err = tx.ExecContext(ctx, queryInsertLine, req.ID, itemID, reqLine.Qty, calculatedPrice)
			if err != nil {
				log.Printf("❌ UpdateOrder: Error inserting line: %v", err)
				return nil, fmt.Errorf("failed to insert line: %w", err)
			}

			// Reserve stock
			queryUpdateStock := `
				UPDATE items
				SET stock_reserved = stock_reserved + $1
				WHERE id = $2
			`
			_, err = tx.ExecContext(ctx, queryUpdateStock, reqLine.Qty, itemID)
			if err != nil {
				log.Printf("❌ UpdateOrder: Error reserving stock: %v", err)
				return nil, fmt.Errorf("failed to reserve stock: %w", err)
			}
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("❌ UpdateOrder: Error committing transaction: %v", err)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Fetch updated order with lines
	log.Printf("✅ UpdateOrder: Successfully updated order_id=%d", req.ID)
	return r.GetByID(ctx, req.ID)
}

