//go:build !vips
// +build !vips

package storage

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests pin the behavior of the novips (default) build: the transformer
// is constructable and validates options, but it cannot actually transform
// images — every real transform attempt returns an error. The vips build has
// its own suite (transform_test.go). This file is the regression guard that
// ensures the !vips build compiles and behaves correctly on its own; it would
// have caught the duplicate-symbol bug that previously broke `go build -tags
// vips` (transform_types.go redeclared NewImageTransformerWithOptions and
// ValidateOptions).

// TestNewImageTransformerWithOptions_NotInitialized confirms that without the
// vips build tag the transformer reports itself as uninitialized.
func TestNewImageTransformerWithOptions_NotInitialized(t *testing.T) {
	transformer := NewImageTransformerWithOptions(TransformerOptions{
		MaxWidth:       4096,
		MaxHeight:      2160,
		MaxTotalPixels: DefaultMaxTotalPixels,
		BucketSize:     DefaultBucketSize,
	})

	assert.False(t, transformer.initialized, "novips transformer must not be initialized")
	assert.Equal(t, 4096, transformer.maxWidth)
	assert.Equal(t, 2160, transformer.maxHeight)
}

// TestNewImageTransformerWithOptions_Defaults confirms defaults are applied.
func TestNewImageTransformerWithOptions_Defaults(t *testing.T) {
	transformer := NewImageTransformerWithOptions(TransformerOptions{})

	assert.False(t, transformer.initialized)
	assert.Equal(t, MaxTransformDimension, transformer.maxWidth)
	assert.Equal(t, MaxTransformDimension, transformer.maxHeight)
	assert.Equal(t, DefaultMaxTotalPixels, transformer.maxTotalPixels)
	assert.Equal(t, DefaultBucketSize, transformer.bucketSize)
}

// TestNovips_InitShutdown_Nop confirms the no-op lifecycle functions are safe
// to call even when vips is not compiled in.
func TestNovips_InitShutdown_Nop(t *testing.T) {
	assert.NotPanics(t, func() {
		InitVips()
		ShutdownVips()
	})
}

// TestNovips_Transform_RequiresVips confirms an actual transform request fails
// with a clear error on the novips build.
func TestNovips_Transform_RequiresVips(t *testing.T) {
	transformer := NewImageTransformerWithOptions(TransformerOptions{})
	opts := &TransformOptions{Width: 100, Height: 100}

	result, err := transformer.Transform(bytes.NewReader(smallPNG), "image/png", opts)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "vips")
}

// TestNovips_Transform_NoOpPassesThrough confirms that when no transform is
// requested (nil or empty opts) the call is a no-op rather than an error,
// matching the vips build's behavior.
func TestNovips_Transform_NoOpPassesThrough(t *testing.T) {
	transformer := NewImageTransformerWithOptions(TransformerOptions{})

	t.Run("nil opts", func(t *testing.T) {
		result, err := transformer.Transform(bytes.NewReader(smallPNG), "image/png", nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("empty opts", func(t *testing.T) {
		result, err := transformer.Transform(
			bytes.NewReader(smallPNG),
			"image/png",
			&TransformOptions{},
		)
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

// TestNovips_TransformReader_RequiresVips mirrors the above for TransformReader.
func TestNovips_TransformReader_RequiresVips(t *testing.T) {
	transformer := NewImageTransformerWithOptions(TransformerOptions{})
	opts := &TransformOptions{Width: 100, Height: 100}

	rc, contentType, size, err := transformer.TransformReader(
		bytes.NewReader(smallPNG), "image/png", opts,
	)
	require.Error(t, err)
	assert.Nil(t, rc)
	assert.Empty(t, contentType)
	assert.Equal(t, int64(0), size)
}

// TestNovips_TransformReader_NoOpPassesThrough confirms nil/empty opts are
// a no-op for the reader variant.
func TestNovips_TransformReader_NoOpPassesThrough(t *testing.T) {
	transformer := NewImageTransformerWithOptions(TransformerOptions{})

	t.Run("nil opts", func(t *testing.T) {
		rc, contentType, size, err := transformer.TransformReader(
			bytes.NewReader(smallPNG), "image/png", nil,
		)
		require.NoError(t, err)
		assert.Nil(t, rc)
		assert.Empty(t, contentType)
		assert.Equal(t, int64(0), size)
	})
}

// TestNovips_ValidateOptions_StillWorks confirms validation logic is present
// on the novips build (it is shared behavior, independent of vips).
func TestNovips_ValidateOptions_StillWorks(t *testing.T) {
	transformer := NewImageTransformerWithOptions(TransformerOptions{BucketSize: 50})

	t.Run("negative width rejected", func(t *testing.T) {
		err := transformer.ValidateOptions(&TransformOptions{Width: -1})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidDimensions)
	})

	t.Run("over max width rejected", func(t *testing.T) {
		transformer := NewImageTransformerWithOptions(TransformerOptions{MaxWidth: 100})
		err := transformer.ValidateOptions(&TransformOptions{Width: 500})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrImageTooLarge)
	})

	t.Run("unsupported format rejected", func(t *testing.T) {
		err := transformer.ValidateOptions(&TransformOptions{Format: "bmp"})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUnsupportedFormat)
	})

	t.Run("nil opts ok", func(t *testing.T) {
		assert.NoError(t, transformer.ValidateOptions(nil))
	})

	t.Run("buckets dimensions", func(t *testing.T) {
		opts := &TransformOptions{Width: 123, Height: 456}
		_ = transformer.ValidateOptions(opts)
		// BucketDimension(123, 50) == 100, BucketDimension(456, 50) == 450
		assert.Equal(t, 100, opts.Width)
		assert.Equal(t, 450, opts.Height)
	})
}

// TestNovips_TransformErrors_AreSentinels confirms the exported sentinel errors
// are usable with errors.Is (the validation paths rely on this).
func TestNovips_TransformErrors_AreSentinels(t *testing.T) {
	transformer := NewImageTransformerWithOptions(TransformerOptions{})

	// Drive each sentinel through ValidateOptions and assert errors.Is wrapping.
	cases := []struct {
		name string
		opts *TransformOptions
		want error
	}{
		{"invalid dimensions", &TransformOptions{Width: -1}, ErrInvalidDimensions},
		{"image too large", &TransformOptions{Width: 999_999}, ErrImageTooLarge},
		{"unsupported format", &TransformOptions{Format: "gif"}, ErrUnsupportedFormat},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := transformer.ValidateOptions(tc.opts)
			require.Error(t, err)
			assert.True(t, errors.Is(err, tc.want))
		})
	}
}

// smallPNG is a minimal 1x1 PNG used as transform input in these tests. It is
// never actually decoded because the novips transform fails before any image
// processing occurs; it just needs to be valid enough to serve as a reader.
var smallPNG = []byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
}
