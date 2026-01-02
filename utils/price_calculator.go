package utils

import "strings"

// OrderType constants
const (
	OrderTypeDetal     = "detal"
	OrderTypeMayorista = "mayorista"
)

// PriceConfig holds price configuration for different product types and sizes
type PriceConfig struct {
	// Retail prices (precios al detalle)
	RetailPrices map[string]map[string]int // [productType][size] = price in cents
	// Wholesale prices (precios por mayor)
	WholesalePrices map[string]map[string]int // [productType][size] = price in cents
}

var priceConfig = &PriceConfig{
	RetailPrices: map[string]map[string]int{
		"CA": { // Camiseta
			"M":   10000,
			"S":   10000,
			"XS":  10000,
			"MN":  8000,
			"IT":  8000,
		},
		"Buso": { // Buso (cualquier tipo que no sea CA)
			"XL":  22000,
			"L":   15000,
			"M":   12000,
			"S":   12000,
			"XS":  12000,
			"MN":  8000,
			"IT":  8000,
		},
	},
	WholesalePrices: map[string]map[string]int{
		"CA": { // Camiseta
			"M":   9000,
			"S":   9000,
			"XS":  9000,
			"MN":  6000,
			"IT":  6000,
		},
		"Buso": { // Buso
			"XL":  20000,
			"L":   12500,
			"M":   9500,
			"S":   9500,
			"XS":  9500,
			"MN":  6500,
			"IT":  6500,
		},
	},
}

// NormalizeSize normalizes size values to standard format
// Mini -> MN, Intermedio -> IT
// This function is exported so it can be used by other packages
func NormalizeSize(size string) string {
	sizeUpper := strings.ToUpper(strings.TrimSpace(size))
	
	// Normalize size aliases
	if sizeUpper == "MINI" || sizeUpper == "MN" {
		return "MN"
	}
	if sizeUpper == "INTERMEDIO" || sizeUpper == "IT" {
		return "IT"
	}
	
	return sizeUpper
}

// normalizeSize is an internal alias for NormalizeSize
func normalizeSize(size string) string {
	return NormalizeSize(size)
}

// getProductType returns the product type key based on busoType
func getProductType(busoType string) string {
	if busoType == "CA" {
		return "CA"
	}
	return "Buso"
}

// CalculatePrice calculates the price in cents based on buso_type (hoodie_type), size, and order type
// busoType: 'CA' for Camiseta, any other value for Buso
// size: size of the product
// orderType: "detal" for retail prices, "mayorista" for wholesale prices (case-insensitive)
// Returns price in cents
func CalculatePrice(busoType string, size string, orderType string) int {
	normalizedSize := normalizeSize(size)
	productType := getProductType(busoType)
	
	// Normalize orderType to lowercase for comparison
	normalizedOrderType := strings.ToLower(strings.TrimSpace(orderType))
	
	// Select price map based on order type
	var priceMap map[string]int
	if normalizedOrderType == OrderTypeMayorista {
		priceMap = priceConfig.WholesalePrices[productType]
	} else {
		// Default to retail prices (detal)
		priceMap = priceConfig.RetailPrices[productType]
	}
	
	// Get price for the specific size
	if priceMap != nil {
		if price, exists := priceMap[normalizedSize]; exists {
			return price
		}
	}
	
	// Default prices if size doesn't match
	if normalizedOrderType == OrderTypeMayorista {
		if productType == "CA" {
			return 9000 // Default wholesale price for Camiseta
		}
		return 9500 // Default wholesale price for Buso
	}
	
	// Default retail prices (detal)
	if productType == "CA" {
		return 10000 // Default retail price for Camiseta
	}
	return 12000 // Default retail price for Buso
}

// CalculatePriceLegacy calculates the price using the old method (retail/detal only)
// This is kept for backward compatibility with existing code
// busoType: 'CA' for Camiseta, any other value for Buso
// Returns price in cents (retail/detal price)
func CalculatePriceLegacy(busoType string, size string) int {
	return CalculatePrice(busoType, size, OrderTypeDetal)
}

