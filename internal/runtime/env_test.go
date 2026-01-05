package runtime

import (
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestBuildEnv_BasicEnvironmentSetup(t *testing.T) {
	req := ExecutionRequest{
		ID:        uuid.New(),
		Name:      "test-function",
		Namespace: "default",
	}

	env := buildEnv(req, RuntimeTypeFunction, "", "", "", nil, nil)

	// Verify essential Deno variables are set
	assert.Contains(t, env, "DENO_DIR=/tmp/deno")
	assert.Contains(t, env, "HOME=/tmp")
}

func TestBuildEnv_RuntimeTypeFunction(t *testing.T) {
	executionID := uuid.New()
	req := ExecutionRequest{
		ID:        executionID,
		Name:      "my-function",
		Namespace: "prod",
	}

	userToken := "user-token-123"
	serviceToken := "service-token-456"

	env := buildEnv(req, RuntimeTypeFunction, "https://api.example.com", userToken, serviceToken, nil, nil)

	// Verify function-specific variables
	assert.Contains(t, env, "FLUXBASE_URL=https://api.example.com")
	assert.Contains(t, env, "FLUXBASE_EXECUTION_ID="+executionID.String())
	assert.Contains(t, env, "FLUXBASE_FUNCTION_NAME=my-function")
	assert.Contains(t, env, "FLUXBASE_FUNCTION_NAMESPACE=prod")
	assert.Contains(t, env, "FLUXBASE_USER_TOKEN=user-token-123")
	assert.Contains(t, env, "FLUXBASE_SERVICE_TOKEN=service-token-456")
	assert.Contains(t, env, "FLUXBASE_FUNCTION_CANCELLED=false")
}

func TestBuildEnv_RuntimeTypeJob(t *testing.T) {
	jobID := uuid.New()
	req := ExecutionRequest{
		ID:        jobID,
		Name:      "data-processor",
		Namespace: "staging",
	}

	userToken := "job-token-789"
	serviceToken := "service-token-abc"

	env := buildEnv(req, RuntimeTypeJob, "https://api.example.com", userToken, serviceToken, nil, nil)

	// Verify job-specific variables
	assert.Contains(t, env, "FLUXBASE_URL=https://api.example.com")
	assert.Contains(t, env, "FLUXBASE_JOB_ID="+jobID.String())
	assert.Contains(t, env, "FLUXBASE_JOB_NAME=data-processor")
	assert.Contains(t, env, "FLUXBASE_JOB_NAMESPACE=staging")
	assert.Contains(t, env, "FLUXBASE_JOB_TOKEN=job-token-789")
	assert.Contains(t, env, "FLUXBASE_SERVICE_TOKEN=service-token-abc")
	assert.Contains(t, env, "FLUXBASE_JOB_CANCELLED=false")
}

func TestBuildEnv_CancellationSignal(t *testing.T) {
	req := ExecutionRequest{
		ID:        uuid.New(),
		Name:      "test",
		Namespace: "default",
	}

	t.Run("function not cancelled", func(t *testing.T) {
		cancelSignal := NewCancelSignal()
		env := buildEnv(req, RuntimeTypeFunction, "", "", "", cancelSignal, nil)
		assert.Contains(t, env, "FLUXBASE_FUNCTION_CANCELLED=false")
	})

	t.Run("function cancelled", func(t *testing.T) {
		cancelSignal := NewCancelSignal()
		cancelSignal.Cancel()
		env := buildEnv(req, RuntimeTypeFunction, "", "", "", cancelSignal, nil)
		assert.Contains(t, env, "FLUXBASE_FUNCTION_CANCELLED=true")
	})

	t.Run("job not cancelled", func(t *testing.T) {
		cancelSignal := NewCancelSignal()
		env := buildEnv(req, RuntimeTypeJob, "", "", "", cancelSignal, nil)
		assert.Contains(t, env, "FLUXBASE_JOB_CANCELLED=false")
	})

	t.Run("job cancelled", func(t *testing.T) {
		cancelSignal := NewCancelSignal()
		cancelSignal.Cancel()
		env := buildEnv(req, RuntimeTypeJob, "", "", "", cancelSignal, nil)
		assert.Contains(t, env, "FLUXBASE_JOB_CANCELLED=true")
	})

	t.Run("nil cancel signal", func(t *testing.T) {
		env := buildEnv(req, RuntimeTypeFunction, "", "", "", nil, nil)
		assert.Contains(t, env, "FLUXBASE_FUNCTION_CANCELLED=false")
	})
}

func TestBuildEnv_Secrets(t *testing.T) {
	req := ExecutionRequest{
		ID:        uuid.New(),
		Name:      "test",
		Namespace: "default",
	}

	t.Run("legacy secrets with prefix", func(t *testing.T) {
		secrets := map[string]string{
			"api_key":    "key-123",
			"db_pass":    "pass-456",
			"api_secret": "secret-789",
		}

		env := buildEnv(req, RuntimeTypeFunction, "", "", "", nil, secrets)

		assert.Contains(t, env, "FLUXBASE_SECRET_API_KEY=key-123")
		assert.Contains(t, env, "FLUXBASE_SECRET_DB_PASS=pass-456")
		assert.Contains(t, env, "FLUXBASE_SECRET_API_SECRET=secret-789")
	})

	t.Run("raw FLUXBASE_ secrets", func(t *testing.T) {
		secrets := map[string]string{
			"FLUXBASE_USER_API_KEY":     "user-key",
			"FLUXBASE_SETTING_MAX_SIZE": "1000",
		}

		env := buildEnv(req, RuntimeTypeFunction, "", "", "", nil, secrets)

		assert.Contains(t, env, "FLUXBASE_USER_API_KEY=user-key")
		assert.Contains(t, env, "FLUXBASE_SETTING_MAX_SIZE=1000")
	})

	t.Run("mixed secrets", func(t *testing.T) {
		secrets := map[string]string{
			"api_key":                   "key-123",
			"FLUXBASE_CUSTOM_SETTING":   "value",
			"db_password":               "pass",
			"FLUXBASE_USER_TOKEN_EXTRA": "extra",
		}

		env := buildEnv(req, RuntimeTypeFunction, "", "", "", nil, secrets)

		assert.Contains(t, env, "FLUXBASE_SECRET_API_KEY=key-123")
		assert.Contains(t, env, "FLUXBASE_CUSTOM_SETTING=value")
		assert.Contains(t, env, "FLUXBASE_SECRET_DB_PASSWORD=pass")
		assert.Contains(t, env, "FLUXBASE_USER_TOKEN_EXTRA=extra")
	})

	t.Run("empty secrets", func(t *testing.T) {
		env := buildEnv(req, RuntimeTypeFunction, "", "", "", nil, map[string]string{})

		// Should not contain any FLUXBASE_SECRET_ variables
		for _, e := range env {
			assert.False(t, strings.HasPrefix(e, "FLUXBASE_SECRET_"))
		}
	})

	t.Run("nil secrets", func(t *testing.T) {
		env := buildEnv(req, RuntimeTypeFunction, "", "", "", nil, nil)

		// Should not contain any FLUXBASE_SECRET_ variables
		for _, e := range env {
			assert.False(t, strings.HasPrefix(e, "FLUXBASE_SECRET_"))
		}
	})
}

func TestBuildEnv_BlockedVariables(t *testing.T) {
	req := ExecutionRequest{
		ID:        uuid.New(),
		Name:      "test",
		Namespace: "default",
	}

	// Set blocked variables in the environment
	blockedVars := []string{
		"FLUXBASE_AUTH_JWT_SECRET",
		"FLUXBASE_DATABASE_PASSWORD",
		"FLUXBASE_DATABASE_ADMIN_PASSWORD",
		"FLUXBASE_STORAGE_S3_SECRET_KEY",
		"FLUXBASE_STORAGE_S3_ACCESS_KEY",
		"FLUXBASE_EMAIL_SMTP_PASSWORD",
		"FLUXBASE_SECURITY_SETUP_TOKEN",
		"FLUXBASE_ENCRYPTION_KEY",
	}

	for _, v := range blockedVars {
		t.Setenv(v, "secret-value")
	}

	env := buildEnv(req, RuntimeTypeFunction, "", "", "", nil, nil)

	// Verify blocked variables are not in the environment
	for _, blockedVar := range blockedVars {
		for _, e := range env {
			if strings.HasPrefix(e, blockedVar+"=") {
				t.Errorf("Blocked variable %s should not be in environment", blockedVar)
			}
		}
	}
}

func TestBuildEnv_AllowedFluxbaseVariables(t *testing.T) {
	req := ExecutionRequest{
		ID:        uuid.New(),
		Name:      "test",
		Namespace: "default",
	}

	// Set allowed FLUXBASE_ variables
	t.Setenv("FLUXBASE_DEBUG", "true")
	t.Setenv("FLUXBASE_CUSTOM_VAR", "custom-value")
	t.Setenv("FLUXBASE_TIMEOUT", "30s")

	env := buildEnv(req, RuntimeTypeFunction, "", "", "", nil, nil)

	// Verify allowed variables are passed through
	assert.Contains(t, env, "FLUXBASE_DEBUG=true")
	assert.Contains(t, env, "FLUXBASE_CUSTOM_VAR=custom-value")
	assert.Contains(t, env, "FLUXBASE_TIMEOUT=30s")
}

func TestBuildEnv_SystemVariables(t *testing.T) {
	req := ExecutionRequest{
		ID:        uuid.New(),
		Name:      "test",
		Namespace: "default",
	}

	// Set system variables
	t.Setenv("PATH", "/usr/bin:/bin")
	t.Setenv("SSL_CERT_FILE", "/etc/ssl/certs/ca-certificates.crt")
	t.Setenv("KUBERNETES_SERVICE_HOST", "10.0.0.1")

	env := buildEnv(req, RuntimeTypeFunction, "", "", "", nil, nil)

	// Verify system variables are included
	assert.Contains(t, env, "PATH=/usr/bin:/bin")
	assert.Contains(t, env, "SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt")
	assert.Contains(t, env, "KUBERNETES_SERVICE_HOST=10.0.0.1")
}

func TestBuildEnv_EmptyTokens(t *testing.T) {
	req := ExecutionRequest{
		ID:        uuid.New(),
		Name:      "test",
		Namespace: "default",
	}

	t.Run("function with empty tokens", func(t *testing.T) {
		env := buildEnv(req, RuntimeTypeFunction, "", "", "", nil, nil)

		// Verify token variables are not present when empty
		for _, e := range env {
			assert.False(t, strings.HasPrefix(e, "FLUXBASE_USER_TOKEN="))
			assert.False(t, strings.HasPrefix(e, "FLUXBASE_SERVICE_TOKEN="))
		}
	})

	t.Run("job with empty tokens", func(t *testing.T) {
		env := buildEnv(req, RuntimeTypeJob, "", "", "", nil, nil)

		// Verify token variables are not present when empty
		for _, e := range env {
			assert.False(t, strings.HasPrefix(e, "FLUXBASE_JOB_TOKEN="))
			assert.False(t, strings.HasPrefix(e, "FLUXBASE_SERVICE_TOKEN="))
		}
	})
}

func TestBuildEnv_EmptyPublicURL(t *testing.T) {
	req := ExecutionRequest{
		ID:        uuid.New(),
		Name:      "test",
		Namespace: "default",
	}

	env := buildEnv(req, RuntimeTypeFunction, "", "", "", nil, nil)

	// Verify FLUXBASE_URL is not present when empty
	for _, e := range env {
		assert.False(t, strings.HasPrefix(e, "FLUXBASE_URL="))
	}
}

func TestAllowedEnvVars_Function(t *testing.T) {
	tests := []struct {
		name        string
		secretNames []string
		expected    []string
	}{
		{
			name:        "no secrets",
			secretNames: []string{},
			expected: []string{
				"FLUXBASE_URL",
				"FLUXBASE_USER_TOKEN",
				"FLUXBASE_SERVICE_TOKEN",
				"FLUXBASE_EXECUTION_ID",
				"FLUXBASE_FUNCTION_NAME",
				"FLUXBASE_FUNCTION_NAMESPACE",
				"FLUXBASE_FUNCTION_CANCELLED",
			},
		},
		{
			name:        "legacy secrets",
			secretNames: []string{"api_key", "db_password"},
			expected: []string{
				"FLUXBASE_URL",
				"FLUXBASE_SECRET_API_KEY",
				"FLUXBASE_SECRET_DB_PASSWORD",
			},
		},
		{
			name:        "raw FLUXBASE_ secrets",
			secretNames: []string{"FLUXBASE_USER_KEY", "FLUXBASE_SETTING_VALUE"},
			expected: []string{
				"FLUXBASE_URL",
				"FLUXBASE_USER_KEY",
				"FLUXBASE_SETTING_VALUE",
			},
		},
		{
			name:        "mixed secrets",
			secretNames: []string{"api_key", "FLUXBASE_CUSTOM", "db_pass"},
			expected: []string{
				"FLUXBASE_URL",
				"FLUXBASE_SECRET_API_KEY",
				"FLUXBASE_CUSTOM",
				"FLUXBASE_SECRET_DB_PASS",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := allowedEnvVars(RuntimeTypeFunction, tt.secretNames)

			// Verify all expected variables are present
			for _, exp := range tt.expected {
				assert.Contains(t, result, exp)
			}
		})
	}
}

func TestAllowedEnvVars_Job(t *testing.T) {
	tests := []struct {
		name        string
		secretNames []string
		expected    []string
	}{
		{
			name:        "no secrets",
			secretNames: []string{},
			expected: []string{
				"FLUXBASE_URL",
				"FLUXBASE_JOB_TOKEN",
				"FLUXBASE_SERVICE_TOKEN",
				"FLUXBASE_JOB_ID",
				"FLUXBASE_JOB_NAME",
				"FLUXBASE_JOB_NAMESPACE",
				"FLUXBASE_JOB_CANCELLED",
			},
		},
		{
			name:        "with secrets",
			secretNames: []string{"secret1", "FLUXBASE_RAW"},
			expected: []string{
				"FLUXBASE_URL",
				"FLUXBASE_SECRET_SECRET1",
				"FLUXBASE_RAW",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := allowedEnvVars(RuntimeTypeJob, tt.secretNames)

			// Verify all expected variables are present
			for _, exp := range tt.expected {
				assert.Contains(t, result, exp)
			}
		})
	}
}

func TestAllowedEnvVars_UnknownRuntimeType(t *testing.T) {
	result := allowedEnvVars(RuntimeType(99), []string{"secret"})
	assert.Equal(t, "", result)
}

func TestAllowedEnvVars_EmptySecretNames(t *testing.T) {
	result := allowedEnvVars(RuntimeTypeFunction, nil)

	// Should still have base variables
	assert.Contains(t, result, "FLUXBASE_URL")
	assert.Contains(t, result, "FLUXBASE_USER_TOKEN")

	// Should not contain any FLUXBASE_SECRET_ variables
	assert.NotContains(t, result, "FLUXBASE_SECRET_")
}

func TestAllowedEnvVars_SecretNameFormatting(t *testing.T) {
	secretNames := []string{
		"lowercase_secret",
		"MixedCase_Secret",
		"UPPERCASE_SECRET",
	}

	result := allowedEnvVars(RuntimeTypeFunction, secretNames)

	// All secrets should be uppercased
	assert.Contains(t, result, "FLUXBASE_SECRET_LOWERCASE_SECRET")
	assert.Contains(t, result, "FLUXBASE_SECRET_MIXEDCASE_SECRET")
	assert.Contains(t, result, "FLUXBASE_SECRET_UPPERCASE_SECRET")
}

func TestBuildEnv_Integration(t *testing.T) {
	// Full integration test with all features
	executionID := uuid.New()
	req := ExecutionRequest{
		ID:        executionID,
		Name:      "payment-processor",
		Namespace: "production",
	}

	secrets := map[string]string{
		"stripe_key":            "sk_live_123",
		"FLUXBASE_RATE_LIMIT":   "1000",
		"database_url":          "postgres://...",
		"FLUXBASE_FEATURE_FLAG": "true",
	}

	cancelSignal := NewCancelSignal()

	// Set some environment variables
	t.Setenv("FLUXBASE_DEBUG", "true")
	t.Setenv("FLUXBASE_AUTH_JWT_SECRET", "blocked-secret")
	t.Setenv("PATH", "/usr/local/bin")

	env := buildEnv(
		req,
		RuntimeTypeFunction,
		"https://api.prod.example.com",
		"user-token-xyz",
		"service-token-abc",
		cancelSignal,
		secrets,
	)

	// Verify everything is present and correct
	assert.Contains(t, env, "DENO_DIR=/tmp/deno")
	assert.Contains(t, env, "HOME=/tmp")
	assert.Contains(t, env, "PATH=/usr/local/bin")
	assert.Contains(t, env, "FLUXBASE_DEBUG=true")
	assert.Contains(t, env, "FLUXBASE_URL=https://api.prod.example.com")
	assert.Contains(t, env, "FLUXBASE_EXECUTION_ID="+executionID.String())
	assert.Contains(t, env, "FLUXBASE_FUNCTION_NAME=payment-processor")
	assert.Contains(t, env, "FLUXBASE_FUNCTION_NAMESPACE=production")
	assert.Contains(t, env, "FLUXBASE_USER_TOKEN=user-token-xyz")
	assert.Contains(t, env, "FLUXBASE_SERVICE_TOKEN=service-token-abc")
	assert.Contains(t, env, "FLUXBASE_FUNCTION_CANCELLED=false")
	assert.Contains(t, env, "FLUXBASE_SECRET_STRIPE_KEY=sk_live_123")
	assert.Contains(t, env, "FLUXBASE_RATE_LIMIT=1000")
	assert.Contains(t, env, "FLUXBASE_SECRET_DATABASE_URL=postgres://...")
	assert.Contains(t, env, "FLUXBASE_FEATURE_FLAG=true")

	// Verify blocked variable is NOT present
	for _, e := range env {
		assert.False(t, strings.HasPrefix(e, "FLUXBASE_AUTH_JWT_SECRET="))
	}
}

func TestBuildEnv_SystemVariablesNotSetIfMissing(t *testing.T) {
	req := ExecutionRequest{
		ID:        uuid.New(),
		Name:      "test",
		Namespace: "default",
	}

	// Clear any system variables
	systemVars := []string{
		"SSL_CERT_FILE",
		"SSL_CERT_DIR",
		"CURL_CA_BUNDLE",
		"RESOLV_CONF",
		"LOCALDOMAIN",
		"RES_OPTIONS",
		"HOSTALIASES",
		"KUBERNETES_SERVICE_HOST",
		"KUBERNETES_SERVICE_PORT",
	}

	for _, v := range systemVars {
		os.Unsetenv(v)
	}

	env := buildEnv(req, RuntimeTypeFunction, "", "", "", nil, nil)

	// Verify system variables are not in env when not set
	for _, sysVar := range systemVars {
		for _, e := range env {
			if strings.HasPrefix(e, sysVar+"=") {
				t.Errorf("Unset system variable %s should not be in environment", sysVar)
			}
		}
	}

	// PATH should still be present if it exists
	if path := os.Getenv("PATH"); path != "" {
		assert.Contains(t, env, "PATH="+path)
	}
}
