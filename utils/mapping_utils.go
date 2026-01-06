package utils

import (
	"strings"
)

// MapColorToCode maps color names to their corresponding codes
// Input is normalized to lowercase before mapping
// Returns uppercase code
func MapColorToCode(color string) string {
	colorLower := strings.ToLower(strings.TrimSpace(color))

	colorMap := map[string]string{
		"amarillo jaspeado":    "AM_JS",
		"azul cielo":           "AC",
		"amarillo":             "AM",
		"fucsia":               "FS",
		"rosado":               "RS",
		"tabaco":               "TA",
		"azul cielo estampado": "AC_ES",
		"azul petróleo":        "AP",
		"rojo":                 "RO",
		"verde limón":          "VL",
		"café":                 "CF",
		"naranja":              "NA",
		"tela tipo franela":    "TE_CA",
		"gris jaspeado":        "GR_JS",
		"moraleche":            "ML",
		"negro":                "NG",
		"palo de rosa":         "PR",
		"rosa claro":           "RP",
		"rosado estampado":     "RS_ES",
		"rosado jaspeado":      "RS_JS",
		"verde sapo":           "VS",
		"verde militar":        "VM",
	}

	if code, exists := colorMap[colorLower]; exists {
		return code
	}

	// If not found, return uppercase version of input
	return strings.ToUpper(colorLower)
}

// MapHoodieTypeToCode maps hoodie type names to their corresponding codes
// Input is normalized to lowercase before mapping
// Returns uppercase code
func MapHoodieTypeToCode(hoodieType string) string {
	hoodieLower := strings.ToLower(strings.TrimSpace(hoodieType))

	hoodieMap := map[string]string{
		"buso estándar":       "BU",
		"buso tipo esqueleto": "BE",
		"camiseta":            "CA",
		"impermeable":         "IM",
		"camiseta halloween":  "HW",
		"pañoleta":            "PA",
		"buso sin mangas":     "BC",
	}

	if code, exists := hoodieMap[hoodieLower]; exists {
		return code
	}

	// If not found, return uppercase version of input
	return strings.ToUpper(hoodieLower)
}

// MapImageTypeToCode maps image type names to their corresponding codes
// Input is normalized to lowercase before mapping
// Returns uppercase code
func MapImageTypeToCode(imageType string) string {
	imageLower := strings.ToLower(strings.TrimSpace(imageType))

	imageMap := map[string]string{
		"buso pequeño (tallas mini - intermedio)": "IT",
		"buso estándar (tallas xs - s - m - l)":   "DP",
		"buso grande (tallas xl)":                 "XL",
	}

	if code, exists := imageMap[imageLower]; exists {
		return code
	}

	// If not found, return uppercase version of input
	return strings.ToUpper(imageLower)
}

// MapCodeToColor maps color codes back to their readable names
// Input is normalized to uppercase before mapping
// Returns lowercase readable name
func MapCodeToColor(code string) string {
	codeUpper := strings.ToUpper(strings.TrimSpace(code))

	codeToColorMap := map[string]string{
		"AM_JS": "amarillo jaspeado",
		"AC":    "azul cielo",
		"AM":    "amarillo",
		"FS":    "fucsia",
		"RS":    "rosado",
		"TA":    "tabaco",
		"AC_ES": "azul cielo estampado",
		"AP":    "azul petróleo",
		"RO":    "rojo",
		"VL":    "verde limón",
		"CF":    "café",
		"NA":    "naranja",
		"TE_CA": "tela tipo franela",
		"GR_JS": "gris jaspeado",
		"ML":    "moraleche",
		"NG":    "negro",
		"PR":    "palo de rosa",
		"RP":    "rosa claro",
		"RS_ES": "rosado estampado",
		"RS_JS": "rosado jaspeado",
		"VS":    "verde sapo",
		"VM":    "verde militar",
	}

	if color, exists := codeToColorMap[codeUpper]; exists {
		return color
	}

	// If not found, return lowercase version of input
	return strings.ToLower(codeUpper)
}

// MapCodeToHoodieType maps hoodie type codes back to their readable names
// Input is normalized to uppercase before mapping
// Returns lowercase readable name
func MapCodeToHoodieType(code string) string {
	codeUpper := strings.ToUpper(strings.TrimSpace(code))

	codeToHoodieMap := map[string]string{
		"BU": "buso estándar",
		"BE": "buso tipo esqueleto",
		"CA": "camiseta",
		"IM": "impermeable",
		"HW": "camiseta halloween",
		"PA": "pañoleta",
		"BC": "buso sin mangas",
	}

	if hoodieType, exists := codeToHoodieMap[codeUpper]; exists {
		return hoodieType
	}

	// If not found, return lowercase version of input
	return strings.ToLower(codeUpper)
}

