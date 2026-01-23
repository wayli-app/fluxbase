package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateIntent_RequiredTable(t *testing.T) {
	rules := []IntentRule{
		{
			Keywords:      []string{"restaurant", "cafe", "food"},
			RequiredTable: "my_place_visits",
		},
	}
	validator := NewIntentValidator(rules, nil, "")

	// Should fail - wrong table used
	result := validator.ValidateIntent("Show me restaurants I visited", "SELECT * FROM my_trips", []string{"my_trips"})
	assert.False(t, result.Valid)
	assert.Contains(t, result.Errors[0], "should use table 'my_place_visits'")

	// Should pass - correct table used
	result = validator.ValidateIntent("Show me restaurants I visited", "SELECT * FROM my_place_visits", []string{"my_place_visits"})
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)

	// Should pass - no matching keywords
	result = validator.ValidateIntent("Show me my trips", "SELECT * FROM my_trips", []string{"my_trips"})
	assert.True(t, result.Valid)
}

func TestValidateIntent_ForbiddenTable(t *testing.T) {
	rules := []IntentRule{
		{
			Keywords:       []string{"restaurant", "cafe"},
			RequiredTable:  "my_place_visits",
			ForbiddenTable: "my_trips",
		},
	}
	validator := NewIntentValidator(rules, nil, "")

	// Should fail - using forbidden table
	result := validator.ValidateIntent("What restaurant did I visit?", "SELECT * FROM my_trips", []string{"my_trips"})
	assert.False(t, result.Valid)
	assert.Contains(t, result.Errors[0], "should NOT use table 'my_trips'")
	assert.Contains(t, result.Suggestions[0], "my_place_visits")

	// Should pass - using correct table
	result = validator.ValidateIntent("What restaurant did I visit?", "SELECT * FROM my_place_visits", []string{"my_place_visits"})
	assert.True(t, result.Valid)
}

func TestValidateIntent_CaseInsensitive(t *testing.T) {
	rules := []IntentRule{
		{
			Keywords:      []string{"restaurant"},
			RequiredTable: "my_place_visits",
		},
	}
	validator := NewIntentValidator(rules, nil, "")

	// Should match regardless of case
	result := validator.ValidateIntent("RESTAURANT recommendations?", "SELECT * FROM my_trips", []string{"my_trips"})
	assert.False(t, result.Valid)
}

func TestValidateIntent_NoRules(t *testing.T) {
	validator := NewIntentValidator(nil, nil, "")

	// Should always pass when no rules configured
	result := validator.ValidateIntent("Show me anything", "SELECT * FROM any_table", []string{"any_table"})
	assert.True(t, result.Valid)
}

func TestValidateIntent_MatchedKeywords(t *testing.T) {
	rules := []IntentRule{
		{
			Keywords:      []string{"restaurant", "food", "dining"},
			RequiredTable: "my_place_visits",
		},
	}
	validator := NewIntentValidator(rules, nil, "")

	result := validator.ValidateIntent("Show me food and restaurant places", "SELECT * FROM my_place_visits", []string{"my_place_visits"})
	assert.True(t, result.Valid)
	assert.Contains(t, result.MatchedKeywords, "restaurant")
	assert.Contains(t, result.MatchedKeywords, "food")
}

func TestValidateRequiredColumns_MissingColumns(t *testing.T) {
	requiredCols := RequiredColumnsMap{
		"my_trips": {"id", "title", "image_url"},
	}
	validator := NewIntentValidator(nil, requiredCols, "")

	// Should fail - missing required column
	result := validator.ValidateRequiredColumns("SELECT id, title FROM my_trips", []string{"my_trips"})
	assert.False(t, result.Valid)
	assert.Contains(t, result.Errors[0], "image_url")

	// Should pass - all required columns present
	result = validator.ValidateRequiredColumns("SELECT id, title, image_url FROM my_trips", []string{"my_trips"})
	assert.True(t, result.Valid)
}

func TestValidateRequiredColumns_SelectStar(t *testing.T) {
	requiredCols := RequiredColumnsMap{
		"my_trips": {"id", "title", "image_url"},
	}
	validator := NewIntentValidator(nil, requiredCols, "")

	// Should pass - SELECT * includes all columns
	result := validator.ValidateRequiredColumns("SELECT * FROM my_trips", []string{"my_trips"})
	assert.True(t, result.Valid)
}

