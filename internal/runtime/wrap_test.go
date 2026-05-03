package runtime

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// wrapCode Tests
// =============================================================================

func TestWrapCode_FunctionType(t *testing.T) {
	r := NewRuntime(RuntimeTypeFunction, "secret", "http://localhost:8080")
	req := ExecutionRequest{
		ID:        uuid.New(),
		Name:      "test-function",
		Namespace: "default",
	}
	userCode := `export function handler(req) { return new Response("Hello"); }`

	result := r.wrapCode(userCode, req)

	assert.Contains(t, result, "Fluxbase Edge Function Runtime Bridge")
	assert.Contains(t, result, "_fluxbase")
	assert.Contains(t, result, "_fluxbaseService")
	assert.Contains(t, result, "_functionUtils")
	assert.Contains(t, result, "..._functionUtils")
	assert.Contains(t, result, "_tenantUtils")
}

func TestWrapCode_JobType(t *testing.T) {
	r := NewRuntime(RuntimeTypeJob, "secret", "http://localhost:8080")
	req := ExecutionRequest{
		ID:        uuid.New(),
		Name:      "test-job",
		Namespace: "default",
	}
	userCode := `export function handler(req) { return { success: true }; }`

	result := r.wrapCode(userCode, req)

	assert.Contains(t, result, "Fluxbase Job Runtime Bridge")
	assert.Contains(t, result, "_fluxbase")
	assert.Contains(t, result, "_fluxbaseService")
	assert.Contains(t, result, "_jobUtils")
	assert.Contains(t, result, "..._jobUtils")
	assert.Contains(t, result, "_tenantUtils")
}

func TestWrapCode_UnknownType(t *testing.T) {
	// Test default case - unknown runtime type returns user code unchanged
	r := &DenoRuntime{
		runtimeType: RuntimeType(99), // Unknown type
	}
	req := ExecutionRequest{
		ID:        uuid.New(),
		Name:      "test",
		Namespace: "default",
	}
	userCode := `const x = 1;`

	result := r.wrapCode(userCode, req)

	assert.Equal(t, userCode, result)
}

// =============================================================================
// wrapFunctionCode Tests
// =============================================================================

func TestWrapFunctionCode_BasicStructure(t *testing.T) {
	r := NewRuntime(RuntimeTypeFunction, "secret", "http://localhost:8080")
	req := ExecutionRequest{
		ID:        uuid.New(),
		Name:      "my-function",
		Namespace: "default",
	}
	userCode := `export function handler(req) { return new Response("Hello"); }`

	result := r.wrapFunctionCode(userCode, req)

	// Check for required runtime components
	t.Run("contains environment setup", func(t *testing.T) {
		assert.Contains(t, result, "_fluxbaseUrl")
		assert.Contains(t, result, "_userToken")
		assert.Contains(t, result, "_serviceToken")
	})

	t.Run("contains secrets helper", func(t *testing.T) {
		assert.Contains(t, result, "const secrets = {")
		assert.Contains(t, result, "_normalize(key)")
		assert.Contains(t, result, "getUser(key)")
		assert.Contains(t, result, "getSystem(key)")
		assert.Contains(t, result, "getRequired(key)")
	})

	t.Run("contains SDK clients", func(t *testing.T) {
		assert.Contains(t, result, "_createFluxbaseClient")
		assert.Contains(t, result, "UserClient")
		assert.Contains(t, result, "ServiceClient")
	})

	t.Run("contains function utilities", func(t *testing.T) {
		assert.Contains(t, result, "reportProgress")
		assert.Contains(t, result, "checkCancellation")
		assert.Contains(t, result, "isCancelled")
		assert.Contains(t, result, "getExecutionContext")
		assert.Contains(t, result, "getPayload")
	})

	t.Run("contains AI utilities", func(t *testing.T) {
		assert.Contains(t, result, "ai: {")
		assert.Contains(t, result, "async chat(options)")
		assert.Contains(t, result, "async embed(options)")
		assert.Contains(t, result, "async listProviders()")
	})

	t.Run("contains handler execution", func(t *testing.T) {
		assert.Contains(t, result, "typeof handler === 'function'")
		assert.Contains(t, result, "typeof default_handler === 'function'")
		assert.Contains(t, result, "typeof main === 'function'")
	})

	t.Run("contains result handling", func(t *testing.T) {
		assert.Contains(t, result, "__RESULT__::")
		assert.Contains(t, result, "Deno.exit(0)")
		assert.Contains(t, result, "Deno.exit(1)")
	})
}

