package ai

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// EmbeddingService coordinates embedding generation using configured providers
type EmbeddingService struct {
	provider     EmbeddingProvider
	providerMu   sync.RWMutex
	defaultModel string
	rateLimiter  *embeddingRateLimiter
	cacheEnabled bool
	cacheResults map[string]*cachedEmbedding
	cacheMu      sync.RWMutex
	cacheTTL     time.Duration
}

// EmbeddingServiceConfig contains configuration for the embedding service
type EmbeddingServiceConfig struct {
	Provider     ProviderConfig
	DefaultModel string
	RateLimitRPM int // Requests per minute (0 = no limit)
	CacheEnabled bool
	CacheTTL     time.Duration
}

// cachedEmbedding stores a cached embedding result
type cachedEmbedding struct {
	embedding []float32
	expiresAt time.Time
}

// embeddingRateLimiter provides simple rate limiting
type embeddingRateLimiter struct {
	mu        sync.Mutex
	tokens    int
	maxTokens int
	lastReset time.Time
	window    time.Duration
}

// NewEmbeddingService creates a new embedding service
func NewEmbeddingService(cfg EmbeddingServiceConfig) (*EmbeddingService, error) {
	provider, err := NewEmbeddingProvider(cfg.Provider)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding provider: %w", err)
	}

	// Use provider default model if not specified
	defaultModel := cfg.DefaultModel
	if defaultModel == "" {
		defaultModel = provider.DefaultModel()
	}

	var rateLimiter *embeddingRateLimiter
	if cfg.RateLimitRPM > 0 {
		rateLimiter = &embeddingRateLimiter{
			tokens:    cfg.RateLimitRPM,
			maxTokens: cfg.RateLimitRPM,
			lastReset: time.Now(),
			window:    time.Minute,
		}
	}

	cacheTTL := cfg.CacheTTL
	if cacheTTL == 0 {
		cacheTTL = 15 * time.Minute
	}

	service := &EmbeddingService{
		provider:     provider,
		defaultModel: defaultModel,
		rateLimiter:  rateLimiter,
		cacheEnabled: cfg.CacheEnabled,
		cacheResults: make(map[string]*cachedEmbedding),
		cacheTTL:     cacheTTL,
	}

	// Start cache cleanup goroutine if caching is enabled
	if cfg.CacheEnabled {
		go service.cleanupCache()
	}

	return service, nil
}

// Embed generates embeddings for the given texts
func (s *EmbeddingService) Embed(ctx context.Context, texts []string, model string) (*EmbeddingResponse, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts provided for embedding")
	}

	// Use default model if not specified
	if model == "" {
		model = s.defaultModel
	}

	// Check rate limit
	if s.rateLimiter != nil {
		if !s.rateLimiter.allow() {
			return nil, fmt.Errorf("embedding rate limit exceeded")
		}
	}

	// Try to get cached embeddings
	var uncachedTexts []string
	var uncachedIndices []int
	embeddings := make([][]float32, len(texts))

	if s.cacheEnabled {
		for i, text := range texts {
			cacheKey := s.cacheKey(text, model)
			if cached := s.getFromCache(cacheKey); cached != nil {
				embeddings[i] = cached
			} else {
				uncachedTexts = append(uncachedTexts, text)
				uncachedIndices = append(uncachedIndices, i)
			}
		}

		// All embeddings were cached
		if len(uncachedTexts) == 0 {
			return &EmbeddingResponse{
				Embeddings: embeddings,
				Model:      model,
				Dimensions: len(embeddings[0]),
			}, nil
		}
	} else {
		uncachedTexts = texts
		for i := range texts {
			uncachedIndices = append(uncachedIndices, i)
		}
	}

	// Get embeddings from provider
	s.providerMu.RLock()
	provider := s.provider
	s.providerMu.RUnlock()

	resp, err := provider.Embed(ctx, uncachedTexts, model)
	if err != nil {
		return nil, fmt.Errorf("embedding failed: %w", err)
	}

	// Merge cached and new embeddings
	for i, idx := range uncachedIndices {
		embeddings[idx] = resp.Embeddings[i]

		// Cache the new embedding
		if s.cacheEnabled {
			cacheKey := s.cacheKey(uncachedTexts[i], model)
			s.addToCache(cacheKey, resp.Embeddings[i])
		}
	}

	return &EmbeddingResponse{
		Embeddings: embeddings,
		Model:      resp.Model,
		Dimensions: resp.Dimensions,
		Usage:      resp.Usage,
	}, nil
}

