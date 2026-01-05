package repository

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"
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
// For manual transactions, source='manual' and source_id=NULL
// For sale transactions, source='sale' and source_id must be provided
func (r *FinanceTransactionRepository) Create(ctx context.Context, req *models.CreateFinanceTransactionRequest) (*models.FinanceTransaction, error) {
	log.Printf("💰 CreateFinanceTransaction: type=%s, amount=%d", req.Type, req.Amount)

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

	// Validate destination
	if strings.TrimSpace(req.Destination) == "" {
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

	// For manual transactions, source='manual' and source_id=NULL
	source := "manual"
	var sourceID sql.NullInt64

	// Insert into finance_transactions
	queryInsert := `
		INSERT INTO finance_transactions (type, source, source_id, occurred_at, amount, destination, category, counterparty, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, type, source, source_id, occurred_at, amount, destination, category, counterparty, notes, created_at
	`

	var transaction models.FinanceTransaction
	var category, counterparty, notes sql.NullString
	var sourceIDScan sql.NullInt64

	err := db.DB.QueryRowContext(ctx, queryInsert,
		req.Type,
		source,
		sourceID,
		occurredAt,
		req.Amount,
		req.Destination,
		sql.NullString{String: req.Category, Valid: req.Category != ""},
		sql.NullString{String: req.Counterparty, Valid: req.Counterparty != ""},
		sql.NullString{String: req.Notes, Valid: req.Notes != ""},
	).Scan(
		&transaction.ID,
		&transaction.Type,
		&transaction.Source,
		&sourceIDScan,
		&transaction.OccurredAt,
		&transaction.Amount,
		&transaction.Destination,
		&category,
		&counterparty,
		&notes,
		&transaction.CreatedAt,
	)

	if err != nil {
		log.Printf("❌ CreateFinanceTransaction: Error inserting transaction: %v", err)
		return nil, fmt.Errorf("failed to insert finance transaction: %w", err)
	}

	transaction.Source = source
	if sourceIDScan.Valid {
		transaction.SourceID = &sourceIDScan.Int64
	}
	if category.Valid {
		transaction.Category = category.String
	}
	if counterparty.Valid {
		transaction.Counterparty = counterparty.String
	}
	if notes.Valid {
		transaction.Notes = notes.String
	}

	log.Printf("✅ CreateFinanceTransaction: Successfully created transaction id=%d", transaction.ID)
	return &transaction, nil
}

// cursorData represents the cursor structure for pagination
type cursorData struct {
	OccurredAt string `json:"occurredAt"`
	ID         int64  `json:"id"`
}

// encodeCursor encodes occurredAt and id into a base64 cursor string
func encodeCursor(occurredAt time.Time, id int64) string {
	data := cursorData{
		OccurredAt: occurredAt.Format(time.RFC3339Nano),
		ID:         id,
	}
	jsonData, _ := json.Marshal(data)
	return base64.URLEncoding.EncodeToString(jsonData)
}

// decodeCursor decodes a base64 cursor string into occurredAt and id
func decodeCursor(cursor string) (time.Time, int64, error) {
	jsonData, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("invalid cursor format: %w", err)
	}
	var data cursorData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return time.Time{}, 0, fmt.Errorf("invalid cursor format: %w", err)
	}
	occurredAt, err := time.Parse(time.RFC3339Nano, data.OccurredAt)
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("invalid cursor timestamp: %w", err)
	}
	return occurredAt, data.ID, nil
}

