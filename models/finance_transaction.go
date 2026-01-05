package models

// FinanceTransaction represents a financial transaction in the database
type FinanceTransaction struct {
	ID          int64  `json:"id"`
	Type        string `json:"type"` // 'income' or 'expense'
	Source      string `json:"source"`
	SourceID    *int64 `json:"sourceId,omitempty"` // nullable for manual transactions
	OccurredAt  string `json:"occurredAt"`
	Amount      int64  `json:"amount"`
	Destination string `json:"destination"`
	Category    string `json:"category,omitempty"`
	Counterparty string `json:"counterparty,omitempty"`
	Notes       string `json:"notes,omitempty"`
	CreatedAt   string `json:"createdAt"`
}

// CreateFinanceTransactionRequest represents the request body for creating a finance transaction
// Example: {
//   "type": "expense",
//   "amount": 45000,
//   "destination": "Caja",
//   "category": "materiales",
//   "counterparty": "Proveedor telas",
//   "notes": "Franela 10m"
// }
type CreateFinanceTransactionRequest struct {
	Type        string `json:"type"`                  // 'income' or 'expense'
	Amount      int64  `json:"amount"`                // required, must be > 0
	Destination string `json:"destination"`           // required
	Category    string `json:"category,omitempty"`    // optional
	Counterparty string `json:"counterparty,omitempty"` // optional
	Notes       string `json:"notes,omitempty"`       // optional
	OccurredAt  string `json:"occurredAt,omitempty"`  // optional, defaults to now
}

// FinanceTransactionListRequest represents query parameters for listing transactions
type FinanceTransactionListRequest struct {
	From       *string `json:"from,omitempty"`       // YYYY-MM-DD
	To         *string `json:"to,omitempty"`         // YYYY-MM-DD
	Type       *string `json:"type,omitempty"`      // 'income' or 'expense'
	Source     *string `json:"source,omitempty"`    // 'sale' or 'manual'
	Destination *string `json:"destination,omitempty"` // account name
	Category   *string `json:"category,omitempty"` // category name
	Q          *string `json:"q,omitempty"`         // text search in notes and counterparty
	Limit      int     `json:"limit,omitempty"`     // default 50, max 200
	Cursor     *string `json:"cursor,omitempty"`    // pagination cursor
}

// FinanceTransactionListResponse represents the response for listing transactions
type FinanceTransactionListResponse struct {
	Transactions []FinanceTransaction `json:"transactions"`
	Pagination   PaginationInfo      `json:"pagination"`
}

// PaginationInfo represents pagination metadata
type PaginationInfo struct {
	Limit     int     `json:"limit"`
	NextCursor *string `json:"nextCursor,omitempty"`
}

// FinanceSummaryResponse represents the summary/balance response
type FinanceSummaryResponse struct {
	Currency            string                    `json:"currency"`
	BalanceAllTime     int64                     `json:"balanceAllTime"`
	ByDestinationAllTime []DestinationBalance    `json:"byDestinationAllTime"`
	Range              *SummaryRange             `json:"range,omitempty"`
	ByDestinationRange []DestinationRangeBalance `json:"byDestinationRange,omitempty"`
}

// DestinationBalance represents balance for a destination
type DestinationBalance struct {
	Destination string `json:"destination"`
	Balance     int64  `json:"balance"`
}

// SummaryRange represents balance calculations for a date range
type SummaryRange struct {
	From           string `json:"from"`
	To             string `json:"to"`
	OpeningBalance int64  `json:"openingBalance"`
	Income         int64  `json:"income"`
	Expense        int64  `json:"expense"`
	Net            int64  `json:"net"`
	ClosingBalance int64  `json:"closingBalance"`
}

// DestinationRangeBalance represents balance by destination for a date range
type DestinationRangeBalance struct {
	Destination string `json:"destination"`
	Income      int64  `json:"income"`
	Expense     int64  `json:"expense"`
	Net         int64  `json:"net"`
}

// FinanceDashboardRequest represents query parameters for dashboard
type FinanceDashboardRequest struct {
	Period      *string `json:"period,omitempty"`      // 'month', 'quarter', 'year'
	From        *string `json:"from,omitempty"`         // YYYY-MM-DD
	To          *string `json:"to,omitempty"`           // YYYY-MM-DD
	CompareWith *string `json:"compareWith,omitempty"`  // 'previous', 'last_year'
}

// FinanceDashboardResponse represents the dashboard response
type FinanceDashboardResponse struct {
	Currency      string          `json:"currency"`
	Period        PeriodInfo      `json:"period"`
	CurrentPeriod PeriodMetrics   `json:"currentPeriod"`
	Comparison    *ComparisonData `json:"comparison,omitempty"`
	CashFlow      CashFlowData    `json:"cashFlow"`
	ByCategory    CategoryBreakdown `json:"byCategory"`
	ByCounterparty CounterpartyBreakdown `json:"byCounterparty"`
	ByDestination DestinationBreakdown `json:"byDestination"`
	TopTransactions TopTransactions `json:"topTransactions"`
	KPIs          KPIs            `json:"kpis"`
	Trends        Trends          `json:"trends"`
}

