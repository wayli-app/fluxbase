package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/config"
)

// Unit tests for pure decision/builder helpers across internal/ai that were at
// 0.0% -short coverage. All deterministic; no DB/network/LLM/FS. Fixtures use
// the package's own State/container types and small struct literals.

// =============================================================================
// SchemaBuilder.isTableAllowed
// =============================================================================
//
// Contract (schema_builder.go:666): if a specific-table list exists for the
// schema (len>0), the table must be in it (no fallthrough to the schema
// allowlist). Otherwise, the schema allowlist decides. Receiver unused (pure).

func TestSchemaBuilder_IsTableAllowed(t *testing.T) {
	t.Parallel()
	b := &SchemaBuilder{}

	t.Run("specific table list wins — table present", func(t *testing.T) {
		t.Parallel()
		allowed := map[string]bool{"public": true}
		tables := map[string][]string{"public": {"users", "orders"}}
		assert.True(t, b.isTableAllowed("public", "users", allowed, tables))
	})

	t.Run("specific table list wins — table absent", func(t *testing.T) {
		t.Parallel()
		// public has a table list, so even though public is in allowedSchemaSet,
		// a non-listed table is rejected (no fallthrough).
		allowed := map[string]bool{"public": true}
		tables := map[string][]string{"public": {"users"}}
		assert.False(t, b.isTableAllowed("public", "secret", allowed, tables))
	})

	t.Run("no table list → schema allowlist admits", func(t *testing.T) {
		t.Parallel()
		allowed := map[string]bool{"app": true}
		// app has no entry in tablesBySchema → any table in app allowed.
		assert.True(t, b.isTableAllowed("app", "anything", allowed, map[string][]string{}))
	})

	t.Run("schema not in allowlist and no table list → denied", func(t *testing.T) {
		t.Parallel()
		allowed := map[string]bool{"app": true}
		assert.False(t, b.isTableAllowed("other", "t", allowed, map[string][]string{}))
	})

	t.Run("empty table list falls through to schema allowlist", func(t *testing.T) {
		t.Parallel()
		// An empty slice (len 0) for the schema does NOT count as "specific
		// table list"; the schema allowlist decides. Pin this boundary.
		allowed := map[string]bool{"public": true}
		tables := map[string][]string{"public": {}}
		assert.True(t, b.isTableAllowed("public", "anything", allowed, tables))
	})
}

// =============================================================================
// buildSupervisorTurnMetadata (reads in-memory *State)
// =============================================================================

func TestBuildSupervisorTurnMetadata_EmptyState(t *testing.T) {
	t.Parallel()
	state := NewState()
	meta := buildSupervisorTurnMetadata(state)
	// No plan, no outputs, no language, no report → empty map.
	assert.Empty(t, meta)
}

func TestBuildSupervisorTurnMetadata_Populated(t *testing.T) {
	t.Parallel()
	state := NewState()
	state.Set(SupervisorPlanKey, &SupervisorPlan{UserLanguage: "en", Route: []string{"sql"}})
	state.AppendAgentOutput("sql_agent", "SELECT 1")
	state.SetUserLanguage("en")
	state.Set(VerifyReportKey, &VerifyReport{LanguageOK: true, GroundingOK: false, Issues: []string{"x"}})

	meta := buildSupervisorTurnMetadata(state)
	require.Contains(t, meta, "supervisor_plan")
	require.Contains(t, meta, "agent_outputs")
	require.Contains(t, meta, "user_language")
	require.Contains(t, meta, "verify_report")

	// agent_outputs maps agent name → content.
	ao, ok := meta["agent_outputs"].(map[string]string)
	require.True(t, ok, "agent_outputs should be map[string]string")
	assert.Equal(t, "SELECT 1", ao["sql_agent"])

	// verify_report is a map with the three keys.
	vr, ok := meta["verify_report"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, vr["language_ok"])
	assert.Equal(t, false, vr["grounding_ok"])
}

func TestBuildSupervisorTurnMetadata_NonPlanValueIgnored(t *testing.T) {
	t.Parallel()
	// If the SupervisorPlanKey holds the wrong type, it's silently skipped.
	state := NewState()
	state.Set(SupervisorPlanKey, "not-a-plan")
	meta := buildSupervisorTurnMetadata(state)
	assert.NotContains(t, meta, "supervisor_plan")
}

// =============================================================================
// parseMCPQueryResult (JSON parse + struct build; receiver unused)
// =============================================================================

