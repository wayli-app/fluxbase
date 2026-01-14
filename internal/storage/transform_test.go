package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// FitMode Constants Tests
// =============================================================================

func TestFitMode_Constants(t *testing.T) {
	tests := []struct {
		mode     FitMode
		expected string
	}{
		{FitCover, "cover"},
		{FitContain, "contain"},
		{FitFill, "fill"},
		{FitInside, "inside"},
		{FitOutside, "outside"},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.mode))
		})
	}
}

// =============================================================================
// Error Variables Tests
// =============================================================================

func TestTransformErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		contains string
	}{
		{"ErrInvalidDimensions", ErrInvalidDimensions, "invalid image dimensions"},
		{"ErrUnsupportedFormat", ErrUnsupportedFormat, "unsupported output format"},
		{"ErrNotAnImage", ErrNotAnImage, "not an image"},
		{"ErrTransformFailed", ErrTransformFailed, "transformation failed"},
		{"ErrImageTooLarge", ErrImageTooLarge, "exceeds maximum"},
		{"ErrVipsNotInitialized", ErrVipsNotInitialized, "not initialized"},
		{"ErrTooManyPixels", ErrTooManyPixels, "pixel count exceeds"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Error(t, tt.err)
			assert.Contains(t, tt.err.Error(), tt.contains)
		})
	}
}

// =============================================================================
// BucketDimension Tests
// =============================================================================

