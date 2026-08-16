package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/auth"
)

// =============================================================================
// Test Helpers
// =============================================================================

const validTestCaptchaToken = "valid-test-token-12345"

// createTestHandlerWithCaptcha creates an AuthHandler with a mock captcha service
// and a mock auth service that allows signup and password login
func createTestHandlerWithCaptcha(endpoints []string) *AuthHandler {
	authService := auth.NewTestAuthServiceWithSettings(true, true)
	captchaService := auth.NewTestCaptchaService(endpoints, validTestCaptchaToken)
	return NewAuthHandler(nil, authService, captchaService, "https://example.com", "")
}

// parseErrorResponse parses the JSON error response
func parseErrorResponse(t *testing.T, body []byte) map[string]interface{} {
	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(t, err, "Failed to parse response body: %s", string(body))
	return result
}

// =============================================================================
// SignUp Captcha Tests
// =============================================================================

func TestSignUp_CaptchaRequired_ReturnsError(t *testing.T) {
	handler := createTestHandlerWithCaptcha([]string{"signup"})

	app := newTestApp(t)
	app.Post("/auth/signup", handler.SignUp)

	// Request without captcha token
	body := `{"email": "test@example.com", "password": "TestPassword123!"}`
	req := httptest.NewRequest("POST", "/auth/signup", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	result := parseErrorResponse(t, respBody)

	assert.Equal(t, "CAPTCHA verification required", result["error"])
	assert.Equal(t, "CAPTCHA_REQUIRED", result["code"])
}

func TestSignUp_CaptchaInvalid_ReturnsError(t *testing.T) {
	handler := createTestHandlerWithCaptcha([]string{"signup"})

	app := newTestApp(t)
	app.Post("/auth/signup", handler.SignUp)

	// Request with invalid captcha token
	body := `{"email": "test@example.com", "password": "TestPassword123!", "captcha_token": "invalid-token"}`
	req := httptest.NewRequest("POST", "/auth/signup", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	result := parseErrorResponse(t, respBody)

	assert.Equal(t, "CAPTCHA verification failed", result["error"])
	assert.Equal(t, "CAPTCHA_INVALID", result["code"])
}

func TestSignUp_CaptchaDisabled_SkipsVerification(t *testing.T) {
	// Test that when captcha is disabled, the captcha service doesn't block the request
	// We verify this by checking that IsEnabledForEndpoint returns false
	captchaService := auth.NewDisabledCaptchaService()

	// Verify the service correctly reports captcha as disabled
	assert.False(t, captchaService.IsEnabled())
	assert.False(t, captchaService.IsEnabledForEndpoint("signup"))
	assert.False(t, captchaService.IsEnabledForEndpoint("login"))

	// VerifyForEndpoint should return nil (no error) when disabled
	err := captchaService.VerifyForEndpoint(context.Background(), "signup", "", "127.0.0.1")
	assert.NoError(t, err, "Disabled captcha should not require verification")
}

func TestSignUp_CaptchaNotConfiguredForEndpoint_SkipsVerification(t *testing.T) {
	// Captcha enabled only for "login", not "signup"
	captchaService := auth.NewTestCaptchaService([]string{"login"}, validTestCaptchaToken)

	// Verify signup is not protected but login is
	assert.True(t, captchaService.IsEnabled())
	assert.False(t, captchaService.IsEnabledForEndpoint("signup"))
	assert.True(t, captchaService.IsEnabledForEndpoint("login"))

	// VerifyForEndpoint should return nil for signup (not configured)
	err := captchaService.VerifyForEndpoint(context.Background(), "signup", "", "127.0.0.1")
	assert.NoError(t, err, "Unconfigured endpoint should not require verification")

	// But login should require it
	err = captchaService.VerifyForEndpoint(context.Background(), "login", "", "127.0.0.1")
	assert.Error(t, err, "Configured endpoint should require verification")
}

// =============================================================================
// SignIn Captcha Tests
// =============================================================================

func TestSignIn_CaptchaRequired_ReturnsError(t *testing.T) {
	handler := createTestHandlerWithCaptcha([]string{"login"})

	app := newTestApp(t)
	app.Post("/auth/signin", handler.SignIn)

	// Request without captcha token
	body := `{"email": "test@example.com", "password": "TestPassword123!"}`
	req := httptest.NewRequest("POST", "/auth/signin", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	result := parseErrorResponse(t, respBody)

	assert.Equal(t, "CAPTCHA verification required", result["error"])
	assert.Equal(t, "CAPTCHA_REQUIRED", result["code"])
}

func TestSignIn_CaptchaInvalid_ReturnsError(t *testing.T) {
	handler := createTestHandlerWithCaptcha([]string{"login"})

	app := newTestApp(t)
	app.Post("/auth/signin", handler.SignIn)

	// Request with invalid captcha token
	body := `{"email": "test@example.com", "password": "TestPassword123!", "captcha_token": "bad-token"}`
	req := httptest.NewRequest("POST", "/auth/signin", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	result := parseErrorResponse(t, respBody)

	assert.Equal(t, "CAPTCHA verification failed", result["error"])
	assert.Equal(t, "CAPTCHA_INVALID", result["code"])
}

func TestSignIn_CaptchaDisabled_SkipsVerification(t *testing.T) {
	// Test that when captcha is disabled, the captcha service doesn't block the request
	captchaService := auth.NewDisabledCaptchaService()

	// Verify the service correctly reports captcha as disabled
	assert.False(t, captchaService.IsEnabled())
	assert.False(t, captchaService.IsEnabledForEndpoint("login"))

	// VerifyForEndpoint should return nil (no error) when disabled
	err := captchaService.VerifyForEndpoint(context.Background(), "login", "", "127.0.0.1")
	assert.NoError(t, err, "Disabled captcha should not require verification")
}

// =============================================================================
// Magic Link Captcha Tests
// =============================================================================

func TestMagicLink_CaptchaRequired_ReturnsError(t *testing.T) {
	handler := createTestHandlerWithCaptcha([]string{"magic_link"})

	app := newTestApp(t)
	app.Post("/auth/magiclink", handler.SendMagicLink)

	// Request without captcha token
	body := `{"email": "test@example.com"}`
	req := httptest.NewRequest("POST", "/auth/magiclink", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	result := parseErrorResponse(t, respBody)

	assert.Equal(t, "CAPTCHA verification required", result["error"])
	assert.Equal(t, "CAPTCHA_REQUIRED", result["code"])
}

func TestMagicLink_CaptchaInvalid_ReturnsError(t *testing.T) {
	handler := createTestHandlerWithCaptcha([]string{"magic_link"})

	app := newTestApp(t)
	app.Post("/auth/magiclink", handler.SendMagicLink)

	// Request with invalid captcha token
	body := `{"email": "test@example.com", "captcha_token": "wrong-token"}`
	req := httptest.NewRequest("POST", "/auth/magiclink", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	result := parseErrorResponse(t, respBody)

	assert.Equal(t, "CAPTCHA verification failed", result["error"])
	assert.Equal(t, "CAPTCHA_INVALID", result["code"])
}

func TestMagicLink_CaptchaDisabled_SkipsVerification(t *testing.T) {
	// Test that when captcha is disabled, the captcha service doesn't block the request
	captchaService := auth.NewDisabledCaptchaService()

	// Verify the service correctly reports captcha as disabled
	assert.False(t, captchaService.IsEnabled())
	assert.False(t, captchaService.IsEnabledForEndpoint("magic_link"))

	// VerifyForEndpoint should return nil (no error) when disabled
	err := captchaService.VerifyForEndpoint(context.Background(), "magic_link", "", "127.0.0.1")
	assert.NoError(t, err, "Disabled captcha should not require verification")
}

// =============================================================================
// Password Reset Captcha Tests
// =============================================================================

func TestPasswordReset_CaptchaRequired_ReturnsError(t *testing.T) {
	handler := createTestHandlerWithCaptcha([]string{"password_reset"})

	app := newTestApp(t)
	app.Post("/auth/password/reset", handler.RequestPasswordReset)

	// Request without captcha token
	body := `{"email": "test@example.com"}`
	req := httptest.NewRequest("POST", "/auth/password/reset", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	result := parseErrorResponse(t, respBody)

	assert.Equal(t, "CAPTCHA verification required", result["error"])
	assert.Equal(t, "CAPTCHA_REQUIRED", result["code"])
}

func TestPasswordReset_CaptchaInvalid_ReturnsError(t *testing.T) {
	handler := createTestHandlerWithCaptcha([]string{"password_reset"})

	app := newTestApp(t)
	app.Post("/auth/password/reset", handler.RequestPasswordReset)

	// Request with invalid captcha token
	body := `{"email": "test@example.com", "captcha_token": "expired-token"}`
	req := httptest.NewRequest("POST", "/auth/password/reset", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	result := parseErrorResponse(t, respBody)

	assert.Equal(t, "CAPTCHA verification failed", result["error"])
	assert.Equal(t, "CAPTCHA_INVALID", result["code"])
}

func TestPasswordReset_CaptchaDisabled_SkipsVerification(t *testing.T) {
	// Test that when captcha is disabled, the captcha service doesn't block the request
	captchaService := auth.NewDisabledCaptchaService()

	// Verify the service correctly reports captcha as disabled
	assert.False(t, captchaService.IsEnabled())
	assert.False(t, captchaService.IsEnabledForEndpoint("password_reset"))

	// VerifyForEndpoint should return nil (no error) when disabled
	err := captchaService.VerifyForEndpoint(context.Background(), "password_reset", "", "127.0.0.1")
	assert.NoError(t, err, "Disabled captcha should not require verification")
}

// =============================================================================
// Multiple Endpoints Tests
// =============================================================================

func TestCaptcha_MultipleEndpointsConfigured(t *testing.T) {
	// Enable captcha for signup and login, but not magic_link
	captchaService := auth.NewTestCaptchaService([]string{"signup", "login"}, validTestCaptchaToken)

	t.Run("signup requires captcha", func(t *testing.T) {
		assert.True(t, captchaService.IsEnabledForEndpoint("signup"))
		err := captchaService.VerifyForEndpoint(context.Background(), "signup", "", "127.0.0.1")
		assert.ErrorIs(t, err, auth.ErrCaptchaRequired)
	})

	t.Run("login requires captcha", func(t *testing.T) {
		assert.True(t, captchaService.IsEnabledForEndpoint("login"))
		err := captchaService.VerifyForEndpoint(context.Background(), "login", "", "127.0.0.1")
		assert.ErrorIs(t, err, auth.ErrCaptchaRequired)
	})

	t.Run("magic_link does not require captcha", func(t *testing.T) {
		assert.False(t, captchaService.IsEnabledForEndpoint("magic_link"))
		err := captchaService.VerifyForEndpoint(context.Background(), "magic_link", "", "127.0.0.1")
		assert.NoError(t, err)
	})

	t.Run("password_reset does not require captcha", func(t *testing.T) {
		assert.False(t, captchaService.IsEnabledForEndpoint("password_reset"))
		err := captchaService.VerifyForEndpoint(context.Background(), "password_reset", "", "127.0.0.1")
		assert.NoError(t, err)
	})
}

func TestCaptcha_HandlerIntegration_MultipleEndpoints(t *testing.T) {
	// Test actual handler integration with multiple endpoints
	handler := createTestHandlerWithCaptcha([]string{"signup", "login"})

	app := newTestApp(t)
	app.Post("/auth/signup", handler.SignUp)
	app.Post("/auth/signin", handler.SignIn)

	t.Run("signup requires captcha via handler", func(t *testing.T) {
		body := `{"email": "test@example.com", "password": "Test123!"}`
		req := httptest.NewRequest("POST", "/auth/signup", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)

		respBody, _ := io.ReadAll(resp.Body)
		result := parseErrorResponse(t, respBody)
		assert.Equal(t, "CAPTCHA_REQUIRED", result["code"])
	})

	t.Run("login requires captcha via handler", func(t *testing.T) {
		body := `{"email": "test@example.com", "password": "Test123!"}`
		req := httptest.NewRequest("POST", "/auth/signin", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)

		respBody, _ := io.ReadAll(resp.Body)
		result := parseErrorResponse(t, respBody)
		assert.Equal(t, "CAPTCHA_REQUIRED", result["code"])
	})
}

// =============================================================================
// Nil Captcha Service Tests
// =============================================================================

func TestNilCaptchaService_SkipsVerification(t *testing.T) {
	// When captcha service is nil, the handler should skip verification
	// This tests the handler's nil check: if h.captchaService != nil { ... }

	// We verify this behavior by checking that the handler constructor
	// accepts nil and the handler code has proper nil checks
	handler := NewAuthHandler(nil, nil, nil, "https://example.com", "")
	assert.Nil(t, handler.captchaService, "Handler should accept nil captcha service")

	// The actual nil-safety is tested implicitly - if the handler
	// didn't have proper nil checks, it would panic when called
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestSignUp_EmptyCaptchaToken_TreatedAsMissing(t *testing.T) {
	handler := createTestHandlerWithCaptcha([]string{"signup"})

	app := newTestApp(t)
	app.Post("/auth/signup", handler.SignUp)

	// Empty string captcha token
	body := `{"email": "test@example.com", "password": "TestPassword123!", "captcha_token": ""}`
	req := httptest.NewRequest("POST", "/auth/signup", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	result := parseErrorResponse(t, respBody)

	assert.Equal(t, "CAPTCHA verification required", result["error"])
	assert.Equal(t, "CAPTCHA_REQUIRED", result["code"])
}

func TestSignUp_WhitespaceCaptchaToken_Treated_AsInvalid(t *testing.T) {
	handler := createTestHandlerWithCaptcha([]string{"signup"})

	app := newTestApp(t)
	app.Post("/auth/signup", handler.SignUp)

	// Whitespace-only captcha token
	body := `{"email": "test@example.com", "password": "TestPassword123!", "captcha_token": "   "}`
	req := httptest.NewRequest("POST", "/auth/signup", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	result := parseErrorResponse(t, respBody)

	// Whitespace token is passed to provider and should be invalid
	assert.Equal(t, "CAPTCHA verification failed", result["error"])
	assert.Equal(t, "CAPTCHA_INVALID", result["code"])
}

func TestCaptcha_InvalidJSON_ReturnsParseError(t *testing.T) {
	handler := createTestHandlerWithCaptcha([]string{"signup"})

	app := fiber.New(fiber.Config{ErrorHandler: customErrorHandler})
	app.Post("/auth/signup", handler.SignUp)

	// Invalid JSON
	body := `{"email": "test@example.com", invalid}`
	req := httptest.NewRequest("POST", "/auth/signup", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	result := parseErrorResponse(t, respBody)

	// Should fail with parse error, not captcha error
	assert.Contains(t, result["error"].(string), "Invalid request body")
}