// EmbedSingle generates an embedding for a single text
func (s *EmbeddingService) EmbedSingle(ctx context.Context, text string, model string) ([]float32, error) {
	resp, err := s.Embed(ctx, []string{text}, model)
	if err != nil {
		return nil, err
	}
	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return resp.Embeddings[0], nil
}

// SupportedModels returns the models supported by the current provider
func (s *EmbeddingService) SupportedModels() []EmbeddingModel {
	s.providerMu.RLock()
	defer s.providerMu.RUnlock()
	return s.provider.SupportedModels()
}

// DefaultModel returns the default embedding model
func (s *EmbeddingService) DefaultModel() string {
	return s.defaultModel
}

// SetProvider updates the embedding provider
func (s *EmbeddingService) SetProvider(cfg ProviderConfig) error {
	provider, err := NewEmbeddingProvider(cfg)
	if err != nil {
		return fmt.Errorf("failed to create new embedding provider: %w", err)
	}

	s.providerMu.Lock()
	s.provider = provider
	s.providerMu.Unlock()

	return nil
}

// IsConfigured returns whether the service has a configured provider
func (s *EmbeddingService) IsConfigured() bool {
	s.providerMu.RLock()
	defer s.providerMu.RUnlock()
	return s.provider != nil
}

// cacheKey generates a cache key for a text and model
func (s *EmbeddingService) cacheKey(text, model string) string {
	// Simple hash-based key
	return fmt.Sprintf("%s:%s", model, text)
}

// getFromCache retrieves an embedding from cache
func (s *EmbeddingService) getFromCache(key string) []float32 {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	cached, exists := s.cacheResults[key]
	if !exists || time.Now().After(cached.expiresAt) {
		return nil
	}
	return cached.embedding
}

// addToCache adds an embedding to cache
func (s *EmbeddingService) addToCache(key string, embedding []float32) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	s.cacheResults[key] = &cachedEmbedding{
		embedding: embedding,
		expiresAt: time.Now().Add(s.cacheTTL),
	}
}

// cleanupCache periodically removes expired cache entries
func (s *EmbeddingService) cleanupCache() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.cacheMu.Lock()
		now := time.Now()
		for key, cached := range s.cacheResults {
			if now.After(cached.expiresAt) {
				delete(s.cacheResults, key)
			}
		}
		s.cacheMu.Unlock()
	}
}

// allow checks if a request is allowed under rate limiting
func (rl *embeddingRateLimiter) allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	if now.Sub(rl.lastReset) >= rl.window {
		rl.tokens = rl.maxTokens
		rl.lastReset = now
	}

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}
	return false
}

// EmbeddingServiceFromConfig creates an EmbeddingService from AI config
func EmbeddingServiceFromConfig(aiCfg interface{}) (*EmbeddingService, error) {
	// Type assert to get the relevant fields
	type aiConfig interface {
		GetProviderType() string
		GetProviderEnabled() bool
		GetEmbeddingModel() string
		GetOpenAIConfig() OpenAIConfig
		GetAzureConfig() AzureConfig
		GetOllamaConfig() OllamaConfig
	}

	// Try direct field access via reflection or interface
	// This is a simplified approach - in practice you'd pass the config directly
	log.Debug().Msg("Creating embedding service from config")

	return nil, fmt.Errorf("use NewEmbeddingService with explicit config instead")
}
