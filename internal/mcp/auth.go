package mcp

import (
	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/gofiber/fiber/v2"
)

// AuthContext contains authentication information for MCP requests
type AuthContext struct {
	// UserID is the authenticated user's ID (nil for anonymous or service key auth)
	UserID *string

	// UserEmail is the authenticated user's email (empty for service key auth)
	UserEmail string

	// UserRole is the user's role (anon, authenticated, admin, service_role, etc.)
	UserRole string

	// AuthType indicates how the request was authenticated (jwt, clientkey, service_key)
	AuthType string

	// ClientKeyID is the ID of the client key (if authenticated via client key)
	ClientKeyID string

	// ClientKeyName is the name of the client key (if authenticated via client key)
	ClientKeyName string

	// Scopes are the permissions granted (from client key or inferred from role)
	Scopes []string

	// IsServiceRole indicates if this is a service role (bypasses RLS)
	IsServiceRole bool

	// AllowedNamespaces lists the namespaces this context can access.
	// nil = all namespaces allowed (no restrictions)
	// empty slice = default namespace only
	// populated slice = specific namespaces allowed
	AllowedNamespaces []string

	// IsImpersonating indicates if this request is using impersonation
	IsImpersonating bool

	// ImpersonationAdminID is the ID of the admin who initiated impersonation (if IsImpersonating is true)
	ImpersonationAdminID string

	// ImpersonationSessionID is the session ID for tracking impersonation (if IsImpersonating is true)
	ImpersonationSessionID string

	// Metadata contains additional context-specific data that tools may need.
	// This is used to pass chatbot-specific configuration (e.g., HTTPAllowedDomains)
	// to MCP tools without polluting the core auth fields.
	Metadata map[string]any
}

// HasScope checks if the auth context has a specific scope
func (ctx *AuthContext) HasScope(scope string) bool {
	// Service role has all scopes
	if ctx.IsServiceRole {
		return true
	}

	// Check for wildcard scope
	for _, s := range ctx.Scopes {
		if s == "*" {
			return true
		}
	}

	// Check for specific scope
	for _, s := range ctx.Scopes {
		if s == scope {
			return true
		}
	}

	return false
}

// HasScopes checks if the auth context has all specified scopes
func (ctx *AuthContext) HasScopes(scopes ...string) bool {
	for _, scope := range scopes {
		if !ctx.HasScope(scope) {
			return false
		}
	}
	return true
}

// HasAnyScope checks if the auth context has any of the specified scopes
func (ctx *AuthContext) HasAnyScope(scopes ...string) bool {
	for _, scope := range scopes {
		if ctx.HasScope(scope) {
			return true
		}
	}
	return false
}

// IsAuthenticated returns true if the request is authenticated
func (ctx *AuthContext) IsAuthenticated() bool {
	return ctx.UserID != nil || ctx.AuthType == "service_key"
}

// GetMetadata returns a metadata value by key, or nil if not found
func (ctx *AuthContext) GetMetadata(key string) any {
	if ctx.Metadata == nil {
		return nil
	}
	return ctx.Metadata[key]
}

// GetMetadataStringSlice returns a metadata value as a string slice, or nil if not found or wrong type
func (ctx *AuthContext) GetMetadataStringSlice(key string) []string {
	val := ctx.GetMetadata(key)
	if val == nil {
		return nil
	}
	if slice, ok := val.([]string); ok {
		return slice
	}
	return nil
}

// GetMetadataString returns a metadata value as a string, or empty string if not found or wrong type
func (ctx *AuthContext) GetMetadataString(key string) string {
	val := ctx.GetMetadata(key)
	if val == nil {
		return ""
	}
	if str, ok := val.(string); ok {
		return str
	}
	return ""
}

// HasNamespaceAccess checks if the auth context can access a specific namespace.
// Returns true if:
//   - AllowedNamespaces is nil (no restrictions, all namespaces allowed)
//   - The namespace is in the AllowedNamespaces list
//   - AllowedNamespaces is empty and namespace is "default"
//   - IsServiceRole is true (service role bypasses all restrictions)
func (ctx *AuthContext) HasNamespaceAccess(namespace string) bool {
	// Service role bypasses all namespace checks
	if ctx.IsServiceRole {
		return true
	}

	// nil = all namespaces allowed (no restrictions)
	if ctx.AllowedNamespaces == nil {
		return true
	}

	// empty slice = default namespace only
	if len(ctx.AllowedNamespaces) == 0 {
		return namespace == "default"
	}

	// Check if namespace is in the allowed list
	return auth.IsNamespaceAllowed(namespace, ctx.AllowedNamespaces)
}

