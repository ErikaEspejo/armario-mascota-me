package service

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
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
	// Background color for PNG transparency flattening
	// Using white (#FFFFFF) as default
	backgroundColor = "#FFFFFF"
)

// getBackgroundColor returns the background color for flattening transparent images
func getBackgroundColor() color.Color {
	// Parse hex color #FFFFFF (white)
	// R: 255, G: 255, B: 255, A: 255
	return color.RGBA{R: 255, G: 255, B: 255, A: 255}
}

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

	// Flatten transparent images onto a solid background
	// JPEG doesn't support transparency, so we need to flatten PNG images with alpha channel
	var processedImg image.Image = img
	
	// Check if image might have transparency (PNG format or NRGBA type)
	needsFlattening := false
	if format == "png" {
		needsFlattening = true
	} else if _, ok := img.(*image.NRGBA); ok {
		needsFlattening = true
	} else if rgba, ok := img.(*image.RGBA); ok {
		// Quick check: if it's RGBA, check if it might have transparency
		// We'll flatten it anyway to be safe, as it's a PNG-like format
		needsFlattening = true
		_ = rgba // avoid unused variable warning
	}

	if needsFlattening {
		log.Printf("🖼️  Image may have transparency, flattening onto background color %s", backgroundColor)
		bounds := img.Bounds()
		// Create a new image with solid background color
		bgImg := imaging.New(bounds.Dx(), bounds.Dy(), getBackgroundColor())
		// Overlay the original image on top of the background
		processedImg = imaging.Overlay(bgImg, img, image.Pt(0, 0), 1.0)
	}

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
	bounds := processedImg.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	var resizedImg image.Image = processedImg
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
		resizedImg = imaging.Resize(processedImg, newWidth, newHeight, imaging.Lanczos)
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