func TestWrapFunctionCode_ImportsExtracted(t *testing.T) {
	r := NewRuntime(RuntimeTypeFunction, "secret", "http://localhost:8080")
	req := ExecutionRequest{
		ID:        uuid.New(),
		Name:      "test-function",
		Namespace: "default",
	}
	userCode := `import { something } from 'module';
export function handler(req) { return new Response("Hello"); }`

	result := r.wrapFunctionCode(userCode, req)

	// Import should appear at the top
	lines := strings.Split(result, "\n")
	foundImportSection := false
	for _, line := range lines[:10] { // Check first 10 lines
		if strings.Contains(line, "import { something }") {
			foundImportSection = true
			break
		}
	}
	assert.True(t, foundImportSection, "Import should be near the top of wrapped code")

	// User code section should be separate
	assert.Contains(t, result, "User function code")
}

func TestWrapFunctionCode_RequestJSONEmbedded(t *testing.T) {
	r := NewRuntime(RuntimeTypeFunction, "secret", "http://localhost:8080")
	req := ExecutionRequest{
		ID:        uuid.New(),
		Name:      "test-function",
		Namespace: "default",
		UserID:    "user-123",
		UserEmail: "user@example.com",
		UserRole:  "authenticated",
	}
	userCode := `export function handler(req) { return new Response("Hello"); }`

	result := r.wrapFunctionCode(userCode, req)

	// Check that request context is embedded
	assert.Contains(t, result, "test-function")
	assert.Contains(t, result, req.ID.String())
}

// =============================================================================
// wrapJobCode Tests
// =============================================================================

func TestWrapJobCode_BasicStructure(t *testing.T) {
	r := NewRuntime(RuntimeTypeJob, "secret", "http://localhost:8080")
	req := ExecutionRequest{
		ID:        uuid.New(),
		Name:      "my-job",
		Namespace: "default",
	}
	userCode := `export function handler(req) { return { success: true }; }`

	result := r.wrapJobCode(userCode, req)

	t.Run("contains environment setup", func(t *testing.T) {
		assert.Contains(t, result, "_fluxbaseUrl")
		assert.Contains(t, result, "_jobToken")
		assert.Contains(t, result, "_serviceToken")
	})

	t.Run("contains secrets helper", func(t *testing.T) {
		assert.Contains(t, result, "const secrets = {")
	})

	t.Run("contains SDK clients", func(t *testing.T) {
		assert.Contains(t, result, "_createFluxbaseClient")
	})

	t.Run("contains job utilities", func(t *testing.T) {
		assert.Contains(t, result, "reportProgress")
		assert.Contains(t, result, "checkCancellation")
		assert.Contains(t, result, "isCancelled")
		assert.Contains(t, result, "getJobContext")
		assert.Contains(t, result, "getJobPayload")
	})

	t.Run("contains AI utilities", func(t *testing.T) {
		assert.Contains(t, result, "ai: {")
		assert.Contains(t, result, "async chat(options)")
		assert.Contains(t, result, "async embed(options)")
	})

	t.Run("contains handler execution", func(t *testing.T) {
		assert.Contains(t, result, "typeof handler === 'function'")
		assert.Contains(t, result, "typeof default_handler === 'function'")
		assert.Contains(t, result, "typeof main === 'function'")
	})

	t.Run("contains result handling", func(t *testing.T) {
		assert.Contains(t, result, "__RESULT__::")
		assert.Contains(t, result, "success: true")
		assert.Contains(t, result, "success: false")
	})
}

func TestWrapJobCode_ImportsExtracted(t *testing.T) {
	r := NewRuntime(RuntimeTypeJob, "secret", "http://localhost:8080")
	req := ExecutionRequest{
		ID:        uuid.New(),
		Name:      "test-job",
		Namespace: "default",
	}
	userCode := `import { z } from 'zod';
export function handler(req) { return { success: true }; }`

	result := r.wrapJobCode(userCode, req)

	// Import should appear at the top
	lines := strings.Split(result, "\n")
	foundImportSection := false
	for _, line := range lines[:10] {
		if strings.Contains(line, "import { z }") {
			foundImportSection = true
			break
		}
	}
	assert.True(t, foundImportSection, "Import should be near the top of wrapped code")

	assert.Contains(t, result, "User job code")
}

