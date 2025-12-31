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
// Input is normalized to uppercase before mapping
// Returns lowercase readable name
func MapCodeToImageType(code string) string {
	codeUpper := strings.ToUpper(strings.TrimSpace(code))

	codeToImageMap := map[string]string{
		"IT": "buso pequeño (tallas mini - intermedio)",
		"DP": "buso estándar (tallas xs - s - m - l)",
		"XL": "buso grande (tallas xl)",
	}

	if imageType, exists := codeToImageMap[codeUpper]; exists {
		return imageType
	}

	// If not found, return lowercase version of input
	return strings.ToLower(codeUpper)
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