// List retrieves finance transactions with filters and cursor pagination
func (r *FinanceTransactionRepository) List(ctx context.Context, req *models.FinanceTransactionListRequest) (*models.FinanceTransactionListResponse, error) {
	log.Printf("📦 ListFinanceTransactions: Fetching transactions with filters")

	// Set default limit
	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	// Build query with filters
	query := `
		SELECT id, type, source, source_id, occurred_at, amount, destination, category, counterparty, notes, created_at
		FROM finance_transactions
		WHERE 1=1
	`
	var args []interface{}
	argIndex := 1

	// Date range filters
	if req.From != nil && *req.From != "" {
		fromDate, err := time.Parse("2006-01-02", *req.From)
		if err != nil {
			return nil, fmt.Errorf("invalid from date format: %w", err)
		}
		query += fmt.Sprintf(" AND occurred_at >= $%d", argIndex)
		args = append(args, fromDate)
		argIndex++
	}

	if req.To != nil && *req.To != "" {
		toDate, err := time.Parse("2006-01-02", *req.To)
		if err != nil {
			return nil, fmt.Errorf("invalid to date format: %w", err)
		}
		// Set to end of day
		toDate = time.Date(toDate.Year(), toDate.Month(), toDate.Day(), 23, 59, 59, 999999999, toDate.Location())
		query += fmt.Sprintf(" AND occurred_at <= $%d", argIndex)
		args = append(args, toDate)
		argIndex++
	}

	// Type filter
	if req.Type != nil && *req.Type != "" {
		query += fmt.Sprintf(" AND type = $%d", argIndex)
		args = append(args, *req.Type)
		argIndex++
	}

	// Source filter
	if req.Source != nil && *req.Source != "" {
		query += fmt.Sprintf(" AND source = $%d", argIndex)
		args = append(args, *req.Source)
		argIndex++
	}

	// Destination filter
	if req.Destination != nil && *req.Destination != "" {
		query += fmt.Sprintf(" AND destination = $%d", argIndex)
		args = append(args, *req.Destination)
		argIndex++
	}

	// Category filter
	if req.Category != nil && *req.Category != "" {
		query += fmt.Sprintf(" AND category = $%d", argIndex)
		args = append(args, *req.Category)
		argIndex++
	}

	// Text search filter (q) - search in notes and counterparty
	if req.Q != nil && *req.Q != "" {
		searchTerm := "%" + *req.Q + "%"
		query += fmt.Sprintf(" AND (notes ILIKE $%d OR counterparty ILIKE $%d)", argIndex, argIndex)
		args = append(args, searchTerm)
		argIndex++
	}

	// Cursor pagination
	if req.Cursor != nil && *req.Cursor != "" {
		cursorOccurredAt, cursorID, err := decodeCursor(*req.Cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
		query += fmt.Sprintf(" AND (occurred_at, id) < ($%d, $%d)", argIndex, argIndex+1)
		args = append(args, cursorOccurredAt, cursorID)
		argIndex += 2
	}

	// Order and limit (fetch limit+1 to check if there's a next page)
	query += fmt.Sprintf(" ORDER BY occurred_at DESC, id DESC LIMIT $%d", argIndex)
	args = append(args, limit+1)
	argIndex++

	rows, err := db.DB.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("❌ ListFinanceTransactions: Error fetching transactions: %v", err)
		return nil, fmt.Errorf("failed to fetch transactions: %w", err)
	}
	defer rows.Close()

	var transactions []models.FinanceTransaction
	var nextCursor *string

	for rows.Next() {
		var transaction models.FinanceTransaction
		var category, counterparty, notes sql.NullString
		var sourceID sql.NullInt64
		var occurredAt time.Time

		err := rows.Scan(
			&transaction.ID,
			&transaction.Type,
			&transaction.Source,
			&sourceID,
			&occurredAt,
			&transaction.Amount,
			&transaction.Destination,
			&category,
			&counterparty,
			&notes,
			&transaction.CreatedAt,
		)
		if err != nil {
			log.Printf("❌ ListFinanceTransactions: Error scanning transaction: %v", err)
			continue
		}

		transaction.OccurredAt = occurredAt.Format(time.RFC3339)
		if sourceID.Valid {
			transaction.SourceID = &sourceID.Int64
		}
		if category.Valid {
			transaction.Category = category.String
		}
		if counterparty.Valid {
			transaction.Counterparty = counterparty.String
		}
		if notes.Valid {
			transaction.Notes = notes.String
		}

		transactions = append(transactions, transaction)
	}

	if err := rows.Err(); err != nil {
		log.Printf("❌ ListFinanceTransactions: Error iterating transactions: %v", err)
		return nil, fmt.Errorf("failed to iterate transactions: %w", err)
	}

	// Check if there's a next page
	if len(transactions) > limit {
		// Remove the extra item and create cursor from it
		lastTransaction := transactions[limit]
		lastOccurredAt, _ := time.Parse(time.RFC3339, lastTransaction.OccurredAt)
		cursor := encodeCursor(lastOccurredAt, lastTransaction.ID)
		nextCursor = &cursor
		transactions = transactions[:limit]
	}

	log.Printf("✅ ListFinanceTransactions: Successfully fetched %d transactions", len(transactions))

	return &models.FinanceTransactionListResponse{
		Transactions: transactions,
		Pagination: models.PaginationInfo{
			Limit:      limit,
			NextCursor: nextCursor,
		},
	}, nil
}

