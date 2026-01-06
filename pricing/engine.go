package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"armario-mascota-me/db"
	"armario-mascota-me/models"
	"armario-mascota-me/utils"
)

// PricingConfig represents the pricing configuration structure
type PricingConfig struct {
	Currency    string                 `json:"currency"`
	Groups      map[string]GroupConfig `json:"groups"`
	SizeBuckets map[string]string      `json:"sizeBuckets"`
	Pricebook   map[string]map[string]PriceEntry `json:"pricebook"`
	Rules       []Rule                 `json:"rules"`
}

type GroupConfig struct {
	IncludeTypes []string `json:"includeTypes"`
	ExcludeTypes []string `json:"excludeTypes"`
}

type PriceEntry struct {
	Retail    int64 `json:"retail"`
	Wholesale int64 `json:"wholesale"`
}

type Rule struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Active     bool                   `json:"active"`
	Priority   int                    `json:"priority"`
	Type       string                 `json:"type"`
	Conditions map[string]interface{} `json:"conditions"`
	Action     map[string]interface{} `json:"action,omitempty"`
}

// OrderLineInput represents input data for pricing calculation
type OrderLineInput struct {
	LineID     int64
	ItemID     int64
	Qty        int
	HoodieType string
	Size       string
	SKU        string
}

// Engine handles pricing calculations based on JSON configuration
type Engine struct {
	config *PricingConfig
}

var engineInstance *Engine

// NewEngine creates a new pricing engine instance
func NewEngine(configPath string) (*Engine, error) {
	if engineInstance != nil {
		return engineInstance, nil
	}

	// Resolve config path
	if !filepath.IsAbs(configPath) {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
		configPath = filepath.Join(wd, configPath)
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read pricing config: %w", err)
	}

	// Parse JSON
	var config PricingConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse pricing config: %w", err)
	}

	// Validate config
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid pricing config: %w", err)
	}

	// Sort rules by priority (highest first)
	sort.Slice(config.Rules, func(i, j int) bool {
		return config.Rules[i].Priority > config.Rules[j].Priority
	})

	engine := &Engine{
		config: &config,
	}

	engineInstance = engine
	log.Printf("✅ PricingEngine: Successfully loaded pricing config from %s", configPath)
	return engine, nil
}

func validateConfig(config *PricingConfig) error {
	if config.Currency == "" {
		return fmt.Errorf("currency is required")
	}
	if len(config.Groups) == 0 {
		return fmt.Errorf("groups are required")
	}
	if len(config.Pricebook) == 0 {
		return fmt.Errorf("pricebook is required")
	}
	return nil
}

// GetEngine returns the singleton pricing engine instance
func GetEngine() *Engine {
	return engineInstance
}

// getGroupForProductType determines which group a product type belongs to
func (e *Engine) getGroupForProductType(productType string) string {
	for groupName, groupConfig := range e.config.Groups {
		// Check if product type is in includeTypes
		for _, includeType := range groupConfig.IncludeTypes {
			if includeType == productType {
				// Check if it's not excluded
				isExcluded := false
				for _, excludeType := range groupConfig.ExcludeTypes {
					if excludeType == productType {
						isExcluded = true
						break
					}
				}
				if !isExcluded {
					return groupName
				}
			}
		}
	}
	return ""
}

// getSizeBucket maps a size to its bucket
func (e *Engine) getSizeBucket(size string) string {
	normalizedSize := utils.NormalizeSize(size)
	if bucket, exists := e.config.SizeBuckets[normalizedSize]; exists {
		return bucket
	}
	// Default: return normalized size if not found
	return normalizedSize
}

// isEligibleForWholesaleCount checks if a product type is eligible for wholesale count
func (e *Engine) isEligibleForWholesaleCount(productType string) bool {
	group := e.getGroupForProductType(productType)
	return group == "BUSOS" || group == "CAMISETAS"
}

