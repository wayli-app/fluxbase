package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePageProfiles_ValidJSON_ReturnsProfiles(t *testing.T) {
	jsonStr := `[
		{"page": "orders", "agents": ["sql", "action"], "tables": ["orders", "order_items"]},
		{"page": "docs", "agents": ["kb"], "kbs": ["product-docs"]}
	]`
	profiles, err := ParsePageProfilesJSON(jsonStr)
	require.NoError(t, err)
	require.Len(t, profiles, 2)

	orders := profiles["orders"]
	require.NotNil(t, orders)
	assert.Equal(t, "orders", orders.Page)
	assert.Equal(t, []string{"sql", "action"}, orders.Agents)
	assert.Equal(t, []string{"orders", "order_items"}, orders.Tables)

	docs := profiles["docs"]
	require.NotNil(t, docs)
	assert.Equal(t, []string{"product-docs"}, docs.KBs)
}

func TestParsePageProfiles_MissingPage_ReturnsError(t *testing.T) {
	jsonStr := `[{"agents": ["sql"]}]`
	_, err := ParsePageProfilesJSON(jsonStr)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing 'page'")
}

func TestParsePageProfiles_DuplicatePage_ReturnsError(t *testing.T) {
	jsonStr := `[
		{"page": "orders"},
		{"page": "orders"}
	]`
	_, err := ParsePageProfilesJSON(jsonStr)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate page name")
}

func TestParsePageProfiles_InvalidPageName_ReturnsError(t *testing.T) {
	jsonStr := `[{"page": "../etc/passwd"}]`
	_, err := ParsePageProfilesJSON(jsonStr)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must match")
}

func TestParsePageProfiles_UnknownAgent_ReturnsError(t *testing.T) {
	jsonStr := `[{"page": "x", "agents": ["not-real"]}]`
	_, err := ParsePageProfilesJSON(jsonStr)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown agent")
}

func TestParsePageProfiles_MalformedJSON_ReturnsError(t *testing.T) {
	_, err := ParsePageProfilesJSON(`[not json`)
	require.Error(t, err)
}

func TestPageProfiles_Resolve_KnownPage_ReturnsProfile(t *testing.T) {
	p := PageProfiles{
		"orders": &PageProfile{Page: "orders"},
	}
	profile := p.Resolve("orders")
	require.NotNil(t, profile)
	assert.Equal(t, "orders", profile.Page)
}

func TestPageProfiles_Resolve_UnknownPage_ReturnsNil(t *testing.T) {
	p := PageProfiles{
		"orders": &PageProfile{Page: "orders"},
	}
	assert.Nil(t, p.Resolve("unknown"))
}

func TestPageProfiles_Resolve_EmptyContext_ReturnsNil(t *testing.T) {
	p := PageProfiles{
		"orders": &PageProfile{Page: "orders"},
	}
	assert.Nil(t, p.Resolve(""))
}

func TestPageProfiles_Resolve_NilMap_ReturnsNil(t *testing.T) {
	var p PageProfiles
	assert.Nil(t, p.Resolve("anything"))
}

func TestPageProfile_HasAgent_RespectsWhitelist(t *testing.T) {
	p := &PageProfile{Page: "x", Agents: []string{"sql", "kb"}}
	assert.True(t, p.HasAgent("sql"))
	assert.True(t, p.HasAgent("kb"))
	assert.False(t, p.HasAgent("action"))
	assert.False(t, p.HasAgent("chat"))
}

func TestPageProfile_HasAgent_EmptyWhitelist_PermitsAll(t *testing.T) {
	p := &PageProfile{Page: "x"}
	for _, a := range []string{"sql", "kb", "action", "chat"} {
		assert.True(t, p.HasAgent(a), "expected %q to be allowed", a)
	}
}

func TestPageProfile_HasAgent_NilProfile_PermitsAll(t *testing.T) {
	var p *PageProfile
	assert.True(t, p.HasAgent("sql"))
}

func TestPageProfile_ResolvedTables_OverridesGlobal(t *testing.T) {
	p := &PageProfile{Page: "x", Tables: []string{"t1", "t2"}}
	assert.Equal(t, []string{"t1", "t2"}, p.ResolvedTables([]string{"global"}))
}