// Summary calculates financial summary and balances
func (r *FinanceTransactionRepository) Summary(ctx context.Context, from, to *string) (*models.FinanceSummaryResponse, error) {
	log.Printf("📊 SummaryFinanceTransactions: Calculating summary (from=%v, to=%v)", from, to)

	response := &models.FinanceSummaryResponse{
		Currency: "COP",
	}

	// Calculate balanceAllTime
	queryAllTime := `
		SELECT 
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE -amount END), 0) as balance_all_time
		FROM finance_transactions
	`
	var balanceAllTime int64
	err := db.DB.QueryRowContext(ctx, queryAllTime).Scan(&balanceAllTime)
	if err != nil {
		log.Printf("❌ SummaryFinanceTransactions: Error calculating balanceAllTime: %v", err)
		return nil, fmt.Errorf("failed to calculate balance all time: %w", err)
	}
	response.BalanceAllTime = balanceAllTime

	// Calculate byDestinationAllTime
	queryByDestination := `
		SELECT 
			destination,
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE -amount END), 0) as balance
		FROM finance_transactions
		GROUP BY destination
		ORDER BY destination
	`
	rows, err := db.DB.QueryContext(ctx, queryByDestination)
	if err != nil {
		log.Printf("❌ SummaryFinanceTransactions: Error calculating byDestinationAllTime: %v", err)
		return nil, fmt.Errorf("failed to calculate by destination all time: %w", err)
	}
	defer rows.Close()

	var byDestinationAllTime []models.DestinationBalance
	for rows.Next() {
		var db models.DestinationBalance
		if err := rows.Scan(&db.Destination, &db.Balance); err != nil {
			log.Printf("❌ SummaryFinanceTransactions: Error scanning destination balance: %v", err)
			continue
		}
		byDestinationAllTime = append(byDestinationAllTime, db)
	}
	response.ByDestinationAllTime = byDestinationAllTime

	// If date range is provided, calculate range-specific metrics
	if from != nil && *from != "" && to != nil && *to != "" {
		fromDate, err := time.Parse("2006-01-02", *from)
		if err != nil {
			return nil, fmt.Errorf("invalid from date format: %w", err)
		}
		toDate, err := time.Parse("2006-01-02", *to)
		if err != nil {
			return nil, fmt.Errorf("invalid to date format: %w", err)
		}
		toDate = time.Date(toDate.Year(), toDate.Month(), toDate.Day(), 23, 59, 59, 999999999, toDate.Location())

		// Calculate opening balance (before from date)
		queryOpeningBalance := `
			SELECT COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE -amount END), 0) as opening_balance
			FROM finance_transactions
			WHERE occurred_at < $1
		`
		var openingBalance int64
		err = db.DB.QueryRowContext(ctx, queryOpeningBalance, fromDate).Scan(&openingBalance)
		if err != nil {
			log.Printf("❌ SummaryFinanceTransactions: Error calculating openingBalance: %v", err)
			return nil, fmt.Errorf("failed to calculate opening balance: %w", err)
		}

		// Calculate income, expense, and net in range
		queryRange := `
			SELECT 
				COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) as income,
				COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) as expense
			FROM finance_transactions
			WHERE occurred_at >= $1 AND occurred_at <= $2
		`
		var income, expense int64
		err = db.DB.QueryRowContext(ctx, queryRange, fromDate, toDate).Scan(&income, &expense)
		if err != nil {
			log.Printf("❌ SummaryFinanceTransactions: Error calculating range metrics: %v", err)
			return nil, fmt.Errorf("failed to calculate range metrics: %w", err)
		}

		net := income - expense
		closingBalance := openingBalance + net

		response.Range = &models.SummaryRange{
			From:           *from,
			To:             *to,
			OpeningBalance: openingBalance,
			Income:         income,
			Expense:        expense,
			Net:            net,
			ClosingBalance: closingBalance,
		}

		// Calculate byDestinationRange
		queryByDestinationRange := `
			SELECT 
				destination,
				COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) as income,
				COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) as expense
			FROM finance_transactions
			WHERE occurred_at >= $1 AND occurred_at <= $2
			GROUP BY destination
			ORDER BY destination
		`
		rows, err = db.DB.QueryContext(ctx, queryByDestinationRange, fromDate, toDate)
		if err != nil {
			log.Printf("❌ SummaryFinanceTransactions: Error calculating byDestinationRange: %v", err)
			return nil, fmt.Errorf("failed to calculate by destination range: %w", err)
		}
		defer rows.Close()

		var byDestinationRange []models.DestinationRangeBalance
		for rows.Next() {
			var drb models.DestinationRangeBalance
			if err := rows.Scan(&drb.Destination, &drb.Income, &drb.Expense); err != nil {
				log.Printf("❌ SummaryFinanceTransactions: Error scanning destination range balance: %v", err)
				continue
			}
			drb.Net = drb.Income - drb.Expense
			byDestinationRange = append(byDestinationRange, drb)
		}
		response.ByDestinationRange = byDestinationRange
	}

	log.Printf("✅ SummaryFinanceTransactions: Successfully calculated summary")
	return response, nil
}

