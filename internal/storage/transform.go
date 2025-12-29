package storage

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/davidbyttow/govips/v2/vips"
)

// FitMode defines how the image should be fit within the target dimensions
type FitMode string

const (
	FitCover   FitMode = "cover"   // Resize to cover target dimensions, cropping if needed
	FitContain FitMode = "contain" // Resize to fit within target dimensions, letterboxing if needed
	FitFill    FitMode = "fill"    // Stretch to exactly fill target dimensions
	FitInside  FitMode = "inside"  // Resize to fit within target, only scale down
	FitOutside FitMode = "outside" // Resize to be at least as large as target
)

// TransformOptions contains parameters for image transformation
type TransformOptions struct {
	Width   int     // Target width in pixels (0 = auto based on height)
	Height  int     // Target height in pixels (0 = auto based on width)
	Format  string  // Output format: webp, jpg, jpeg, png, avif (empty = same as input)
	Quality int     // Output quality 1-100 (default 80)
	Fit     FitMode // How to fit the image (default cover)
}

// TransformResult contains the result of an image transformation
type TransformResult struct {
	Data        []byte
	ContentType string
	Width       int
	Height      int
}

var (
	ErrInvalidDimensions  = errors.New("invalid image dimensions")
	ErrUnsupportedFormat  = errors.New("unsupported output format")
	ErrNotAnImage         = errors.New("file is not an image")
	ErrTransformFailed    = errors.New("image transformation failed")
	ErrImageTooLarge      = errors.New("image exceeds maximum allowed dimensions")
	ErrVipsNotInitialized = errors.New("vips library not initialized")
	ErrTooManyPixels      = errors.New("total pixel count exceeds maximum")
)

// MaxTransformDimension is the maximum allowed dimension for transformed images
const MaxTransformDimension = 8192

// DefaultMaxTotalPixels is the default maximum total pixel count (16 megapixels)
const DefaultMaxTotalPixels = 16_000_000

// DefaultBucketSize is the default dimension bucketing size (50px)
const DefaultBucketSize = 50

// BucketDimension rounds a dimension to the nearest bucket size
// This reduces cache key variations and provides DoS protection
func BucketDimension(dim int, bucketSize int) int {
	if dim <= 0 || bucketSize <= 0 {
		return dim
	}
	return ((dim + bucketSize/2) / bucketSize) * bucketSize
}

// SupportedOutputFormats lists the supported output formats
var SupportedOutputFormats = map[string]bool{
	"webp": true,
	"jpg":  true,
	"jpeg": true,
	"png":  true,
	"avif": true,
}

// SupportedInputMimeTypes lists MIME types that can be transformed
var SupportedInputMimeTypes = map[string]bool{
	"image/jpeg":    true,
	"image/png":     true,
	"image/webp":    true,
	"image/gif":     true,
	"image/tiff":    true,
	"image/bmp":     true,
	"image/svg+xml": true,
	"image/avif":    true,
}

// ImageTransformer handles image transformation operations
type ImageTransformer struct {
	initialized    bool
	maxWidth       int
	maxHeight      int
	maxTotalPixels int
	bucketSize     int
}

// vipsStartupLock is used to ensure vips is only started once
var vipsInstance *ImageTransformer

// TransformerOptions configures the image transformer
type TransformerOptions struct {
	MaxWidth       int
	MaxHeight      int
	MaxTotalPixels int
	BucketSize     int
}

// NewImageTransformer creates a new image transformer
// Note: vips.Startup() should be called once at application startup
func NewImageTransformer(maxWidth, maxHeight int) *ImageTransformer {
	return NewImageTransformerWithOptions(TransformerOptions{
		MaxWidth:  maxWidth,
		MaxHeight: maxHeight,
	})
}

// NewImageTransformerWithOptions creates a new image transformer with full options
func NewImageTransformerWithOptions(opts TransformerOptions) *ImageTransformer {
	if opts.MaxWidth <= 0 {
		opts.MaxWidth = MaxTransformDimension
	}
	if opts.MaxHeight <= 0 {
		opts.MaxHeight = MaxTransformDimension
	}
	if opts.MaxTotalPixels <= 0 {
		opts.MaxTotalPixels = DefaultMaxTotalPixels
	}
	if opts.BucketSize <= 0 {
		opts.BucketSize = DefaultBucketSize
	}

	return &ImageTransformer{
		initialized:    true,
		maxWidth:       opts.MaxWidth,
		maxHeight:      opts.MaxHeight,
		maxTotalPixels: opts.MaxTotalPixels,
		bucketSize:     opts.BucketSize,
	}
}

// InitVips initializes the vips library. Call this once at application startup.
func InitVips() {
	vips.Startup(nil)
}

// ShutdownVips shuts down the vips library. Call this at application shutdown.
func ShutdownVips() {
	vips.Shutdown()
}

// CanTransform checks if the given content type can be transformed
func CanTransform(contentType string) bool {
	// Normalize content type (remove charset, etc.)
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = strings.TrimSpace(contentType[:idx])
	}
	return SupportedInputMimeTypes[strings.ToLower(contentType)]
}

