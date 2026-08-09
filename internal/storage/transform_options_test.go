package storage

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseTransformOptions tests parsing of transform options from query parameters
func TestParseTransformOptions(t *testing.T) {
	tests := []struct {
		name     string
		width    int
		height   int
		format   string
		quality  int
		fit      string
		wantNil  bool
		wantOpts *TransformOptions
	}{
		{
			name:    "no options specified",
			width:   0,
			height:  0,
			format:  "",
			quality: 0,
			fit:     "",
			wantNil: true,
		},
		{
			name:  "width only",
			width: 800,
			wantOpts: &TransformOptions{
				Width: 800,
				Fit:   FitCover,
			},
		},
		{
			name:   "height only",
			height: 600,
			wantOpts: &TransformOptions{
				Height: 600,
				Fit:    FitCover,
			},
		},
		{
			name:   "width and height",
			width:  1920,
			height: 1080,
			wantOpts: &TransformOptions{
				Width:  1920,
				Height: 1080,
				Fit:    FitCover,
			},
		},
		{
			name:    "all options specified",
			width:   1280,
			height:  720,
			format:  "webp",
			quality: 90,
			fit:     "contain",
			wantOpts: &TransformOptions{
				Width:   1280,
				Height:  720,
				Format:  "webp",
				Quality: 90,
				Fit:     FitContain,
			},
		},
		{
			name:   "format conversion to jpg",
			width:  800,
			height: 600,
			format: "jpg",
			wantOpts: &TransformOptions{
				Width:  800,
				Height: 600,
				Format: "jpg",
				Fit:    FitCover,
			},
		},
		{
			name:   "format conversion to png",
			width:  800,
			height: 600,
			format: "png",
			wantOpts: &TransformOptions{
				Width:  800,
				Height: 600,
				Format: "png",
				Fit:    FitCover,
			},
		},
		{
			name:    "quality only",
			quality: 95,
			wantOpts: &TransformOptions{
				Quality: 95,
				Fit:     FitCover,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ParseTransformOptions(tt.width, tt.height, tt.format, tt.quality, tt.fit)

			if tt.wantNil {
				assert.Nil(t, opts)
			} else {
				assert.NotNil(t, opts)
				assert.Equal(t, tt.wantOpts.Width, opts.Width)
				assert.Equal(t, tt.wantOpts.Height, opts.Height)
				assert.Equal(t, tt.wantOpts.Format, opts.Format)
				assert.Equal(t, tt.wantOpts.Quality, opts.Quality)
				assert.Equal(t, tt.wantOpts.Fit, opts.Fit)
			}
		})
	}
}

// TestParseTransformOptions_FitModeNormalization tests fit mode string normalization
func TestParseTransformOptions_FitModeNormalization(t *testing.T) {
	tests := []struct {
		name    string
		fit     string
		wantFit FitMode
	}{
		{
			name:    "cover",
			fit:     "cover",
			wantFit: FitCover,
		},
		{
			name:    "contain",
			fit:     "contain",
			wantFit: FitContain,
		},
		{
			name:    "fill",
			fit:     "fill",
			wantFit: FitFill,
		},
		{
			name:    "inside",
			fit:     "inside",
			wantFit: FitInside,
		},
		{
			name:    "outside",
			fit:     "outside",
			wantFit: FitOutside,
		},
		{
			name:    "uppercase COVER",
			fit:     "COVER",
			wantFit: FitCover,
		},
		{
			name:    "mixed case CoNtAiN",
			fit:     "CoNtAiN",
			wantFit: FitContain,
		},
		{
			name:    "invalid fit mode defaults to cover",
			fit:     "invalid",
			wantFit: FitCover,
		},
		{
			name:    "empty fit mode defaults to cover",
			fit:     "",
			wantFit: FitCover,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ParseTransformOptions(800, 600, "jpg", 80, tt.fit)
			require.NotNil(t, opts)
			assert.Equal(t, tt.wantFit, opts.Fit)
		})
	}
}

