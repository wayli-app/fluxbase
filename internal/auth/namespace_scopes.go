package auth

import (
	"strings"
)

// ParseNamespaceScope parses a scope into its component parts.
// Scope format: "action:resource:namespace_pattern"
// Examples:
//   - "*" → ("*", "*", "*")
//   - "execute:functions" → ("execute", "functions", "*")
//   - "execute:functions:prod" → ("execute", "functions", "prod")
//   - "execute:functions:prod-*" → ("execute", "functions", "prod-*")
//
// Returns: action, resource, namespacePattern
func ParseNamespaceScope(scope string) (action, resource, namespacePattern string) {
	// Wildcard scope
	if scope == "*" {
		return "*", "*", "*"
	}

	parts := strings.Split(scope, ":")
	if len(parts) == 0 {
		return "", "", ""
	}

	// action:resource (no namespace = all namespaces)
	if len(parts) == 2 {
		return parts[0], parts[1], "*"
	}

	// action:resource:namespace
	if len(parts) >= 3 {
		namespace := strings.Join(parts[2:], ":") // Handle colons in namespace
		return parts[0], parts[1], namespace
	}

	// Single part (malformed, but handle gracefully)
	return parts[0], "", "*"
}

// MatchNamespacePattern checks if a namespace matches a pattern.
// Patterns supported:
//   - "*" → matches all namespaces
//   - "prod" → exact match only
//   - "prod-*" → prefix match (namespaces starting with "prod-")
//
// Note: Only prefix wildcards are supported to prevent ReDoS attacks.
func MatchNamespacePattern(namespace, pattern string) bool {
	// Wildcard matches everything
	if pattern == "*" {
		return true
	}

	// Exact match
	if pattern == namespace {
		return true
	}

	// Prefix match (ends with *)
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(namespace, prefix)
	}

	return false
}

// HasScopeForNamespace checks if the given scopes grant access to a specific
// resource in a specific namespace.
//
// Parameters:
//   - scopes: List of scope strings (e.g., ["execute:functions:prod", "read:tables"])
//   - action: The action being performed (e.g., "execute", "read", "write")
//   - resource: The resource type (e.g., "functions", "rpc", "jobs")
//   - namespace: The specific namespace to check access for
//
// Returns true if any scope grants access to the resource in the namespace.
func HasScopeForNamespace(scopes []string, action, resource, namespace string) bool {
	for _, scope := range scopes {
		scopeAction, scopeResource, scopeNamespace := ParseNamespaceScope(scope)

		// Check if action matches (or wildcard)
		if scopeAction != "*" && scopeAction != action {
			continue
		}

		// Check if resource matches (or wildcard)
		if scopeResource != "*" && scopeResource != resource {
			continue
		}

		// Check if namespace matches the pattern
		if MatchNamespacePattern(namespace, scopeNamespace) {
			return true
		}
	}

	return false
}

// ExtractAllowedNamespaces derives the list of allowed namespaces from scopes
// for a specific resource type.
//
// Parameters:
//   - scopes: List of scope strings
//   - resource: The resource type to check (e.g., "functions", "rpc", "jobs")
//
// Returns:
//   - nil: All namespaces allowed (wildcard or no restrictions)
//   - []string{}: Only default namespace allowed
//   - []string{"prod", "dev"}: Specific namespaces allowed
//
// Note: This function returns explicit namespace lists, not patterns.
// Wildcard patterns (e.g., "prod-*") in scopes will result in nil (all allowed).
func ExtractAllowedNamespaces(scopes []string, resource string) []string {
	hasWildcard := false
	allowedNamespaces := make(map[string]bool)

	for _, scope := range scopes {
		action, scopeResource, namespacePattern := ParseNamespaceScope(scope)

		// Global wildcard grants all access
		if scope == "*" {
			return nil
		}

		// Check if scope applies to this resource
		// Must be execute/read/write for the resource, or wildcard action
		validAction := action == "execute" || action == "read" || action == "write" || action == "*"
		validResource := scopeResource == resource || scopeResource == "*"

		if !validAction || !validResource {
			continue
		}

		// Wildcard namespace pattern
		if namespacePattern == "*" || strings.HasSuffix(namespacePattern, "*") {
			hasWildcard = true
			break
		}

		// Specific namespace
		allowedNamespaces[namespacePattern] = true
	}

	// If any wildcard found, allow all namespaces
	if hasWildcard {
		return nil
	}

	// No scopes matched this resource
	if len(allowedNamespaces) == 0 {
		// Return empty slice to indicate default-only access
		// (only if there were scopes but none matched)
		if len(scopes) > 0 {
			return []string{}
		}
		// No scopes at all = allow all (backward compatibility)
		return nil
	}

	// Return specific namespace list
	result := make([]string, 0, len(allowedNamespaces))
	for ns := range allowedNamespaces {
		result = append(result, ns)
	}
	return result
}

// IsNamespaceAllowed checks if a namespace is in the allowed list.
// Helper function for filtering operations.
//
// Parameters:
//   - namespace: The namespace to check
//   - allowedNamespaces: The list of allowed namespaces (nil = all allowed)
//
// Returns true if the namespace is allowed.
func IsNamespaceAllowed(namespace string, allowedNamespaces []string) bool {
	// nil = all allowed (no restrictions)
	if allowedNamespaces == nil {
		return true
	}

	// empty array = default namespace only
	if len(allowedNamespaces) == 0 {
		return namespace == "default"
	}

	// Check if namespace is in the allowed list
	for _, allowed := range allowedNamespaces {
		if allowed == namespace || MatchNamespacePattern(namespace, allowed) {
			return true
		}
	}

	return false
}
