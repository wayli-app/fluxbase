//go:build cgo && ocr

package ai

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/otiai10/gosseract/v2"
	"github.com/rs/zerolog/log"
)

// TesseractProvider implements OCR using Tesseract
type TesseractProvider struct {
	name             string
	defaultLanguages []string
	available        bool
	tesseractPath    string
	pdftoppmPath     string
}

// NewTesseractProvider creates a new Tesseract OCR provider
func NewTesseractProvider(cfg OCRProviderConfig) (*TesseractProvider, error) {
	languages := cfg.Languages
	if len(languages) == 0 {
		languages = []string{"eng"}
	}

	// Check if Tesseract is available
	tesseractPath, err := exec.LookPath("tesseract")
	available := err == nil

	// Check if pdftoppm is available (for PDF conversion)
	pdftoppmPath, _ := exec.LookPath("pdftoppm")

	if !available {
		log.Warn().Msg("Tesseract not found in PATH, OCR will be unavailable")
	} else {
		log.Debug().
			Str("tesseract_path", tesseractPath).
			Str("pdftoppm_path", pdftoppmPath).
			Strs("languages", languages).
			Msg("Tesseract provider initialized")
	}

	return &TesseractProvider{
		name:             "tesseract",
		defaultLanguages: languages,
		available:        available,
		tesseractPath:    tesseractPath,
		pdftoppmPath:     pdftoppmPath,
	}, nil
}

func (p *TesseractProvider) Name() string {
	return p.name
}

func (p *TesseractProvider) Type() OCRProviderType {
	return OCRProviderTypeTesseract
}

func (p *TesseractProvider) IsAvailable() bool {
	return p.available
}

func (p *TesseractProvider) ExtractTextFromPDF(ctx context.Context, pdfData []byte, languages []string) (*OCRResult, error) {
	if !p.available {
		return nil, fmt.Errorf("tesseract is not available")
	}

	if len(languages) == 0 {
		languages = p.defaultLanguages
	}

	// Check if pdftoppm is available for PDF conversion
	if p.pdftoppmPath == "" {
		return nil, fmt.Errorf("pdftoppm (poppler-utils) is required for PDF OCR but not found")
	}

	// Convert PDF to images
	images, tmpDir, err := p.pdfToImages(ctx, pdfData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert PDF to images: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if len(images) == 0 {
		return nil, fmt.Errorf("no pages extracted from PDF")
	}

	// Process each image with Tesseract
	var allText strings.Builder
	var totalConfidence float64

	for i, imgPath := range images {
		text, confidence, err := p.ocrImage(imgPath, languages)
		if err != nil {
			log.Warn().Err(err).Int("page", i+1).Msg("OCR failed for page, continuing with others")
			continue
		}

		if text != "" {
			allText.WriteString(text)
			allText.WriteString("\n\n")
			totalConfidence += confidence
		}
	}

	avgConfidence := 0.0
	if len(images) > 0 {
		avgConfidence = totalConfidence / float64(len(images))
	}

	return &OCRResult{
		Text:       strings.TrimSpace(allText.String()),
		Confidence: avgConfidence,
		Pages:      len(images),
		Language:   strings.Join(languages, "+"),
	}, nil
}

func (p *TesseractProvider) ExtractTextFromImage(ctx context.Context, imageData []byte, languages []string) (*OCRResult, error) {
	if !p.available {
		return nil, fmt.Errorf("tesseract is not available")
	}

	if len(languages) == 0 {
		languages = p.defaultLanguages
	}

	// Write image to temp file
	tmpFile, err := os.CreateTemp("", "ocr-image-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(imageData); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	text, confidence, err := p.ocrImage(tmpFile.Name(), languages)
	if err != nil {
		return nil, err
	}

	return &OCRResult{
		Text:       text,
		Confidence: confidence,
		Pages:      1,
		Language:   strings.Join(languages, "+"),
	}, nil
}

func (p *TesseractProvider) Close() error {
	return nil
}

// pdfToImages converts PDF to PNG images using pdftoppm
func (p *TesseractProvider) pdfToImages(ctx context.Context, pdfData []byte) ([]string, string, error) {
	// Create temp directory for images
	tmpDir, err := os.MkdirTemp("", "ocr-pdf-*")
	if err != nil {
		return nil, "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Write PDF to temp file
	pdfPath := filepath.Join(tmpDir, "input.pdf")
	if err := os.WriteFile(pdfPath, pdfData, 0600); err != nil {
		os.RemoveAll(tmpDir)
		return nil, "", fmt.Errorf("failed to write PDF temp file: %w", err)
	}

	// Convert PDF to PNG images using pdftoppm
	// -png: output PNG format
	// -r 300: 300 DPI for good OCR quality
	outputPrefix := filepath.Join(tmpDir, "page")
	cmd := exec.CommandContext(ctx, p.pdftoppmPath, "-png", "-r", "300", pdfPath, outputPrefix)

	if output, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(tmpDir)
		return nil, "", fmt.Errorf("pdftoppm failed: %w, output: %s", err, string(output))
	}

	// Find generated image files
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, "", fmt.Errorf("failed to read temp dir: %w", err)
	}

	var images []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".png") {
			images = append(images, filepath.Join(tmpDir, entry.Name()))
		}
	}

	// Sort images by name to maintain page order
	sort.Strings(images)

	return images, tmpDir, nil
}

// ocrImage runs Tesseract on a single image file
func (p *TesseractProvider) ocrImage(imagePath string, languages []string) (string, float64, error) {
	client := gosseract.NewClient()
	defer client.Close()

	// Set languages
	langStr := strings.Join(languages, "+")
	if err := client.SetLanguage(langStr); err != nil {
		return "", 0, fmt.Errorf("failed to set language: %w", err)
	}

	// Set image
	if err := client.SetImage(imagePath); err != nil {
		return "", 0, fmt.Errorf("failed to set image: %w", err)
	}

	// Extract text
	text, err := client.Text()
	if err != nil {
		return "", 0, fmt.Errorf("OCR failed: %w", err)
	}

	// Estimate confidence based on text quality
	confidence := p.estimateConfidence(text)

	return strings.TrimSpace(text), confidence, nil
}

// estimateConfidence estimates OCR confidence based on text characteristics
func (p *TesseractProvider) estimateConfidence(text string) float64 {
	if len(text) == 0 {
		return 0
	}

	printable := 0
	total := 0
	for _, r := range text {
		total++
		if unicode.IsPrint(r) || unicode.IsSpace(r) {
			printable++
		}
	}

	if total == 0 {
		return 0
	}

	return float64(printable) / float64(total)
}
