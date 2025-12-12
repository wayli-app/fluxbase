// Package e2e tests the AI chatbot configuration parsing
package e2e

import (
	"testing"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/ai"
	"github.com/stretchr/testify/require"
)

// TestParseChatbotConfigAllAnnotations tests parsing of all annotation types
func TestParseChatbotConfigAllAnnotations(t *testing.T) {
	// GIVEN: A chatbot source with all annotations
	code := `/**
 * Test Assistant
 *
 * @fluxbase:allowed-tables tracker_data,locations,venues
 * @fluxbase:allowed-operations SELECT
 * @fluxbase:allowed-schemas public,app
 * @fluxbase:max-tokens 8192
 * @fluxbase:temperature 0.5
 * @fluxbase:persist-conversations true
 * @fluxbase:conversation-ttl 48h
 * @fluxbase:max-turns 100
 * @fluxbase:rate-limit 30/min
 * @fluxbase:daily-limit 1000
 * @fluxbase:token-budget 200000/day
 * @fluxbase:allow-unauthenticated true
 * @fluxbase:public false
 */

export default ` + "`" + `You are a helpful assistant.` + "`" + `;
`

	// WHEN: Parsing the configuration
	config := ai.ParseChatbotConfig(code)

	// THEN: All values are parsed correctly
	require.Equal(t, []string{"tracker_data", "locations", "venues"}, config.AllowedTables)
	require.Equal(t, []string{"SELECT"}, config.AllowedOperations)
	require.Equal(t, []string{"public", "app"}, config.AllowedSchemas)
	require.Equal(t, 8192, config.MaxTokens)
	require.Equal(t, 0.5, config.Temperature)
	require.True(t, config.PersistConversations)
	require.Equal(t, 48*time.Hour, config.ConversationTTL)
	require.Equal(t, 100, config.MaxTurns)
	require.Equal(t, 30, config.RateLimitPerMinute)
	require.Equal(t, 1000, config.DailyRequestLimit)
	require.Equal(t, 200000, config.DailyTokenBudget)
	require.True(t, config.AllowUnauthenticated)
	require.False(t, config.IsPublic)
}

// TestParseChatbotConfigDefaults tests that defaults are applied for missing annotations
func TestParseChatbotConfigDefaults(t *testing.T) {
	// GIVEN: A chatbot source with no annotations
	code := `/**
 * Simple Assistant
 */

export default ` + "`" + `You are a simple assistant.` + "`" + `;
`

	// WHEN: Parsing the configuration
	config := ai.ParseChatbotConfig(code)

	// THEN: Defaults are applied
	defaults := ai.DefaultChatbotConfig()

	require.Equal(t, defaults.AllowedTables, config.AllowedTables)
	require.Equal(t, defaults.AllowedOperations, config.AllowedOperations)
	require.Equal(t, defaults.AllowedSchemas, config.AllowedSchemas)
	require.Equal(t, defaults.MaxTokens, config.MaxTokens)
	require.Equal(t, defaults.Temperature, config.Temperature)
	require.Equal(t, defaults.PersistConversations, config.PersistConversations)
	require.Equal(t, defaults.ConversationTTL, config.ConversationTTL)
	require.Equal(t, defaults.MaxTurns, config.MaxTurns)
	require.Equal(t, defaults.RateLimitPerMinute, config.RateLimitPerMinute)
	require.Equal(t, defaults.DailyRequestLimit, config.DailyRequestLimit)
	require.Equal(t, defaults.DailyTokenBudget, config.DailyTokenBudget)
	require.Equal(t, defaults.AllowUnauthenticated, config.AllowUnauthenticated)
	require.Equal(t, defaults.IsPublic, config.IsPublic)
}

// TestParseChatbotConfigPartialAnnotations tests parsing with only some annotations
func TestParseChatbotConfigPartialAnnotations(t *testing.T) {
	// GIVEN: A chatbot source with partial annotations
	code := `/**
 * Partial Assistant
 *
 * @fluxbase:allowed-tables users,orders
 * @fluxbase:max-tokens 2048
 */

export default ` + "`" + `You help with user data.` + "`" + `;
`

	// WHEN: Parsing the configuration
	config := ai.ParseChatbotConfig(code)
	defaults := ai.DefaultChatbotConfig()

	// THEN: Specified values are parsed, others use defaults
	require.Equal(t, []string{"users", "orders"}, config.AllowedTables)
	require.Equal(t, 2048, config.MaxTokens)

	// Defaults for unspecified
	require.Equal(t, defaults.AllowedOperations, config.AllowedOperations)
	require.Equal(t, defaults.Temperature, config.Temperature)
	require.Equal(t, defaults.PersistConversations, config.PersistConversations)
}

