package ai

import (
	"context"
)

// OCRProviderType represents the type of OCR provider
type OCRProviderType string

const (
	OCRProviderTypeTesseract OCRProviderType = "tesseract"
)

// OCRResult represents the result of OCR processing
type OCRResult struct {
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
	Pages      int     `json:"pages"`
	Language   string  `json:"language,omitempty"`
}

// OCRProvider defines the interface for OCR providers
type OCRProvider interface {
	// Name returns the provider name
	Name() string

	// Type returns the provider type
	Type() OCRProviderType

	// ExtractTextFromPDF extracts text from PDF bytes using OCR
	ExtractTextFromPDF(ctx context.Context, pdfData []byte, languages []string) (*OCRResult, error)

	// ExtractTextFromImage extracts text from image bytes
	ExtractTextFromImage(ctx context.Context, imageData []byte, languages []string) (*OCRResult, error)

	// IsAvailable checks if the provider is properly configured and available
	IsAvailable() bool

	// Close cleans up resources
	Close() error
}

// OCRProviderConfig represents OCR provider configuration
type OCRProviderConfig struct {
	Type      OCRProviderType `json:"type"`
	Languages []string        `json:"languages"` // e.g., ["eng", "deu", "nld"]
}

// NewOCRProvider creates an OCR provider based on configuration
func NewOCRProvider(cfg OCRProviderConfig) (OCRProvider, error) {
	switch cfg.Type {
	case OCRProviderTypeTesseract:
		return NewTesseractProvider(cfg)
	default:
		return NewTesseractProvider(cfg) // Default to Tesseract
	}
}