func TestPageProfile_ResolvedTables_NoOverride_FallsBack(t *testing.T) {
	p := &PageProfile{Page: "x"}
	assert.Equal(t, []string{"global"}, p.ResolvedTables([]string{"global"}))
}

func TestParseChatbotConfig_PageContextsAnnotation(t *testing.T) {
	code := "/**\n" +
		" * Test chatbot with page contexts.\n" +
		" *\n" +
		" * @fluxbase:page-contexts [\n" +
		" *   {\"page\": \"orders\", \"agents\": [\"sql\"], \"tables\": [\"orders\"]},\n" +
		" *   {\"page\": \"docs\", \"agents\": [\"kb\"]}\n" +
		" * ]\n" +
		" */\n" +
		"export default `You are a helpful assistant.`\n"
	config := ParseChatbotConfig(code)
	require.NotNil(t, config.PageProfiles)
	require.Len(t, config.PageProfiles, 2)

	orders := config.PageProfiles["orders"]
	require.NotNil(t, orders)
	assert.Equal(t, []string{"orders"}, orders.Tables)

	docs := config.PageProfiles["docs"]
	require.NotNil(t, docs)
	assert.Equal(t, []string{"kb"}, docs.Agents)
}

func TestParseChatbotConfig_PageContexts_MergesMultipleAnnotations(t *testing.T) {
	code := "/**\n" +
		" * @fluxbase:page-contexts [{\"page\": \"orders\"}]\n" +
		" * @fluxbase:page-contexts [{\"page\": \"docs\"}]\n" +
		" */\n" +
		"export default `test`\n"
	config := ParseChatbotConfig(code)
	require.NotNil(t, config.PageProfiles)
	require.Len(t, config.PageProfiles, 2)
	assert.NotNil(t, config.PageProfiles["orders"])
	assert.NotNil(t, config.PageProfiles["docs"])
}

func TestParseChatbotConfig_ReasoningModeSupervisor(t *testing.T) {
	code := "/**\n * @fluxbase:reasoning-mode supervisor\n */\nexport default `test`\n"
	config := ParseChatbotConfig(code)
	assert.Equal(t, "supervisor", config.ReasoningMode)
}

func TestParseChatbotConfig_ReasoningModeDefault_IsSupervisor(t *testing.T) {
	config := DefaultChatbotConfig()
	assert.Equal(t, "supervisor", config.ReasoningMode)
}

func TestParseChatbotConfig_SupervisorModels(t *testing.T) {
	code := "/**\n * @fluxbase:supervisor-models {\"sql\":\"gpt-4-turbo\",\"chat\":\"gpt-4o-mini\"}\n */\nexport default `test`\n"
	config := ParseChatbotConfig(code)
	require.NotNil(t, config.SupervisorAgentModels)
	assert.Equal(t, "gpt-4-turbo", config.SupervisorAgentModels["sql"])
	assert.Equal(t, "gpt-4o-mini", config.SupervisorAgentModels["chat"])
}

func TestPopulateDerivedFields_DefaultsToSupervisorWhenNoAnnotation(t *testing.T) {
	c := &Chatbot{Code: "export default `test`"}
	c.PopulateDerivedFields()
	assert.Equal(t, "supervisor", c.ReasoningMode)
}

func TestPopulateDerivedFields_PreservesExplicitReactAnnotation(t *testing.T) {
	c := &Chatbot{Code: "/**\n * @fluxbase:reasoning-mode react\n */\nexport default `test`\n"}
	c.PopulateDerivedFields()
	assert.Equal(t, "react", c.ReasoningMode)
}

func TestPopulateDerivedFields_ParsesPageProfilesFromCode(t *testing.T) {
	c := &Chatbot{Code: "/**\n * @fluxbase:page-contexts [{\"page\":\"orders\",\"tables\":[\"orders\"]}]\n */\nexport default `test`\n"}
	c.PopulateDerivedFields()
	require.NotNil(t, c.PageProfiles)
	require.NotNil(t, c.PageProfiles["orders"])
}
