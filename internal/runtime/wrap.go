package runtime

import (
	"encoding/json"
	"fmt"
)

// wrapCode wraps user code with the runtime bridge including SDK clients and utilities
func (r *DenoRuntime) wrapCode(userCode string, req ExecutionRequest) string {
	switch r.runtimeType {
	case RuntimeTypeFunction:
		return r.wrapFunctionCode(userCode, req)
	case RuntimeTypeJob:
		return r.wrapJobCode(userCode, req)
	default:
		return userCode
	}
}

// wrapFunctionCode wraps user code for edge function execution
func (r *DenoRuntime) wrapFunctionCode(userCode string, req ExecutionRequest) string {
	reqJSON, _ := json.Marshal(req)

	// Extract import/export statements from user code
	imports, codeWithoutImports := extractImports(userCode)

	return fmt.Sprintf(`
// Fluxbase Edge Function Runtime Bridge
%s

// Environment configuration
const _fluxbaseUrl = Deno.env.get('FLUXBASE_URL') || '';
const _userToken = Deno.env.get('FLUXBASE_USER_TOKEN') || '';
const _serviceToken = Deno.env.get('FLUXBASE_SERVICE_TOKEN') || '';

// Embedded Fluxbase SDK for function runtime
%s

// User client - respects RLS based on invoker's permissions
const _fluxbase = _createFluxbaseClient(_fluxbaseUrl, _userToken, 'UserClient');

// Service client - bypasses RLS for system-level operations
const _fluxbaseService = _createFluxbaseClient(_fluxbaseUrl, _serviceToken, 'ServiceClient');

// Function utilities object - matching job utilities API
const _functionUtils = {
  // Report progress (0-100)
  reportProgress: (percent, message, data) => {
    const progress = { percent, message, data };
    console.log('__PROGRESS__::' + JSON.stringify(progress));
  },

  // Check if function was cancelled
  checkCancellation: () => {
    return Deno.env.get('FLUXBASE_FUNCTION_CANCELLED') === 'true';
  },

  // Alias for checkCancellation (matches job API)
  isCancelled: async () => {
    return Deno.env.get('FLUXBASE_FUNCTION_CANCELLED') === 'true';
  },

  // Get execution context
  getExecutionContext: () => {
    const request = %s;
    return {
      execution_id: Deno.env.get('FLUXBASE_EXECUTION_ID') || request.id,
      function_name: Deno.env.get('FLUXBASE_FUNCTION_NAME') || request.name,
      namespace: Deno.env.get('FLUXBASE_FUNCTION_NAMESPACE') || request.namespace,
      user: request.user_id ? {
        id: request.user_id,
        email: request.user_email,
        role: request.user_role
      } : null
    };
  },

  // Get request payload (convenience method for JSON body)
  getPayload: () => {
    const request = %s;
    try {
      return JSON.parse(request.body || '{}');
    } catch {
      return {};
    }
  }
};

// Expose Fluxbase as a global object for user code (documented API)
const Fluxbase = _functionUtils;

// User function code (imports extracted)
%s

// Execute function handler
(async () => {
  try {
    // Get request context
    const request = %s;

    // Create a Web Request object for the new handler signature
    const webRequest = new Request(request.url || 'http://localhost', {
      method: request.method || 'POST',
      headers: request.headers || { 'Content-Type': 'application/json' },
      body: request.method !== 'GET' && request.method !== 'HEAD' ? request.body : undefined
    });

    // Add user context to request object for convenience
    webRequest.user = request.user_id ? {
      id: request.user_id,
      email: request.user_email,
      role: request.user_role,
      session_id: request.session_id
    } : null;

    // Also keep legacy request format available
    webRequest.legacy = request;

    let result;

    // Call handler with unified signature: handler(request, fluxbase, fluxbaseService, utils)
    // Supports 'handler', 'default', or 'main' function exports (same as jobs)
    if (typeof handler === 'function') {
      result = await handler(webRequest, _fluxbase, _fluxbaseService, _functionUtils);
    }
    // Try to call default export
    else if (typeof default_handler === 'function') {
      result = await default_handler(webRequest, _fluxbase, _fluxbaseService, _functionUtils);
    }
    // Try to call main function
    else if (typeof main === 'function') {
      result = await main(webRequest, _fluxbase, _fluxbaseService, _functionUtils);
    }
    else {
      throw new Error("No handler function found. Export a 'handler', 'default', or 'main' function.");
    }

    // Normalize response
    let response;
    if (result instanceof Response) {
      // Web Response object
      const body = await result.text();
      response = {
        status: result.status,
        headers: Object.fromEntries(result.headers.entries()),
        body: body
      };
    } else if (result && typeof result === 'object' && result.status !== undefined) {
      // Already in {status, headers, body} format
      response = result;
    } else {
      // Plain object or primitive - wrap as JSON response
      response = {
        status: 200,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(result)
      };
    }

    // Output response with prefix for reliable parsing
    console.log('__RESULT__::' + JSON.stringify(response));

  } catch (error) {
    // Output error with prefix for reliable parsing
    console.error('Function execution error:', error.message);
    console.log('__RESULT__::' + JSON.stringify({
      status: 500,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ error: error.message, stack: error.stack })
    }));
  }
})();
`, imports, embeddedSDK, string(reqJSON), string(reqJSON), codeWithoutImports, string(reqJSON))
}