// Dashboard calculates comprehensive financial dashboard metrics
func (r *FinanceTransactionRepository) Dashboard(ctx context.Context, req *models.FinanceDashboardRequest) (*models.FinanceDashboardResponse, error) {
	log.Printf("📊 DashboardFinanceTransactions: Calculating dashboard metrics")

	// Determine period dates
	var fromDate, toDate time.Time
	var periodType string
	var periodLabel string

	if req.From != nil && *req.From != "" && req.To != nil && *req.To != "" {
		// Use provided dates
		var err error
		fromDate, err = time.Parse("2006-01-02", *req.From)
		if err != nil {
			return nil, fmt.Errorf("invalid from date format: %w", err)
		}
		toDate, err = time.Parse("2006-01-02", *req.To)
		if err != nil {
			return nil, fmt.Errorf("invalid to date format: %w", err)
		}
		toDate = time.Date(toDate.Year(), toDate.Month(), toDate.Day(), 23, 59, 59, 999999999, toDate.Location())
		periodType = "custom"
		periodLabel = fmt.Sprintf("%s - %s", *req.From, *req.To)
	} else {
		// Determine period based on period type (default: month)
		periodTypeStr := "month"
		if req.Period != nil && *req.Period != "" {
			periodTypeStr = *req.Period
		}
		periodType = periodTypeStr
		now := time.Now()

		switch periodTypeStr {
		case "quarter":
			// First day of current quarter
			quarter := (int(now.Month()) - 1) / 3
			fromDate = time.Date(now.Year(), time.Month(quarter*3+1), 1, 0, 0, 0, 0, now.Location())
			// Last day of current quarter
			toDate = time.Date(now.Year(), time.Month((quarter+1)*3+1), 0, 23, 59, 59, 999999999, now.Location())
			periodLabel = fmt.Sprintf("Q%d %d", quarter+1, now.Year())
		case "year":
			// First day of current year
			fromDate = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
			// Last day of current year
			toDate = time.Date(now.Year(), 12, 31, 23, 59, 59, 999999999, now.Location())
			periodLabel = fmt.Sprintf("%d", now.Year())
		default: // month
			// First day of current month
			fromDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
			// Last day of current month
			toDate = time.Date(now.Year(), now.Month()+1, 0, 23, 59, 59, 999999999, now.Location())
			monthNames := []string{"Enero", "Febrero", "Marzo", "Abril", "Mayo", "Junio", "Julio", "Agosto", "Septiembre", "Octubre", "Noviembre", "Diciembre"}
			periodLabel = fmt.Sprintf("%s %d", monthNames[now.Month()-1], now.Year())
		}
	}

	response := &models.FinanceDashboardResponse{
		Currency: "COP",
		Period: models.PeriodInfo{
			Type:  periodType,
			From:  fromDate.Format("2006-01-02"),
			To:    toDate.Format("2006-01-02"),
			Label: periodLabel,
		},
	}

	// Calculate current period metrics
	currentMetrics, err := r.calculatePeriodMetrics(ctx, fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate current period metrics: %w", err)
	}
	response.CurrentPeriod = *currentMetrics

	// Calculate comparison if requested
	if req.CompareWith != nil && *req.CompareWith != "" {
		var compareFrom, compareTo time.Time
		var compareType string

		switch *req.CompareWith {
		case "last_year":
			// Same period last year
			compareFrom = time.Date(fromDate.Year()-1, fromDate.Month(), fromDate.Day(), 0, 0, 0, 0, fromDate.Location())
			compareTo = time.Date(toDate.Year()-1, toDate.Month(), toDate.Day(), 23, 59, 59, 999999999, toDate.Location())
			compareType = "last_year"
		default: // previous
			// Previous period of same duration
			duration := toDate.Sub(fromDate)
			compareTo = fromDate.Add(-time.Nanosecond)
			compareFrom = compareTo.Add(-duration)
			compareType = "previous"
		}

		previousMetrics, err := r.calculatePeriodMetrics(ctx, compareFrom, compareTo)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate previous period metrics: %w", err)
		}

		// Calculate changes
		changes := r.calculateChanges(currentMetrics, previousMetrics)

		response.Comparison = &models.ComparisonData{
			Type: compareType,
			PreviousPeriod: *previousMetrics,
			PreviousPeriodInfo: models.PeriodInfo{
				Type: periodType,
				From: compareFrom.Format("2006-01-02"),
				To:   compareTo.Format("2006-01-02"),
			},
			Changes: changes,
		}
	}

	// Calculate cash flow time series
	cashFlow, err := r.calculateCashFlow(ctx, fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate cash flow: %w", err)
	}
	response.CashFlow = *cashFlow

	// Calculate breakdown by category
	byCategory, err := r.calculateCategoryBreakdown(ctx, fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate category breakdown: %w", err)
	}
	response.ByCategory = *byCategory

	// Calculate breakdown by counterparty
	byCounterparty, err := r.calculateCounterpartyBreakdown(ctx, fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate counterparty breakdown: %w", err)
	}
	response.ByCounterparty = *byCounterparty

	// Calculate breakdown by destination
	byDestination, err := r.calculateDestinationBreakdown(ctx, fromDate, toDate, currentMetrics.Net)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate destination breakdown: %w", err)
	}
	response.ByDestination = *byDestination

	// Get top transactions
	topTransactions, err := r.getTopTransactions(ctx, fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get top transactions: %w", err)
	}
	response.TopTransactions = *topTransactions

	// Calculate KPIs
	kpis := r.calculateKPIs(currentMetrics, fromDate, toDate, byCategory)
	response.KPIs = kpis

	// Calculate trends
	var trends models.Trends
	if response.Comparison != nil {
		trends = r.calculateTrends(currentMetrics, &response.Comparison.PreviousPeriod)
	} else {
		trends = models.Trends{
			IncomeTrend:       "stable",
			ExpenseTrend:      "stable",
			NetTrend:          "stable",
			ProfitMarginTrend: "stable",
		}
	}
	response.Trends = trends

	log.Printf("✅ DashboardFinanceTransactions: Successfully calculated dashboard")
	return response, nil
}

