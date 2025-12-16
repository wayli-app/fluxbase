package ai

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// IntentRule defines a mapping between user keywords and required/forbidden tables
type IntentRule struct {
	Keywords       []string `json:"keywords"`
	RequiredTable  string   `json:"requiredTable,omitempty"`
	ForbiddenTable string   `json:"forbiddenTable,omitempty"`
}

// RequiredColumnsMap maps table names to required column lists
type RequiredColumnsMap map[string][]string

// Chatbot represents an AI chatbot definition
type Chatbot struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Description  string `json:"description,omitempty"`
	Code         string `json:"code"`          // Full code content
	OriginalCode string `json:"original_code"` // Code before bundling
	IsBundled    bool   `json:"is_bundled"`
	BundleError  string `json:"bundle_error,omitempty"`

	// Parsed from annotations
	AllowedTables      []string `json:"allowed_tables"`
	AllowedOperations  []string `json:"allowed_operations"`
	AllowedSchemas     []string `json:"allowed_schemas"`
	HTTPAllowedDomains []string `json:"http_allowed_domains"`

	// Intent validation (parsed from annotations)
	IntentRules     []IntentRule       `json:"intent_rules,omitempty"`
	RequiredColumns RequiredColumnsMap `json:"required_columns,omitempty"`
	DefaultTable    string             `json:"default_table,omitempty"`

	// Runtime config
	Enabled     bool    `json:"enabled"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
	Model       string  `json:"model,omitempty"` // Model name from @fluxbase:model annotation
	ProviderID  *string `json:"provider_id,omitempty"`

	// Conversation config
	PersistConversations bool `json:"persist_conversations"`
	ConversationTTLHours int  `json:"conversation_ttl_hours"`
	MaxConversationTurns int  `json:"max_conversation_turns"`

	// Rate limiting (per user, per chatbot)
	RateLimitPerMinute int `json:"rate_limit_per_minute"`
	DailyRequestLimit  int `json:"daily_request_limit"`
	DailyTokenBudget   int `json:"daily_token_budget"`

	// Access control
	AllowUnauthenticated bool `json:"allow_unauthenticated"`
	IsPublic             bool `json:"is_public"`

	Version   int       `json:"version"`
	Source    string    `json:"source"` // "filesystem" or "api"
	CreatedBy *string   `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ChatbotConfig represents the parsed configuration from annotations
type ChatbotConfig struct {
	// Data access
	AllowedTables      []string
	AllowedOperations  []string
	AllowedSchemas     []string
	HTTPAllowedDomains []string

	// Intent validation
	IntentRules     []IntentRule
	RequiredColumns RequiredColumnsMap
	DefaultTable    string

	// Model settings
	MaxTokens   int
	Temperature float64
	Model       string

	// Conversation settings
	PersistConversations bool
	ConversationTTL      time.Duration
	MaxTurns             int

	// Rate limiting
	RateLimitPerMinute int
	DailyRequestLimit  int
	DailyTokenBudget   int

	// Access control
	AllowUnauthenticated bool
	IsPublic             bool

	// Metadata
	Version int
}

// DefaultChatbotConfig returns the default configuration for a chatbot
func DefaultChatbotConfig() ChatbotConfig {
	return ChatbotConfig{
		AllowedTables:        []string{},
		AllowedOperations:    []string{"SELECT"},
		AllowedSchemas:       []string{"public"},
		HTTPAllowedDomains:   []string{},
		MaxTokens:            4096,
		Temperature:          0.7,
		PersistConversations: false,
		ConversationTTL:      24 * time.Hour,
		MaxTurns:             50,
		RateLimitPerMinute:   20,
		DailyRequestLimit:    500,
		DailyTokenBudget:     100000,
		AllowUnauthenticated: false,
		IsPublic:             true,
		Version:              1,
	}
}