// PeriodInfo represents period information
type PeriodInfo struct {
	Type  string `json:"type"`  // 'month', 'quarter', 'year', 'custom'
	From  string `json:"from"` // YYYY-MM-DD
	To    string `json:"to"`   // YYYY-MM-DD
	Label string `json:"label"`
}

// PeriodMetrics represents metrics for a period
type PeriodMetrics struct {
	Income            int64   `json:"income"`
	Expense           int64   `json:"expense"`
	Net               int64   `json:"net"`
	TransactionCount int     `json:"transactionCount"`
	AverageTransaction float64 `json:"averageTransaction"`
	ProfitMargin      float64 `json:"profitMargin"`
}

// ComparisonData represents comparison with another period
type ComparisonData struct {
	Type          string        `json:"type"` // 'previous', 'last_year'
	PreviousPeriod PeriodMetrics `json:"previousPeriod"`
	PreviousPeriodInfo PeriodInfo `json:"previousPeriodInfo"`
	Changes       PeriodChanges `json:"changes"`
}

// PeriodChanges represents percentage changes between periods
type PeriodChanges struct {
	IncomeChange       float64 `json:"incomeChange"`
	ExpenseChange      float64 `json:"expenseChange"`
	NetChange          float64 `json:"netChange"`
	ProfitMarginChange float64 `json:"profitMarginChange"`
}

// CashFlowData represents cash flow time series
type CashFlowData struct {
	Daily   []DailyCashFlow   `json:"daily"`
	Weekly  []WeeklyCashFlow  `json:"weekly"`
	Monthly []MonthlyCashFlow `json:"monthly"`
}

// DailyCashFlow represents daily cash flow
type DailyCashFlow struct {
	Date    string `json:"date"` // YYYY-MM-DD
	Income  int64  `json:"income"`
	Expense int64  `json:"expense"`
	Net     int64  `json:"net"`
}

// WeeklyCashFlow represents weekly cash flow
type WeeklyCashFlow struct {
	Week    string `json:"week"` // YYYY-Www
	Income  int64  `json:"income"`
	Expense int64  `json:"expense"`
	Net     int64  `json:"net"`
}

// MonthlyCashFlow represents monthly cash flow
type MonthlyCashFlow struct {
	Month   string `json:"month"` // YYYY-MM
	Income  int64  `json:"income"`
	Expense int64  `json:"expense"`
	Net     int64  `json:"net"`
}

// CategoryBreakdown represents breakdown by category
type CategoryBreakdown struct {
	Income  []CategoryAmount `json:"income"`
	Expense []CategoryAmount `json:"expense"`
}

// CategoryAmount represents amount by category
type CategoryAmount struct {
	Category  string  `json:"category"`
	Amount    int64   `json:"amount"`
	Percentage float64 `json:"percentage"`
	Count     int     `json:"count"`
}

// CounterpartyBreakdown represents breakdown by counterparty
type CounterpartyBreakdown struct {
	TopExpenses []CounterpartyAmount `json:"topExpenses"`
	TopIncomes  []CounterpartyAmount `json:"topIncomes"`
}

// CounterpartyAmount represents amount by counterparty
type CounterpartyAmount struct {
	Counterparty string `json:"counterparty"`
	Amount       int64  `json:"amount"`
	Count        int    `json:"count"`
}

// DestinationBreakdown represents breakdown by destination
type DestinationBreakdown struct {
	Destinations []DestinationMetrics `json:"destinations"`
	TotalNet     int64                `json:"totalNet"`
}

// DestinationMetrics represents metrics for a destination
type DestinationMetrics struct {
	Destination string  `json:"destination"`
	Income      int64   `json:"income"`
	Expense     int64   `json:"expense"`
	Net         int64   `json:"net"`
	Percentage  float64 `json:"percentage"`
}

// TopTransactions represents top transactions
type TopTransactions struct {
	LargestIncomes  []TopTransaction `json:"largestIncomes"`
	LargestExpenses []TopTransaction `json:"largestExpenses"`
}

// TopTransaction represents a top transaction
type TopTransaction struct {
	ID         int64  `json:"id"`
	Amount     int64  `json:"amount"`
	Destination string `json:"destination"`
	Category   string `json:"category,omitempty"`
	OccurredAt string `json:"occurredAt"`
}

// KPIs represents key performance indicators
type KPIs struct {
	ProfitMargin          float64 `json:"profitMargin"`
	ExpenseRatio          float64 `json:"expenseRatio"`
	AverageDailyNet       float64 `json:"averageDailyNet"`
	AverageTransactionSize float64 `json:"averageTransactionSize"`
	TransactionsPerDay    float64 `json:"transactionsPerDay"`
	LargestExpenseCategory string  `json:"largestExpenseCategory"`
	LargestIncomeCategory  string  `json:"largestIncomeCategory"`
}

// Trends represents trend indicators
type Trends struct {
	IncomeTrend       string `json:"incomeTrend"`       // 'increasing', 'decreasing', 'stable'
	ExpenseTrend     string `json:"expenseTrend"`       // 'increasing', 'decreasing', 'stable'
	NetTrend         string `json:"netTrend"`           // 'increasing', 'decreasing', 'stable'
	ProfitMarginTrend string `json:"profitMarginTrend"`  // 'improving', 'declining', 'stable'
}