// Helper function to calculate period metrics
func (r *FinanceTransactionRepository) calculatePeriodMetrics(ctx context.Context, from, to time.Time) (*models.PeriodMetrics, error) {
	query := `
		SELECT 
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) as income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) as expense,
			COUNT(*) as transaction_count,
			COALESCE(AVG(amount), 0) as avg_transaction
		FROM finance_transactions
		WHERE occurred_at >= $1 AND occurred_at <= $2
	`

	var income, expense int64
	var transactionCount int
	var avgTransaction float64

	err := db.DB.QueryRowContext(ctx, query, from, to).Scan(&income, &expense, &transactionCount, &avgTransaction)
	if err != nil {
		return nil, err
	}

	net := income - expense
	var profitMargin float64
	if income > 0 {
		profitMargin = (float64(net) / float64(income)) * 100
	}

	return &models.PeriodMetrics{
		Income:            income,
		Expense:           expense,
		Net:               net,
		TransactionCount:  transactionCount,
		AverageTransaction: avgTransaction,
		ProfitMargin:      profitMargin,
	}, nil
}

// Helper function to calculate changes between periods
func (r *FinanceTransactionRepository) calculateChanges(current, previous *models.PeriodMetrics) models.PeriodChanges {
	var incomeChange, expenseChange, netChange, profitMarginChange float64

	if previous.Income > 0 {
		incomeChange = ((float64(current.Income) - float64(previous.Income)) / float64(previous.Income)) * 100
	}
	if previous.Expense > 0 {
		expenseChange = ((float64(current.Expense) - float64(previous.Expense)) / float64(previous.Expense)) * 100
	}
	if previous.Net != 0 {
		netChange = ((float64(current.Net) - float64(previous.Net)) / float64(abs(previous.Net))) * 100
	}
	profitMarginChange = current.ProfitMargin - previous.ProfitMargin

	return models.PeriodChanges{
		IncomeChange:       incomeChange,
		ExpenseChange:      expenseChange,
		NetChange:          netChange,
		ProfitMarginChange: profitMarginChange,
	}
}

