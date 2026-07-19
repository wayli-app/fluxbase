package ai

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// IntentRule defines a mapping between user keywords and required/forbidden tables/tools
type IntentRule struct {
	Keywords       []string `json:"keywords"`
	RequiredTable  string   `json:"requiredTable,omitempty"`
	ForbiddenTable string   `json:"forbiddenTable,omitempty"`
	RequiredTool   string   `json:"requiredTool,omitempty"`
	ForbiddenTool  string   `json:"forbiddenTool,omitempty"`
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
	AllowUnauthenticated bool     `json:"allow_unauthenticated"`
	IsPublic             bool     `json:"is_public"`
	RequireRoles         []string `json:"require_roles,omitempty"` // Required roles to access (OR semantics)

	// Response language
	ResponseLanguage string `json:"response_language"` // "auto" (default), ISO code, or language name

	// Logging
	DisableExecutionLogs bool `json:"disable_execution_logs"` // If true, skip creating execution logs

	// Settings
	RequiredSettings []string `json:"required_settings,omitempty"` // Setting keys this chatbot requires

	// MCP integration
	MCPTools     []string `json:"mcp_tools,omitempty"`      // Allowed MCP tools (e.g., query_table, insert_record)
	UseMCPSchema bool     `json:"use_mcp_schema,omitempty"` // If true, fetch schema from MCP resources

	// RAG/Knowledge Base settings
	KnowledgeBases         []string `json:"knowledge_bases,omitempty"`
	RAGMaxChunks           int      `json:"rag_max_chunks"`
	RAGSimilarityThreshold float64  `json:"rag_similarity_threshold"`
	RAGGraphBoostWeight    float64  `json:"rag_graph_boost_weight,omitempty"` // 0=off (default), 1=fully entity-driven
	RAGTable               string   `json:"rag_table,omitempty"`              // User table for vector search
	RAGColumn              string   `json:"rag_column,omitempty"`             // Vector column in RAG table
	RAGContentColumn       string   `json:"rag_content_column,omitempty"`     // Text content column in RAG table

	// Agent behavior settings
	ReasoningMode     string       `json:"reasoning_mode,omitempty"`      // "supervisor" (default), "react", "strict", "none"
	MaxToolIterations int          `json:"max_tool_iterations,omitempty"` // Max tool calling iterations (default: 5)
	ShowReasoning     bool         `json:"show_reasoning,omitempty"`      // If true, expose agent reasoning to users
	PageProfiles      PageProfiles `json:"page_profiles,omitempty"`       // Per-page routing/config overrides (Level 2 page-aware chatbots)
	// SupervisorAgentModels optionally overrides the model used per specialist
	// agent in supervisor mode. Keys are agent names: "supervisor", "sql",
	// "kb", "action", "chat", "verifier". Missing keys fall
	// back to the chatbot's main Model.
	SupervisorAgentModels map[string]string `json:"supervisor_agent_models,omitempty"`

	// WebSearchEnabled turns on the Web Agent specialist. Requires a
	// configured Tavily integration at the tenant/instance level.
	WebSearchEnabled bool `json:"web_search_enabled,omitempty"`
	// WebSearchDomains optionally restricts the Web Agent to results from
	// these domains only. Empty = no restriction.
	WebSearchDomains []string `json:"web_search_domains,omitempty"`

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
	RequireRoles         []string // Required roles to access (OR semantics)

	// RAG/Knowledge Base settings
	KnowledgeBases         []string // Knowledge base names to link
	RAGMaxChunks           int      // Max chunks to retrieve per query
	RAGSimilarityThreshold float64  // Minimum similarity score (0-1)
	RAGGraphBoostWeight    float64  // Graph-boost weight (0=off, 1=fully entity-driven); falls back to global config
	RAGTable               string   // User table for vector search (optional)
	RAGColumn              string   // Vector column in RAG table
	RAGContentColumn       string   // Text content column in RAG table

	// Response language
	ResponseLanguage string // "auto" (default), ISO code, or language name

	// Logging
	DisableExecutionLogs bool

	// Settings
	RequiredSettings []string // Setting keys this chatbot requires

	// MCP integration
	MCPTools     []string // Allowed MCP tools (e.g., query_table, insert_record)
	UseMCPSchema bool     // If true, fetch schema from MCP resources

	// Agent behavior settings
	ReasoningMode         string            // "supervisor" (default), "react", "strict", "none"
	MaxToolIterations     int               // Max tool calling iterations (default: 5)
	ShowReasoning         bool              // If true, expose agent reasoning to users
	PageProfiles          PageProfiles      // Per-page routing/config overrides
	SupervisorAgentModels map[string]string // Optional per-agent model overrides in supervisor mode

	// Web Search (Web Agent specialist). Off by default. When enabled
	// and a Tavily integration exists at the tenant/instance level, the
	// supervisor routes current-info questions to the Web Agent.
	WebSearchEnabled bool     // parsed from @fluxbase:web-search
	WebSearchDomains []string // parsed from @fluxbase:web-search-domains (optional allowlist)

	// Metadata
	Version int
}