// TestValidateOptions tests validation of transform options
func TestValidateOptions(t *testing.T) {
	tests := []struct {
		name      string
		maxWidth  int
		maxHeight int
		maxPixels int
		opts      *TransformOptions
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "nil options",
			maxWidth:  1920,
			maxHeight: 1080,
			maxPixels: 16000000,
			opts:      nil,
			wantErr:   false,
		},
		{
			name:      "valid options within limits",
			maxWidth:  1920,
			maxHeight: 1080,
			maxPixels: 16000000,
			opts: &TransformOptions{
				Width:   800,
				Height:  600,
				Format:  "webp",
				Quality: 80,
			},
			wantErr: false,
		},
		{
			name:      "negative dimensions",
			maxWidth:  1920,
			maxHeight: 1080,
			maxPixels: 16000000,
			opts: &TransformOptions{
				Width:  -100,
				Height: -200,
			},
			wantErr: true,
			errMsg:  "invalid image dimensions",
		},
		{
			name:      "width exceeds maximum",
			maxWidth:  1920,
			maxHeight: 1080,
			maxPixels: 16000000,
			opts: &TransformOptions{
				Width: 4000,
			},
			wantErr: true,
			errMsg:  "exceeds maximum",
		},
		{
			name:      "height exceeds maximum",
			maxWidth:  1920,
			maxHeight: 1080,
			maxPixels: 16000000,
			opts: &TransformOptions{
				Height: 3000,
			},
			wantErr: true,
			errMsg:  "exceeds maximum",
		},
		{
			name:      "total pixels exceed maximum",
			maxWidth:  1920,
			maxHeight: 1080,
			maxPixels: 1000000, // 1 megapixel
			opts: &TransformOptions{
				Width:  1200,
				Height: 900, // 1.08 megapixels
			},
			wantErr: true,
			errMsg:  "pixel count exceeds",
		},
		{
			name:      "unsupported format",
			maxWidth:  1920,
			maxHeight: 1080,
			maxPixels: 16000000,
			opts: &TransformOptions{
				Format: "bmp",
			},
			wantErr: true,
			errMsg:  "unsupported output format",
		},
		{
			name:      "quality below range",
			maxWidth:  1920,
			maxHeight: 1080,
			maxPixels: 16000000,
			opts: &TransformOptions{
				Quality: -10,
			},
			wantErr: false, // Quality is normalized to 80
		},
		{
			name:      "quality above range",
			maxWidth:  1920,
			maxHeight: 1080,
			maxPixels: 16000000,
			opts: &TransformOptions{
				Quality: 150,
			},
			wantErr: false, // Quality is normalized to 80
		},
		{
			name:      "valid all supported formats",
			maxWidth:  1920,
			maxHeight: 1080,
			maxPixels: 16000000,
			opts: &TransformOptions{
				Width:  800,
				Height: 600,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transformer := NewImageTransformerWithOptions(TransformerOptions{
				MaxWidth:       tt.maxWidth,
				MaxHeight:      tt.maxHeight,
				MaxTotalPixels: tt.maxPixels,
			})

			err := transformer.ValidateOptions(tt.opts)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateOptions_FormatNormalization tests format string normalization
func TestValidateOptions_FormatNormalization(t *testing.T) {
	transformer := NewImageTransformerWithOptions(TransformerOptions{})

	tests := []struct {
		name     string
		format   string
		expected string
	}{
		{
			name:     "lowercase webp",
			format:   "webp",
			expected: "webp",
		},
		{
			name:     "uppercase WEBP",
			format:   "WEBP",
			expected: "webp",
		},
		{
			name:     "mixed case WeBp",
			format:   "WeBp",
			expected: "webp",
		},
		{
			name:     "jpeg normalized to jpg",
			format:   "jpeg",
			expected: "jpeg",
		},
		{
			name:     "jpg",
			format:   "jpg",
			expected: "jpg",
		},
		{
			name:     "png",
			format:   "png",
			expected: "png",
		},
		{
			name:     "avif",
			format:   "avif",
			expected: "avif",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &TransformOptions{
				Width:  800,
				Height: 600,
				Format: tt.format,
			}

			err := transformer.ValidateOptions(opts)

			// Only test supported formats
			if tt.expected != "" {
				if strings.EqualFold(tt.format, "bmp") {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tt.expected, opts.Format)
				}
			}
		})
	}
}

// NOTE: TestValidateOptions_QualityNormalization is build-specific — the vips
// and novips ValidateOptions implementations normalize out-of-range quality
// values differently (vips: <=0 -> 80, >100 -> 100 clamp; novips: <0 or >100
// -> 80, leaving 0 untouched). It lives in transform_test.go (vips) and
// transform_novips_test.go (!vips), not here, so this unconstrained file does
// not assert a single contract that one build would violate.

// TestBucketDimension tests dimension bucketing for DoS protection
func TestBucketDimension(t *testing.T) {
	tests := []struct {
		name       string
		dim        int
		bucketSize int
		expected   int
	}{
		{
			name:       "exactly on bucket boundary",
			dim:        100,
			bucketSize: 50,
			expected:   100,
		},
		{
			name:       "rounds up to next bucket",
			dim:        125,
			bucketSize: 50,
			expected:   150, // 125 + 25 = 150
		},
		{
			name:       "rounds down to next bucket",
			dim:        124,
			bucketSize: 50,
			expected:   100, // 124 - 24 = 100
		},
		{
			name:       "small dimension",
			dim:        25,
			bucketSize: 50,
			expected:   50,
		},
		{
			name:       "zero dimension",
			dim:        0,
			bucketSize: 50,
			expected:   0,
		},
		{
			name:       "negative dimension",
			dim:        -100,
			bucketSize: 50,
			expected:   -100,
		},
		{
			name:       "zero bucket size",
			dim:        800,
			bucketSize: 0,
			expected:   800,
		},
		{
			name:       "negative bucket size",
			dim:        800,
			bucketSize: -50,
			expected:   800,
		},
		{
			name:       "large dimension",
			dim:        1950,
			bucketSize: 50,
			expected:   1950, // Rounds to 1950 (1950 + 25 = 1975, but 1975 > 1950)
		},
		{
			name:       "odd dimension",
			dim:        801,
			bucketSize: 50,
			expected:   800,
		},
		{
			name:       "custom bucket size 100",
			dim:        1234,
			bucketSize: 100,
			expected:   1200,
		},
		{
			name:       "custom bucket size 25",
			dim:        38,
			bucketSize: 25,
			expected:   50,
		},
		{
			name:       "bucket size 1 (no bucketing)",
			dim:        123,
			bucketSize: 1,
			expected:   123,
		},
		{
			name:       "large dimension with bucketing",
			dim:        8100,
			bucketSize: 50,
			expected:   8100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BucketDimension(tt.dim, tt.bucketSize)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestValidateOptions_DimensionBucketing tests that validation applies dimension bucketing
func TestValidateOptions_DimensionBucketing(t *testing.T) {
	transformer := NewImageTransformerWithOptions(TransformerOptions{
		BucketSize: 100,
	})

	opts := &TransformOptions{
		Width:  1234,
		Height: 5678,
	}

	err := transformer.ValidateOptions(opts)
	assert.NoError(t, err)

	// Dimensions should be bucketed
	assert.Equal(t, 1200, opts.Width)
	assert.Equal(t, 5700, opts.Height)
}

// TestCanTransform tests content type transformation support detection
func TestCanTransform(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		wantCan     bool
	}{
		{
			name:        "jpeg lowercase",
			contentType: "image/jpeg",
			wantCan:     true,
		},
		{
			name:        "jpeg uppercase",
			contentType: "IMAGE/JPEG",
			wantCan:     true,
		},
		{
			name:        "png",
			contentType: "image/png",
			wantCan:     true,
		},
		{
			name:        "webp",
			contentType: "image/webp",
			wantCan:     true,
		},
		{
			name:        "gif",
			contentType: "image/gif",
			wantCan:     true,
		},
		{
			name:        "svg",
			contentType: "image/svg+xml",
			wantCan:     true,
		},
		{
			name:        "avif",
			contentType: "image/avif",
			wantCan:     true,
		},
		{
			name:        "tiff",
			contentType: "image/tiff",
			wantCan:     true,
		},
		{
			name:        "bmp",
			contentType: "image/bmp",
			wantCan:     true,
		},
		{
			name:        "with charset parameter",
			contentType: "image/jpeg; charset=utf-8",
			wantCan:     true,
		},
		{
			name:        "with boundary parameter",
			contentType: "image/png; boundary=something",
			wantCan:     true,
		},
		{
			name:        "with spaces - needs trim",
			contentType: " image/jpeg ",
			wantCan:     false, // CanTransform doesn't trim spaces
		},
		{
			name:        "mixed case Image/Png",
			contentType: "Image/Png",
			wantCan:     true,
		},
		{
			name:        "pdf not supported",
			contentType: "application/pdf",
			wantCan:     false,
		},
		{
			name:        "text not supported",
			contentType: "text/plain",
			wantCan:     false,
		},
		{
			name:        "video not supported",
			contentType: "video/mp4",
			wantCan:     false,
		},
		{
			name:        "heic not supported",
			contentType: "image/heic",
			wantCan:     false,
		},
		{
			name:        "empty string",
			contentType: "",
			wantCan:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CanTransform(tt.contentType)
			assert.Equal(t, tt.wantCan, result)
		})
	}
}

// TestNewImageTransformerWithOptions_DefaultValues tests default value handling
func TestNewImageTransformerWithOptions_DefaultValues(t *testing.T) {
	tests := []struct {
		name           string
		maxWidth       int
		maxHeight      int
		maxTotalPixels int
		bucketSize     int
		expectedWidth  int
		expectedHeight int
		expectedPixels int
		expectedBucket int
	}{
		{
			name:           "all zero values use defaults",
			maxWidth:       0,
			maxHeight:      0,
			maxTotalPixels: 0,
			bucketSize:     0,
			expectedWidth:  MaxTransformDimension,
			expectedHeight: MaxTransformDimension,
			expectedPixels: DefaultMaxTotalPixels,
			expectedBucket: DefaultBucketSize,
		},
		{
			name:           "custom values are respected",
			maxWidth:       4096,
			maxHeight:      2160,
			maxTotalPixels: 8000000,
			bucketSize:     100,
			expectedWidth:  4096,
			expectedHeight: 2160,
			expectedPixels: 8000000,
			expectedBucket: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transformer := NewImageTransformerWithOptions(TransformerOptions{
				MaxWidth:       tt.maxWidth,
				MaxHeight:      tt.maxHeight,
				MaxTotalPixels: tt.maxTotalPixels,
				BucketSize:     tt.bucketSize,
			})

			assert.Equal(t, tt.expectedWidth, transformer.maxWidth)
			assert.Equal(t, tt.expectedHeight, transformer.maxHeight)
			assert.Equal(t, tt.expectedPixels, transformer.maxTotalPixels)
			assert.Equal(t, tt.expectedBucket, transformer.bucketSize)
		})
	}
}

// TestSupportedFormatsAndTypes tests the supported format/type maps
func TestSupportedFormatsAndTypes(t *testing.T) {
	// Test supported output formats
	supportedFormats := []string{"webp", "jpg", "jpeg", "png", "avif"}
	for _, format := range supportedFormats {
		t.Run("format_supported_"+format, func(t *testing.T) {
			assert.True(t, SupportedOutputFormats[format], "Format %s should be supported", format)
		})
	}

	// Test unsupported formats
	unsupportedFormats := []string{"bmp", "tiff", "gif", "svg"}
	for _, format := range unsupportedFormats {
		t.Run("format_unsupported_"+format, func(t *testing.T) {
			assert.False(t, SupportedOutputFormats[format], "Format %s should not be supported", format)
		})
	}

	// Test supported input MIME types
	supportedTypes := []string{
		"image/jpeg", "image/png", "image/webp", "image/gif",
		"image/tiff", "image/bmp", "image/svg+xml", "image/avif",
	}
	for _, mimeType := range supportedTypes {
		t.Run("mime_supported_"+mimeType, func(t *testing.T) {
			assert.True(t, SupportedInputMimeTypes[mimeType], "MIME type %s should be supported", mimeType)
		})
	}
}

// TestTransformErrorValues tests that error variables are properly defined
func TestTransformErrorValues(t *testing.T) {
	errors := []struct {
		name string
		err  error
	}{
		{"ErrInvalidDimensions", ErrInvalidDimensions},
		{"ErrUnsupportedFormat", ErrUnsupportedFormat},
		{"ErrNotAnImage", ErrNotAnImage},
		{"ErrTransformFailed", ErrTransformFailed},
		{"ErrImageTooLarge", ErrImageTooLarge},
		{"ErrVipsNotInitialized", ErrVipsNotInitialized},
		{"ErrTooManyPixels", ErrTooManyPixels},
	}

	for _, tt := range errors {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.err)
			assert.NotEmpty(t, tt.err.Error())
		})
	}
}