// CalculateOrderPricing calculates pricing for an order based on its lines
func (e *Engine) CalculateOrderPricing(ctx context.Context, orderID int64) (*models.PricingBreakdown, error) {
	// Get order lines with product information
	lines, err := e.getOrderLines(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order lines: %w", err)
	}

	if len(lines) == 0 {
		return &models.PricingBreakdown{
			Total:        0,
			Lines:        []models.PricingLine{},
			AppliedRules: []string{},
			OrderType:    "detal",
		}, nil
	}

	// Calculate global eligible quantity (BUSOS + CAMISETAS only)
	globalQtyEligible := 0
	for _, line := range lines {
		if e.isEligibleForWholesaleCount(line.HoodieType) {
			globalQtyEligible += line.Qty
		}
	}

	log.Printf("💰 CalculateOrderPricing: Order %d has %d eligible units (BUSOS+CAMISETAS)", orderID, globalQtyEligible)

	// Check if wholesale override applies (priority 1000)
	wholesaleOverride := false
	for _, rule := range e.config.Rules {
		if !rule.Active {
			continue
		}
		if rule.Type == "wholesale_override" && rule.Priority == 1000 {
			if minQty, ok := rule.Conditions["minQty"].(float64); ok {
				if globalQtyEligible >= int(minQty) {
					wholesaleOverride = true
					log.Printf("💰 Wholesale override applies: %d >= %d", globalQtyEligible, int(minQty))
					break
				}
			}
		}
	}

	// Calculate pricing
	var breakdown *models.PricingBreakdown
	if wholesaleOverride {
		breakdown = e.calculateWholesalePricing(lines)
		breakdown.OrderType = "mayorista"
	} else {
		breakdown = e.calculateRetailWithBundles(lines, globalQtyEligible)
		breakdown.OrderType = "detal"
	}

	log.Printf("✅ CalculateOrderPricing: Order %d total = %d, orderType = %s", orderID, breakdown.Total, breakdown.OrderType)
	return breakdown, nil
}

