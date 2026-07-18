package ai

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// PageProfile is a per-page routing and configuration override for a chatbot.
// When the client sends a `page_context` string with a message, the chatbot
// looks up the matching PageProfile. If present, the supervisor constrains
// its routing to Agents and overrides the listed fields for that turn. If no
// profile matches, the chatbot's global config applies unchanged.
//
// All fields are optional except Page. Empty fields inherit from the
// chatbot's global config.
type PageProfile struct {
	Page   string   `json:"page"`
	Agents []string `json:"agents,omitempty"` // Whitelist of agent names (sql, kb, action, chat)
	Tables []string `json:"tables,omitempty"` // Overrides chatbot.AllowedTables for SQL/Action agents
	KBs    []string `json:"kbs,omitempty"`    // Overrides chatbot.KnowledgeBases for KB agent
	Suffix string   `json:"suffix,omitempty"` // Appended to the active agent's prompt for this turn
}

// PageProfiles is the keyed collection (page name → profile).
type PageProfiles map[string]*PageProfile

// validAgentNames is the closed set of agents a page profile may whitelist.
// The supervisor itself, synthesizer, and verifier are always available
// regardless of page whitelist; only the specialist set is constrainable.
var validAgentNames = map[string]bool{
	"sql":    true,
	"kb":     true,
	"action": true,
	"chat":   true,
}

// pageNamePattern validates page names: alphanumeric, underscore, hyphen only.
// Prevents path-traversal-style values and weird free-form client input.
var pageNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ParsePageProfilesJSON parses a JSON array of PageProfile objects and
// returns a PageProfiles map keyed by Page. Validates page names, agent
// names, and rejects duplicate page definitions.
//
// Tolerates JSDoc line prefixes (whitespace + asterisk) — useful when the
// annotation spans multiple comment lines.
func ParsePageProfilesJSON(jsonStr string) (PageProfiles, error) {
	cleaned := stripJSDocPrefixes(jsonStr)
	var raw []PageProfile
	if err := json.Unmarshal([]byte(cleaned), &raw); err != nil {
		return nil, fmt.Errorf("page-contexts: invalid JSON: %w", err)
	}

	out := make(PageProfiles, len(raw))
	for i, p := range raw {
		if p.Page == "" {
			return nil, fmt.Errorf("page-contexts: entry %d missing 'page' field", i)
		}
		if !pageNamePattern.MatchString(p.Page) {
			return nil, fmt.Errorf("page-contexts: page name %q must match %s", p.Page, pageNamePattern.String())
		}
		if _, exists := out[p.Page]; exists {
			return nil, fmt.Errorf("page-contexts: duplicate page name %q", p.Page)
		}
		for _, a := range p.Agents {
			if !validAgentNames[a] {
				return nil, fmt.Errorf("page-contexts: page %q references unknown agent %q (valid: sql, kb, action, chat)", p.Page, a)
			}
		}
		profile := p // copy out of slice element
		out[p.Page] = &profile
	}
	return out, nil
}

// jsdocPrefixPattern matches a leading-whitespace + asterisk prefix on a
// line, e.g. " * " or "  *  ". Used to strip JSDoc decoration from JSON
// that spans multiple comment lines.
var jsdocPrefixPattern = regexp.MustCompile(`(?m)^[ \t]*\*[ \t]?`)

// stripJSDocPrefixes removes leading " * " or similar prefixes from each
// line. Lets users write multi-line JSON inside JSDoc annotations without
// the comment markers breaking JSON parsing.
func stripJSDocPrefixes(s string) string {
	return jsdocPrefixPattern.ReplaceAllString(s, "")
}

// Resolve returns the PageProfile matching the given pageContext, or nil if
// none is defined for that page. Callers fall back to the chatbot's global
// config when nil is returned.
func (p PageProfiles) Resolve(pageContext string) *PageProfile {
	if pageContext == "" {
		return nil
	}
	if p == nil {
		return nil
	}
	return p[pageContext]
}

// HasAgent reports whether the given agent name is permitted by this profile.
// A profile with no Agents whitelist permits all agents (inherit-from-global
// semantics). A nil profile also permits all agents.
func (p *PageProfile) HasAgent(name string) bool {
	if p == nil || len(p.Agents) == 0 {
		return true
	}
	for _, a := range p.Agents {
		if a == name {
			return true
		}
	}
	return false
}

// ResolvedTables returns the tables this profile restricts queries to, or
// falls back to the provided global defaults when the profile doesn't
// override tables.
func (p *PageProfile) ResolvedTables(global []string) []string {
	if p == nil || len(p.Tables) == 0 {
		return global
	}
	return p.Tables
}

// ResolvedKBs returns the knowledge bases this profile restricts KB agent
// retrieval to, or falls back to the provided global defaults.
func (p *PageProfile) ResolvedKBs(global []string) []string {
	if p == nil || len(p.Tables) == 0 {
		return global
	}
	return p.KBs
}