// TestParseDescription tests description extraction from JSDoc
func TestParseDescription(t *testing.T) {
	testCases := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name: "simple_description",
			code: `/**
 * SQL Assistant
 *
 * @fluxbase:allowed-tables users
 */
export default ` + "`" + `prompt` + "`" + `;`,
			expected: "SQL Assistant",
		},
		{
			name: "description_with_dash",
			code: `/**
 * Location History - Track your movements
 *
 * @fluxbase:allowed-tables locations
 */
export default ` + "`" + `prompt` + "`" + `;`,
			expected: "Location History - Track your movements",
		},
		{
			name: "no_description",
			code: `/**
 * @fluxbase:allowed-tables users
 */
export default ` + "`" + `prompt` + "`" + `;`,
			expected: "",
		},
		{
			name: "multi_line_description_takes_first",
			code: `/**
 * First Line Description
 * Second line that is not included
 *
 * @fluxbase:allowed-tables users
 */
export default ` + "`" + `prompt` + "`" + `;`,
			expected: "First Line Description",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// WHEN: Parsing the description
			description := ai.ParseDescription(tc.code)

			// THEN: Expected description is extracted
			require.Equal(t, tc.expected, description)
		})
	}
}

// TestParseSystemPrompt tests system prompt extraction
func TestParseSystemPrompt(t *testing.T) {
	testCases := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "simple_prompt",
			code:     `export default ` + "`" + `You are a helpful assistant.` + "`" + `;`,
			expected: "You are a helpful assistant.",
		},
		{
			name: "multiline_prompt",
			code: `export default ` + "`" + `You are a helpful assistant.

## Guidelines
1. Be helpful
2. Be accurate
` + "`" + `;`,
			expected: `You are a helpful assistant.

## Guidelines
1. Be helpful
2. Be accurate`,
		},
		{
			name:     "prompt_with_variable",
			code:     `export default ` + "`" + `You help user {{user_id}} with queries.` + "`" + `;`,
			expected: "You help user {{user_id}} with queries.",
		},
		{
			name:     "no_export_default",
			code:     `const prompt = "Hello"`,
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// WHEN: Parsing the system prompt
			prompt := ai.ParseSystemPrompt(tc.code)

			// THEN: Expected prompt is extracted
			require.Equal(t, tc.expected, prompt)
		})
	}
}

// TestChatbotApplyConfig tests applying config to a chatbot
func TestChatbotApplyConfig(t *testing.T) {
	// GIVEN: A chatbot and config
	chatbot := &ai.Chatbot{}
	config := ai.ChatbotConfig{
		AllowedTables:        []string{"users", "orders"},
		AllowedOperations:    []string{"SELECT", "INSERT"},
		AllowedSchemas:       []string{"public", "app"},
		MaxTokens:            4096,
		Temperature:          0.8,
		PersistConversations: true,
		ConversationTTL:      72 * time.Hour,
		MaxTurns:             30,
		RateLimitPerMinute:   15,
		DailyRequestLimit:    250,
		DailyTokenBudget:     50000,
		AllowUnauthenticated: true,
		IsPublic:             false,
	}

	// WHEN: Applying config
	chatbot.ApplyConfig(config)

	// THEN: All values are applied
	require.Equal(t, config.AllowedTables, chatbot.AllowedTables)
	require.Equal(t, config.AllowedOperations, chatbot.AllowedOperations)
	require.Equal(t, config.AllowedSchemas, chatbot.AllowedSchemas)
	require.Equal(t, config.MaxTokens, chatbot.MaxTokens)
	require.Equal(t, config.Temperature, chatbot.Temperature)
	require.Equal(t, config.PersistConversations, chatbot.PersistConversations)
	require.Equal(t, 72, chatbot.ConversationTTLHours) // Converted from duration
	require.Equal(t, config.MaxTurns, chatbot.MaxConversationTurns)
	require.Equal(t, config.RateLimitPerMinute, chatbot.RateLimitPerMinute)
	require.Equal(t, config.DailyRequestLimit, chatbot.DailyRequestLimit)
	require.Equal(t, config.DailyTokenBudget, chatbot.DailyTokenBudget)
	require.Equal(t, config.AllowUnauthenticated, chatbot.AllowUnauthenticated)
	require.Equal(t, config.IsPublic, chatbot.IsPublic)
}

