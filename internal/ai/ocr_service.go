package ai

import (
	"context"
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
)

// OCRService manages OCR providers
type OCRService struct {
	provider         OCRProvider
	defaultLanguages []string
	mu               sync.RWMutex
	enabled          bool
}

// OCRServiceConfig contains configuration for the OCR service
type OCRServiceConfig struct {
	Enabled          bool
	ProviderType     OCRProviderType
	DefaultLanguages []string
}

// NewOCRService creates a new OCR service
func NewOCRService(cfg OCRServiceConfig) (*OCRService, error) {
	if !cfg.Enabled {
		log.Info().Msg("OCR service disabled")
		return &OCRService{enabled: false}, nil
	}

	// Set default languages if not provided
	languages := cfg.DefaultLanguages
	if len(languages) == 0 {
		languages = []string{"eng"}
	}

	provider, err := NewOCRProvider(OCRProviderConfig{
		Type:      cfg.ProviderType,
		Languages: languages,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create OCR provider: %w", err)
	}

	if !provider.IsAvailable() {
		log.Warn().Str("provider", string(cfg.ProviderType)).Msg("OCR provider not available, OCR will be disabled")
		return &OCRService{enabled: false}, nil
	}

	log.Info().
		Str("provider", provider.Name()).
		Strs("languages", languages).
		Msg("OCR service initialized")

	return &OCRService{
		provider:         provider,
		defaultLanguages: languages,
		enabled:          true,
	}, nil
}

// ExtractTextFromPDF attempts OCR on a PDF document
// If languages is empty, uses the service's default languages
func (s *OCRService) ExtractTextFromPDF(ctx context.Context, pdfData []byte, languages []string) (*OCRResult, error) {
	if !s.enabled {
		return nil, fmt.Errorf("OCR service is not enabled")
	}

	s.mu.RLock()
	provider := s.provider
	defaultLangs := s.defaultLanguages
	s.mu.RUnlock()

	// Use provided languages or fall back to defaults
	if len(languages) == 0 {
		languages = defaultLangs
	}

	result, err := provider.ExtractTextFromPDF(ctx, pdfData, languages)
	if err != nil {
		return nil, fmt.Errorf("OCR extraction failed: %w", err)
	}

	log.Debug().
		Int("pages", result.Pages).
		Float64("confidence", result.Confidence).
		Int("text_length", len(result.Text)).
		Msg("OCR extraction completed")

	return result, nil
}

// ExtractTextFromImage attempts OCR on an image
func (s *OCRService) ExtractTextFromImage(ctx context.Context, imageData []byte, languages []string) (*OCRResult, error) {
	if !s.enabled {
		return nil, fmt.Errorf("OCR service is not enabled")
	}

	s.mu.RLock()
	provider := s.provider
	defaultLangs := s.defaultLanguages
	s.mu.RUnlock()

	if len(languages) == 0 {
		languages = defaultLangs
	}

	return provider.ExtractTextFromImage(ctx, imageData, languages)
}

// IsEnabled returns whether OCR is enabled and available
func (s *OCRService) IsEnabled() bool {
	return s.enabled
}

// GetDefaultLanguages returns the configured default languages
func (s *OCRService) GetDefaultLanguages() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.defaultLanguages
}

// Close cleans up the OCR service resources
func (s *OCRService) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.provider != nil {
		return s.provider.Close()
	}
	return nil
}
