package realtime

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Unit tests for the pure helpers extracted from handler.go: the channel-prefix
// classifiers (isAdminChannel/isFluxbaseChannel) and the two config parsers
// (parseLogSubscription, parsePostgresChangesConfig). All are deterministic and
// I/O-free. Convention matches handler_test.go: testify, package realtime.

// =============================================================================
// Channel-prefix classifiers
// =============================================================================
//
// Contract source: handler_helpers.go. Admin channels ("realtime:admin:*") and
// fluxbase channels ("fluxbase:*") are broadcast-only. The check is byte-prefix
// based and must not panic on channels shorter than the prefix.

func TestIsAdminChannel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		channel string
		want    bool
	}{
		{"exact prefix only", "realtime:admin:", true},
		{"typical admin channel", "realtime:admin:metrics", true},
		{"long admin channel", "realtime:admin:system:health", true},
		{"empty is not admin", "", false},
		{"too short is not admin", "realtime", false},
		{"shorter than prefix", "realtime:admin", false},
		{"prefix substring not at start", "x-realtime:admin:y", false},
		{"other channel", "realtime:public:users", false},
		{"fluxbase channel is not admin", "fluxbase:events", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, isAdminChannel(tt.channel))
		})
	}
}

func TestIsFluxbaseChannel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		channel string
		want    bool
	}{
		{"exact prefix only", "fluxbase:", true},
		{"typical fluxbase channel", "fluxbase:events", true},
		{"long fluxbase channel", "fluxbase:system:config", true},
		{"empty is not fluxbase", "", false},
		{"too short", "flux", false},
		{"shorter than prefix", "fluxbas", false},
		{"prefix substring not at start", "x-fluxbase:y", false},
		{"other channel", "realtime:public:users", false},
		{"admin channel is not fluxbase", "realtime:admin:metrics", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, isFluxbaseChannel(tt.channel))
		})
	}
}

// =============================================================================
// parsePostgresChangesConfig
// =============================================================================
//
// Contract source: handler_helpers.go. Prefers msg.Config ({event, schema,
// table, filter}); falls back to legacy top-level fields when no table parses.
// Defaults: schema -> "public", event -> "*". filter is never defaulted.

func TestParsePostgresChangesConfig(t *testing.T) {
	t.Parallel()

	mustJSON := func(t *testing.T, v interface{}) json.RawMessage {
		t.Helper()
		b, err := json.Marshal(v)
		assert.NoError(t, err)
		return b
	}

	t.Run("config format wins when present", func(t *testing.T) {
		t.Parallel()
		msg := ClientMessage{
			Config: mustJSON(t, PostgresChangesConfig{
				Event: "INSERT", Schema: "auth", Table: "users", Filter: "id=eq.1",
			}),
			// Legacy fields set too — config must win.
			Event: "DELETE", Schema: "public", Table: "orders", Filter: "x=eq.2",
		}
		event, schema, table, filter := parsePostgresChangesConfig(msg)
		assert.Equal(t, "INSERT", event)
		assert.Equal(t, "auth", schema)
		assert.Equal(t, "users", table)
		assert.Equal(t, "id=eq.1", filter)
	})

	t.Run("legacy fields used when config absent", func(t *testing.T) {
		t.Parallel()
		msg := ClientMessage{
			Event: "UPDATE", Schema: "app", Table: "settings", Filter: "key=eq.foo",
		}
		event, schema, table, filter := parsePostgresChangesConfig(msg)
		assert.Equal(t, "UPDATE", event)
		assert.Equal(t, "app", schema)
		assert.Equal(t, "settings", table)
		assert.Equal(t, "key=eq.foo", filter)
	})

	t.Run("legacy fields used when config has empty table", func(t *testing.T) {
		t.Parallel()
		// Config present but table empty -> falls back to legacy fields.
		msg := ClientMessage{
			Config: mustJSON(t, map[string]string{"event": "INSERT"}), // no table
			Event:  "*", Table: "legacy_table", Schema: "legacy_schema",
		}
		_, schema, table, _ := parsePostgresChangesConfig(msg)
		assert.Equal(t, "legacy_table", table)
		assert.Equal(t, "legacy_schema", schema)
	})

	t.Run("schema defaults to public when empty", func(t *testing.T) {
		t.Parallel()
		msg := ClientMessage{Table: "users"} // no schema anywhere
		_, schema, _, _ := parsePostgresChangesConfig(msg)
		assert.Equal(t, "public", schema)
	})

	t.Run("event defaults to star when empty", func(t *testing.T) {
		t.Parallel()
		msg := ClientMessage{Table: "users"} // no event anywhere
		event, _, _, _ := parsePostgresChangesConfig(msg)
		assert.Equal(t, "*", event)
	})

	t.Run("filter not defaulted when empty", func(t *testing.T) {
		t.Parallel()
		msg := ClientMessage{Table: "users"}
		_, _, _, filter := parsePostgresChangesConfig(msg)
		assert.Empty(t, filter)
	})

	t.Run("malformed config falls back to legacy", func(t *testing.T) {
		t.Parallel()
		msg := ClientMessage{
			Config: json.RawMessage(`{not valid json`),
			Table:  "fallback_table",
		}
		_, _, table, _ := parsePostgresChangesConfig(msg)
		assert.Equal(t, "fallback_table", table)
	})

	t.Run("empty message yields defaults and empty table", func(t *testing.T) {
		t.Parallel()
		event, schema, table, filter := parsePostgresChangesConfig(ClientMessage{})
		assert.Equal(t, "*", event)
		assert.Equal(t, "public", schema)
		assert.Empty(t, table)
		assert.Empty(t, filter)
	})
}