// Annotation patterns for parsing chatbot configuration
var (
	// @fluxbase:allowed-tables tracker_data,locations,venues
	allowedTablesPattern = regexp.MustCompile(`@fluxbase:allowed-tables\s+([^\n*]+)`)

	// @fluxbase:allowed-operations SELECT,INSERT
	allowedOperationsPattern = regexp.MustCompile(`@fluxbase:allowed-operations\s+([^\n*]+)`)

	// @fluxbase:allowed-schemas public,app
	allowedSchemasPattern = regexp.MustCompile(`@fluxbase:allowed-schemas\s+([^\n*]+)`)

	// @fluxbase:http-allowed-domains pelias.wayli.app,api.example.com
	httpAllowedDomainsPattern = regexp.MustCompile(`@fluxbase:http-allowed-domains\s+([^\n*]+)`)

	// @fluxbase:max-tokens 4096
	maxTokensPattern = regexp.MustCompile(`@fluxbase:max-tokens\s+(\d+)`)

	// @fluxbase:temperature 0.7
	temperaturePattern = regexp.MustCompile(`@fluxbase:temperature\s+([\d.]+)`)

	// @fluxbase:model gpt-4-turbo
	modelPattern = regexp.MustCompile(`@fluxbase:model\s+([^\n*]+)`)

	// @fluxbase:persist-conversations true
	persistConversationsPattern = regexp.MustCompile(`@fluxbase:persist-conversations\s+(true|false)`)

	// @fluxbase:conversation-ttl 24h
	conversationTTLPattern = regexp.MustCompile(`@fluxbase:conversation-ttl\s+([^\n*]+)`)

	// @fluxbase:max-turns 50
	maxTurnsPattern = regexp.MustCompile(`@fluxbase:max-turns\s+(\d+)`)

	// @fluxbase:rate-limit 20/min
	rateLimitPattern = regexp.MustCompile(`@fluxbase:rate-limit\s+(\d+)/min`)

	// @fluxbase:daily-limit 500
	dailyLimitPattern = regexp.MustCompile(`@fluxbase:daily-limit\s+(\d+)`)

	// @fluxbase:token-budget 100000/day
	tokenBudgetPattern = regexp.MustCompile(`@fluxbase:token-budget\s+(\d+)/day`)

	// @fluxbase:allow-unauthenticated true
	allowUnauthenticatedPattern = regexp.MustCompile(`@fluxbase:allow-unauthenticated\s+(true|false)`)

	// @fluxbase:public false
	publicPattern = regexp.MustCompile(`@fluxbase:public\s+(true|false)`)

	// @fluxbase:version 2
	versionPattern = regexp.MustCompile(`@fluxbase:version\s+(\d+)`)

	// Extract description from JSDoc: first line after /**
	descriptionPattern = regexp.MustCompile(`/\*\*\s*\n\s*\*\s*([^\n@]+)`)

	// Extract system prompt from export default
	systemPromptPattern = regexp.MustCompile("(?s)export\\s+default\\s+`([^`]+)`")

	// @fluxbase:intent-rules [{"keywords":["restaurant"],"requiredTable":"my_places"}]
	// Note: We just match the start and use extractBalancedJSON for the full array
	intentRulesPattern = regexp.MustCompile(`@fluxbase:intent-rules\s+(\[)`)

	// @fluxbase:required-columns my_trips=id,title,image_url
	requiredColumnsPattern = regexp.MustCompile(`@fluxbase:required-columns\s+([^\n*]+)`)

	// @fluxbase:default-table my_place_visits
	defaultTablePattern = regexp.MustCompile(`@fluxbase:default-table\s+([^\n*\s]+)`)
)

