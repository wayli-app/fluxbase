package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseChatbotConfig_IntentRules(t *testing.T) {
	code := "/**\n" +
		" * Test chatbot\n" +
		" *\n" +
		" * @fluxbase:allowed-tables my_trips,my_place_visits\n" +
		" * @fluxbase:intent-rules [{\"keywords\":[\"restaurant\",\"cafe\"],\"requiredTable\":\"my_place_visits\",\"forbiddenTable\":\"my_trips\"},{\"keywords\":[\"trip\",\"travel\"],\"requiredTable\":\"my_trips\"}]\n" +
		" */\n" +
		"\n" +
		"export default `You are a helpful assistant.`;\n"

	config := ParseChatbotConfig(code)

	assert.Len(t, config.IntentRules, 2)

	// First rule
	assert.Equal(t, []string{"restaurant", "cafe"}, config.IntentRules[0].Keywords)
	assert.Equal(t, "my_place_visits", config.IntentRules[0].RequiredTable)
	assert.Equal(t, "my_trips", config.IntentRules[0].ForbiddenTable)

	// Second rule
	assert.Equal(t, []string{"trip", "travel"}, config.IntentRules[1].Keywords)
	assert.Equal(t, "my_trips", config.IntentRules[1].RequiredTable)
	assert.Equal(t, "", config.IntentRules[1].ForbiddenTable)
}

func TestParseChatbotConfig_RequiredColumns(t *testing.T) {
	code := "/**\n" +
		" * Test chatbot\n" +
		" *\n" +
		" * @fluxbase:required-columns my_trips=id,title,image_url my_place_visits=poi_name,city\n" +
		" */\n" +
		"\n" +
		"export default `You are a helpful assistant.`;\n"

	config := ParseChatbotConfig(code)

	assert.Len(t, config.RequiredColumns, 2)
	assert.Equal(t, []string{"id", "title", "image_url"}, config.RequiredColumns["my_trips"])
	assert.Equal(t, []string{"poi_name", "city"}, config.RequiredColumns["my_place_visits"])
}

func TestParseChatbotConfig_DefaultTable(t *testing.T) {
	code := "/**\n" +
		" * Test chatbot\n" +
		" *\n" +
		" * @fluxbase:default-table my_place_visits\n" +
		" */\n" +
		"\n" +
		"export default `You are a helpful assistant.`;\n"

	config := ParseChatbotConfig(code)

	assert.Equal(t, "my_place_visits", config.DefaultTable)
}

func TestParseChatbotConfig_AllIntentAnnotations(t *testing.T) {
	code := "/**\n" +
		" * Location Assistant\n" +
		" *\n" +
		" * @fluxbase:allowed-tables my_trips,my_place_visits,my_poi_summary\n" +
		" * @fluxbase:allowed-operations SELECT\n" +
		" * @fluxbase:default-table my_place_visits\n" +
		" * @fluxbase:required-columns my_trips=id,title,image_url\n" +
		" * @fluxbase:intent-rules [{\"keywords\":[\"restaurant\",\"cafe\",\"food\"],\"requiredTable\":\"my_place_visits\",\"forbiddenTable\":\"my_trips\"}]\n" +
		" */\n" +
		"\n" +
		"export default `You are a location assistant.`;\n"

	config := ParseChatbotConfig(code)

	// Check all intent-related fields
	assert.Equal(t, "my_place_visits", config.DefaultTable)
	assert.Len(t, config.IntentRules, 1)
	assert.Equal(t, []string{"restaurant", "cafe", "food"}, config.IntentRules[0].Keywords)
	assert.Len(t, config.RequiredColumns, 1)
	assert.Equal(t, []string{"id", "title", "image_url"}, config.RequiredColumns["my_trips"])
}

func TestParseChatbotConfig_NoIntentAnnotations(t *testing.T) {
	code := "/**\n" +
		" * Simple chatbot\n" +
		" *\n" +
		" * @fluxbase:allowed-tables users\n" +
		" * @fluxbase:allowed-operations SELECT\n" +
		" */\n" +
		"\n" +
		"export default `You are a helpful assistant.`;\n"

	config := ParseChatbotConfig(code)

	// Should be nil/empty when no intent annotations
	assert.Nil(t, config.IntentRules)
	assert.Nil(t, config.RequiredColumns)
	assert.Equal(t, "", config.DefaultTable)
}

func TestParseChatbotConfig_InvalidIntentRulesJSON(t *testing.T) {
	code := "/**\n" +
		" * Test chatbot\n" +
		" *\n" +
		" * @fluxbase:intent-rules not-valid-json\n" +
		" */\n" +
		"\n" +
		"export default `You are a helpful assistant.`;\n"

	config := ParseChatbotConfig(code)

	// Should be nil when JSON is invalid
	assert.Nil(t, config.IntentRules)
}

