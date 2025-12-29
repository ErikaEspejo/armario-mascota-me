package utils

import (
	"fmt"
	"regexp"
	"strings"

	"armario-mascota-me/models"
)

// ParseFileName parses a filename following the pattern:
// COLOR1_COLOR2-BUSO-TIPOIMAGENIDDECORACION-BASE.PNG
// Example: RO_RO-BE-IT0001-C.PNG
func ParseFileName(filename string) (*models.DesignAsset, error) {
	// Remove extension (case-insensitive)
	extRegex := regexp.MustCompile(`\.(png|jpg|jpeg)$`)
	nameWithoutExt := extRegex.ReplaceAllString(strings.ToLower(filename), "")
	
	// Split by hyphen
	parts := strings.Split(nameWithoutExt, "-")
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid filename format: expected 4 parts separated by '-', got %d parts", len(parts))
	}

	// Part 0: COLOR1_COLOR2
	colorParts := strings.Split(parts[0], "_")
	if len(colorParts) != 2 {
		return nil, fmt.Errorf("invalid color format: expected COLOR1_COLOR2, got %s", parts[0])
	}
	colorPrimary := strings.ToUpper(colorParts[0])
	colorSecondary := strings.ToUpper(colorParts[1])

	// Part 1: BUSO
	busoType := strings.ToUpper(parts[1])

	// Part 2: TIPOIMAGENIDDECORACION
	// Extract image type (IT, DP, or XL) and decoration ID
	imageTypeRegex := regexp.MustCompile(`^(IT|DP|XL)(\d+)$`)
	matches := imageTypeRegex.FindStringSubmatch(strings.ToUpper(parts[2]))
	if len(matches) != 3 {
		return nil, fmt.Errorf("invalid image type and decoration ID format: expected TIPOIMAGENIDDECORACION (e.g., IT0001), got %s", parts[2])
	}
	imageType := matches[1]
	decoID := matches[2]

	// Part 3: BASE
	decoBase := strings.ToUpper(parts[3])

	// Validate base value
	if decoBase != "C" && decoBase != "0" && decoBase != "N" {
		return nil, fmt.Errorf("invalid base value: expected C, 0, or N, got %s", decoBase)
	}

	return &models.DesignAsset{
		ColorPrimary:   colorPrimary,
		ColorSecondary: colorSecondary,
		BusoType:       busoType,
		ImageType:      imageType,
		DecoID:         decoID,
		DecoBase:       decoBase,
	}, nil
}