// TestTransformOptions_FitModeConstants tests fit mode constants
func TestTransformOptions_FitModeConstants(t *testing.T) {
	modes := []FitMode{
		FitCover,
		FitContain,
		FitFill,
		FitInside,
		FitOutside,
	}

	expectedValues := []string{
		"cover",
		"contain",
		"fill",
		"inside",
		"outside",
	}

	for i, mode := range modes {
		modeStr := string(mode)
		t.Run("fit_mode_"+modeStr, func(t *testing.T) {
			assert.Equal(t, expectedValues[i], modeStr)
		})
	}
}

// TestValidateOptions_MissingFitModeDefaultsToCover tests empty fit mode
func TestValidateOptions_MissingFitModeDefaultsToCover(t *testing.T) {
	transformer := NewImageTransformerWithOptions(TransformerOptions{})

	opts := &TransformOptions{
		Width:  800,
		Height: 600,
		Fit:    "",
	}

	err := transformer.ValidateOptions(opts)
	assert.NoError(t, err)
	assert.Equal(t, FitCover, opts.Fit)
}

// TestTransformOptions_Constants tests constant values
func TestTransformOptions_Constants(t *testing.T) {
	assert.Equal(t, 8192, MaxTransformDimension, "MaxTransformDimension should be 8192")
	assert.Equal(t, 16_000_000, DefaultMaxTotalPixels, "DefaultMaxTotalPixels should be 16 million")
	assert.Equal(t, 50, DefaultBucketSize, "DefaultBucketSize should be 50")
}
