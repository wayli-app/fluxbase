package mcp

import (
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

	return ctx
}

// inferScopesFromRole infers default scopes based on user role
// This provides baseline access for JWT-authenticated users without explicit scopes
func inferScopesFromRole(role string) []string {
	switch role {
	case "admin", "dashboard_admin":
		// Admins get full access
		return []string{"*"}
	case "authenticated":
		// Authenticated users get read access to most things
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

	// Schema scopes (for resources)
	ScopeReadSchema = "read:schema"
)