// getOrderLines retrieves order lines with product information
func (e *Engine) getOrderLines(ctx context.Context, orderID int64) ([]OrderLineInput, error) {
	query := `
		SELECT rol.id, rol.item_id, rol.qty,
		       COALESCE(da.hoodie_type, '') as hoodie_type,
		       i.size, i.sku
		FROM reserved_order_lines rol
		INNER JOIN items i ON rol.item_id = i.id
		LEFT JOIN design_assets da ON i.design_asset_id = da.id
		WHERE rol.reserved_order_id = $1
		ORDER BY rol.id ASC
	`

	rows, err := db.DB.QueryContext(ctx, query, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []OrderLineInput
	for rows.Next() {
		var line OrderLineInput
		err := rows.Scan(
			&line.LineID,
			&line.ItemID,
			&line.Qty,
			&line.HoodieType,
			&line.Size,
			&line.SKU,
		)
		if err != nil {
			return nil, err
		}
		log.Printf("💰 getOrderLines: Line %d - ItemID=%d, Size=%s (normalized=%s), HoodieType=%s, Qty=%d", 
			line.LineID, line.ItemID, line.Size, utils.NormalizeSize(line.Size), line.HoodieType, line.Qty)
		lines = append(lines, line)
	}

	return lines, rows.Err()
}

// calculateWholesalePricing calculates wholesale pricing for all eligible items
func (e *Engine) calculateWholesalePricing(lines []OrderLineInput) *models.PricingBreakdown {
	breakdown := &models.PricingBreakdown{
		Total:        0,
		Lines:        []models.PricingLine{},
		AppliedRules: []string{"WHOLESALE_GLOBAL_6PLUS"},
	}

	for _, line := range lines {
		group := e.getGroupForProductType(line.HoodieType)
		sizeBucket := e.getSizeBucket(line.Size)

		var unitPrice int64
		if group == "BUSOS" || group == "CAMISETAS" {
			// Get wholesale price
			if pricebook, exists := e.config.Pricebook[group]; exists {
				if priceEntry, exists := pricebook[sizeBucket]; exists {
					unitPrice = priceEntry.Wholesale
				}
			}
		} else {
			// For IM/PA, use retail price (they don't participate in wholesale)
			if pricebook, exists := e.config.Pricebook["BUSOS"]; exists {
				if priceEntry, exists := pricebook[sizeBucket]; exists {
					unitPrice = priceEntry.Retail
				}
			}
		}

		// Default fallback prices
		if unitPrice == 0 {
			if group == "CAMISETAS" {
				unitPrice = 9000 // Default wholesale for camisetas
			} else {
				unitPrice = 9500 // Default wholesale for busos
			}
		}

		lineTotal := int64(line.Qty) * unitPrice
		breakdown.Total += lineTotal

		breakdown.Lines = append(breakdown.Lines, models.PricingLine{
			LineID:      line.LineID,
			ItemID:      line.ItemID,
			Qty:         line.Qty,
			QtyInBundle: 0,
			QtyRetail:   line.Qty,
			UnitPrice:   unitPrice,
			LineTotal:   lineTotal,
			RuleIDs:     []string{"WHOLESALE_GLOBAL_6PLUS"},
		})
	}

	return breakdown
}

// calculateRetailWithBundles calculates retail pricing with bundle promotions
func (e *Engine) calculateRetailWithBundles(lines []OrderLineInput, globalQtyEligible int) *models.PricingBreakdown {
	breakdown := &models.PricingBreakdown{
		Total:        0,
		Lines:        []models.PricingLine{},
		AppliedRules: []string{},
	}

	// Group lines by group and size bucket for bundle processing
	type LineKey struct {
		Group     string
		SizeBucket string
		LineID    int64
	}

	// Process bundles first
	bundleRules := e.getBundleRules()
	
	// Create a map to track remaining quantities after bundles
	remainingQty := make(map[int64]int)
	for _, line := range lines {
		remainingQty[line.LineID] = line.Qty
	}

	// Track bundle applications
	bundleApplications := make(map[int64]int) // lineID -> qty in bundles
	bundleRuleIDs := make(map[int64][]string)  // lineID -> rule IDs applied

	// Apply bundle rules
	for _, rule := range bundleRules {
		if !rule.Active {
			continue
		}

		// Check onlyIfCartQtyBelow condition
		if onlyIfBelow, ok := rule.Conditions["onlyIfCartQtyBelow"].(float64); ok {
			if globalQtyEligible >= int(onlyIfBelow) {
				log.Printf("💰 Bundle rule %s skipped: cart qty %d >= %d", rule.ID, globalQtyEligible, int(onlyIfBelow))
				continue
			}
		}

		group, _ := rule.Conditions["group"].(string)
		sizes, _ := rule.Conditions["sizes"].([]interface{})
		requiredQty, _ := rule.Conditions["requiredQty"].(float64)
		mixSizes, _ := rule.Conditions["mixSizes"].(bool)
		_, _ = rule.Action["bundleTotalPrice"].(float64) // Will be used later when calculating bundle totals

		// Find eligible lines
		var eligibleLines []OrderLineInput
		log.Printf("💰 Bundle rule %s: Checking rule - group=%s, sizes=%v, mixSizes=%v, requiredQty=%d", 
			rule.ID, group, sizes, mixSizes, int(requiredQty))
		for _, line := range lines {
			lineGroup := e.getGroupForProductType(line.HoodieType)
			lineSizeBucket := e.getSizeBucket(line.Size)

			if lineGroup != group {
				log.Printf("💰 Bundle rule %s: Line %d skipped - group mismatch (lineGroup=%s, ruleGroup=%s)", 
					rule.ID, line.LineID, lineGroup, group)
				continue
			}

			// Check if size matches
			sizeMatch := false
			for _, size := range sizes {
				if sizeStr, ok := size.(string); ok {
					if mixSizes {
						// For mixSizes, check if size bucket matches
						if e.getSizeBucket(sizeStr) == lineSizeBucket {
							sizeMatch = true
							log.Printf("💰 Bundle rule %s: Line %d (size=%s, bucket=%s) matches rule size %s (bucket=%s) - mixSizes=true", 
								rule.ID, line.LineID, line.Size, lineSizeBucket, sizeStr, e.getSizeBucket(sizeStr))
							break
						}
					} else {
						// For non-mix, check exact size match
						normalizedRuleSize := utils.NormalizeSize(sizeStr)
						normalizedLineSize := utils.NormalizeSize(line.Size)
						if normalizedRuleSize == normalizedLineSize {
							sizeMatch = true
							log.Printf("💰 Bundle rule %s: Line %d (size=%s normalized=%s) matches rule size %s (normalized=%s) - mixSizes=false", 
								rule.ID, line.LineID, line.Size, normalizedLineSize, sizeStr, normalizedRuleSize)
							break
						}
					}
				}
			}

			if sizeMatch && remainingQty[line.LineID] > 0 {
				log.Printf("💰 Bundle rule %s: Line %d is eligible - size=%s, remainingQty=%d", 
					rule.ID, line.LineID, line.Size, remainingQty[line.LineID])
				eligibleLines = append(eligibleLines, line)
			} else if sizeMatch {
				log.Printf("💰 Bundle rule %s: Line %d matched size but has no remaining qty (remainingQty=%d)", 
					rule.ID, line.LineID, remainingQty[line.LineID])
			}
		}

		log.Printf("💰 Bundle rule %s: Found %d eligible lines", rule.ID, len(eligibleLines))

		// Sort eligible lines deterministically (by lineID)
		sort.Slice(eligibleLines, func(i, j int) bool {
			return eligibleLines[i].LineID < eligibleLines[j].LineID
		})

		// Apply bundles
		// If mixSizes is false, sizes in the rule can be mixed with each other, but not with sizes outside the list
		// If mixSizes is true, sizes can be mixed based on size buckets
		if !mixSizes {
			// mixSizes=false: Can mix sizes within the rule's sizes list, but not with other sizes
			// All eligible lines can be combined since they all match sizes in the rule
			totalEligibleQty := 0
			for _, line := range eligibleLines {
				totalEligibleQty += remainingQty[line.LineID]
			}

			log.Printf("💰 Bundle rule %s: Total eligible qty=%d, requiredQty=%d (mixSizes=false, can mix sizes within rule)", 
				rule.ID, totalEligibleQty, int(requiredQty))

			bundlesCount := totalEligibleQty / int(requiredQty)
			if bundlesCount > 0 {
				log.Printf("💰 Bundle rule %s: Applying %d bundles (mixSizes=false, totalQty=%d, requiredQty=%d)", 
					rule.ID, bundlesCount, totalEligibleQty, int(requiredQty))
				// Distribute bundle quantities deterministically across all eligible lines
				qtyToDistribute := bundlesCount * int(requiredQty)
				distributed := 0

				for i := range eligibleLines {
					if distributed >= qtyToDistribute {
						break
					}
					line := eligibleLines[i]
					available := remainingQty[line.LineID]
					if available > 0 {
						toTake := qtyToDistribute - distributed
						if toTake > available {
							toTake = available
						}
						remainingQty[line.LineID] -= toTake
						bundleApplications[line.LineID] += toTake
						if bundleRuleIDs[line.LineID] == nil {
							bundleRuleIDs[line.LineID] = []string{}
						}
						bundleRuleIDs[line.LineID] = append(bundleRuleIDs[line.LineID], rule.ID)
						distributed += toTake
						log.Printf("💰 Bundle rule %s: Applied %d units from line %d (size=%s) to bundle", 
							rule.ID, toTake, line.LineID, line.Size)
					}
				}
			}
		} else {
			// mixSizes is true - can mix sizes in bundles
			totalEligibleQty := 0
			for _, line := range eligibleLines {
				totalEligibleQty += remainingQty[line.LineID]
			}

			bundlesCount := totalEligibleQty / int(requiredQty)
			if bundlesCount > 0 {
				log.Printf("💰 Bundle rule %s: Applying %d bundles (mixSizes=true, can mix sizes)", 
					rule.ID, bundlesCount)
				// Distribute bundle quantities deterministically
				qtyToDistribute := bundlesCount * int(requiredQty)
				distributed := 0

				for i := range eligibleLines {
					if distributed >= qtyToDistribute {
						break
					}
					line := eligibleLines[i]
					available := remainingQty[line.LineID]
					if available > 0 {
						toTake := qtyToDistribute - distributed
						if toTake > available {
							toTake = available
						}
						remainingQty[line.LineID] -= toTake
						bundleApplications[line.LineID] += toTake
						if bundleRuleIDs[line.LineID] == nil {
							bundleRuleIDs[line.LineID] = []string{}
						}
						bundleRuleIDs[line.LineID] = append(bundleRuleIDs[line.LineID], rule.ID)
						distributed += toTake
					}
				}
			}
		}

		// Track bundle total (will be distributed to lines later)
		if len(eligibleLines) > 0 {
			breakdown.AppliedRules = append(breakdown.AppliedRules, rule.ID)
		}
	}

	// Calculate bundle totals by rule first
	bundleTotalsByRule := make(map[string]int64) // ruleID -> total bundle price
	for _, rule := range bundleRules {
		if !rule.Active {
			continue
		}
		if bundleTotalPrice, ok := rule.Action["bundleTotalPrice"].(float64); ok {
			if requiredQty, ok := rule.Conditions["requiredQty"].(float64); ok {
				// Count total qty in bundles for this rule
				totalQtyInBundles := 0
				for lineID, qty := range bundleApplications {
					if contains(bundleRuleIDs[lineID], rule.ID) {
						totalQtyInBundles += qty
					}
				}
				bundlesCount := totalQtyInBundles / int(requiredQty)
				if bundlesCount > 0 {
					bundleTotalsByRule[rule.ID] = int64(bundlesCount) * int64(bundleTotalPrice)
				}
			}
		}
	}

	// Calculate retail pricing for remaining quantities and bundle pricing
	for _, line := range lines {
		group := e.getGroupForProductType(line.HoodieType)
		sizeBucket := e.getSizeBucket(line.Size)
		qtyInBundle := bundleApplications[line.LineID]
		qtyRetail := remainingQty[line.LineID]

		// Get retail price
		var retailPrice int64
		if group != "" {
			if pricebook, exists := e.config.Pricebook[group]; exists {
				if priceEntry, exists := pricebook[sizeBucket]; exists {
					retailPrice = priceEntry.Retail
				}
			}
		}

		// Default fallback prices
		if retailPrice == 0 {
			if group == "CAMISETAS" {
				retailPrice = 10000 // Default retail for camisetas
			} else if group == "BUSOS" {
				retailPrice = 12000 // Default retail for busos
			} else {
				// For IM/PA or unknown groups, use a default price
				// Try to get price from BUSOS pricebook as fallback
				if pricebook, exists := e.config.Pricebook["BUSOS"]; exists {
					if priceEntry, exists := pricebook[sizeBucket]; exists {
						retailPrice = priceEntry.Retail
					}
				}
				if retailPrice == 0 {
					retailPrice = 12000 // Ultimate fallback
				}
			}
		}

		// Calculate bundle unit price if this line is in a bundle
		var bundleUnitPrice int64
		ruleIDs := bundleRuleIDs[line.LineID]
		if len(ruleIDs) == 0 {
			ruleIDs = []string{}
		}

		if qtyInBundle > 0 && len(ruleIDs) > 0 {
			// Find the bundle rule to get bundleTotalPrice and requiredQty
			ruleID := ruleIDs[0]
			for _, rule := range bundleRules {
				if rule.ID == ruleID {
					if bundleTotalPrice, ok := rule.Action["bundleTotalPrice"].(float64); ok {
						if requiredQty, ok := rule.Conditions["requiredQty"].(float64); ok {
							// Bundle unit price = bundleTotalPrice / requiredQty
							bundleUnitPrice = int64(bundleTotalPrice) / int64(requiredQty)
							log.Printf("💰 Bundle unit price for line %d: %d (bundleTotal=%d, requiredQty=%d)", 
								line.LineID, bundleUnitPrice, int64(bundleTotalPrice), int64(requiredQty))
							break
						}
					}
				}
			}
		}

		// Calculate totals
		retailTotal := int64(qtyRetail) * retailPrice
		bundleTotal := int64(qtyInBundle) * bundleUnitPrice
		lineTotal := retailTotal + bundleTotal
		breakdown.Total += lineTotal

		// Determine effective unit price for display
		// Priority: If line has units in bundle, show bundle price (promo price)
		// Otherwise show retail price
		var effectiveUnitPrice int64
		if qtyInBundle > 0 {
			// Line has units in bundle - use bundle unit price (promo price)
			// This ensures items affected by promo show the promo price
			effectiveUnitPrice = bundleUnitPrice
			if effectiveUnitPrice == 0 {
				// Fallback if bundleUnitPrice wasn't calculated
				effectiveUnitPrice = retailPrice
			}
		} else {
			// No units in bundle - use retail price
			effectiveUnitPrice = retailPrice
		}

		breakdown.Lines = append(breakdown.Lines, models.PricingLine{
			LineID:      line.LineID,
			ItemID:      line.ItemID,
			Qty:         line.Qty,
			QtyInBundle: qtyInBundle,
			QtyRetail:   qtyRetail,
			UnitPrice:   effectiveUnitPrice, // Effective unit price (bundle price for bundle units, retail for others)
			LineTotal:   lineTotal,
			RuleIDs:     ruleIDs,
		})
	}

	return breakdown
}

// getBundleRules returns active bundle rules sorted by priority
func (e *Engine) getBundleRules() []Rule {
	var bundleRules []Rule
	for _, rule := range e.config.Rules {
		if rule.Active && rule.Type == "bundle_fixed_total" {
			bundleRules = append(bundleRules, rule)
		}
	}
	return bundleRules
}

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// UpdateOrderType updates the order_type field in reserved_orders based on pricing calculation
func (e *Engine) UpdateOrderType(ctx context.Context, orderID int64, orderType string) error {
	query := `UPDATE reserved_orders SET order_type = $1 WHERE id = $2`
	_, err := db.DB.ExecContext(ctx, query, strings.ToLower(orderType), orderID)
	if err != nil {
		return fmt.Errorf("failed to update order_type: %w", err)
	}
	log.Printf("✅ UpdateOrderType: Updated order %d order_type to %s", orderID, orderType)
	return nil
}