func TestValidateRequiredColumns_NoRequiredColumns(t *testing.T) {
	validator := NewIntentValidator(nil, nil, "")

	// Should always pass when no required columns configured
	result := validator.ValidateRequiredColumns("SELECT id FROM my_trips", []string{"my_trips"})
	assert.True(t, result.Valid)
}

func TestValidateRequiredColumns_TableNotConfigured(t *testing.T) {
	requiredCols := RequiredColumnsMap{
		"my_trips": {"id", "title"},
	}
	validator := NewIntentValidator(nil, requiredCols, "")

	// Should pass - table not in required columns config
	result := validator.ValidateRequiredColumns("SELECT name FROM my_place_visits", []string{"my_place_visits"})
	assert.True(t, result.Valid)
}

func TestValidateRequiredColumns_WithSchemaPrefix(t *testing.T) {
	requiredCols := RequiredColumnsMap{
		"my_trips": {"id", "title"},
	}
	validator := NewIntentValidator(nil, requiredCols, "")

	// Should work with schema prefix
	result := validator.ValidateRequiredColumns("SELECT name FROM public.my_trips", []string{"public.my_trips"})
	assert.False(t, result.Valid) // Missing required columns
}

func TestGetDefaultTable(t *testing.T) {
	validator := NewIntentValidator(nil, nil, "my_place_visits")
	assert.Equal(t, "my_place_visits", validator.GetDefaultTable())

	validator = NewIntentValidator(nil, nil, "")
	assert.Equal(t, "", validator.GetDefaultTable())
}

func TestHasIntentRules(t *testing.T) {
	rules := []IntentRule{{Keywords: []string{"test"}, RequiredTable: "test_table"}}
	validator := NewIntentValidator(rules, nil, "")
	assert.True(t, validator.HasIntentRules())

	validator = NewIntentValidator(nil, nil, "")
	assert.False(t, validator.HasIntentRules())
}

func TestHasRequiredColumns(t *testing.T) {
	requiredCols := RequiredColumnsMap{"table": {"col1"}}
	validator := NewIntentValidator(nil, requiredCols, "")
	assert.True(t, validator.HasRequiredColumns())

	validator = NewIntentValidator(nil, nil, "")
	assert.False(t, validator.HasRequiredColumns())
}

func TestValidateIntent_WithSchemaPrefix(t *testing.T) {
	rules := []IntentRule{
		{
			Keywords:       []string{"restaurant", "cafe", "food"},
			RequiredTable:  "my_place_visits",
			ForbiddenTable: "my_trips",
		},
	}
	validator := NewIntentValidator(rules, nil, "")

	// Should pass when SQL uses schema-prefixed table name but rule uses simple name
	result := validator.ValidateIntent("Show me restaurants", "SELECT * FROM public.my_place_visits", []string{"public.my_place_visits"})
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)

	// Should fail when using wrong table (even with schema prefix)
	result = validator.ValidateIntent("Show me restaurants", "SELECT * FROM public.my_trips", []string{"public.my_trips"})
	assert.False(t, result.Valid)
	assert.Contains(t, result.Errors[0], "should NOT use table 'my_trips'")
}

// Test the extractSelectedColumns function
func TestExtractSelectedColumns(t *testing.T) {
	// Simple select
	cols, err := extractSelectedColumns("SELECT id, name, email FROM users")
	assert.NoError(t, err)
	assert.True(t, cols["id"])
	assert.True(t, cols["name"])
	assert.True(t, cols["email"])

	// Select with aliases
	cols, err = extractSelectedColumns("SELECT id, name AS user_name FROM users")
	assert.NoError(t, err)
	assert.True(t, cols["id"])
	assert.True(t, cols["user_name"])

	// Select star
	cols, err = extractSelectedColumns("SELECT * FROM users")
	assert.NoError(t, err)
	assert.True(t, cols["*"])

	// Invalid SQL
	_, err = extractSelectedColumns("NOT VALID SQL")
	assert.Error(t, err)
}