// =============================================================================
// parseLogSubscription
// =============================================================================
//
// Contract source: handler_helpers.go. Tries, in order: msg.Config as
// LogSubscriptionConfig ({execution_id, type}); msg.Config as legacy
// PostgresChangesConfig ({schema, table}); msg.Payload map with string
// execution_id/type fields. Returns empty strings when nothing parses.

func TestParseLogSubscription(t *testing.T) {
	t.Parallel()

	mustJSON := func(t *testing.T, v interface{}) json.RawMessage {
		t.Helper()
		b, err := json.Marshal(v)
		assert.NoError(t, err)
		return b
	}

	t.Run("new config format", func(t *testing.T) {
		t.Parallel()
		msg := ClientMessage{
			Config: mustJSON(t, LogSubscriptionConfig{
				ExecutionID: "exec-123", Type: "function",
			}),
		}
		id, typ := parseLogSubscription(msg)
		assert.Equal(t, "exec-123", id)
		assert.Equal(t, "function", typ)
	})

	t.Run("legacy config format (schema/table)", func(t *testing.T) {
		t.Parallel()
		msg := ClientMessage{
			Config: mustJSON(t, PostgresChangesConfig{
				Schema: "exec-456", Table: "job",
			}),
		}
		id, typ := parseLogSubscription(msg)
		assert.Equal(t, "exec-456", id)
		assert.Equal(t, "job", typ)
	})

	t.Run("new config wins over legacy when both would parse", func(t *testing.T) {
		t.Parallel()
		// LogSubscriptionConfig parses successfully (has execution_id) so the
		// legacy PostgresChangesConfig fallback is not tried.
		msg := ClientMessage{
			Config: mustJSON(t, LogSubscriptionConfig{
				ExecutionID: "from-new", Type: "rpc",
			}),
		}
		id, typ := parseLogSubscription(msg)
		assert.Equal(t, "from-new", id)
		assert.Equal(t, "rpc", typ)
	})

	t.Run("payload fallback when config has no execution_id", func(t *testing.T) {
		t.Parallel()
		msg := ClientMessage{
			Config: mustJSON(t, LogSubscriptionConfig{}), // empty execution_id
			Payload: mustJSON(t, map[string]interface{}{
				"execution_id": "from-payload", "type": "function",
			}),
		}
		id, typ := parseLogSubscription(msg)
		assert.Equal(t, "from-payload", id)
		assert.Equal(t, "function", typ)
	})

	t.Run("payload fallback when no config", func(t *testing.T) {
		t.Parallel()
		msg := ClientMessage{
			Payload: mustJSON(t, map[string]interface{}{
				"execution_id": "payload-only", "type": "job",
			}),
		}
		id, typ := parseLogSubscription(msg)
		assert.Equal(t, "payload-only", id)
		assert.Equal(t, "job", typ)
	})

	t.Run("empty message returns empty strings", func(t *testing.T) {
		t.Parallel()
		id, typ := parseLogSubscription(ClientMessage{})
		assert.Empty(t, id)
		assert.Empty(t, typ)
	})

	t.Run("malformed config falls back to payload", func(t *testing.T) {
		t.Parallel()
		msg := ClientMessage{
			Config: json.RawMessage(`{not json`),
			Payload: mustJSON(t, map[string]interface{}{
				"execution_id": "after-bad-config",
			}),
		}
		id, _ := parseLogSubscription(msg)
		assert.Equal(t, "after-bad-config", id)
	})

	t.Run("non-string payload fields ignored", func(t *testing.T) {
		t.Parallel()
		// Numeric execution_id is not a string -> ignored.
		msg := ClientMessage{
			Payload: mustJSON(t, map[string]interface{}{
				"execution_id": 12345,
			}),
		}
		id, _ := parseLogSubscription(msg)
		assert.Empty(t, id)
	})

	t.Run("config type NOT preserved when execution_id comes from payload", func(t *testing.T) {
		t.Parallel()
		// Documents a subtle real behavior: when config has a Type but no
		// ExecutionID, the legacy-config fallback re-parses the same bytes as
		// PostgresChangesConfig (which has no Table) and clobbers executionType
		// with "". The payload then supplies executionID but its "type" field is
		// absent here, so typ ends up empty. Callers that need a type must
		// include execution_id in the same config object.
		msg := ClientMessage{
			Config: mustJSON(t, LogSubscriptionConfig{Type: "from-config"}),
			Payload: mustJSON(t, map[string]interface{}{
				"execution_id": "from-payload",
			}),
		}
		id, typ := parseLogSubscription(msg)
		assert.Equal(t, "from-payload", id)
		assert.Empty(t, typ, "legacy fallback clobbers type when execution_id is absent from config")
	})
}
