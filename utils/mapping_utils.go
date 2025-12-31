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
