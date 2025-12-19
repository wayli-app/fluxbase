//go:build !cgo || !ocr

package ai

import (
	"context"
	"fmt"
)

// TesseractProvider is a stub for environments without Tesseract/CGO support
type TesseractProvider struct {
	name string
}

// NewTesseractProvider creates a stub provider that reports unavailability
func NewTesseractProvider(cfg OCRProviderConfig) (*TesseractProvider, error) {
	return &TesseractProvider{
		name: "tesseract (unavailable)",
	}, nil
}

func (p *TesseractProvider) Name() string {
	return p.name
}

func (p *TesseractProvider) Type() OCRProviderType {
	return OCRProviderTypeTesseract
}

func (p *TesseractProvider) IsAvailable() bool {
	return false
}

func (p *TesseractProvider) ExtractTextFromPDF(ctx context.Context, pdfData []byte, languages []string) (*OCRResult, error) {
	return nil, fmt.Errorf("OCR not available: built without Tesseract support")
}

func (p *TesseractProvider) ExtractTextFromImage(ctx context.Context, imageData []byte, languages []string) (*OCRResult, error) {
	return nil, fmt.Errorf("OCR not available: built without Tesseract support")
}

func (p *TesseractProvider) Close() error {
	return nil
}
