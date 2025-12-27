package runtime

import (
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
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

	// Test with RuntimeTypeFunction
	req := ExecutionRequest{
		ID:        uuid.New(),
		Name:      "test-function",
		Namespace: "default",
	}
	env := buildEnv(req, RuntimeTypeFunction, "http://localhost:8080", "user-token", "service-token", nil, nil)

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
	blockedVarsToCheck := []string{
		"FLUXBASE_AUTH_JWT_SECRET",
		"FLUXBASE_DATABASE_PASSWORD",
		"FLUXBASE_STORAGE_S3_SECRET_KEY",
		"FLUXBASE_EMAIL_SMTP_PASSWORD",
		"FLUXBASE_SECURITY_SETUP_TOKEN",
		"FLUXBASE_DATABASE_ADMIN_PASSWORD",
		"FLUXBASE_STORAGE_S3_ACCESS_KEY",
	}

	for _, key := range blockedVarsToCheck {
		if _, ok := envMap[key]; ok {
			t.Errorf("Expected environment variable %s to be blocked, but it was included", key)
		}
	}

	// Test system variables behavior
	os.Setenv("PATH", "/usr/bin")
	os.Setenv("HOME", "/home/user")
	os.Setenv("RANDOM_VAR", "should-be-excluded")
	defer func() {
		os.Unsetenv("PATH")
		os.Unsetenv("HOME")
		os.Unsetenv("RANDOM_VAR")
	}()

	env = buildEnv(req, RuntimeTypeFunction, "http://localhost:8080", "user-token", "service-token", nil, nil)
	envMap = make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// PATH is intentionally included for subprocess operation (finding executables)
	if envMap["PATH"] != "/usr/bin" {
		t.Errorf("Expected PATH=/usr/bin (for subprocess operation), got PATH=%s", envMap["PATH"])
	}
	// HOME is intentionally set to /tmp for Deno runtime requirements (overrides any existing value)
	if envMap["HOME"] != "/tmp" {
		t.Errorf("Expected HOME=/tmp (for Deno), got HOME=%s", envMap["HOME"])
	}
	// Random non-system, non-FLUXBASE variables should be excluded
	if _, ok := envMap["RANDOM_VAR"]; ok {
		t.Error("Expected RANDOM_VAR to be excluded, but it was included")
	}

	// Test that function-specific variables are included
	if _, ok := envMap["FLUXBASE_EXECUTION_ID"]; !ok {
		t.Error("Expected FLUXBASE_EXECUTION_ID to be included")
	}
	if envMap["FLUXBASE_FUNCTION_NAME"] != "test-function" {
		t.Errorf("Expected FLUXBASE_FUNCTION_NAME=test-function, got %s", envMap["FLUXBASE_FUNCTION_NAME"])
	}
	if envMap["FLUXBASE_USER_TOKEN"] != "user-token" {
		t.Errorf("Expected FLUXBASE_USER_TOKEN=user-token, got %s", envMap["FLUXBASE_USER_TOKEN"])
	}
	if envMap["FLUXBASE_SERVICE_TOKEN"] != "service-token" {
		t.Errorf("Expected FLUXBASE_SERVICE_TOKEN=service-token, got %s", envMap["FLUXBASE_SERVICE_TOKEN"])
	}
}

func TestBuildEnvForJob(t *testing.T) {
	// Test with RuntimeTypeJob
	req := ExecutionRequest{
		ID:        uuid.New(),
		Name:      "test-job",
		Namespace: "default",
	}
	env := buildEnv(req, RuntimeTypeJob, "http://localhost:8080", "job-token", "service-token", nil, nil)

	// Convert to map for easier testing
	envMap := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Test that job-specific variables are included
	if _, ok := envMap["FLUXBASE_JOB_ID"]; !ok {
		t.Error("Expected FLUXBASE_JOB_ID to be included")
	}
	if envMap["FLUXBASE_JOB_NAME"] != "test-job" {
		t.Errorf("Expected FLUXBASE_JOB_NAME=test-job, got %s", envMap["FLUXBASE_JOB_NAME"])
	}
	if envMap["FLUXBASE_JOB_TOKEN"] != "job-token" {
		t.Errorf("Expected FLUXBASE_JOB_TOKEN=job-token, got %s", envMap["FLUXBASE_JOB_TOKEN"])
	}
	if envMap["FLUXBASE_SERVICE_TOKEN"] != "service-token" {
		t.Errorf("Expected FLUXBASE_SERVICE_TOKEN=service-token, got %s", envMap["FLUXBASE_SERVICE_TOKEN"])
	}
}