func TestValidateToolCall_RequiredTool(t *testing.T) {
	rules := []IntentRule{
		{
			Keywords:     []string{"similar", "like this", "recommend based on"},
			RequiredTool: "vector_search",
		},
	}
	validator := NewIntentValidator(rules, nil, "")

	// Should fail - wrong tool used
	result := validator.ValidateToolCall("Find places similar to this cafe", "query_table")
	assert.False(t, result.Valid)
	assert.Contains(t, result.Errors[0], "should use tool 'vector_search'")
	assert.Contains(t, result.Errors[0], "not 'query_table'")

	// Should pass - correct tool used
	result = validator.ValidateToolCall("Find places similar to this cafe", "vector_search")
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)

	// Should pass - no matching keywords
	result = validator.ValidateToolCall("Show me all restaurants", "query_table")
	assert.True(t, result.Valid)
}

func TestValidateToolCall_ForbiddenTool(t *testing.T) {
	rules := []IntentRule{
		{
			Keywords:      []string{"visited", "been to", "have I been"},
			ForbiddenTool: "http_request",
		},
	}
	validator := NewIntentValidator(rules, nil, "")

	// Should fail - using forbidden tool
	result := validator.ValidateToolCall("Places I've visited", "http_request")
	assert.False(t, result.Valid)
	assert.Contains(t, result.Errors[0], "should NOT use tool 'http_request'")

	// Should pass - using different tool
	result = validator.ValidateToolCall("Places I've visited", "query_table")
	assert.True(t, result.Valid)
}

func TestValidateToolCall_RequiredAndForbiddenTool(t *testing.T) {
	rules := []IntentRule{
		{
			Keywords:      []string{"similar", "like this"},
			RequiredTool:  "vector_search",
			ForbiddenTool: "http_request",
		},
	}
	validator := NewIntentValidator(rules, nil, "")

	// Should fail - using forbidden tool
	result := validator.ValidateToolCall("Find similar places", "http_request")
	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 2) // Both forbidden and required violations

	// Should pass - using required tool
	result = validator.ValidateToolCall("Find similar places", "vector_search")
	assert.True(t, result.Valid)
}

func TestValidateToolCall_CaseInsensitive(t *testing.T) {
	rules := []IntentRule{
		{
			Keywords:     []string{"similar"},
			RequiredTool: "vector_search",
		},
	}
	validator := NewIntentValidator(rules, nil, "")

	// Should match regardless of keyword case
	result := validator.ValidateToolCall("SIMILAR places please", "query_table")
	assert.False(t, result.Valid)
}

func TestValidateToolCall_NoToolRules(t *testing.T) {
	// Rules with only table constraints, no tool constraints
	rules := []IntentRule{
		{
			Keywords:      []string{"restaurant"},
			RequiredTable: "my_place_visits",
		},
	}
	validator := NewIntentValidator(rules, nil, "")

	// Should pass - no tool rules apply
	result := validator.ValidateToolCall("Show me restaurants", "any_tool")
	assert.True(t, result.Valid)
}

func TestValidateToolCall_NoRules(t *testing.T) {
	validator := NewIntentValidator(nil, nil, "")

	// Should always pass when no rules configured
	result := validator.ValidateToolCall("Any message", "any_tool")
	assert.True(t, result.Valid)
}

func TestValidateToolCall_MatchedKeywords(t *testing.T) {
	rules := []IntentRule{
		{
			Keywords:     []string{"similar", "like this", "recommend"},
			RequiredTool: "vector_search",
		},
	}
	validator := NewIntentValidator(rules, nil, "")

	result := validator.ValidateToolCall("Find similar places, recommend some like this", "vector_search")
	assert.True(t, result.Valid)
	assert.Contains(t, result.MatchedKeywords, "similar")
	assert.Contains(t, result.MatchedKeywords, "like this")
	assert.Contains(t, result.MatchedKeywords, "recommend")
}

func TestValidateToolCall_SuggestionWithRequiredTool(t *testing.T) {
	rules := []IntentRule{
		{
			Keywords:      []string{"similar"},
			RequiredTool:  "vector_search",
			ForbiddenTool: "http_request",
		},
	}
	validator := NewIntentValidator(rules, nil, "")

	// When forbidden tool is used and required tool exists, suggestion should mention required tool
	result := validator.ValidateToolCall("Find similar places", "http_request")
	assert.False(t, result.Valid)
	assert.Contains(t, result.Suggestions[0], "vector_search")
}