// FilterNamespaces returns only the namespaces that this context can access.
// If AllowedNamespaces is nil (no restrictions), all input namespaces are returned.
func (ctx *AuthContext) FilterNamespaces(namespaces []string) []string {
	// Service role bypasses all namespace checks
	if ctx.IsServiceRole {
		return namespaces
	}

	// nil = all namespaces allowed (no restrictions)
	if ctx.AllowedNamespaces == nil {
		return namespaces
	}

	// Filter to only allowed namespaces
	filtered := make([]string, 0, len(namespaces))
	for _, ns := range namespaces {
		if ctx.HasNamespaceAccess(ns) {
			filtered = append(filtered, ns)
		}
	}
	return filtered
}

// ExtractAuthContext extracts authentication context from Fiber locals
// This should be called after auth middleware has run
func ExtractAuthContext(c *fiber.Ctx) *AuthContext {
	ctx := &AuthContext{
		Scopes: make([]string, 0),
	}

	// Extract auth type
	if authType := c.Locals("auth_type"); authType != nil {
		ctx.AuthType = authType.(string)
	}

	// Extract user info from JWT auth
	if userID := c.Locals("user_id"); userID != nil {
		switch v := userID.(type) {
		case string:
			ctx.UserID = &v
		case *string:
			ctx.UserID = v
		}
	}

	if userEmail := c.Locals("user_email"); userEmail != nil {
		ctx.UserEmail = userEmail.(string)
	}

	if userRole := c.Locals("user_role"); userRole != nil {
		ctx.UserRole = userRole.(string)
	}

	// Check for service role
	if ctx.UserRole == "service_role" || ctx.AuthType == "service_key" {
		ctx.IsServiceRole = true
	}

	// Extract client key info
	if clientKeyID := c.Locals("client_key_id"); clientKeyID != nil {
		ctx.ClientKeyID = clientKeyID.(string)
	}

	if clientKeyName := c.Locals("client_key_name"); clientKeyName != nil {
		ctx.ClientKeyName = clientKeyName.(string)
	}

	// Extract scopes from client key or service key
	if scopes := c.Locals("client_key_scopes"); scopes != nil {
		if scopeSlice, ok := scopes.([]string); ok {
			ctx.Scopes = scopeSlice
		}
	}

	if scopes := c.Locals("service_key_scopes"); scopes != nil {
		if scopeSlice, ok := scopes.([]string); ok {
			ctx.Scopes = scopeSlice
		}
	}

	// If no explicit scopes but authenticated via JWT, infer default scopes based on role
	if len(ctx.Scopes) == 0 && ctx.AuthType == "jwt" && ctx.UserID != nil {
		ctx.Scopes = inferScopesFromRole(ctx.UserRole)
	}

	// Extract allowed namespaces from Fiber locals (set by middleware)
	if allowedNS := c.Locals("allowed_namespaces"); allowedNS != nil {
		if nsSlice, ok := allowedNS.([]string); ok {
			ctx.AllowedNamespaces = nsSlice
		}
	}

	// If not explicitly set, derive from scopes
	// Note: We don't derive for all resources, just check for general namespace patterns
	if ctx.AllowedNamespaces == nil && len(ctx.Scopes) > 0 {
		// Check if any scopes have namespace restrictions
		hasNamespaceScopes := false
		for _, scope := range ctx.Scopes {
			_, _, nsPattern := auth.ParseNamespaceScope(scope)
			if nsPattern != "*" {
				hasNamespaceScopes = true
				break
			}
		}
		// If namespace-scoped permissions exist but AllowedNamespaces not set,
		// we'll leave it as nil (all allowed) for now. The HasNamespaceAccess
		// method will check scopes dynamically.
		_ = hasNamespaceScopes
	}

	// Extract impersonation context from Fiber locals (if set by handler)
	if isImpersonating := c.Locals("is_impersonating"); isImpersonating != nil {
		if impersonating, ok := isImpersonating.(bool); ok {
			ctx.IsImpersonating = impersonating
		}
	}

	if adminID := c.Locals("impersonation_admin_id"); adminID != nil {
		if id, ok := adminID.(string); ok {
			ctx.ImpersonationAdminID = id
		}
	}

	if sessionID := c.Locals("impersonation_session_id"); sessionID != nil {
		if id, ok := sessionID.(string); ok {
			ctx.ImpersonationSessionID = id
		}
	}

	return ctx
}