// Helper function for absolute value
func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// Helper function to calculate cash flow time series
func (r *FinanceTransactionRepository) calculateCashFlow(ctx context.Context, from, to time.Time) (*models.CashFlowData, error) {
	cashFlow := &models.CashFlowData{}

	// Daily cash flow
	dailyQuery := `
		SELECT 
			DATE(occurred_at) as date,
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) as income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) as expense
		FROM finance_transactions
		WHERE occurred_at >= $1 AND occurred_at <= $2
		GROUP BY DATE(occurred_at)
		ORDER BY date
	`

	rows, err := db.DB.QueryContext(ctx, dailyQuery, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var dcf models.DailyCashFlow
		var date time.Time
		if err := rows.Scan(&date, &dcf.Income, &dcf.Expense); err != nil {
			continue
		}
		dcf.Date = date.Format("2006-01-02")
		dcf.Net = dcf.Income - dcf.Expense
		cashFlow.Daily = append(cashFlow.Daily, dcf)
	}

	// Weekly cash flow
	weeklyQuery := `
		SELECT 
			TO_CHAR(occurred_at, 'IYYY-"W"IW') as week,
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) as income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) as expense
		FROM finance_transactions
		WHERE occurred_at >= $1 AND occurred_at <= $2
		GROUP BY TO_CHAR(occurred_at, 'IYYY-"W"IW')
		ORDER BY week
	`

	rows, err = db.DB.QueryContext(ctx, weeklyQuery, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var wcf models.WeeklyCashFlow
		if err := rows.Scan(&wcf.Week, &wcf.Income, &wcf.Expense); err != nil {
			continue
		}
		wcf.Net = wcf.Income - wcf.Expense
		cashFlow.Weekly = append(cashFlow.Weekly, wcf)
	}

	// Monthly cash flow
	monthlyQuery := `
		SELECT 
			TO_CHAR(occurred_at, 'YYYY-MM') as month,
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) as income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) as expense
		FROM finance_transactions
		WHERE occurred_at >= $1 AND occurred_at <= $2
		GROUP BY TO_CHAR(occurred_at, 'YYYY-MM')
		ORDER BY month
	`

	rows, err = db.DB.QueryContext(ctx, monthlyQuery, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var mcf models.MonthlyCashFlow
		if err := rows.Scan(&mcf.Month, &mcf.Income, &mcf.Expense); err != nil {
			continue
		}
		mcf.Net = mcf.Income - mcf.Expense
		cashFlow.Monthly = append(cashFlow.Monthly, mcf)
	}

	return cashFlow, nil
}