// ParseChatbotConfig parses chatbot configuration from TypeScript source code
func ParseChatbotConfig(code string) ChatbotConfig {
	config := DefaultChatbotConfig()

	// Parse allowed tables
	if matches := allowedTablesPattern.FindStringSubmatch(code); len(matches) > 1 {
		config.AllowedTables = parseCSV(matches[1])
	}

	// Parse allowed operations
	if matches := allowedOperationsPattern.FindStringSubmatch(code); len(matches) > 1 {
		config.AllowedOperations = parseCSV(matches[1])
	}

	// Parse allowed schemas
	if matches := allowedSchemasPattern.FindStringSubmatch(code); len(matches) > 1 {
		config.AllowedSchemas = parseCSV(matches[1])
	}

	// Parse HTTP allowed domains
	if matches := httpAllowedDomainsPattern.FindStringSubmatch(code); len(matches) > 1 {
		config.HTTPAllowedDomains = parseCSV(matches[1])
	}

	// Parse max tokens
	if matches := maxTokensPattern.FindStringSubmatch(code); len(matches) > 1 {
		if v, err := strconv.Atoi(matches[1]); err == nil {
			config.MaxTokens = v
		}
	}

	// Parse temperature
	if matches := temperaturePattern.FindStringSubmatch(code); len(matches) > 1 {
		if v, err := strconv.ParseFloat(matches[1], 64); err == nil {
			config.Temperature = v
		}
	}

	// Parse model
	if matches := modelPattern.FindStringSubmatch(code); len(matches) > 1 {
		config.Model = strings.TrimSpace(matches[1])
	}

	// Parse persist conversations
	if matches := persistConversationsPattern.FindStringSubmatch(code); len(matches) > 1 {
		config.PersistConversations = matches[1] == "true"
	}

	// Parse conversation TTL
	if matches := conversationTTLPattern.FindStringSubmatch(code); len(matches) > 1 {
		if d, err := time.ParseDuration(strings.TrimSpace(matches[1])); err == nil {
			config.ConversationTTL = d
		}
	}

	// Parse max turns
	if matches := maxTurnsPattern.FindStringSubmatch(code); len(matches) > 1 {
		if v, err := strconv.Atoi(matches[1]); err == nil {
			config.MaxTurns = v
		}
	}

	// Parse rate limit
	if matches := rateLimitPattern.FindStringSubmatch(code); len(matches) > 1 {
		if v, err := strconv.Atoi(matches[1]); err == nil {
			config.RateLimitPerMinute = v
		}
	}

	// Parse daily limit
	if matches := dailyLimitPattern.FindStringSubmatch(code); len(matches) > 1 {
		if v, err := strconv.Atoi(matches[1]); err == nil {
			config.DailyRequestLimit = v
		}
	}

	// Parse token budget
	if matches := tokenBudgetPattern.FindStringSubmatch(code); len(matches) > 1 {
		if v, err := strconv.Atoi(matches[1]); err == nil {
			config.DailyTokenBudget = v
		}
	}

	// Parse allow unauthenticated
	if matches := allowUnauthenticatedPattern.FindStringSubmatch(code); len(matches) > 1 {
		config.AllowUnauthenticated = matches[1] == "true"
	}

	// Parse public
	if matches := publicPattern.FindStringSubmatch(code); len(matches) > 1 {
		config.IsPublic = matches[1] == "true"
	}

	// Parse version
	if matches := versionPattern.FindStringSubmatch(code); len(matches) > 1 {
		if v, err := strconv.Atoi(matches[1]); err == nil && v > 0 {
			config.Version = v
		}
	}

	// Parse intent rules (JSON array) - supports multiple annotations, merges all rules
	allIntentLocs := intentRulesPattern.FindAllStringIndex(code, -1)
	for _, loc := range allIntentLocs {
		// Find the opening bracket position
		bracketIdx := strings.Index(code[loc[0]:], "[")
		if bracketIdx >= 0 {
			jsonStr := extractBalancedJSON(code, loc[0]+bracketIdx)
			if jsonStr != "" {
				var rules []IntentRule
				if err := json.Unmarshal([]byte(jsonStr), &rules); err == nil {
					config.IntentRules = append(config.IntentRules, rules...)
				}
			}
		}
	}

	// Parse required columns (format: table1=col1,col2 table2=col1,col2,col3)
	// Supports multiple annotations, merges all column requirements
	allColMatches := requiredColumnsPattern.FindAllStringSubmatch(code, -1)
	for _, matches := range allColMatches {
		if len(matches) > 1 {
			parsed := parseRequiredColumns(matches[1])
			if len(parsed) > 0 {
				if config.RequiredColumns == nil {
					config.RequiredColumns = make(RequiredColumnsMap)
				}
				for table, cols := range parsed {
					config.RequiredColumns[table] = cols
				}
			}
		}
	}

	// Parse default table
	if matches := defaultTablePattern.FindStringSubmatch(code); len(matches) > 1 {
		config.DefaultTable = strings.TrimSpace(matches[1])
	}

	return config
}

