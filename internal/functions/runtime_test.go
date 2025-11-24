package functions

import (
	"os"
	"strings"
	"testing"
)

func TestBuildEnvForFunction(t *testing.T) {
	// Set up test environment variables
	testVars := map[string]string{
		// Should be included
		"FLUXBASE_BASE_URL":         "http://localhost:8080",
		"FLUXBASE_SERVICE_ROLE_KEY": "test-service-key",
		"FLUXBASE_ANON_KEY":         "test-anon-key",
		"FLUXBASE_DEBUG":            "true",
		// Should be blocked
		"FLUXBASE_AUTH_JWT_SECRET":         "super-secret",
		"FLUXBASE_DATABASE_PASSWORD":       "db-password",
		"FLUXBASE_STORAGE_S3_SECRET_KEY":   "s3-secret",
		"FLUXBASE_EMAIL_SMTP_PASSWORD":     "smtp-password",
		"FLUXBASE_SECURITY_SETUP_TOKEN":    "setup-token",
		"FLUXBASE_DATABASE_ADMIN_PASSWORD": "admin-password",
		"FLUXBASE_STORAGE_S3_ACCESS_KEY":   "s3-access-key",
	}

	// Set environment variables
	for key, value := range testVars {
		os.Setenv(key, value)
	}
	defer func() {
		// Clean up
		for key := range testVars {
			os.Unsetenv(key)
		}
	}()

	// Call buildEnvForFunction
	env := buildEnvForFunction()

	// Convert to map for easier testing
	envMap := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Test that allowed variables are included
	allowedVars := []string{
		"FLUXBASE_BASE_URL",
		"FLUXBASE_SERVICE_ROLE_KEY",
		"FLUXBASE_ANON_KEY",
		"FLUXBASE_DEBUG",
	}

	for _, key := range allowedVars {
		if value, ok := envMap[key]; !ok {
			t.Errorf("Expected environment variable %s to be included, but it was not", key)
		} else if value != testVars[key] {
			t.Errorf("Expected %s=%s, got %s=%s", key, testVars[key], key, value)
		}
	}

	// Test that blocked variables are excluded
	blockedVars := []string{
		"FLUXBASE_AUTH_JWT_SECRET",
		"FLUXBASE_DATABASE_PASSWORD",
		"FLUXBASE_STORAGE_S3_SECRET_KEY",
		"FLUXBASE_EMAIL_SMTP_PASSWORD",
		"FLUXBASE_SECURITY_SETUP_TOKEN",
		"FLUXBASE_DATABASE_ADMIN_PASSWORD",
		"FLUXBASE_STORAGE_S3_ACCESS_KEY",
	}

	for _, key := range blockedVars {
		if _, ok := envMap[key]; ok {
			t.Errorf("Expected environment variable %s to be blocked, but it was included", key)
		}
	}

	// Test that non-FLUXBASE variables are not included
	os.Setenv("PATH", "/usr/bin")
	os.Setenv("HOME", "/home/user")
	defer func() {
		os.Unsetenv("PATH")
		os.Unsetenv("HOME")
	}()

	env = buildEnvForFunction()
	envMap = make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	if _, ok := envMap["PATH"]; ok {
		t.Error("Expected PATH to be excluded, but it was included")
	}
	if _, ok := envMap["HOME"]; ok {
		t.Error("Expected HOME to be excluded, but it was included")
	}
}