// DefaultChatbotConfig returns the default configuration for a chatbot
func DefaultChatbotConfig() ChatbotConfig {
	return ChatbotConfig{
		AllowedTables:          []string{},
		AllowedOperations:      []string{"SELECT"},
		AllowedSchemas:         []string{"public"},
		HTTPAllowedDomains:     []string{},
		MaxTokens:              4096,
		Temperature:            0.7,
		PersistConversations:   false,
		ConversationTTL:        24 * time.Hour,
		MaxTurns:               50,
		RateLimitPerMinute:     20,
		DailyRequestLimit:      500,
		DailyTokenBudget:       100000,
		AllowUnauthenticated:   false,
		IsPublic:               true,
		KnowledgeBases:         []string{},
		RAGMaxChunks:           5,
		RAGSimilarityThreshold: 0.7,
		ResponseLanguage:       "auto",
		ReasoningMode:          "supervisor", // Default: multi-agent supervisor pipeline. Pin "react" for the legacy ReAct loop.
		MaxToolIterations:      5,
		ShowReasoning:          false,
		Version:                1,
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

	// @fluxbase:http-allowed-domains api.fake-domain.com,api.fake-example.com
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

	// RAG/Knowledge Base annotations
	// @fluxbase:knowledge-base support-docs,faq-base
	knowledgeBasePattern = regexp.MustCompile(`@fluxbase:knowledge-base\s+([^\n*]+)`)

	// @fluxbase:rag-max-chunks 5
	ragMaxChunksPattern = regexp.MustCompile(`@fluxbase:rag-max-chunks\s+(\d+)`)

	// @fluxbase:rag-similarity-threshold 0.7
	ragThresholdPattern = regexp.MustCompile(`@fluxbase:rag-similarity-threshold\s+([\d.]+)`)

	// @fluxbase:rag-graph-boost-weight 0.3 (0=off, 1=fully entity-driven)
	ragGraphBoostPattern = regexp.MustCompile(`@fluxbase:rag-graph-boost-weight\s+([\d.]+)`)

	// @fluxbase:rag-table documents (for user-table RAG)
	ragTablePattern = regexp.MustCompile(`@fluxbase:rag-table\s+([^\n*\s]+)`)

	// @fluxbase:rag-column embedding (vector column in rag-table)
	ragColumnPattern = regexp.MustCompile(`@fluxbase:rag-column\s+([^\n*\s]+)`)

	// @fluxbase:rag-content-column content (text column to retrieve)
	ragContentColumnPattern = regexp.MustCompile(`@fluxbase:rag-content-column\s+([^\n*\s]+)`)

	// @fluxbase:response-language auto | en | German | Deutsch
	responseLanguagePattern = regexp.MustCompile(`@fluxbase:response-language\s+([^\n*]+)`)

	// @fluxbase:disable-execution-logs true
	disableExecutionLogsPattern = regexp.MustCompile(`@fluxbase:disable-execution-logs(?:\s+(true|false))?`)

	// @fluxbase:required-settings pelias.endpoint,google.maps.api_key
	requiredSettingsPattern = regexp.MustCompile(`@fluxbase:required-settings\s+([^\n*]+)`)

	// MCP integration annotations
	// @fluxbase:mcp-tools query_table,insert_record,invoke_function
	mcpToolsPattern = regexp.MustCompile(`@fluxbase:mcp-tools\s+([^\n*]+)`)

	// @fluxbase:use-mcp-schema (or @fluxbase:use-mcp-schema true)
	useMCPSchemaPattern = regexp.MustCompile(`@fluxbase:use-mcp-schema(?:\s+(true|false))?`)

	// Agent behavior annotations
	// @fluxbase:reasoning-mode supervisor|react|strict|none
	reasoningModePattern = regexp.MustCompile(`@fluxbase:reasoning-mode\s+(supervisor|react|strict|none)`)

	// @fluxbase:page-contexts [{...}]
	// JSON array of page profiles. See PageProfile struct in page_context.go.
	pageContextsPattern = regexp.MustCompile(`@fluxbase:page-contexts\s+(\[)`)

	// @fluxbase:supervisor-models {"sql":"gpt-4-turbo","chat":"gpt-4o-mini"}
	// Optional per-agent model overrides for supervisor mode.
	supervisorModelsPattern = regexp.MustCompile(`@fluxbase:supervisor-models\s+(\{)`)

	// @fluxbase:max-iterations 10
	maxToolIterationsPattern = regexp.MustCompile(`@fluxbase:max-iterations\s+(\d+)`)

	// @fluxbase:show-reasoning true
	showReasoningPattern = regexp.MustCompile(`@fluxbase:show-reasoning\s+(true|false)`)

	// @fluxbase:web-search enabled=true
	// Enables the Web Agent specialist for this chatbot. The supervisor
	// will route current-info questions to it when a Tavily integration
	// is configured at the tenant/instance level.
	webSearchPattern = regexp.MustCompile(`@fluxbase:web-search\s+(enabled|disabled|true|false)`)

	// @fluxbase:web-search-domains wikipedia.org,mdn.dev
	// Optional allowlist for web search results. Domains not in this list
	// are filtered out by the Web Agent before passing results to the LLM.
	webSearchDomainsPattern = regexp.MustCompile(`@fluxbase:web-search-domains\s+(.+)`)
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

	// Parse RAG/Knowledge Base annotations
	if matches := knowledgeBasePattern.FindStringSubmatch(code); len(matches) > 1 {
		config.KnowledgeBases = parseCSV(matches[1])
	}

	// Parse RAG max chunks
	if matches := ragMaxChunksPattern.FindStringSubmatch(code); len(matches) > 1 {
		if v, err := strconv.Atoi(matches[1]); err == nil && v > 0 {
			config.RAGMaxChunks = v
		}
	}

	// Parse RAG similarity threshold
	if matches := ragThresholdPattern.FindStringSubmatch(code); len(matches) > 1 {
		if v, err := strconv.ParseFloat(matches[1], 64); err == nil && v >= 0 && v <= 1 {
			config.RAGSimilarityThreshold = v
		}
	}

	// Parse RAG graph boost weight
	if matches := ragGraphBoostPattern.FindStringSubmatch(code); len(matches) > 1 {
		if v, err := strconv.ParseFloat(matches[1], 64); err == nil && v >= 0 && v <= 1 {
			config.RAGGraphBoostWeight = v
		}
	}

	// Parse RAG table (user-table RAG)
	if matches := ragTablePattern.FindStringSubmatch(code); len(matches) > 1 {
		config.RAGTable = strings.TrimSpace(matches[1])
	}

	// Parse RAG column
	if matches := ragColumnPattern.FindStringSubmatch(code); len(matches) > 1 {
		config.RAGColumn = strings.TrimSpace(matches[1])
	}

	// Parse RAG content column
	if matches := ragContentColumnPattern.FindStringSubmatch(code); len(matches) > 1 {
		config.RAGContentColumn = strings.TrimSpace(matches[1])
	}

	// Parse response language
	if matches := responseLanguagePattern.FindStringSubmatch(code); len(matches) > 1 {
		config.ResponseLanguage = strings.TrimSpace(matches[1])
	}

	// Parse disable-execution-logs flag
	if matches := disableExecutionLogsPattern.FindStringSubmatch(code); matches != nil {
		// If no value specified or value is "true", disable logs
		if len(matches) <= 1 || matches[1] == "" || matches[1] == "true" {
			config.DisableExecutionLogs = true
		}
	}

	// Parse required settings
	if matches := requiredSettingsPattern.FindStringSubmatch(code); len(matches) > 1 {
		config.RequiredSettings = parseCSV(matches[1])
	}

	// Parse MCP tools
	if matches := mcpToolsPattern.FindStringSubmatch(code); len(matches) > 1 {
		config.MCPTools = parseCSV(matches[1])
	}

	// Parse use-mcp-schema flag
	if matches := useMCPSchemaPattern.FindStringSubmatch(code); matches != nil {
		// If no value specified or value is "true", enable MCP schema
		if len(matches) <= 1 || matches[1] == "" || matches[1] == "true" {
			config.UseMCPSchema = true
		}
	}

	// Parse reasoning mode
	if matches := reasoningModePattern.FindStringSubmatch(code); len(matches) > 1 {
		config.ReasoningMode = matches[1]
	}

	// Parse page contexts (JSON array of PageProfile objects)
	// Supports multiple annotations — later ones merge with earlier ones (last wins per page name).
	allPageLocs := pageContextsPattern.FindAllStringIndex(code, -1)
	for _, loc := range allPageLocs {
		bracketIdx := strings.Index(code[loc[0]:], "[")
		if bracketIdx >= 0 {
			jsonStr := extractBalancedJSON(code, loc[0]+bracketIdx)
			if jsonStr != "" {
				profiles, err := ParsePageProfilesJSON(jsonStr)
				if err != nil {
					// ponytail: skip malformed block, don't fail the whole parse
					continue
				}
				if config.PageProfiles == nil {
					config.PageProfiles = make(PageProfiles)
				}
				for name, p := range profiles {
					config.PageProfiles[name] = p
				}
			}
		}
	}

	// Parse supervisor per-agent model overrides (JSON object)
	allModelLocs := supervisorModelsPattern.FindAllStringIndex(code, -1)
	for _, loc := range allModelLocs {
		braceIdx := strings.Index(code[loc[0]:], "{")
		if braceIdx >= 0 {
			jsonStr := extractBalancedJSONObject(code, loc[0]+braceIdx)
			if jsonStr != "" {
				var models map[string]string
				if err := json.Unmarshal([]byte(jsonStr), &models); err == nil {
					if config.SupervisorAgentModels == nil {
						config.SupervisorAgentModels = make(map[string]string)
					}
					for k, v := range models {
						config.SupervisorAgentModels[k] = v
					}
				}
			}
		}
	}

	// Parse max tool iterations
	if matches := maxToolIterationsPattern.FindStringSubmatch(code); len(matches) > 1 {
		if v, err := strconv.Atoi(matches[1]); err == nil && v > 0 {
			config.MaxToolIterations = v
		}
	}

	// Parse show reasoning flag
	if matches := showReasoningPattern.FindStringSubmatch(code); len(matches) > 1 {
		config.ShowReasoning = matches[1] == "true"
	}

	// Parse web-search enable/disable.
	// Accepted forms: enabled, disabled, true, false. Default is disabled
	// (annotation absent). This is opt-in per chatbot to avoid surprise
	// Tavily API charges.
	if matches := webSearchPattern.FindStringSubmatch(code); len(matches) > 1 {
		v := strings.ToLower(matches[1])
		config.WebSearchEnabled = v == "enabled" || v == "true"
	}

	// Parse optional web-search domain allowlist. Comma-separated. Empty
	// when annotation absent (no restriction). Scheme prefixes are stripped
	// (https://, http://) so users can paste URLs verbatim.
	if matches := webSearchDomainsPattern.FindStringSubmatch(code); len(matches) > 1 {
		raw := strings.TrimSpace(matches[1])
		if raw != "" {
			for _, d := range strings.Split(raw, ",") {
				d = strings.TrimSpace(d)
				d = strings.TrimPrefix(d, "https://")
				d = strings.TrimPrefix(d, "http://")
				d = strings.TrimSpace(d)
				if d != "" {
					config.WebSearchDomains = append(config.WebSearchDomains, d)
				}
			}
		}
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
	return extractBalanced(s, startIdx, '[', ']')
}

// extractBalancedJSONObject extracts a balanced JSON object starting from the
// given position. startIdx should point to the opening brace '{'.
func extractBalancedJSONObject(s string, startIdx int) string {
	return extractBalanced(s, startIdx, '{', '}')
}

// extractBalanced extracts a balanced bracketed region (array or object),
// respecting string literals and escape sequences. Returns "" if unbalanced.
func extractBalanced(s string, startIdx int, open, close byte) string {
	if startIdx >= len(s) || s[startIdx] != open {
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

		switch c {
		case open:
			depth++
		case close:
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
	c.ResponseLanguage = config.ResponseLanguage
	c.DisableExecutionLogs = config.DisableExecutionLogs
	c.RequiredSettings = config.RequiredSettings
	c.MCPTools = config.MCPTools
	c.UseMCPSchema = config.UseMCPSchema

	// RAG/Knowledge Base settings
	c.KnowledgeBases = config.KnowledgeBases
	c.RAGMaxChunks = config.RAGMaxChunks
	c.RAGSimilarityThreshold = config.RAGSimilarityThreshold
	c.RAGGraphBoostWeight = config.RAGGraphBoostWeight
	c.RAGTable = config.RAGTable
	c.RAGColumn = config.RAGColumn
	c.RAGContentColumn = config.RAGContentColumn

	// Agent behavior settings
	c.ReasoningMode = config.ReasoningMode
	c.MaxToolIterations = config.MaxToolIterations
	c.ShowReasoning = config.ShowReasoning
	c.PageProfiles = config.PageProfiles
	c.SupervisorAgentModels = config.SupervisorAgentModels
	c.WebSearchEnabled = config.WebSearchEnabled
	c.WebSearchDomains = config.WebSearchDomains

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

	// Parse agent behavior annotations from code if not already set on the
	// chatbot (DB-loaded chatbots don't have these columns). This is a
	// best-effort parse — if it fails, defaults apply (supervisor mode).
	if c.Code != "" {
		// ReasoningMode: parse from code if empty, else default to "supervisor"
		if c.ReasoningMode == "" {
			if matches := reasoningModePattern.FindStringSubmatch(c.Code); len(matches) > 1 {
				c.ReasoningMode = matches[1]
			} else {
				c.ReasoningMode = "supervisor"
			}
		}

		// PageProfiles: parse from code (no DB column for this)
		if c.PageProfiles == nil {
			if locs := pageContextsPattern.FindAllStringIndex(c.Code, -1); len(locs) > 0 {
				merged := make(PageProfiles)
				for _, loc := range locs {
					bracketIdx := strings.Index(c.Code[loc[0]:], "[")
					if bracketIdx < 0 {
						continue
					}
					jsonStr := extractBalancedJSON(c.Code, loc[0]+bracketIdx)
					if jsonStr == "" {
						continue
					}
					profiles, err := ParsePageProfilesJSON(jsonStr)
					if err != nil {
						continue
					}
					for name, p := range profiles {
						merged[name] = p
					}
				}
				if len(merged) > 0 {
					c.PageProfiles = merged
				}
			}
		}

		// SupervisorAgentModels: parse from code (no DB column)
		if c.SupervisorAgentModels == nil {
			if locs := supervisorModelsPattern.FindAllStringIndex(c.Code, -1); len(locs) > 0 {
				merged := make(map[string]string)
				for _, loc := range locs {
					braceIdx := strings.Index(c.Code[loc[0]:], "{")
					if braceIdx < 0 {
						continue
					}
					jsonStr := extractBalancedJSONObject(c.Code, loc[0]+braceIdx)
					if jsonStr == "" {
						continue
					}
					var models map[string]string
					if err := json.Unmarshal([]byte(jsonStr), &models); err != nil {
						continue
					}
					for k, v := range models {
						merged[k] = v
					}
				}
				if len(merged) > 0 {
					c.SupervisorAgentModels = merged
				}
			}

			// Web-search enable + optional domain allowlist (no DB column)
			if matches := webSearchPattern.FindStringSubmatch(c.Code); len(matches) > 1 {
				v := strings.ToLower(matches[1])
				c.WebSearchEnabled = v == "enabled" || v == "true"
			}
			if matches := webSearchDomainsPattern.FindStringSubmatch(c.Code); len(matches) > 1 {
				raw := strings.TrimSpace(matches[1])
				if raw != "" {
					for _, d := range strings.Split(raw, ",") {
						d = strings.TrimSpace(d)
						d = strings.TrimPrefix(d, "https://")
						d = strings.TrimPrefix(d, "http://")
						d = strings.TrimSpace(d)
						if d != "" {
							c.WebSearchDomains = append(c.WebSearchDomains, d)
						}
					}
				}
			}
		}
	}
}

// QualifiedTable represents a table with its schema
type QualifiedTable struct {
	Schema string
	Table  string
}

// ParseQualifiedTables extracts schema and table from allowed-tables annotation.
// Tables can be specified as "table" (defaults to public schema) or "schema.table".
// Returns a list of qualified tables.
func ParseQualifiedTables(allowedTables []string, defaultSchema string) []QualifiedTable {
	if defaultSchema == "" {
		defaultSchema = "public"
	}

	result := make([]QualifiedTable, 0, len(allowedTables))
	for _, table := range allowedTables {
		if strings.Contains(table, ".") {
			parts := strings.SplitN(table, ".", 2)
			result = append(result, QualifiedTable{
				Schema: parts[0],
				Table:  parts[1],
			})
		} else {
			result = append(result, QualifiedTable{
				Schema: defaultSchema,
				Table:  table,
			})
		}
	}
	return result
}

// GroupTablesBySchema groups qualified tables by schema for efficient filtering.
// Returns a map of schema -> []table names.
func GroupTablesBySchema(tables []QualifiedTable) map[string][]string {
	result := make(map[string][]string)
	for _, qt := range tables {
		result[qt.Schema] = append(result[qt.Schema], qt.Table)
	}
	return result
}

// HasMCPTools returns true if the chatbot has MCP tools configured
func (c *Chatbot) HasMCPTools() bool {
	return len(c.MCPTools) > 0
}

// AccessLevel defines the level of access a chatbot has to a knowledge base
type AccessLevel string

const (
	// AccessLevelFull - chatbot can retrieve all chunks from the KB
	AccessLevelFull AccessLevel = "full"
	// AccessLevelFiltered - chatbot retrieves chunks matching filter_expression
	AccessLevelFiltered AccessLevel = "filtered"
	// AccessLevelTiered - chatbot retrieves based on priority ordering
	AccessLevelTiered AccessLevel = "tiered"
)

// DefaultTraceIDGenerator implements trace ID generation
type DefaultTraceIDGenerator struct{}

// NewTraceIDGenerator creates a new trace ID generator
func NewTraceIDGenerator() *DefaultTraceIDGenerator {
	return &DefaultTraceIDGenerator{}
}

// GenerateTraceID generates a unique trace ID
func (t *DefaultTraceIDGenerator) GenerateTraceID() string {
	return fmt.Sprintf("trace_%x", uuid.New().String())
}

// GenerateSpanID generates a unique span ID
func (t *DefaultTraceIDGenerator) GenerateSpanID() string {
	return fmt.Sprintf("span_%x", uuid.New().String())
}