// MapCodeToImageType maps image type codes back to their readable names
// Supports both old format (IT, DP, XL) and new format (ItMn, MnSML, etc.)
// Input is normalized before mapping
// Returns comma-separated readable names (e.g., "Intermedio,Mini")
func MapCodeToImageType(code string) string {
	codeTrimmed := strings.TrimSpace(code)
	if codeTrimmed == "" {
		return ""
	}

	// Check if code contains lowercase letters (indicates new format)
	hasLowercase := false
	for _, char := range codeTrimmed {
		if char >= 'a' && char <= 'z' {
			hasLowercase = true
			break
		}
	}

	// New format: parse concatenated codes (e.g., "ItMn", "MnSML")
	if hasLowercase {
		// Mapping from codes to readable names
		codeToNameMap := map[string]string{
			"Mn": "Mini",
			"It": "Intermedio",
			"X":  "XS",
			"S":  "S",
			"M":  "M",
			"L":  "L",
			"H":  "XL",
		}

		var result []string
		remaining := codeTrimmed

		// Try to match codes in order (longest first to avoid partial matches)
		// Order matters: "Mn" must come before "M", "It" must come before any single char
		codes := []string{"Mn", "It", "X", "S", "M", "L", "H"}
		
		for len(remaining) > 0 {
			matched := false
			for _, codeKey := range codes {
				if strings.HasPrefix(remaining, codeKey) {
					if name, exists := codeToNameMap[codeKey]; exists {
						result = append(result, name)
						remaining = remaining[len(codeKey):]
						matched = true
						break
					}
				}
			}
			if !matched {
				// Skip unknown character
				remaining = remaining[1:]
			}
		}

		if len(result) > 0 {
			return strings.Join(result, ",")
		}
	}

	// Old format: check for old format codes (for backward compatibility)
	codeUpper := strings.ToUpper(codeTrimmed)
	codeToImageMap := map[string]string{
		"IT": "buso pequeño (tallas mini - intermedio)",
		"DP": "buso estándar (tallas xs - s - m - l)",
		"XL": "buso grande (tallas xl)",
	}

	if imageType, exists := codeToImageMap[codeUpper]; exists {
		return imageType
	}

	// If no matches found, return lowercase version of input
	return strings.ToLower(codeTrimmed)
}

// MapCodeToDecoBase maps deco base codes back to their readable names
// Input is normalized to uppercase before mapping
// Returns capitalized readable name
func MapCodeToDecoBase(code string) string {
	codeUpper := strings.ToUpper(strings.TrimSpace(code))

	codeToDecoBaseMap := map[string]string{
		"0": "N/A",
		"C": "Círculo",
		"N": "Nube",
	}

	if decoBase, exists := codeToDecoBaseMap[codeUpper]; exists {
		return decoBase
	}

	// If not found, return uppercase version of input
	return codeUpper
}

// ParseImageTypeSizes parses comma-separated size values and returns concatenated codes
// Input format: "Intermedio,Mini,XS" or "Mini,S,M,L"
// Returns: "ItMnX" or "MnSML"
// Mapping:
//   - Mini -> Mn
//   - Intermedio -> It
//   - XS -> X
//   - S -> S
//   - M -> M
//   - L -> L
//   - XL -> H
func ParseImageTypeSizes(imageType string) string {
	// Normalize input to lowercase and trim
	imageTypeLower := strings.ToLower(strings.TrimSpace(imageType))
	
	// Split by comma
	parts := strings.Split(imageTypeLower, ",")
	
	// Mapping from input values to codes
	sizeMap := map[string]string{
		"mini":       "Mn",
		"intermedio": "It",
		"xs":         "X",
		"s":          "S",
		"m":          "M",
		"l":          "L",
		"xl":         "H",
	}
	
	// Track seen codes to avoid duplicates
	seenCodes := make(map[string]bool)
	var result strings.Builder
	
	// Process each part
	for _, part := range parts {
		partTrimmed := strings.TrimSpace(part)
		if partTrimmed == "" {
			continue
		}
		
		// Get code from map
		if code, exists := sizeMap[partTrimmed]; exists {
			// Only add if not already seen
			if !seenCodes[code] {
				result.WriteString(code)
				seenCodes[code] = true
			}
		}
		// Unknown values are ignored
	}
	
	return result.String()
}