func TestBucketDimension(t *testing.T) {
	tests := []struct {
		name       string
		dim        int
		bucketSize int
		expected   int
	}{
		{"exact bucket", 100, 50, 100},
		{"round up", 126, 50, 150},
		{"round down", 124, 50, 100},
		{"midpoint rounds up", 125, 50, 150},
		{"small dimension", 30, 50, 50},
		{"zero dimension", 0, 50, 0},
		{"negative dimension", -10, 50, -10},
		{"zero bucket size", 100, 0, 100},
		{"negative bucket size", 100, -50, 100},
		{"large dimension", 8100, 50, 8100},
		{"bucket size 100", 250, 100, 300},
		{"bucket size 25", 38, 25, 50},
		{"bucket size 1 (no bucketing)", 123, 1, 123},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BucketDimension(tt.dim, tt.bucketSize)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// SupportedOutputFormats Tests
// =============================================================================

func TestSupportedOutputFormats(t *testing.T) {
	supported := []string{"webp", "jpg", "jpeg", "png", "avif"}
	unsupported := []string{"gif", "bmp", "tiff", "svg", "ico", "heic"}

	for _, format := range supported {
		t.Run("supported_"+format, func(t *testing.T) {
			assert.True(t, SupportedOutputFormats[format])
		})
	}

	for _, format := range unsupported {
		t.Run("unsupported_"+format, func(t *testing.T) {
			assert.False(t, SupportedOutputFormats[format])
		})
	}
}

// =============================================================================
// SupportedInputMimeTypes Tests
// =============================================================================

func TestSupportedInputMimeTypes(t *testing.T) {
	supported := []string{
		"image/jpeg",
		"image/png",
		"image/webp",
		"image/gif",
		"image/tiff",
		"image/bmp",
		"image/svg+xml",
		"image/avif",
	}
	unsupported := []string{
		"text/plain",
		"application/pdf",
		"video/mp4",
		"audio/mpeg",
		"image/heic",
	}

	for _, mime := range supported {
		t.Run("supported_"+mime, func(t *testing.T) {
			assert.True(t, SupportedInputMimeTypes[mime])
		})
	}

	for _, mime := range unsupported {
		t.Run("unsupported_"+mime, func(t *testing.T) {
			assert.False(t, SupportedInputMimeTypes[mime])
		})
	}
}

// =============================================================================
// CanTransform Tests
// =============================================================================

func TestCanTransform(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		expected    bool
	}{
		{"jpeg", "image/jpeg", true},
		{"png", "image/png", true},
		{"webp", "image/webp", true},
		{"gif", "image/gif", true},
		{"avif", "image/avif", true},
		{"svg", "image/svg+xml", true},
		{"tiff", "image/tiff", true},
		{"bmp", "image/bmp", true},
		{"text", "text/plain", false},
		{"pdf", "application/pdf", false},
		{"video", "video/mp4", false},
		{"jpeg with charset", "image/jpeg; charset=utf-8", true},
		{"png with boundary", "image/png; boundary=something", true},
		{"uppercase", "IMAGE/JPEG", true},
		{"mixed case", "Image/Png", true},
		{"empty", "", false},
		{"heic not supported", "image/heic", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CanTransform(tt.contentType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// NewImageTransformer Tests
// =============================================================================

func TestNewImageTransformer(t *testing.T) {
	t.Run("with valid dimensions", func(t *testing.T) {
		transformer := NewImageTransformer(1920, 1080)

		assert.NotNil(t, transformer)
		assert.True(t, transformer.initialized)
		assert.Equal(t, 1920, transformer.maxWidth)
		assert.Equal(t, 1080, transformer.maxHeight)
	})

	t.Run("with zero dimensions uses defaults", func(t *testing.T) {
		transformer := NewImageTransformer(0, 0)

		assert.Equal(t, MaxTransformDimension, transformer.maxWidth)
		assert.Equal(t, MaxTransformDimension, transformer.maxHeight)
	})

	t.Run("with negative dimensions uses defaults", func(t *testing.T) {
		transformer := NewImageTransformer(-100, -200)

		assert.Equal(t, MaxTransformDimension, transformer.maxWidth)
		assert.Equal(t, MaxTransformDimension, transformer.maxHeight)
	})
}

// =============================================================================
// NewImageTransformerWithOptions Tests
// =============================================================================

func TestNewImageTransformerWithOptions(t *testing.T) {
	t.Run("all options specified", func(t *testing.T) {
		opts := TransformerOptions{
			MaxWidth:       4096,
			MaxHeight:      2160,
			MaxTotalPixels: 8000000,
			BucketSize:     100,
		}

		transformer := NewImageTransformerWithOptions(opts)

		assert.True(t, transformer.initialized)
		assert.Equal(t, 4096, transformer.maxWidth)
		assert.Equal(t, 2160, transformer.maxHeight)
		assert.Equal(t, 8000000, transformer.maxTotalPixels)
		assert.Equal(t, 100, transformer.bucketSize)
	})

	t.Run("zero options use defaults", func(t *testing.T) {
		transformer := NewImageTransformerWithOptions(TransformerOptions{})

		assert.Equal(t, MaxTransformDimension, transformer.maxWidth)
		assert.Equal(t, MaxTransformDimension, transformer.maxHeight)
		assert.Equal(t, DefaultMaxTotalPixels, transformer.maxTotalPixels)
		assert.Equal(t, DefaultBucketSize, transformer.bucketSize)
	})

	t.Run("negative options use defaults", func(t *testing.T) {
		opts := TransformerOptions{
			MaxWidth:       -100,
			MaxHeight:      -200,
			MaxTotalPixels: -1000,
			BucketSize:     -50,
		}

		transformer := NewImageTransformerWithOptions(opts)

		assert.Equal(t, MaxTransformDimension, transformer.maxWidth)
		assert.Equal(t, MaxTransformDimension, transformer.maxHeight)
		assert.Equal(t, DefaultMaxTotalPixels, transformer.maxTotalPixels)
		assert.Equal(t, DefaultBucketSize, transformer.bucketSize)
	})
}

// =============================================================================
// ValidateOptions Tests
// =============================================================================

func TestImageTransformer_ValidateOptions(t *testing.T) {
	transformer := NewImageTransformer(1920, 1080)

	t.Run("nil options returns nil", func(t *testing.T) {
		err := transformer.ValidateOptions(nil)
		assert.NoError(t, err)
	})

	t.Run("valid options", func(t *testing.T) {
		opts := &TransformOptions{
			Width:   800,
			Height:  600,
			Format:  "webp",
			Quality: 80,
			Fit:     FitCover,
		}

		err := transformer.ValidateOptions(opts)
		assert.NoError(t, err)
	})

	t.Run("negative width", func(t *testing.T) {
		opts := &TransformOptions{Width: -100}

		err := transformer.ValidateOptions(opts)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidDimensions)
	})

	t.Run("negative height", func(t *testing.T) {
		opts := &TransformOptions{Height: -100}

		err := transformer.ValidateOptions(opts)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidDimensions)
	})

	t.Run("width exceeds max", func(t *testing.T) {
		opts := &TransformOptions{Width: 5000}

		err := transformer.ValidateOptions(opts)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrImageTooLarge)
	})

	t.Run("height exceeds max", func(t *testing.T) {
		opts := &TransformOptions{Height: 5000}

		err := transformer.ValidateOptions(opts)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrImageTooLarge)
	})

	t.Run("total pixels exceeds max", func(t *testing.T) {
		// Use a transformer with low max pixels
		smallTransformer := NewImageTransformerWithOptions(TransformerOptions{
			MaxWidth:       10000,
			MaxHeight:      10000,
			MaxTotalPixels: 1000000, // 1 megapixel
		})

		opts := &TransformOptions{Width: 2000, Height: 2000} // 4 megapixels

		err := smallTransformer.ValidateOptions(opts)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrTooManyPixels)
	})

	t.Run("unsupported format", func(t *testing.T) {
		opts := &TransformOptions{Format: "gif"}

		err := transformer.ValidateOptions(opts)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUnsupportedFormat)
	})

	t.Run("quality normalized", func(t *testing.T) {
		opts := &TransformOptions{Quality: 0}
		err := transformer.ValidateOptions(opts)
		assert.NoError(t, err)
		assert.Equal(t, 80, opts.Quality) // Default quality

		opts = &TransformOptions{Quality: 150}
		err = transformer.ValidateOptions(opts)
		assert.NoError(t, err)
		assert.Equal(t, 100, opts.Quality) // Capped at 100
	})

	t.Run("fit mode normalized", func(t *testing.T) {
		opts := &TransformOptions{Fit: ""}
		err := transformer.ValidateOptions(opts)
		assert.NoError(t, err)
		assert.Equal(t, FitCover, opts.Fit) // Default fit mode
	})

	t.Run("format normalized to lowercase", func(t *testing.T) {
		opts := &TransformOptions{Format: "WEBP"}
		err := transformer.ValidateOptions(opts)
		assert.NoError(t, err)
		assert.Equal(t, "webp", opts.Format)
	})

	t.Run("dimensions bucketed", func(t *testing.T) {
		opts := &TransformOptions{Width: 823, Height: 617}
		err := transformer.ValidateOptions(opts)
		assert.NoError(t, err)
		assert.Equal(t, 850, opts.Width)  // Bucketed
		assert.Equal(t, 600, opts.Height) // Bucketed
	})
}