// ValidateOptions validates and normalizes transform options
func (t *ImageTransformer) ValidateOptions(opts *TransformOptions) error {
	if opts == nil {
		return nil
	}

	// Validate dimensions
	if opts.Width < 0 || opts.Height < 0 {
		return fmt.Errorf("%w: dimensions cannot be negative", ErrInvalidDimensions)
	}

	// Apply dimension bucketing to reduce cache key variations
	if opts.Width > 0 {
		opts.Width = BucketDimension(opts.Width, t.bucketSize)
	}
	if opts.Height > 0 {
		opts.Height = BucketDimension(opts.Height, t.bucketSize)
	}

	if opts.Width > t.maxWidth {
		return fmt.Errorf("%w: width exceeds maximum of %d", ErrImageTooLarge, t.maxWidth)
	}

	if opts.Height > t.maxHeight {
		return fmt.Errorf("%w: height exceeds maximum of %d", ErrImageTooLarge, t.maxHeight)
	}

	// Check total pixel count to prevent memory exhaustion
	if opts.Width > 0 && opts.Height > 0 {
		totalPixels := opts.Width * opts.Height
		if totalPixels > t.maxTotalPixels {
			return fmt.Errorf("%w: %d pixels exceeds maximum of %d", ErrTooManyPixels, totalPixels, t.maxTotalPixels)
		}
	}

	// Validate format
	if opts.Format != "" {
		format := strings.ToLower(opts.Format)
		if !SupportedOutputFormats[format] {
			return fmt.Errorf("%w: %s", ErrUnsupportedFormat, opts.Format)
		}
		opts.Format = format
	}

	// Normalize quality
	if opts.Quality <= 0 {
		opts.Quality = 80
	} else if opts.Quality > 100 {
		opts.Quality = 100
	}

	// Normalize fit mode
	if opts.Fit == "" {
		opts.Fit = FitCover
	}

	return nil
}

// Transform transforms an image according to the provided options
func (t *ImageTransformer) Transform(data io.Reader, contentType string, opts *TransformOptions) (*TransformResult, error) {
	if !t.initialized {
		return nil, ErrVipsNotInitialized
	}

	if opts == nil {
		return nil, nil
	}

	// Validate options
	if err := t.ValidateOptions(opts); err != nil {
		return nil, err
	}

	// Check if transformation is needed
	if opts.Width == 0 && opts.Height == 0 && opts.Format == "" {
		return nil, nil // No transformation needed
	}

	// Check if content type is supported
	if !CanTransform(contentType) {
		return nil, fmt.Errorf("%w: %s", ErrNotAnImage, contentType)
	}

	// Read all data into memory for vips processing
	imageData, err := io.ReadAll(data)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	// Load the image
	image, err := vips.NewImageFromBuffer(imageData)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTransformFailed, err)
	}
	defer image.Close()

	// Get original dimensions
	origWidth := image.Width()
	origHeight := image.Height()

	// Calculate target dimensions
	targetWidth, targetHeight := t.calculateDimensions(origWidth, origHeight, opts.Width, opts.Height, opts.Fit)

	// Resize if needed
	if targetWidth != origWidth || targetHeight != origHeight {
		// Calculate scale factors
		hScale := float64(targetWidth) / float64(origWidth)
		vScale := float64(targetHeight) / float64(origHeight)

		// For cover mode, we may need to crop after resize
		if opts.Fit == FitCover && (opts.Width > 0 && opts.Height > 0) {
			// Scale to cover the target dimensions
			scale := max(hScale, vScale)
			if err := image.Resize(scale, vips.KernelLanczos3); err != nil {
				return nil, fmt.Errorf("%w: resize failed: %v", ErrTransformFailed, err)
			}

			// Crop to exact dimensions if needed
			if image.Width() > targetWidth || image.Height() > targetHeight {
				left := (image.Width() - targetWidth) / 2
				top := (image.Height() - targetHeight) / 2
				if err := image.ExtractArea(left, top, targetWidth, targetHeight); err != nil {
					return nil, fmt.Errorf("%w: crop failed: %v", ErrTransformFailed, err)
				}
			}
		} else if opts.Fit == FitFill {
			// Stretch to exactly fit - resize width and height separately
			if err := image.Resize(hScale, vips.KernelLanczos3); err != nil {
				return nil, fmt.Errorf("%w: resize width failed: %v", ErrTransformFailed, err)
			}
			// Note: vips Resize only does uniform scaling, for true stretch we'd need more complex handling
		} else {
			// For contain, inside, outside - uniform scaling
			scale := min(hScale, vScale)
			if opts.Fit == FitOutside {
				scale = max(hScale, vScale)
			}
			if opts.Fit == FitInside && scale > 1 {
				scale = 1 // Only scale down
			}
			if err := image.Resize(scale, vips.KernelLanczos3); err != nil {
				return nil, fmt.Errorf("%w: resize failed: %v", ErrTransformFailed, err)
			}
		}
	}

	// Determine output format
	outputFormat := t.determineOutputFormat(contentType, opts.Format)

	// Export to the target format
	var outputData []byte
	var outputContentType string

	exportParams := t.getExportParams(outputFormat, opts.Quality)

	switch outputFormat {
	case "webp":
		outputData, _, err = image.ExportWebp(exportParams.(*vips.WebpExportParams))
		outputContentType = "image/webp"
	case "jpg", "jpeg":
		outputData, _, err = image.ExportJpeg(exportParams.(*vips.JpegExportParams))
		outputContentType = "image/jpeg"
	case "png":
		outputData, _, err = image.ExportPng(exportParams.(*vips.PngExportParams))
		outputContentType = "image/png"
	case "avif":
		outputData, _, err = image.ExportAvif(exportParams.(*vips.AvifExportParams))
		outputContentType = "image/avif"
	default:
		// Fall back to original format or JPEG
		outputData, _, err = image.ExportJpeg(&vips.JpegExportParams{Quality: opts.Quality})
		outputContentType = "image/jpeg"
	}

	if err != nil {
		return nil, fmt.Errorf("%w: export failed: %v", ErrTransformFailed, err)
	}

	return &TransformResult{
		Data:        outputData,
		ContentType: outputContentType,
		Width:       image.Width(),
		Height:      image.Height(),
	}, nil
}