// Helper function to calculate category breakdown
func (r *FinanceTransactionRepository) calculateCategoryBreakdown(ctx context.Context, from, to time.Time) (*models.CategoryBreakdown, error) {
	breakdown := &models.CategoryBreakdown{}

	// Income by category
	incomeQuery := `
		SELECT 
			COALESCE(category, 'sin_categoria') as category,
			SUM(amount) as amount,
			COUNT(*) as count
		FROM finance_transactions
		WHERE occurred_at >= $1 AND occurred_at <= $2 AND type = 'income'
		GROUP BY category
		ORDER BY amount DESC
	`

	rows, err := db.DB.QueryContext(ctx, incomeQuery, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var totalIncome int64
	var incomeCategories []models.CategoryAmount
	for rows.Next() {
		var ca models.CategoryAmount
		if err := rows.Scan(&ca.Category, &ca.Amount, &ca.Count); err != nil {
			continue
		}
		totalIncome += ca.Amount
		incomeCategories = append(incomeCategories, ca)
	}

	// Calculate percentages
	for i := range incomeCategories {
		if totalIncome > 0 {
			incomeCategories[i].Percentage = (float64(incomeCategories[i].Amount) / float64(totalIncome)) * 100
		}
	}
	breakdown.Income = incomeCategories

	// Expense by category
	expenseQuery := `
		SELECT 
			COALESCE(category, 'sin_categoria') as category,
			SUM(amount) as amount,
			COUNT(*) as count
		FROM finance_transactions
		WHERE occurred_at >= $1 AND occurred_at <= $2 AND type = 'expense'
		GROUP BY category
		ORDER BY amount DESC
	`

	rows, err = db.DB.QueryContext(ctx, expenseQuery, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var totalExpense int64
	var expenseCategories []models.CategoryAmount
	for rows.Next() {
		var ca models.CategoryAmount
		if err := rows.Scan(&ca.Category, &ca.Amount, &ca.Count); err != nil {
			continue
		}
		totalExpense += ca.Amount
		expenseCategories = append(expenseCategories, ca)
	}

	// Calculate percentages
	for i := range expenseCategories {
		if totalExpense > 0 {
			expenseCategories[i].Percentage = (float64(expenseCategories[i].Amount) / float64(totalExpense)) * 100
		}
	}
	breakdown.Expense = expenseCategories

	return breakdown, nil
}

// Helper function to calculate counterparty breakdown
func (r *FinanceTransactionRepository) calculateCounterpartyBreakdown(ctx context.Context, from, to time.Time) (*models.CounterpartyBreakdown, error) {
	breakdown := &models.CounterpartyBreakdown{}

	// Top expenses by counterparty
	expenseQuery := `
		SELECT 
			counterparty,
			SUM(amount) as amount,
			COUNT(*) as count
		FROM finance_transactions
		WHERE occurred_at >= $1 AND occurred_at <= $2 AND type = 'expense' AND counterparty IS NOT NULL
		GROUP BY counterparty
		ORDER BY amount DESC
		LIMIT 10
	`

	rows, err := db.DB.QueryContext(ctx, expenseQuery, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ca models.CounterpartyAmount
		if err := rows.Scan(&ca.Counterparty, &ca.Amount, &ca.Count); err != nil {
			continue
		}
		breakdown.TopExpenses = append(breakdown.TopExpenses, ca)
	}

	// Top incomes by counterparty
	incomeQuery := `
		SELECT 
			counterparty,
			SUM(amount) as amount,
			COUNT(*) as count
		FROM finance_transactions
		WHERE occurred_at >= $1 AND occurred_at <= $2 AND type = 'income' AND counterparty IS NOT NULL
		GROUP BY counterparty
		ORDER BY amount DESC
		LIMIT 10
	`

	rows, err = db.DB.QueryContext(ctx, incomeQuery, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ca models.CounterpartyAmount
		if err := rows.Scan(&ca.Counterparty, &ca.Amount, &ca.Count); err != nil {
			continue
		}
		breakdown.TopIncomes = append(breakdown.TopIncomes, ca)
	}

	return breakdown, nil
}

// Helper function to calculate destination breakdown
func (r *FinanceTransactionRepository) calculateDestinationBreakdown(ctx context.Context, from, to time.Time, totalNet int64) (*models.DestinationBreakdown, error) {
	breakdown := &models.DestinationBreakdown{TotalNet: totalNet}

	query := `
		SELECT 
			destination,
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) as income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) as expense
		FROM finance_transactions
		WHERE occurred_at >= $1 AND occurred_at <= $2
		GROUP BY destination
		ORDER BY destination
	`

	rows, err := db.DB.QueryContext(ctx, query, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var dm models.DestinationMetrics
		if err := rows.Scan(&dm.Destination, &dm.Income, &dm.Expense); err != nil {
			continue
		}
		dm.Net = dm.Income - dm.Expense
		if totalNet != 0 {
			dm.Percentage = (float64(dm.Net) / float64(abs(totalNet))) * 100
		}
		breakdown.Destinations = append(breakdown.Destinations, dm)
	}

	return breakdown, nil
}

// Helper function to get top transactions
func (r *FinanceTransactionRepository) getTopTransactions(ctx context.Context, from, to time.Time) (*models.TopTransactions, error) {
	topTransactions := &models.TopTransactions{}

	// Largest incomes
	incomeQuery := `
		SELECT id, amount, destination, category, occurred_at
		FROM finance_transactions
		WHERE occurred_at >= $1 AND occurred_at <= $2 AND type = 'income'
		ORDER BY amount DESC
		LIMIT 10
	`

	rows, err := db.DB.QueryContext(ctx, incomeQuery, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tt models.TopTransaction
		var category sql.NullString
		var occurredAt time.Time
		if err := rows.Scan(&tt.ID, &tt.Amount, &tt.Destination, &category, &occurredAt); err != nil {
			continue
		}
		if category.Valid {
			tt.Category = category.String
		}
		tt.OccurredAt = occurredAt.Format(time.RFC3339)
		topTransactions.LargestIncomes = append(topTransactions.LargestIncomes, tt)
	}

	// Largest expenses
	expenseQuery := `
		SELECT id, amount, destination, category, occurred_at
		FROM finance_transactions
		WHERE occurred_at >= $1 AND occurred_at <= $2 AND type = 'expense'
		ORDER BY amount DESC
		LIMIT 10
	`

	rows, err = db.DB.QueryContext(ctx, expenseQuery, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tt models.TopTransaction
		var category sql.NullString
		var occurredAt time.Time
		if err := rows.Scan(&tt.ID, &tt.Amount, &tt.Destination, &category, &occurredAt); err != nil {
			continue
		}
		if category.Valid {
			tt.Category = category.String
		}
		tt.OccurredAt = occurredAt.Format(time.RFC3339)
		topTransactions.LargestExpenses = append(topTransactions.LargestExpenses, tt)
	}

	return topTransactions, nil
}

// Helper function to calculate KPIs
func (r *FinanceTransactionRepository) calculateKPIs(metrics *models.PeriodMetrics, from, to time.Time, byCategory *models.CategoryBreakdown) models.KPIs {
	kpis := models.KPIs{
		ProfitMargin:          metrics.ProfitMargin,
		AverageTransactionSize: metrics.AverageTransaction,
	}

	// Expense ratio
	if metrics.Income > 0 {
		kpis.ExpenseRatio = (float64(metrics.Expense) / float64(metrics.Income)) * 100
	}

	// Average daily net
	days := int(to.Sub(from).Hours()/24) + 1
	if days > 0 {
		kpis.AverageDailyNet = float64(metrics.Net) / float64(days)
		kpis.TransactionsPerDay = float64(metrics.TransactionCount) / float64(days)
	}

	// Largest categories
	if len(byCategory.Expense) > 0 {
		kpis.LargestExpenseCategory = byCategory.Expense[0].Category
	}
	if len(byCategory.Income) > 0 {
		kpis.LargestIncomeCategory = byCategory.Income[0].Category
	}

	return kpis
}

// Helper function to calculate trends
func (r *FinanceTransactionRepository) calculateTrends(current, previous *models.PeriodMetrics) models.Trends {
	trends := models.Trends{}

	// Income trend
	if current.Income > previous.Income {
		trends.IncomeTrend = "increasing"
	} else if current.Income < previous.Income {
		trends.IncomeTrend = "decreasing"
	} else {
		trends.IncomeTrend = "stable"
	}

	// Expense trend
	if current.Expense > previous.Expense {
		trends.ExpenseTrend = "increasing"
	} else if current.Expense < previous.Expense {
		trends.ExpenseTrend = "decreasing"
	} else {
		trends.ExpenseTrend = "stable"
	}

	// Net trend
	if current.Net > previous.Net {
		trends.NetTrend = "increasing"
	} else if current.Net < previous.Net {
		trends.NetTrend = "decreasing"
	} else {
		trends.NetTrend = "stable"
	}

	// Profit margin trend
	if current.ProfitMargin > previous.ProfitMargin {
		trends.ProfitMarginTrend = "improving"
	} else if current.ProfitMargin < previous.ProfitMargin {
		trends.ProfitMarginTrend = "declining"
	} else {
		trends.ProfitMarginTrend = "stable"
	}

	return trends
}