// =============================================================================
// calculateDimensions Tests
// =============================================================================

func TestImageTransformer_calculateDimensions(t *testing.T) {
	transformer := NewImageTransformer(8192, 8192)

	tests := []struct {
		name         string
		origWidth    int
		origHeight   int
		targetWidth  int
		targetHeight int
		fit          FitMode
		expectWidth  int
		expectHeight int
	}{
		{
			name:         "both dimensions specified",
			origWidth:    1000,
			origHeight:   500,
			targetWidth:  800,
			targetHeight: 600,
			fit:          FitCover,
			expectWidth:  800,
			expectHeight: 600,
		},
		{
			name:         "only width specified - landscape",
			origWidth:    1000,
			origHeight:   500,
			targetWidth:  500,
			targetHeight: 0,
			fit:          FitCover,
			expectWidth:  500,
			expectHeight: 250,
		},
		{
			name:         "only width specified - portrait",
			origWidth:    500,
			origHeight:   1000,
			targetWidth:  250,
			targetHeight: 0,
			fit:          FitCover,
			expectWidth:  250,
			expectHeight: 500,
		},
		{
			name:         "only height specified - landscape",
			origWidth:    1000,
			origHeight:   500,
			targetWidth:  0,
			targetHeight: 250,
			fit:          FitCover,
			expectWidth:  500,
			expectHeight: 250,
		},
		{
			name:         "only height specified - portrait",
			origWidth:    500,
			origHeight:   1000,
			targetWidth:  0,
			targetHeight: 500,
			fit:          FitCover,
			expectWidth:  250,
			expectHeight: 500,
		},
		{
			name:         "no resize needed",
			origWidth:    800,
			origHeight:   600,
			targetWidth:  0,
			targetHeight: 0,
			fit:          FitCover,
			expectWidth:  800,
			expectHeight: 600,
		},
		{
			name:         "square image with width only",
			origWidth:    1000,
			origHeight:   1000,
			targetWidth:  500,
			targetHeight: 0,
			fit:          FitCover,
			expectWidth:  500,
			expectHeight: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			width, height := transformer.calculateDimensions(tt.origWidth, tt.origHeight, tt.targetWidth, tt.targetHeight, tt.fit)
			assert.Equal(t, tt.expectWidth, width)
			assert.Equal(t, tt.expectHeight, height)
		})
	}
}