func TestRuntimeType(t *testing.T) {
	if RuntimeTypeFunction.String() != "function" {
		t.Errorf("Expected RuntimeTypeFunction.String() = 'function', got '%s'", RuntimeTypeFunction.String())
	}
	if RuntimeTypeJob.String() != "job" {
		t.Errorf("Expected RuntimeTypeJob.String() = 'job', got '%s'", RuntimeTypeJob.String())
	}
}

func TestCancelSignal(t *testing.T) {
	signal := NewCancelSignal()

	if signal.IsCancelled() {
		t.Error("Expected new signal to not be cancelled")
	}

	signal.Cancel()

	if !signal.IsCancelled() {
		t.Error("Expected signal to be cancelled after Cancel()")
	}

	// Verify context is done
	select {
	case <-signal.Context().Done():
		// Good, context was cancelled
	default:
		t.Error("Expected context to be done after Cancel()")
	}
}

func TestBuildEnvWithSecrets(t *testing.T) {
	req := ExecutionRequest{
		ID:        uuid.New(),
		Name:      "test-function",
		Namespace: "default",
	}

	t.Run("secrets are injected as FLUXBASE_SECRET_NAME", func(t *testing.T) {
		secrets := map[string]string{
			"API_KEY":     "sk-1234567890",
			"DB_PASSWORD": "supersecret",
			"OAUTH_TOKEN": "oauth-xyz",
		}

		env := buildEnv(req, RuntimeTypeFunction, "http://localhost:8080", "user-token", "service-token", nil, secrets)

		envMap := make(map[string]string)
		for _, e := range env {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				envMap[parts[0]] = parts[1]
			}
		}

		// Check that secrets are injected with correct prefix
		if envMap["FLUXBASE_SECRET_API_KEY"] != "sk-1234567890" {
			t.Errorf("Expected FLUXBASE_SECRET_API_KEY=sk-1234567890, got %s", envMap["FLUXBASE_SECRET_API_KEY"])
		}
		if envMap["FLUXBASE_SECRET_DB_PASSWORD"] != "supersecret" {
			t.Errorf("Expected FLUXBASE_SECRET_DB_PASSWORD=supersecret, got %s", envMap["FLUXBASE_SECRET_DB_PASSWORD"])
		}
		if envMap["FLUXBASE_SECRET_OAUTH_TOKEN"] != "oauth-xyz" {
			t.Errorf("Expected FLUXBASE_SECRET_OAUTH_TOKEN=oauth-xyz, got %s", envMap["FLUXBASE_SECRET_OAUTH_TOKEN"])
		}
	})

	t.Run("secret names are uppercased", func(t *testing.T) {
		secrets := map[string]string{
			"lowercase_key": "value1",
			"MixedCase_Key": "value2",
		}

		env := buildEnv(req, RuntimeTypeFunction, "http://localhost:8080", "user-token", "service-token", nil, secrets)

		envMap := make(map[string]string)
		for _, e := range env {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				envMap[parts[0]] = parts[1]
			}
		}

		// Names should be uppercased
		if envMap["FLUXBASE_SECRET_LOWERCASE_KEY"] != "value1" {
			t.Errorf("Expected FLUXBASE_SECRET_LOWERCASE_KEY=value1, got %s", envMap["FLUXBASE_SECRET_LOWERCASE_KEY"])
		}
		if envMap["FLUXBASE_SECRET_MIXEDCASE_KEY"] != "value2" {
			t.Errorf("Expected FLUXBASE_SECRET_MIXEDCASE_KEY=value2, got %s", envMap["FLUXBASE_SECRET_MIXEDCASE_KEY"])
		}
	})

	t.Run("empty secrets map does not add any secret vars", func(t *testing.T) {
		secrets := map[string]string{}

		env := buildEnv(req, RuntimeTypeFunction, "http://localhost:8080", "user-token", "service-token", nil, secrets)

		for _, e := range env {
			if strings.HasPrefix(e, "FLUXBASE_SECRET_") {
				t.Errorf("Expected no FLUXBASE_SECRET_ vars with empty secrets map, but found: %s", e)
			}
		}
	})

	t.Run("nil secrets map does not add any secret vars", func(t *testing.T) {
		env := buildEnv(req, RuntimeTypeFunction, "http://localhost:8080", "user-token", "service-token", nil, nil)

		for _, e := range env {
			if strings.HasPrefix(e, "FLUXBASE_SECRET_") {
				t.Errorf("Expected no FLUXBASE_SECRET_ vars with nil secrets, but found: %s", e)
			}
		}
	})

	t.Run("secrets work for job runtime type", func(t *testing.T) {
		jobReq := ExecutionRequest{
			ID:        uuid.New(),
			Name:      "test-job",
			Namespace: "default",
		}
		secrets := map[string]string{
			"JOB_SECRET": "job-secret-value",
		}

		env := buildEnv(jobReq, RuntimeTypeJob, "http://localhost:8080", "job-token", "service-token", nil, secrets)

		envMap := make(map[string]string)
		for _, e := range env {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				envMap[parts[0]] = parts[1]
			}
		}

		if envMap["FLUXBASE_SECRET_JOB_SECRET"] != "job-secret-value" {
			t.Errorf("Expected FLUXBASE_SECRET_JOB_SECRET=job-secret-value, got %s", envMap["FLUXBASE_SECRET_JOB_SECRET"])
		}
	})

	t.Run("secrets with special characters in values", func(t *testing.T) {
		secrets := map[string]string{
			"SPECIAL":     "p@$$w0rd!#$%^&*()",
			"JSON_SECRET": `{"key": "value"}`,
			"MULTILINE":   "line1\nline2",
			"EQUALS_SIGN": "key=value=more",
			"UNICODE":     "日本語🔐",
		}

		env := buildEnv(req, RuntimeTypeFunction, "http://localhost:8080", "user-token", "service-token", nil, secrets)

		envMap := make(map[string]string)
		for _, e := range env {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				envMap[parts[0]] = parts[1]
			}
		}

		if envMap["FLUXBASE_SECRET_SPECIAL"] != "p@$$w0rd!#$%^&*()" {
			t.Errorf("Special characters not preserved: got %s", envMap["FLUXBASE_SECRET_SPECIAL"])
		}
		if envMap["FLUXBASE_SECRET_JSON_SECRET"] != `{"key": "value"}` {
			t.Errorf("JSON not preserved: got %s", envMap["FLUXBASE_SECRET_JSON_SECRET"])
		}
		if envMap["FLUXBASE_SECRET_EQUALS_SIGN"] != "key=value=more" {
			t.Errorf("Equals signs not preserved: got %s", envMap["FLUXBASE_SECRET_EQUALS_SIGN"])
		}
		if envMap["FLUXBASE_SECRET_UNICODE"] != "日本語🔐" {
			t.Errorf("Unicode not preserved: got %s", envMap["FLUXBASE_SECRET_UNICODE"])
		}
	})
}

