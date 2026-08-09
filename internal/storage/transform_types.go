package storage

import (
	"errors"
	"io"
	"strings"
)

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

// TransformerOptions configures the image transformer
type TransformerOptions struct {
	MaxWidth       int
	MaxHeight      int
	MaxTotalPixels int
	BucketSize     int
}

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

// ImageTransformer handles image transformations
// Fields are defined in platform-specific files (transform.go or transform_novips.go)
type ImageTransformer struct {
	initialized    bool
	maxWidth       int
	maxHeight      int
	maxTotalPixels int
	bucketSize     int
}

// TransformInterface defines the interface for image transformation operations
// This is used by the cache and other components
type TransformInterface interface {
	Transform(data io.Reader, contentType string, opts *TransformOptions) (*TransformResult, error)
	TransformReader(data io.Reader, contentType string, opts *TransformOptions) (io.ReadCloser, string, int64, error)
}

// Ensure ImageTransformer implements the interface
var _ TransformInterface = (*ImageTransformer)(nil)

// CanTransform checks if the given content type can be transformed
func CanTransform(contentType string) bool {
	// Normalize content type (remove charset, etc.)
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = strings.TrimSpace(contentType[:idx])
	}
	return SupportedInputMimeTypes[strings.ToLower(contentType)]
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

// NOTE: NewImageTransformerWithOptions and ValidateOptions are defined in the
// build-tagged files (transform.go for the vips build, transform_novips.go for
// the default/novips build). They must NOT live here — transform_types.go has
// no build constraint, so defining them here duplicates the symbols in the
// vips build and breaks `go build -tags vips`.