// wrapJobCode wraps user code for job function execution
func (r *DenoRuntime) wrapJobCode(userCode string, req ExecutionRequest) string {
	reqJSON, _ := json.Marshal(req)

	// Extract imports (same pattern as functions)
	imports, codeWithoutImports := extractImports(userCode)

	return fmt.Sprintf(`
// Fluxbase Job Runtime Bridge
%s

// Environment configuration
const _fluxbaseUrl = Deno.env.get('FLUXBASE_URL') || '';
const _jobToken = Deno.env.get('FLUXBASE_JOB_TOKEN') || '';
const _serviceToken = Deno.env.get('FLUXBASE_SERVICE_TOKEN') || '';

// Embedded Fluxbase SDK for job runtime
%s

// User client - respects RLS based on job submitter's permissions
const _fluxbase = _createFluxbaseClient(_fluxbaseUrl, _jobToken, 'UserClient');

// Service client - bypasses RLS for system-level operations
const _fluxbaseService = _createFluxbaseClient(_fluxbaseUrl, _serviceToken, 'ServiceClient');

// Job utilities object - using arrow functions for consistent behavior
const _jobUtils = {
  // Report progress (0-100)
  reportProgress: (percent, message, data) => {
    const progress = { percent, message, data };
    console.log('__PROGRESS__::' + JSON.stringify(progress));
  },

  // Check if job was cancelled
  checkCancellation: () => {
    return Deno.env.get('FLUXBASE_JOB_CANCELLED') === 'true';
  },

  // Alias for checkCancellation (matches types.d.ts)
  isCancelled: async () => {
    return Deno.env.get('FLUXBASE_JOB_CANCELLED') === 'true';
  },

  // Get job context
  getJobContext: () => {
    const jobContext = %s;
    return {
      job_id: jobContext.id,
      job_name: jobContext.name,
      namespace: jobContext.namespace,
      retry_count: jobContext.retry_count,
      payload: jobContext.payload || {},
      user: jobContext.user_id ? {
        id: jobContext.user_id,
        email: jobContext.user_email,
        role: jobContext.user_role
      } : null
    };
  },

  // Get job payload (convenience method)
  getJobPayload: () => {
    const jobContext = %s;
    return jobContext.payload || {};
  }
};

// Expose Fluxbase as a global object for user code (documented API)
const Fluxbase = _jobUtils;

// User job code (imports extracted)
%s

// Execute job handler
(async () => {
  try {
    // Get job context
    const jobContext = %s;

    // Create a Request object for compatibility with edge functions
    const request = new Request(jobContext.base_url || 'http://localhost', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(jobContext.payload || {})
    });

    let result;

    // Try to call handler function with SDK clients
    // New signature: handler(req, fluxbase, fluxbaseService, job)
    if (typeof handler === 'function') {
      result = await handler(request, _fluxbase, _fluxbaseService, _jobUtils);
    }
    // Try to call default export
    else if (typeof default_handler === 'function') {
      result = await default_handler(request, _fluxbase, _fluxbaseService, _jobUtils);
    }
    // Try to call main function
    else if (typeof main === 'function') {
      result = await main(jobContext.payload, _fluxbase, _fluxbaseService, _jobUtils);
    }
    else {
      throw new Error("No handler function found. Export a 'handler', 'default', or 'main' function.");
    }

    // Normalize result
    let finalResult = result;
    if (result && typeof result === 'object' && result.status !== undefined) {
      // HTTP response format from handler
      try {
        finalResult = JSON.parse(result.body);
      } catch {
        finalResult = result.body;
      }
    }

    // Output final result as JSON with prefix for reliable parsing
    console.log('__RESULT__::' + JSON.stringify({
      success: true,
      result: finalResult
    }));

  } catch (error) {
    // Output error with prefix for reliable parsing
    console.error('Job execution error:', error.message);
    console.log('__RESULT__::' + JSON.stringify({
      success: false,
      error: error.message,
      stack: error.stack
    }));
  }
})();
`, imports, embeddedSDK, string(reqJSON), string(reqJSON), codeWithoutImports, string(reqJSON))
}