func TestImageTransformer_calculateDimensions_ClampsToMax(t *testing.T) {
	transformer := NewImageTransformer(1000, 1000)

	// Test that calculated dimensions are clamped to max
	width, height := transformer.calculateDimensions(100, 100, 2000, 0, FitCover)

	assert.LessOrEqual(t, width, 1000)
	assert.LessOrEqual(t, height, 1000)
}

// =============================================================================
// determineOutputFormat Tests
// =============================================================================

func TestImageTransformer_determineOutputFormat(t *testing.T) {
	transformer := NewImageTransformer(1920, 1080)

	tests := []struct {
		name            string
		inputType       string
		requestedFormat string
		expected        string
	}{
		{"requested format takes precedence", "image/jpeg", "webp", "webp"},
		{"jpeg input", "image/jpeg", "", "jpg"},
		{"png input", "image/png", "", "png"},
		{"webp input", "image/webp", "", "webp"},
		{"avif input", "image/avif", "", "avif"},
		{"gif defaults to jpg", "image/gif", "", "jpg"},
		{"bmp defaults to jpg", "image/bmp", "", "jpg"},
		{"tiff defaults to jpg", "image/tiff", "", "jpg"},
		{"unknown defaults to jpg", "image/unknown", "", "jpg"},
		{"empty content type", "", "", "jpg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := transformer.determineOutputFormat(tt.inputType, tt.requestedFormat)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// ParseTransformOptions Tests
// =============================================================================

func TestParseTransformOptions(t *testing.T) {
	t.Run("no options returns nil", func(t *testing.T) {
		opts := ParseTransformOptions(0, 0, "", 0, "")
		assert.Nil(t, opts)
	})

	t.Run("width only", func(t *testing.T) {
		opts := ParseTransformOptions(800, 0, "", 0, "")
		require.NotNil(t, opts)
		assert.Equal(t, 800, opts.Width)
		assert.Equal(t, 0, opts.Height)
		assert.Equal(t, FitCover, opts.Fit) // Default
	})

	t.Run("height only", func(t *testing.T) {
		opts := ParseTransformOptions(0, 600, "", 0, "")
		require.NotNil(t, opts)
		assert.Equal(t, 0, opts.Width)
		assert.Equal(t, 600, opts.Height)
	})

	t.Run("format only", func(t *testing.T) {
		opts := ParseTransformOptions(0, 0, "webp", 0, "")
		require.NotNil(t, opts)
		assert.Equal(t, "webp", opts.Format)
	})

	t.Run("quality only", func(t *testing.T) {
		opts := ParseTransformOptions(0, 0, "", 90, "")
		require.NotNil(t, opts)
		assert.Equal(t, 90, opts.Quality)
	})

	t.Run("fit modes", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected FitMode
		}{
			{"cover", FitCover},
			{"COVER", FitCover},
			{"Cover", FitCover},
			{"contain", FitContain},
			{"fill", FitFill},
			{"inside", FitInside},
			{"outside", FitOutside},
			{"invalid", FitCover}, // Default
			{"", FitCover},        // Default
		}

		for _, tc := range testCases {
			t.Run("fit_"+tc.input, func(t *testing.T) {
				opts := ParseTransformOptions(100, 0, "", 0, tc.input)
				require.NotNil(t, opts)
				assert.Equal(t, tc.expected, opts.Fit)
			})
		}
	})

	t.Run("all options", func(t *testing.T) {
		opts := ParseTransformOptions(800, 600, "png", 85, "contain")

		require.NotNil(t, opts)
		assert.Equal(t, 800, opts.Width)
		assert.Equal(t, 600, opts.Height)
		assert.Equal(t, "png", opts.Format)
		assert.Equal(t, 85, opts.Quality)
		assert.Equal(t, FitContain, opts.Fit)
	})
}

// =============================================================================
// TransformOptions Struct Tests
// =============================================================================

func TestTransformOptions_Struct(t *testing.T) {
	t.Run("all fields", func(t *testing.T) {
		opts := TransformOptions{
			Width:   1920,
			Height:  1080,
			Format:  "webp",
			Quality: 90,
			Fit:     FitContain,
		}

		assert.Equal(t, 1920, opts.Width)
		assert.Equal(t, 1080, opts.Height)
		assert.Equal(t, "webp", opts.Format)
		assert.Equal(t, 90, opts.Quality)
		assert.Equal(t, FitContain, opts.Fit)
	})

	t.Run("zero value", func(t *testing.T) {
		opts := TransformOptions{}

		assert.Zero(t, opts.Width)
		assert.Zero(t, opts.Height)
		assert.Empty(t, opts.Format)
		assert.Zero(t, opts.Quality)
		assert.Empty(t, opts.Fit)
	})
}

// =============================================================================
// TransformResult Struct Tests
// =============================================================================

func TestTransformResult_Struct(t *testing.T) {
	t.Run("all fields", func(t *testing.T) {
		result := TransformResult{
			Data:        []byte("image data"),
			ContentType: "image/webp",
			Width:       800,
			Height:      600,
		}

		assert.Equal(t, []byte("image data"), result.Data)
		assert.Equal(t, "image/webp", result.ContentType)
		assert.Equal(t, 800, result.Width)
		assert.Equal(t, 600, result.Height)
	})

	t.Run("zero value", func(t *testing.T) {
		result := TransformResult{}

		assert.Nil(t, result.Data)
		assert.Empty(t, result.ContentType)
		assert.Zero(t, result.Width)
		assert.Zero(t, result.Height)
	})
}

// =============================================================================
// Constants Tests
// =============================================================================

func TestTransformConstants(t *testing.T) {
	t.Run("MaxTransformDimension", func(t *testing.T) {
		assert.Equal(t, 8192, MaxTransformDimension)
	})

	t.Run("DefaultMaxTotalPixels", func(t *testing.T) {
		assert.Equal(t, 16_000_000, DefaultMaxTotalPixels)
	})

	t.Run("DefaultBucketSize", func(t *testing.T) {
		assert.Equal(t, 50, DefaultBucketSize)
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkBucketDimension(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = BucketDimension(823, 50)
	}
}

func BenchmarkCanTransform(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = CanTransform("image/jpeg; charset=utf-8")
	}
}

func BenchmarkParseTransformOptions(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = ParseTransformOptions(800, 600, "webp", 85, "contain")
	}
}

func BenchmarkValidateOptions(b *testing.B) {
	transformer := NewImageTransformer(8192, 8192)
	opts := &TransformOptions{
		Width:   800,
		Height:  600,
		Format:  "webp",
		Quality: 80,
		Fit:     FitCover,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = transformer.ValidateOptions(opts)
	}
}

func BenchmarkCalculateDimensions(b *testing.B) {
	transformer := NewImageTransformer(8192, 8192)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = transformer.calculateDimensions(1920, 1080, 800, 0, FitCover)
	}
}
