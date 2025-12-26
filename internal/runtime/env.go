package runtime

import (
	"fmt"
	"os"
	"strings"
)

// blockedVars are environment variables that should never be exposed to user code
var blockedVars = map[string]bool{
	"FLUXBASE_AUTH_JWT_SECRET":         true,
	"FLUXBASE_DATABASE_PASSWORD":       true,
	"FLUXBASE_DATABASE_ADMIN_PASSWORD": true,
	"FLUXBASE_STORAGE_S3_SECRET_KEY":   true,
	"FLUXBASE_STORAGE_S3_ACCESS_KEY":   true,
	"FLUXBASE_EMAIL_SMTP_PASSWORD":     true,
	"FLUXBASE_SECURITY_SETUP_TOKEN":    true,
}

// buildEnv creates the environment variable list for execution
func buildEnv(req ExecutionRequest, runtimeType RuntimeType, publicURL, userToken, serviceToken string, cancelSignal *CancelSignal) []string {
	env := []string{}

	// Deno requires HOME or DENO_DIR to determine its cache directory.
	// Set DENO_DIR to a temp directory to avoid permission issues.
	// Also set HOME as some Deno internals may still look for it.
	env = append(env, "DENO_DIR=/tmp/deno")
	env = append(env, "HOME=/tmp")

	// Include essential system environment variables for proper subprocess operation
	// These are needed for DNS resolution, SSL certificates, and basic system functionality
	systemVars := []string{
		"PATH",                    // Required for finding executables
		"SSL_CERT_FILE",           // SSL certificate file location
		"SSL_CERT_DIR",            // SSL certificate directory
		"CURL_CA_BUNDLE",          // CA bundle for curl/fetch
		"RESOLV_CONF",             // DNS resolver configuration
		"LOCALDOMAIN",             // Local domain for DNS
		"RES_OPTIONS",             // Resolver options
		"HOSTALIASES",             // Host aliases file
		"KUBERNETES_SERVICE_HOST", // Kubernetes API server (for DNS)
		"KUBERNETES_SERVICE_PORT", // Kubernetes API port
	}

	for _, key := range systemVars {
		if val := os.Getenv(key); val != "" {
			env = append(env, fmt.Sprintf("%s=%s", key, val))
		}
	}

	// Pass all FLUXBASE_* environment variables except blocked ones
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "FLUXBASE_") {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				key := parts[0]
				if !blockedVars[key] {
					env = append(env, e)
				}
			}
		}
	}

	// Add SDK client credentials
	if publicURL != "" {
		env = append(env, fmt.Sprintf("FLUXBASE_URL=%s", publicURL))
	}

	// Add execution-specific environment variables based on runtime type
	switch runtimeType {
	case RuntimeTypeFunction:
		env = append(env, fmt.Sprintf("FLUXBASE_EXECUTION_ID=%s", req.ID))
		env = append(env, fmt.Sprintf("FLUXBASE_FUNCTION_NAME=%s", req.Name))
		env = append(env, fmt.Sprintf("FLUXBASE_FUNCTION_NAMESPACE=%s", req.Namespace))

		if userToken != "" {
			env = append(env, fmt.Sprintf("FLUXBASE_USER_TOKEN=%s", userToken))
		}
		if serviceToken != "" {
			env = append(env, fmt.Sprintf("FLUXBASE_SERVICE_TOKEN=%s", serviceToken))
		}

		// Add cancellation signal
		if cancelSignal != nil && cancelSignal.IsCancelled() {
			env = append(env, "FLUXBASE_FUNCTION_CANCELLED=true")
		} else {
			env = append(env, "FLUXBASE_FUNCTION_CANCELLED=false")
		}

	case RuntimeTypeJob:
		env = append(env, fmt.Sprintf("FLUXBASE_JOB_ID=%s", req.ID))
		env = append(env, fmt.Sprintf("FLUXBASE_JOB_NAME=%s", req.Name))
		env = append(env, fmt.Sprintf("FLUXBASE_JOB_NAMESPACE=%s", req.Namespace))

		if userToken != "" {
			env = append(env, fmt.Sprintf("FLUXBASE_JOB_TOKEN=%s", userToken))
		}
		if serviceToken != "" {
			env = append(env, fmt.Sprintf("FLUXBASE_SERVICE_TOKEN=%s", serviceToken))
		}

		// Add cancellation signal
		if cancelSignal != nil && cancelSignal.IsCancelled() {
			env = append(env, "FLUXBASE_JOB_CANCELLED=true")
		} else {
			env = append(env, "FLUXBASE_JOB_CANCELLED=false")
		}
	}

	return env
}

// allowedEnvVars returns the list of allowed environment variables for Deno permissions
func allowedEnvVars(runtimeType RuntimeType) string {
	switch runtimeType {
	case RuntimeTypeFunction:
		return "FLUXBASE_URL,FLUXBASE_USER_TOKEN,FLUXBASE_SERVICE_TOKEN,FLUXBASE_EXECUTION_ID,FLUXBASE_FUNCTION_NAME,FLUXBASE_FUNCTION_NAMESPACE,FLUXBASE_FUNCTION_CANCELLED"
	case RuntimeTypeJob:
		return "FLUXBASE_URL,FLUXBASE_JOB_TOKEN,FLUXBASE_SERVICE_TOKEN,FLUXBASE_JOB_ID,FLUXBASE_JOB_NAME,FLUXBASE_JOB_NAMESPACE,FLUXBASE_JOB_CANCELLED"
	default:
		return ""
	}
}
