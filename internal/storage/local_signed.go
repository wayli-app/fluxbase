package storage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

// signedURLToken represents the data encoded in a signed URL token
type signedURLToken struct {
	Bucket    string `json:"b"`
	Key       string `json:"k"`
	ExpiresAt int64  `json:"e"`
	Method    string `json:"m"`
	// Transform options (optional, for image downloads)
	TrWidth   int    `json:"tw,omitempty"` // Transform width
	TrHeight  int    `json:"th,omitempty"` // Transform height
	TrFormat  string `json:"tf,omitempty"` // Transform format
	TrQuality int    `json:"tq,omitempty"` // Transform quality
	TrFit     string `json:"ti,omitempty"` // Transform fit mode
}

// SignedTokenResult contains the result of validating a signed URL token
type SignedTokenResult struct {
	Bucket           string
	Key              string
	Method           string
	TransformWidth   int
	TransformHeight  int
	TransformFormat  string
	TransformQuality int
	TransformFit     string
}

// GenerateSignedURL generates a signed URL for temporary access to local storage
func (ls *LocalStorage) GenerateSignedURL(ctx context.Context, bucket, key string, opts *SignedURLOptions) (string, error) {
	if ls.signingSecret == "" {
		return "", fmt.Errorf("signing secret not configured for local storage")
	}
	if ls.baseURL == "" {
		return "", fmt.Errorf("base URL not configured for local storage")
	}

	if opts == nil {
		opts = &SignedURLOptions{
			ExpiresIn: 15 * time.Minute,
			Method:    "GET",
		}
	}
	if opts.ExpiresIn == 0 {
		opts.ExpiresIn = 15 * time.Minute
	}
	if opts.Method == "" {
		opts.Method = "GET"
	}

	// Create token data
	token := signedURLToken{
		Bucket:    bucket,
		Key:       key,
		ExpiresAt: time.Now().Add(opts.ExpiresIn).Unix(),
		Method:    opts.Method,
		// Include transform options if specified
		TrWidth:   opts.TransformWidth,
		TrHeight:  opts.TransformHeight,
		TrFormat:  opts.TransformFormat,
		TrQuality: opts.TransformQuality,
		TrFit:     opts.TransformFit,
	}

	// Encode token to JSON
	tokenJSON, err := json.Marshal(token)
	if err != nil {
		return "", fmt.Errorf("failed to encode token: %w", err)
	}

	// Sign the token with HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(ls.signingSecret))
	mac.Write(tokenJSON)
	signature := mac.Sum(nil)

	// Combine token and signature, then base64 encode
	tokenJSON = append(tokenJSON, signature...)
	encodedToken := base64.URLEncoding.EncodeToString(tokenJSON)

	// Build the signed URL
	signedURL := fmt.Sprintf("%s/api/v1/storage/object?token=%s", ls.baseURL, url.QueryEscape(encodedToken))

	return signedURL, nil
}

// ValidateSignedToken validates a signed URL token and returns the bucket and key
func (ls *LocalStorage) ValidateSignedToken(token string) (bucket, key, method string, err error) {
	if ls.signingSecret == "" {
		return "", "", "", fmt.Errorf("signing secret not configured")
	}

	// Decode the base64 token
	decoded, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid token encoding")
	}

	// Token must be at least 32 bytes (signature length) + some JSON
	if len(decoded) < 33 {
		return "", "", "", fmt.Errorf("invalid token length")
	}

	// Split token and signature (last 32 bytes are the HMAC-SHA256 signature)
	tokenJSON := decoded[:len(decoded)-32]
	providedSig := decoded[len(decoded)-32:]

	// Verify signature
	mac := hmac.New(sha256.New, []byte(ls.signingSecret))
	mac.Write(tokenJSON)
	expectedSig := mac.Sum(nil)

	if !hmac.Equal(providedSig, expectedSig) {
		return "", "", "", fmt.Errorf("invalid token signature")
	}

	// Parse token data
	var tokenData signedURLToken
	if err := json.Unmarshal(tokenJSON, &tokenData); err != nil {
		return "", "", "", fmt.Errorf("invalid token data")
	}

	// Check expiration
	if time.Now().Unix() > tokenData.ExpiresAt {
		return "", "", "", fmt.Errorf("token expired")
	}

	return tokenData.Bucket, tokenData.Key, tokenData.Method, nil
}

// ValidateSignedTokenFull validates a signed URL token and returns the full result including transforms
func (ls *LocalStorage) ValidateSignedTokenFull(token string) (*SignedTokenResult, error) {
	if ls.signingSecret == "" {
		return nil, fmt.Errorf("signing secret not configured")
	}

	// Decode the base64 token
	decoded, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token encoding")
	}

	// Token must be at least 32 bytes (signature length) + some JSON
	if len(decoded) < 33 {
		return nil, fmt.Errorf("invalid token length")
	}

	// Split token and signature (last 32 bytes are the HMAC-SHA256 signature)
	tokenJSON := decoded[:len(decoded)-32]
	providedSig := decoded[len(decoded)-32:]

	// Verify signature
	mac := hmac.New(sha256.New, []byte(ls.signingSecret))
	mac.Write(tokenJSON)
	expectedSig := mac.Sum(nil)

	if !hmac.Equal(providedSig, expectedSig) {
		return nil, fmt.Errorf("invalid token signature")
	}

	// Parse token data
	var tokenData signedURLToken
	if err := json.Unmarshal(tokenJSON, &tokenData); err != nil {
		return nil, fmt.Errorf("invalid token data")
	}

	// Check expiration
	if time.Now().Unix() > tokenData.ExpiresAt {
		return nil, fmt.Errorf("token expired")
	}

	return &SignedTokenResult{
		Bucket:           tokenData.Bucket,
		Key:              tokenData.Key,
		Method:           tokenData.Method,
		TransformWidth:   tokenData.TrWidth,
		TransformHeight:  tokenData.TrHeight,
		TransformFormat:  tokenData.TrFormat,
		TransformQuality: tokenData.TrQuality,
		TransformFit:     tokenData.TrFit,
	}, nil
}