func TestWrapJobCode_JobContextEmbedded(t *testing.T) {
	r := NewRuntime(RuntimeTypeJob, "secret", "http://localhost:8080")
	req := ExecutionRequest{
		ID:         uuid.New(),
		Name:       "test-job",
		Namespace:  "default",
		RetryCount: 3,
		Payload: map[string]interface{}{
			"key": "value",
		},
	}
	userCode := `export function handler(req) { return { success: true }; }`

	result := r.wrapJobCode(userCode, req)

	// Check that job context is embedded
	assert.Contains(t, result, "test-job")
	assert.Contains(t, result, req.ID.String())
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestWrapCode_ProgressReporting(t *testing.T) {
	t.Run("function progress prefix", func(t *testing.T) {
		r := NewRuntime(RuntimeTypeFunction, "secret", "http://localhost:8080")
		req := ExecutionRequest{
			ID:        uuid.New(),
			Name:      "test",
			Namespace: "default",
		}
		result := r.wrapCode("const x = 1;", req)
		assert.Contains(t, result, "__PROGRESS__::")
	})

	t.Run("job progress prefix", func(t *testing.T) {
		r := NewRuntime(RuntimeTypeJob, "secret", "http://localhost:8080")
		req := ExecutionRequest{
			ID:        uuid.New(),
			Name:      "test",
			Namespace: "default",
		}
		result := r.wrapCode("const x = 1;", req)
		assert.Contains(t, result, "__PROGRESS__::")
	})
}

func TestWrapCode_CancellationCheck(t *testing.T) {
	t.Run("function cancellation env var", func(t *testing.T) {
		r := NewRuntime(RuntimeTypeFunction, "secret", "http://localhost:8080")
		req := ExecutionRequest{
			ID:        uuid.New(),
			Name:      "test",
			Namespace: "default",
		}
		result := r.wrapCode("const x = 1;", req)
		assert.Contains(t, result, "FLUXBASE_FUNCTION_CANCELLED")
	})

	t.Run("job cancellation env var", func(t *testing.T) {
		r := NewRuntime(RuntimeTypeJob, "secret", "http://localhost:8080")
		req := ExecutionRequest{
			ID:        uuid.New(),
			Name:      "test",
			Namespace: "default",
		}
		result := r.wrapCode("const x = 1;", req)
		assert.Contains(t, result, "FLUXBASE_JOB_CANCELLED")
	})
}

func TestWrapCode_TenantScopedServiceClient(t *testing.T) {
	tenantID := uuid.New().String()

	t.Run("function service client includes X-FB-Tenant header when TenantID set", func(t *testing.T) {
		r := NewRuntime(RuntimeTypeFunction, "secret", "http://localhost:8080")
		req := ExecutionRequest{
			ID:        uuid.New(),
			Name:      "test-function",
			Namespace: "default",
			TenantID:  tenantID,
		}
		result := r.wrapFunctionCode("const x = 1;", req)

		assert.Contains(t, result, "X-FB-Tenant")
		assert.Contains(t, result, tenantID)
	})

	t.Run("job service client includes X-FB-Tenant header when TenantID set", func(t *testing.T) {
		r := NewRuntime(RuntimeTypeJob, "secret", "http://localhost:8080")
		req := ExecutionRequest{
			ID:        uuid.New(),
			Name:      "test-job",
			Namespace: "default",
			TenantID:  tenantID,
		}
		result := r.wrapJobCode("const x = 1;", req)

		assert.Contains(t, result, "X-FB-Tenant")
		assert.Contains(t, result, tenantID)
	})

	t.Run("function _fluxbaseService is null when no TenantID", func(t *testing.T) {
		r := NewRuntime(RuntimeTypeFunction, "secret", "http://localhost:8080")
		req := ExecutionRequest{
			ID:        uuid.New(),
			Name:      "test-function",
			Namespace: "default",
		}
		result := r.wrapFunctionCode("const x = 1;", req)

		assert.Contains(t, result, "_fluxbaseService = _tenantId")
		assert.Contains(t, result, "? _createFluxbaseClient")
		assert.Contains(t, result, ": null")
	})

	t.Run("forTenant and forCurrentTenant removed from _tenantUtils", func(t *testing.T) {
		r := NewRuntime(RuntimeTypeFunction, "secret", "http://localhost:8080")
		req := ExecutionRequest{
			ID:        uuid.New(),
			Name:      "test-function",
			Namespace: "default",
			TenantID:  tenantID,
		}
		result := r.wrapFunctionCode("const x = 1;", req)

		assert.NotContains(t, result, "forCurrentTenant()")
		assert.NotContains(t, result, "const _tenantUtils = {\n  // Current tenant from execution context")
	})

	t.Run("tenant utilities still expose getCurrentTenantId and hasTenantContext", func(t *testing.T) {
		r := NewRuntime(RuntimeTypeFunction, "secret", "http://localhost:8080")
		req := ExecutionRequest{
			ID:        uuid.New(),
			Name:      "test-function",
			Namespace: "default",
			TenantID:  tenantID,
		}
		result := r.wrapFunctionCode("const x = 1;", req)

		assert.Contains(t, result, "getCurrentTenantId()")
		assert.Contains(t, result, "hasTenantContext()")
	})
}

// stringPtr is a helper function for creating string pointers.
// Note: Currently unused but kept for potential future use.
/*
func stringPtr(s string) *string {
	return &s
}
*/
