package utils

import "strings"

// CalculatePrice calculates the price in cents based on buso_type (hoodie_type) and size
// busoType: 'CA' for Camiseta, any other value for Buso
// Returns price in cents
func CalculatePrice(busoType string, size string) int {
	// Normalize size to uppercase for comparison
	sizeUpper := strings.ToUpper(strings.TrimSpace(size))

	// Normalize size aliases
	// Mini can be "MINI" or "MN"
	// Intermedio can be "INTERMEDIO" or "IT"
	if sizeUpper == "MINI" || sizeUpper == "MN" {
		sizeUpper = "MN"
	} else if sizeUpper == "INTERMEDIO" || sizeUpper == "IT" {
		sizeUpper = "IT"
	}

	// If buso_type is 'CA', it's a Camiseta
	if busoType == "CA" {
		// Camiseta prices
		switch sizeUpper {
		case "M", "S", "XS":
			return 10000
		case "MN", "IT":
			return 8000
		default:
			// Default price for Camiseta if size doesn't match
			return 10000
		}
	}

	// Buso prices (any other buso_type value)
	switch sizeUpper {
	case "XL":
		return 22000
	case "L":
		return 15000
	case "M", "S", "XS":
		return 12000
	case "MN", "IT":
		return 8000
	default:
		// Default price for Buso if size doesn't match
		return 12000
	}
}