// TransformReader transforms an image and returns a reader for the result
func (t *ImageTransformer) TransformReader(data io.Reader, contentType string, opts *TransformOptions) (io.ReadCloser, string, int64, error) {
	result, err := t.Transform(data, contentType, opts)
	if err != nil {
		return nil, "", 0, err
	}

	if result == nil {
		return nil, "", 0, nil
	}

	return io.NopCloser(bytes.NewReader(result.Data)), result.ContentType, int64(len(result.Data)), nil
}

// calculateDimensions calculates the target dimensions based on fit mode
func (t *ImageTransformer) calculateDimensions(origWidth, origHeight, targetWidth, targetHeight int, fit FitMode) (int, int) {
	var calcWidth, calcHeight int

	// If both dimensions are specified, use them
	if targetWidth > 0 && targetHeight > 0 {
		calcWidth, calcHeight = targetWidth, targetHeight
	} else if targetWidth > 0 && targetHeight == 0 {
		// If only width is specified, calculate height to maintain aspect ratio
		ratio := float64(origHeight) / float64(origWidth)
		calcWidth = targetWidth
		calcHeight = int(float64(targetWidth) * ratio)
	} else if targetHeight > 0 && targetWidth == 0 {
		// If only height is specified, calculate width to maintain aspect ratio
		ratio := float64(origWidth) / float64(origHeight)
		calcWidth = int(float64(targetHeight) * ratio)
		calcHeight = targetHeight
	} else {
		// No resize needed
		return origWidth, origHeight
	}

	// Clamp calculated dimensions to max limits to prevent overflow
	if calcWidth > t.maxWidth {
		calcWidth = t.maxWidth
	}
	if calcHeight > t.maxHeight {
		calcHeight = t.maxHeight
	}

	return calcWidth, calcHeight
}

// determineOutputFormat determines the output format based on input and options
func (t *ImageTransformer) determineOutputFormat(inputContentType, requestedFormat string) string {
	if requestedFormat != "" {
		return requestedFormat
	}

	// Default to input format if supported as output
	switch inputContentType {
	case "image/jpeg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/webp":
		return "webp"
	case "image/avif":
		return "avif"
	default:
		// Default to JPEG for unsupported formats (like GIF, BMP, TIFF)
		return "jpg"
	}
}

// getExportParams returns the appropriate export parameters for the format
func (t *ImageTransformer) getExportParams(format string, quality int) interface{} {
	switch format {
	case "webp":
		return &vips.WebpExportParams{
			Quality:  quality,
			Lossless: false,
		}
	case "jpg", "jpeg":
		return &vips.JpegExportParams{
			Quality:        quality,
			OptimizeCoding: true,
		}
	case "png":
		// PNG doesn't have quality per se, but we can control compression
		compression := 6 // Default compression level
		if quality > 80 {
			compression = 9 // Maximum compression for high quality setting
		} else if quality < 50 {
			compression = 3 // Less compression for speed
		}
		return &vips.PngExportParams{
			Compression: compression,
		}
	case "avif":
		return &vips.AvifExportParams{
			Quality:  quality,
			Lossless: false,
			Speed:    5, // Balance between speed and quality
		}
	default:
		return &vips.JpegExportParams{Quality: quality}
	}
}

// ParseTransformOptions parses query parameters into TransformOptions
func ParseTransformOptions(width, height int, format string, quality int, fit string) *TransformOptions {
	// Return nil if no transform options specified
	if width == 0 && height == 0 && format == "" && quality == 0 && fit == "" {
		return nil
	}

	opts := &TransformOptions{
		Width:   width,
		Height:  height,
		Format:  format,
		Quality: quality,
	}

	// Parse fit mode
	switch strings.ToLower(fit) {
	case "cover":
		opts.Fit = FitCover
	case "contain":
		opts.Fit = FitContain
	case "fill":
		opts.Fit = FitFill
	case "inside":
		opts.Fit = FitInside
	case "outside":
		opts.Fit = FitOutside
	default:
		opts.Fit = FitCover
	}

	return opts
}
