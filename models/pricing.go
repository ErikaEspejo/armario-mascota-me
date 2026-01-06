package models

// PricingLine represents pricing information for a single order line
type PricingLine struct {
	LineID      int64    `json:"lineId"`      // ID from reserved_order_lines
	ItemID      int64    `json:"itemId"`      // Item ID
	Qty         int      `json:"qty"`         // Total quantity
	QtyInBundle int      `json:"qtyInBundle"` // Quantity included in bundle promo (if any)
	QtyRetail   int      `json:"qtyRetail"`   // Quantity at retail price
	UnitPrice   int64    `json:"unitPrice"`   // Unit price applied (retail or wholesale)
	LineTotal   int64    `json:"lineTotal"`   // Total for this line
	RuleIDs     []string `json:"ruleIds"`     // IDs of rules applied to this line
}

// PricingBreakdown represents the complete pricing calculation result
type PricingBreakdown struct {
	Total       int64         `json:"total"`       // Total order amount
	Lines       []PricingLine `json:"lines"`       // Pricing breakdown per line
	AppliedRules []string     `json:"appliedRules"` // List of rule IDs applied
	OrderType   string        `json:"orderType"`   // Calculated order type: "mayorista" or "detal"
}

