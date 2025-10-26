package realtime

import (
	"github.com/wayli-app/fluxbase/internal/auth"
)

// AuthServiceAdapter adapts auth.Service to realtime.AuthService interface
type AuthServiceAdapter struct {
	service *auth.Service
}

// NewAuthServiceAdapter creates a new auth service adapter
func NewAuthServiceAdapter(service *auth.Service) *AuthServiceAdapter {
	return &AuthServiceAdapter{
		service: service,
	}
}

// ValidateToken validates a JWT token and returns claims
func (a *AuthServiceAdapter) ValidateToken(token string) (*TokenClaims, error) {
	claims, err := a.service.ValidateToken(token)
	if err != nil {
		return nil, err
	}

	return &TokenClaims{
		UserID:    claims.UserID,
		Email:     claims.Email,
		Role:      claims.Role,
		SessionID: claims.SessionID,
	}, nil
}