// inferScopesFromRole infers default scopes based on user role
// This provides baseline access for JWT-authenticated users without explicit scopes
func inferScopesFromRole(role string) []string {
	switch role {
	case "admin", "dashboard_admin":
		// Admins get full access including DDL operations
		return []string{"*"}
	case "authenticated":
		// Authenticated users get read/write access to most things
		// Note: admin:ddl is NOT included - DDL requires explicit admin role
		return []string{
			"read:tables",
			"write:tables",
			"execute:functions",
			"execute:rpc",
			"read:storage",
			"write:storage",
			"execute:jobs",
		}
	case "anon":
		// Anonymous users get limited read access
		return []string{
			"read:tables",
		}
	default:
		// Unknown roles get minimal access
		return []string{}
	}
}

// Metadata keys for AuthContext.Metadata
const (
	// MetadataKeyHTTPAllowedDomains is the key for allowed domains in AuthContext.Metadata
	MetadataKeyHTTPAllowedDomains = "http_allowed_domains"

	// SQL execution config keys
	MetadataKeyAllowedSchemas    = "allowed_schemas"
	MetadataKeyAllowedTables     = "allowed_tables"
	MetadataKeyAllowedOperations = "allowed_operations"

	// Intent validation config keys (for chatbots)
	MetadataKeyIntentRules     = "intent_rules"
	MetadataKeyRequiredColumns = "required_columns"
	MetadataKeyDefaultTable    = "default_table"

	// Chatbot context keys
	MetadataKeyChatbotID = "chatbot_id"
)

// MCP Scopes
const (
	// Table scopes
	ScopeReadTables  = "read:tables"
	ScopeWriteTables = "write:tables"

	// Function scopes
	ScopeExecuteFunctions = "execute:functions"
	ScopeInvokeFunctions  = "execute:functions" // Alias for execute:functions

	// RPC scopes
	ScopeExecuteRPC = "execute:rpc"
	ScopeInvokeRPC  = "execute:rpc" // Alias for execute:rpc

	// Storage scopes
	ScopeReadStorage  = "read:storage"
	ScopeWriteStorage = "write:storage"

	// Job scopes
	ScopeExecuteJobs = "execute:jobs"
	ScopeSubmitJobs  = "execute:jobs" // Alias for execute:jobs

	// Vector/AI scopes
	ScopeReadVectors   = "read:vectors"
	ScopeSearchVectors = "read:vectors" // Alias for read:vectors

	// HTTP scopes
	ScopeExecuteHTTP = "execute:http" // Make HTTP requests to allowed domains

	// SQL scopes
	ScopeExecuteSQL = "execute:sql" // Execute raw SQL queries (with validation)

	// Schema scopes (for resources)
	ScopeReadSchema = "read:schema"

	// Admin scopes
	ScopeAdminSchemas = "admin:schemas" // Access to internal schemas
	ScopeAdminDDL     = "admin:ddl"     // DDL operations (create/drop tables, etc.)

	// Sync scopes (admin-level code deployment)
	ScopeSyncFunctions  = "sync:functions"  // Create/update edge functions
	ScopeSyncJobs       = "sync:jobs"       // Create/update background jobs
	ScopeSyncRPC        = "sync:rpc"        // Create/update RPC procedures
	ScopeSyncMigrations = "sync:migrations" // Create/apply migrations (DANGEROUS)
	ScopeSyncChatbots   = "sync:chatbots"   // Create/update AI chatbots

	// Branching scopes
	ScopeBranchRead   = "branch:read"   // List and get branch details
	ScopeBranchWrite  = "branch:write"  // Create, delete, reset branches
	ScopeBranchAccess = "branch:access" // Grant/revoke branch access
)