// ParseDescription extracts the chatbot description from JSDoc comments
func ParseDescription(code string) string {
	if matches := descriptionPattern.FindStringSubmatch(code); len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// ParseSystemPrompt extracts the system prompt from the export default template literal
func ParseSystemPrompt(code string) string {
	if matches := systemPromptPattern.FindStringSubmatch(code); len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// parseCSV parses a comma-separated list of values
func parseCSV(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// parseRequiredColumns parses "table1=col1,col2 table2=col3,col4" format
func parseRequiredColumns(s string) RequiredColumnsMap {
	result := make(RequiredColumnsMap)

	// Split by whitespace to get individual table=columns pairs
	pairs := strings.Fields(s)
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			tableName := strings.TrimSpace(parts[0])
			columns := parseCSV(parts[1])
			if tableName != "" && len(columns) > 0 {
				result[tableName] = columns
			}
		}
	}

	return result
}

// extractBalancedJSON extracts a balanced JSON array starting from the given position
// startIdx should point to the opening bracket '['
func extractBalancedJSON(s string, startIdx int) string {
	if startIdx >= len(s) || s[startIdx] != '[' {
		return ""
	}

	depth := 0
	inString := false
	escaped := false

	for i := startIdx; i < len(s); i++ {
		c := s[i]

		if escaped {
			escaped = false
			continue
		}

		if c == '\\' && inString {
			escaped = true
			continue
		}

		if c == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		if c == '[' {
			depth++
		} else if c == ']' {
			depth--
			if depth == 0 {
				return s[startIdx : i+1]
			}
		}
	}

	return "" // Unbalanced
}

// ApplyConfig applies a ChatbotConfig to a Chatbot
func (c *Chatbot) ApplyConfig(config ChatbotConfig) {
	c.AllowedTables = config.AllowedTables
	c.AllowedOperations = config.AllowedOperations
	c.AllowedSchemas = config.AllowedSchemas
	c.HTTPAllowedDomains = config.HTTPAllowedDomains
	c.IntentRules = config.IntentRules
	c.RequiredColumns = config.RequiredColumns
	c.DefaultTable = config.DefaultTable
	c.MaxTokens = config.MaxTokens
	c.Temperature = config.Temperature
	c.Model = config.Model
	c.PersistConversations = config.PersistConversations
	c.ConversationTTLHours = int(config.ConversationTTL.Hours())
	c.MaxConversationTurns = config.MaxTurns
	c.RateLimitPerMinute = config.RateLimitPerMinute
	c.DailyRequestLimit = config.DailyRequestLimit
	c.DailyTokenBudget = config.DailyTokenBudget
	c.AllowUnauthenticated = config.AllowUnauthenticated
	c.IsPublic = config.IsPublic

	// Only override version if explicitly set in annotation
	if config.Version > 0 {
		c.Version = config.Version
	}
}

// ChatbotSummary represents a lightweight chatbot summary for listing
type ChatbotSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	Description string `json:"description,omitempty"`
	Model       string `json:"model,omitempty"`
	Enabled     bool   `json:"enabled"`
	IsPublic    bool   `json:"is_public"`
	Source      string `json:"source"`
	UpdatedAt   string `json:"updated_at"`
}

// ToSummary converts a Chatbot to a ChatbotSummary
func (c *Chatbot) ToSummary() ChatbotSummary {
	return ChatbotSummary{
		ID:          c.ID,
		Name:        c.Name,
		Namespace:   c.Namespace,
		Description: c.Description,
		Model:       c.Model,
		Enabled:     c.Enabled,
		IsPublic:    c.IsPublic,
		Source:      c.Source,
		UpdatedAt:   c.UpdatedAt.Format(time.RFC3339),
	}
}

// PopulateDerivedFields populates fields that are parsed from code but not stored in DB
// This should be called after loading a chatbot from the database
func (c *Chatbot) PopulateDerivedFields() {
	// Parse model from code if not already set
	if c.Model == "" && c.Code != "" {
		if matches := modelPattern.FindStringSubmatch(c.Code); len(matches) > 1 {
			c.Model = strings.TrimSpace(matches[1])
		}
	}
}
