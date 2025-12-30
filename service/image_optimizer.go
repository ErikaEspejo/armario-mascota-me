package service

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
)

const (
	cacheDir = "cache/images"
	// Quality settings
	qualityThumb  = 60
	qualityMedium = 75
	// Size settings (max dimension)
	maxSizeThumb  = 300
	maxSizeMedium = 800
)

// EnsureCacheDir ensures the cache directory exists, creates it if it doesn't
func EnsureCacheDir() error {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}
	return nil
}

// GetCachePath returns the cache file path for a given asset ID and size
func GetCachePath(assetID int, size string) string {
	filename := fmt.Sprintf("design_asset_%d_%s.jpg", assetID, size)
	return filepath.Join(cacheDir, filename)
}

// CacheExists checks if a cached image exists
func CacheExists(cachePath string) bool {
	_, err := os.Stat(cachePath)
	return err == nil
}

// ReadFromCache reads an image from the cache
func ReadFromCache(cachePath string) ([]byte, error) {
	data, err := ioutil.ReadFile(cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read from cache: %w", err)
	}
	return data, nil
}

// SaveToCache saves an image to the cache
func SaveToCache(cachePath string, imageData []byte) error {
	// Ensure parent directory exists
	dir := filepath.Dir(cachePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	if err := ioutil.WriteFile(cachePath, imageData, 0644); err != nil {
		return fmt.Errorf("failed to write to cache: %w", err)
	}

	log.Printf("✓ Image cached: %s", cachePath)
	return nil
}

// OptimizeImage optimizes an image by converting to JPEG and resizing
// imageData: raw image bytes (PNG, JPEG, etc.)
// size: "thumb" or "medium"
// Returns optimized JPEG image bytes
// Note: Using JPEG instead of WebP to avoid CGO dependency. Can be changed to WebP later if needed.
func OptimizeImage(imageData []byte, size string) ([]byte, error) {
	// Decode the image
	img, format, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	log.Printf("📸 Image decoded: format=%s, bounds=%v", format, img.Bounds())

	// Determine max dimension and quality based on size
	var maxDim int
	var quality int

	switch size {
	case "thumb":
		maxDim = maxSizeThumb
		quality = qualityThumb
	case "medium":
		maxDim = maxSizeMedium
		quality = qualityMedium
	default:
		maxDim = maxSizeMedium
		quality = qualityMedium
		log.Printf("⚠️  Unknown size '%s', defaulting to medium", size)
	}

	// Resize image if needed
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	var resizedImg image.Image = img
	if width > maxDim || height > maxDim {
		// Calculate new dimensions maintaining aspect ratio
		var newWidth, newHeight int
		if width > height {
			newWidth = maxDim
			newHeight = int(float64(height) * float64(maxDim) / float64(width))
		} else {
			newHeight = maxDim
			newWidth = int(float64(width) * float64(maxDim) / float64(height))
		}

		log.Printf("🔄 Resizing image: %dx%d -> %dx%d", width, height, newWidth, newHeight)
		resizedImg = imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)
	}

	// Encode to JPEG
	var buf bytes.Buffer
	opts := &jpeg.Options{
		Quality: quality,
	}
	if err := jpeg.Encode(&buf, resizedImg, opts); err != nil {
		return nil, fmt.Errorf("failed to encode to JPEG: %w", err)
	}
	optimizedData := buf.Bytes()

	log.Printf("✓ Image optimized: size=%s, quality=%d, output_size=%d bytes", size, quality, len(optimizedData))
	return optimizedData, nil
}