func TestAllowedEnvVarsWithSecrets(t *testing.T) {
	t.Run("function runtime includes secret names", func(t *testing.T) {
		secretNames := []string{"API_KEY", "DB_PASSWORD"}
		allowed := allowedEnvVars(RuntimeTypeFunction, secretNames)

		if !strings.Contains(allowed, "FLUXBASE_SECRET_API_KEY") {
			t.Error("Expected FLUXBASE_SECRET_API_KEY in allowed vars")
		}
		if !strings.Contains(allowed, "FLUXBASE_SECRET_DB_PASSWORD") {
			t.Error("Expected FLUXBASE_SECRET_DB_PASSWORD in allowed vars")
		}
		// Base function vars should still be present
		if !strings.Contains(allowed, "FLUXBASE_URL") {
			t.Error("Expected FLUXBASE_URL in allowed vars")
		}
		if !strings.Contains(allowed, "FLUXBASE_FUNCTION_NAME") {
			t.Error("Expected FLUXBASE_FUNCTION_NAME in allowed vars")
		}
	})

	t.Run("job runtime includes secret names", func(t *testing.T) {
		secretNames := []string{"JOB_SECRET"}
		allowed := allowedEnvVars(RuntimeTypeJob, secretNames)

		if !strings.Contains(allowed, "FLUXBASE_SECRET_JOB_SECRET") {
			t.Error("Expected FLUXBASE_SECRET_JOB_SECRET in allowed vars")
		}
		// Base job vars should still be present
		if !strings.Contains(allowed, "FLUXBASE_URL") {
			t.Error("Expected FLUXBASE_URL in allowed vars")
		}
		if !strings.Contains(allowed, "FLUXBASE_JOB_NAME") {
			t.Error("Expected FLUXBASE_JOB_NAME in allowed vars")
		}
	})

	t.Run("empty secret names does not add secret vars", func(t *testing.T) {
		allowed := allowedEnvVars(RuntimeTypeFunction, []string{})

		if strings.Contains(allowed, "FLUXBASE_SECRET_") {
			t.Error("Expected no FLUXBASE_SECRET_ in allowed vars with empty slice")
		}
	})

	t.Run("nil secret names does not add secret vars", func(t *testing.T) {
		allowed := allowedEnvVars(RuntimeTypeFunction, nil)

		if strings.Contains(allowed, "FLUXBASE_SECRET_") {
			t.Error("Expected no FLUXBASE_SECRET_ in allowed vars with nil slice")
		}
	})

	t.Run("secret names are uppercased in allowed list", func(t *testing.T) {
		secretNames := []string{"lowercase", "MixedCase"}
		allowed := allowedEnvVars(RuntimeTypeFunction, secretNames)

		if !strings.Contains(allowed, "FLUXBASE_SECRET_LOWERCASE") {
			t.Error("Expected FLUXBASE_SECRET_LOWERCASE (uppercased) in allowed vars")
		}
		if !strings.Contains(allowed, "FLUXBASE_SECRET_MIXEDCASE") {
			t.Error("Expected FLUXBASE_SECRET_MIXEDCASE (uppercased) in allowed vars")
		}
	})
}

func TestEncryptionKeyBlocked(t *testing.T) {
	// Verify that FLUXBASE_ENCRYPTION_KEY is in the blocked list
	if !blockedVars["FLUXBASE_ENCRYPTION_KEY"] {
		t.Error("FLUXBASE_ENCRYPTION_KEY should be in blockedVars")
	}

	// Set the encryption key env var
	os.Setenv("FLUXBASE_ENCRYPTION_KEY", "my-secret-encryption-key")
	defer os.Unsetenv("FLUXBASE_ENCRYPTION_KEY")

	req := ExecutionRequest{
		ID:        uuid.New(),
		Name:      "test-function",
		Namespace: "default",
	}

	env := buildEnv(req, RuntimeTypeFunction, "http://localhost:8080", "user-token", "service-token", nil, nil)

	// Check that FLUXBASE_ENCRYPTION_KEY is not in the env
	for _, e := range env {
		if strings.HasPrefix(e, "FLUXBASE_ENCRYPTION_KEY=") {
			t.Error("FLUXBASE_ENCRYPTION_KEY should be blocked from env")
		}
	}
}