// TestChatbotToSummary tests converting chatbot to summary
func TestChatbotToSummary(t *testing.T) {
	// GIVEN: A chatbot
	now := time.Now()
	chatbot := &ai.Chatbot{
		ID:          "test-id-123",
		Name:        "test-assistant",
		Namespace:   "analytics",
		Description: "Test chatbot",
		Enabled:     true,
		IsPublic:    true,
		Source:      "filesystem",
		UpdatedAt:   now,
	}

	// WHEN: Converting to summary
	summary := chatbot.ToSummary()

	// THEN: Summary has correct values
	require.Equal(t, chatbot.ID, summary.ID)
	require.Equal(t, chatbot.Name, summary.Name)
	require.Equal(t, chatbot.Namespace, summary.Namespace)
	require.Equal(t, chatbot.Description, summary.Description)
	require.Equal(t, chatbot.Enabled, summary.Enabled)
	require.Equal(t, chatbot.IsPublic, summary.IsPublic)
	require.Equal(t, chatbot.Source, summary.Source)
	require.Equal(t, now.Format(time.RFC3339), summary.UpdatedAt)
}

// TestParseChatbotConfigMultipleOperations tests parsing multiple operations
func TestParseChatbotConfigMultipleOperations(t *testing.T) {
	// GIVEN: Code with multiple operations
	code := `/**
 * Full Access Assistant
 *
 * @fluxbase:allowed-operations SELECT,INSERT,UPDATE
 */
export default ` + "`" + `prompt` + "`" + `;`

	// WHEN: Parsing
	config := ai.ParseChatbotConfig(code)

	// THEN: All operations are parsed
	require.Equal(t, []string{"SELECT", "INSERT", "UPDATE"}, config.AllowedOperations)
}

// TestParseChatbotConfigWhitespaceHandling tests that whitespace is handled correctly
func TestParseChatbotConfigWhitespaceHandling(t *testing.T) {
	// GIVEN: Code with extra whitespace in annotations
	code := `/**
 * Whitespace Test
 *
 * @fluxbase:allowed-tables  users , orders , products
 * @fluxbase:allowed-schemas   public  ,  app
 */
export default ` + "`" + `prompt` + "`" + `;`

	// WHEN: Parsing
	config := ai.ParseChatbotConfig(code)

	// THEN: Values are trimmed
	require.Equal(t, []string{"users", "orders", "products"}, config.AllowedTables)
	require.Equal(t, []string{"public", "app"}, config.AllowedSchemas)
}

// TestDefaultChatbotConfig tests the default configuration values
func TestDefaultChatbotConfig(t *testing.T) {
	// WHEN: Getting default config
	defaults := ai.DefaultChatbotConfig()

	// THEN: Defaults match expected values
	require.Empty(t, defaults.AllowedTables, "Default should have no specific tables")
	require.Equal(t, []string{"SELECT"}, defaults.AllowedOperations, "Default should only allow SELECT")
	require.Equal(t, []string{"public"}, defaults.AllowedSchemas, "Default should only allow public schema")
	require.Equal(t, 4096, defaults.MaxTokens, "Default max tokens should be 4096")
	require.Equal(t, 0.7, defaults.Temperature, "Default temperature should be 0.7")
	require.False(t, defaults.PersistConversations, "Default should not persist conversations")
	require.Equal(t, 24*time.Hour, defaults.ConversationTTL, "Default TTL should be 24 hours")
	require.Equal(t, 50, defaults.MaxTurns, "Default max turns should be 50")
	require.Equal(t, 20, defaults.RateLimitPerMinute, "Default rate limit should be 20/min")
	require.Equal(t, 500, defaults.DailyRequestLimit, "Default daily limit should be 500")
	require.Equal(t, 100000, defaults.DailyTokenBudget, "Default token budget should be 100000")
	require.False(t, defaults.AllowUnauthenticated, "Default should require authentication")
	require.True(t, defaults.IsPublic, "Default should be public")
}
