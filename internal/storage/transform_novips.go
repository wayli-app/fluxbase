//go:build !vips
// +build !vips

package storage

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

// NewImageTransformerWithOptions creates a new image transformer with full
// options. Without the vips build tag the transformer is returned
// uninitialized; any actual transform attempt fails (see Transform below).
// This mirrors the validation logic used by the vips build so that option
// validation behaves identically regardless of build flavor.
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
		initialized:    false, // Not initialized without vips
		maxWidth:       opts.MaxWidth,
		maxHeight:      opts.MaxHeight,
		maxTotalPixels: opts.MaxTotalPixels,
		bucketSize:     opts.BucketSize,
	}
}

// ValidateOptions validates and normalizes transform options. The validation
// rules are independent of vips, so the novips build applies the same checks.
func (t *ImageTransformer) ValidateOptions(opts *TransformOptions) error {
	if opts == nil {
		return nil // No transformation requested
	}

	// Validate dimensions
	if opts.Width < 0 || opts.Height < 0 {
		return ErrInvalidDimensions
	}

	if opts.Width > 0 && opts.Width > t.maxWidth {
		return fmt.Errorf("%w: width %d exceeds maximum %d", ErrImageTooLarge, opts.Width, t.maxWidth)
	}

	if opts.Height > 0 && opts.Height > t.maxHeight {
		return fmt.Errorf("%w: height %d exceeds maximum %d", ErrImageTooLarge, opts.Height, t.maxHeight)
	}

	// Calculate total pixels
	totalPixels := opts.Width * opts.Height
	if totalPixels > 0 && totalPixels > t.maxTotalPixels {
		return fmt.Errorf("%w: %dx%d = %d pixels exceeds maximum %d",
			ErrTooManyPixels, opts.Width, opts.Height, totalPixels, t.maxTotalPixels)
	}

	// Validate format
	if opts.Format != "" {
		opts.Format = strings.ToLower(strings.TrimSpace(opts.Format))
		if !SupportedOutputFormats[opts.Format] {
			return ErrUnsupportedFormat
		}
	}

	// Validate quality
	if opts.Quality < 0 || opts.Quality > 100 {
		opts.Quality = 80
	}

	// Normalize fit mode
	if opts.Fit == "" {
		opts.Fit = FitCover
	}

	// Bucket dimensions for caching and DoS protection
	if t.bucketSize > 0 {
		if opts.Width > 0 {
			opts.Width = BucketDimension(opts.Width, t.bucketSize)
		}
		if opts.Height > 0 {
			opts.Height = BucketDimension(opts.Height, t.bucketSize)
		}
	}

	return nil
}

// InitVips is a no-op when vips is not available
func InitVips() {
	// No-op when vips is not compiled in
}

// ShutdownVips is a no-op when vips is not available
func ShutdownVips() {
	// No-op when vips is not compiled in
}

// Transform returns an error when vips is not available
func (t *ImageTransformer) Transform(data io.Reader, contentType string, opts *TransformOptions) (*TransformResult, error) {
	if opts == nil || (opts.Width == 0 && opts.Height == 0 && opts.Format == "") {
		// No transformation requested, return as-is
		return nil, nil
	}

	// Image transformation requested but vips is not available
	return nil, errors.New("image transformation requires vips build tag")
}

// TransformReader returns an error when vips is not available
func (t *ImageTransformer) TransformReader(data io.Reader, contentType string, opts *TransformOptions) (io.ReadCloser, string, int64, error) {
	if opts == nil || (opts.Width == 0 && opts.Height == 0 && opts.Format == "") {
		// No transformation requested, return as-is
		return nil, "", 0, nil
	}

	// Image transformation requested but vips is not available
	return nil, "", 0, errors.New("image transformation requires vips build tag")
}
