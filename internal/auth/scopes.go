package auth

import (
	"fmt"
	"strings"
)

// Scope constants for API key authorization
const (
	// Tables
	ScopeTablesRead  = "tables:read"
	ScopeTablesWrite = "tables:write"

	// Storage
	ScopeStorageRead  = "storage:read"
	ScopeStorageWrite = "storage:write"

	// Functions
	ScopeFunctionsRead    = "functions:read"
	ScopeFunctionsExecute = "functions:execute"

	// Auth
	ScopeAuthRead  = "auth:read"
	ScopeAuthWrite = "auth:write"

	// Client Keys
	ScopeClientKeysRead  = "clientkeys:read"
	ScopeClientKeysWrite = "clientkeys:write"

	// Webhooks
	ScopeWebhooksRead  = "webhooks:read"
	ScopeWebhooksWrite = "webhooks:write"

	// Monitoring
	ScopeMonitoringRead = "monitoring:read"

	// Realtime
	ScopeRealtimeConnect = "realtime:connect"

	// RPC
	ScopeRPCRead    = "rpc:read"
	ScopeRPCExecute = "rpc:execute"

	// Jobs
	ScopeJobsRead  = "jobs:read"
	ScopeJobsWrite = "jobs:write"

	// AI
	ScopeAIRead  = "ai:read"
	ScopeAIWrite = "ai:write"

	// Secrets
	ScopeSecretsRead  = "secrets:read"
	ScopeSecretsWrite = "secrets:write"

	// Migrations
	ScopeMigrationsRead    = "migrations:read"
	ScopeMigrationsExecute = "migrations:execute"

	// Wildcard scope grants all permissions
	ScopeWildcard = "*"
)

// AllScopes contains all valid scopes (excluding wildcard)
var AllScopes = []string{
	ScopeTablesRead,
	ScopeTablesWrite,
	ScopeStorageRead,
	ScopeStorageWrite,
	ScopeFunctionsRead,
	ScopeFunctionsExecute,
	ScopeAuthRead,
	ScopeAuthWrite,
	ScopeClientKeysRead,
	ScopeClientKeysWrite,
	ScopeWebhooksRead,
	ScopeWebhooksWrite,
	ScopeMonitoringRead,
	ScopeRealtimeConnect,
	ScopeRPCRead,
	ScopeRPCExecute,
	ScopeJobsRead,
	ScopeJobsWrite,
	ScopeAIRead,
	ScopeAIWrite,
	ScopeSecretsRead,
	ScopeSecretsWrite,
	ScopeMigrationsRead,
	ScopeMigrationsExecute,
}

// validScopesMap is a lookup map for O(1) scope validation
var validScopesMap map[string]bool

func init() {
	validScopesMap = make(map[string]bool)
	for _, scope := range AllScopes {
		validScopesMap[scope] = true
	}
	// Wildcard is also valid
	validScopesMap[ScopeWildcard] = true
}

// IsValidScope checks if a single scope is valid
func IsValidScope(scope string) bool {
	return validScopesMap[scope]
}

// ValidateScopes checks if all provided scopes are valid
// Returns an error if any scope is invalid or if no scopes are provided
func ValidateScopes(scopes []string) error {
	if len(scopes) == 0 {
		return fmt.Errorf("at least one scope must be specified")
	}

	var invalidScopes []string
	for _, scope := range scopes {
		if !IsValidScope(scope) {
			invalidScopes = append(invalidScopes, scope)
		}
	}

	if len(invalidScopes) > 0 {
		return fmt.Errorf("invalid scopes: %s", strings.Join(invalidScopes, ", "))
	}

	return nil
}

// HasScope checks if a list of scopes contains the required scope
// Returns true if the scopes contain the required scope or the wildcard scope
func HasScope(scopes []string, required string) bool {
	for _, scope := range scopes {
		if scope == required || scope == ScopeWildcard {
			return true
		}
	}
	return false
}

// HasAllScopes checks if a list of scopes contains all required scopes
// Returns true if all required scopes are present (or if wildcard is present)
func HasAllScopes(scopes []string, required []string) bool {
	for _, req := range required {
		if !HasScope(scopes, req) {
			return false
		}
	}
	return true
}
