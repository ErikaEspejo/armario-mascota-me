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
	_ = strings.ToUpper(colorParts[0]) // colorPrimary - not used in current model
	_ = strings.ToUpper(colorParts[1])  // colorSecondary - not used in current model

	// Part 1: BUSO
	_ = strings.ToUpper(parts[1]) // busoType - not used in current model

	// Part 2: TIPOIMAGENIDDECORACION
	// Extract image type (IT, DP, or XL) and decoration ID
	imageTypeRegex := regexp.MustCompile(`^(IT|DP|XL)(\d+)$`)
	matches := imageTypeRegex.FindStringSubmatch(strings.ToUpper(parts[2]))
	if len(matches) != 3 {
		return nil, fmt.Errorf("invalid image type and decoration ID format: expected TIPOIMAGENIDDECORACION (e.g., IT0001), got %s", parts[2])
	}
	_ = matches[1] // imageType - not used in current model
	_ = matches[2] // decoID - not used in current model

	// Part 3: BASE
	decoBase := strings.ToUpper(parts[3])

	// Validate base value
	if decoBase != "C" && decoBase != "0" && decoBase != "N" {
		return nil, fmt.Errorf("invalid base value: expected C, 0, or N, got %s", decoBase)
	}
	_ = decoBase // decoBase - not used in current model

	// Note: DesignAsset model only contains DriveFileID and ImageURL
	// This parser function may be legacy code and is not currently used
	// Returning minimal struct to maintain compatibility
	return &models.DesignAsset{
		DriveFileID: "", // Will be set by caller if needed
		ImageURL:    "", // Will be set by caller if needed
	}, nil
}