func TestParseMCPQueryResult(t *testing.T) {
	t.Parallel()
	h := &ChatHandler{}

	t.Run("valid json array builds QueryResult", func(t *testing.T) {
		t.Parallel()
		content := `[{"id":1,"name":"alice"},{"id":2,"name":"bob"}]`
		got := h.parseMCPQueryResult("query", map[string]any{"table": "users"}, content)
		require.NotNil(t, got)
		assert.Equal(t, 2, got.RowCount)
		assert.Contains(t, got.Query, "MCP query on users")
		assert.Contains(t, got.Summary, "2 row(s)")
	})

	t.Run("invalid json returns nil", func(t *testing.T) {
		t.Parallel()
		got := h.parseMCPQueryResult("query", map[string]any{}, "not json")
		assert.Nil(t, got)
	})

	t.Run("missing table arg still works", func(t *testing.T) {
		t.Parallel()
		// No "table" key → tableName empty; query string still built.
		got := h.parseMCPQueryResult("query", map[string]any{}, `[]`)
		require.NotNil(t, got)
		assert.Equal(t, 0, got.RowCount)
		assert.Contains(t, got.Query, "MCP query on")
	})
}

// =============================================================================
// Storage.buildConfigBasedProvider
// =============================================================================
//
// Builds a synthetic provider from AIConfig when ProviderType is set. Nil/empty
// config → nil. Missing required env (e.g. OpenAI key) → nil. Valid config →
// ProviderRecord with the expected type/name/model.

func TestBuildConfigBasedProvider_NilConfig(t *testing.T) {
	t.Parallel()
	s := &Storage{config: nil}
	assert.Nil(t, s.buildConfigBasedProvider())
}

func TestBuildConfigBasedProvider_EmptyProviderType(t *testing.T) {
	t.Parallel()
	s := &Storage{config: &config.AIConfig{ProviderType: ""}}
	assert.Nil(t, s.buildConfigBasedProvider())
}

func TestBuildConfigBasedProvider_OpenAI_MissingKey(t *testing.T) {
	t.Parallel()
	// ProviderType set but no API key → nil (must not build a broken provider).
	s := &Storage{config: &config.AIConfig{ProviderType: "openai"}}
	assert.Nil(t, s.buildConfigBasedProvider())
}

func TestBuildConfigBasedProvider_OpenAI_Valid(t *testing.T) {
	t.Parallel()
	s := &Storage{config: &config.AIConfig{
		ProviderType:  "openai",
		ProviderName:  "Configured OpenAI",
		ProviderModel: "gpt-4o",
		OpenAIAPIKey:  "sk-test",
	}}
	got := s.buildConfigBasedProvider()
	require.NotNil(t, got)
	assert.Equal(t, "openai", got.ProviderType)
	assert.Equal(t, "config", got.Name) // Name is the literal "config"
	assert.Equal(t, "Configured OpenAI", got.DisplayName)
	assert.Equal(t, "gpt-4o", got.Config["model"])
	assert.Equal(t, "sk-test", got.Config["api_key"])
	assert.Equal(t, "FROM_CONFIG", got.ID)
	assert.True(t, got.ReadOnly, "config-based providers are read-only")
	assert.True(t, got.IsDefault)
}

func TestBuildConfigBasedProvider_OpenAI_DefaultModel(t *testing.T) {
	t.Parallel()
	// No explicit ProviderModel → openai defaults to "gpt-4-turbo".
	s := &Storage{config: &config.AIConfig{
		ProviderType: "openai",
		OpenAIAPIKey: "sk-test",
	}}
	got := s.buildConfigBasedProvider()
	require.NotNil(t, got)
	assert.Equal(t, "gpt-4-turbo", got.Config["model"])
}

func TestBuildConfigBasedProvider_Ollama_RequiresModel(t *testing.T) {
	t.Parallel()
	// Ollama requires OllamaModel; endpoint alone is insufficient.
	s := &Storage{config: &config.AIConfig{
		ProviderType:   "ollama",
		OllamaEndpoint: "http://localhost:11434",
	}}
	assert.Nil(t, s.buildConfigBasedProvider(), "Ollama without OllamaModel must return nil")
}

func TestBuildConfigBasedProvider_Ollama_Valid(t *testing.T) {
	t.Parallel()
	s := &Storage{config: &config.AIConfig{
		ProviderType:   "ollama",
		ProviderName:   "Local Ollama",
		OllamaModel:    "llama3",
		OllamaEndpoint: "http://localhost:11434",
	}}
	got := s.buildConfigBasedProvider()
	require.NotNil(t, got)
	assert.Equal(t, "ollama", got.ProviderType)
	assert.Equal(t, "llama3", got.Config["model"])
	assert.Equal(t, "http://localhost:11434", got.Config["endpoint"])
}

func TestBuildConfigBasedProvider_Ollama_DefaultEndpoint(t *testing.T) {
	t.Parallel()
	// No endpoint → defaults to localhost:11434.
	s := &Storage{config: &config.AIConfig{
		ProviderType: "ollama",
		OllamaModel:  "llama3",
	}}
	got := s.buildConfigBasedProvider()
	require.NotNil(t, got)
	assert.Equal(t, "http://localhost:11434", got.Config["endpoint"])
}

func TestBuildConfigBasedProvider_UnknownType(t *testing.T) {
	t.Parallel()
	s := &Storage{config: &config.AIConfig{ProviderType: "bogus"}}
	assert.Nil(t, s.buildConfigBasedProvider())
}