func TestParseRequiredColumns(t *testing.T) {
	// Single table
	result := parseRequiredColumns("my_trips=id,title,image_url")
	assert.Len(t, result, 1)
	assert.Equal(t, []string{"id", "title", "image_url"}, result["my_trips"])

	// Multiple tables
	result = parseRequiredColumns("my_trips=id,title my_places=name,city")
	assert.Len(t, result, 2)
	assert.Equal(t, []string{"id", "title"}, result["my_trips"])
	assert.Equal(t, []string{"name", "city"}, result["my_places"])

	// Empty input
	result = parseRequiredColumns("")
	assert.Len(t, result, 0)

	// Invalid format (no equals sign)
	result = parseRequiredColumns("invalid-format")
	assert.Len(t, result, 0)
}

func TestApplyConfig_IntentFields(t *testing.T) {
	config := ChatbotConfig{
		IntentRules: []IntentRule{
			{Keywords: []string{"test"}, RequiredTable: "test_table"},
		},
		RequiredColumns: RequiredColumnsMap{
			"table1": {"col1", "col2"},
		},
		DefaultTable: "default_table",
	}

	chatbot := &Chatbot{}
	chatbot.ApplyConfig(config)

	assert.Len(t, chatbot.IntentRules, 1)
	assert.Equal(t, "test_table", chatbot.IntentRules[0].RequiredTable)
	assert.Equal(t, []string{"col1", "col2"}, chatbot.RequiredColumns["table1"])
	assert.Equal(t, "default_table", chatbot.DefaultTable)
}

func TestParseChatbotConfig_MultipleRequiredColumnsAnnotations(t *testing.T) {
	code := "/**\n" +
		" * Test chatbot\n" +
		" *\n" +
		" * @fluxbase:required-columns my_trips=id,title,image_url\n" +
		" * @fluxbase:required-columns my_place_visits=poi_name,city\n" +
		" * @fluxbase:required-columns my_poi_summary=category,count\n" +
		" */\n" +
		"\n" +
		"export default `You are a helpful assistant.`;\n"

	config := ParseChatbotConfig(code)

	assert.Len(t, config.RequiredColumns, 3)
	assert.Equal(t, []string{"id", "title", "image_url"}, config.RequiredColumns["my_trips"])
	assert.Equal(t, []string{"poi_name", "city"}, config.RequiredColumns["my_place_visits"])
	assert.Equal(t, []string{"category", "count"}, config.RequiredColumns["my_poi_summary"])
}

func TestParseChatbotConfig_MultipleIntentRulesAnnotations(t *testing.T) {
	code := "/**\n" +
		" * Test chatbot\n" +
		" *\n" +
		" * @fluxbase:intent-rules [{\"keywords\":[\"restaurant\",\"cafe\"],\"requiredTable\":\"my_place_visits\"}]\n" +
		" * @fluxbase:intent-rules [{\"keywords\":[\"trip\",\"travel\"],\"requiredTable\":\"my_trips\"}]\n" +
		" */\n" +
		"\n" +
		"export default `You are a helpful assistant.`;\n"

	config := ParseChatbotConfig(code)

	assert.Len(t, config.IntentRules, 2)

	// First annotation
	assert.Equal(t, []string{"restaurant", "cafe"}, config.IntentRules[0].Keywords)
	assert.Equal(t, "my_place_visits", config.IntentRules[0].RequiredTable)

	// Second annotation
	assert.Equal(t, []string{"trip", "travel"}, config.IntentRules[1].Keywords)
	assert.Equal(t, "my_trips", config.IntentRules[1].RequiredTable)
}

func TestParseChatbotConfig_ResponseLanguage(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name: "auto (default)",
			code: "/**\n" +
				" * Test chatbot\n" +
				" *\n" +
				" * @fluxbase:allowed-tables users\n" +
				" */\n" +
				"\n" +
				"export default `You are a helpful assistant.`;\n",
			expected: "auto",
		},
		{
			name: "explicit auto",
			code: "/**\n" +
				" * Test chatbot\n" +
				" *\n" +
				" * @fluxbase:response-language auto\n" +
				" */\n" +
				"\n" +
				"export default `You are a helpful assistant.`;\n",
			expected: "auto",
		},
		{
			name: "ISO code",
			code: "/**\n" +
				" * Test chatbot\n" +
				" *\n" +
				" * @fluxbase:response-language de\n" +
				" */\n" +
				"\n" +
				"export default `You are a helpful assistant.`;\n",
			expected: "de",
		},
		{
			name: "language name English",
			code: "/**\n" +
				" * Test chatbot\n" +
				" *\n" +
				" * @fluxbase:response-language German\n" +
				" */\n" +
				"\n" +
				"export default `You are a helpful assistant.`;\n",
			expected: "German",
		},
		{
			name: "language name native",
			code: "/**\n" +
				" * Test chatbot\n" +
				" *\n" +
				" * @fluxbase:response-language Deutsch\n" +
				" */\n" +
				"\n" +
				"export default `You are a helpful assistant.`;\n",
			expected: "Deutsch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ParseChatbotConfig(tt.code)
			assert.Equal(t, tt.expected, config.ResponseLanguage)
		})
	}
}

func TestApplyConfig_ResponseLanguage(t *testing.T) {
	config := ChatbotConfig{
		ResponseLanguage: "German",
	}

	chatbot := &Chatbot{}
	chatbot.ApplyConfig(config)

	assert.Equal(t, "German", chatbot.ResponseLanguage)
}
