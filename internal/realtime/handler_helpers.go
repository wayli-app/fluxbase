package realtime

import "encoding/json"

// Channel-name prefixes used to classify subscriptions. Admin and fluxbase
// channels are broadcast-only (no database table backing); everything else is
// treated as a postgres_changes subscription that needs a schema/table.
const (
	adminChannelPrefix     = "realtime:admin:"
	fluxbaseChannelPrefix  = "fluxbase:"
	defaultSubscribeSchema = "public"
	defaultSubscribeEvent  = "*"
)

// isAdminChannel reports whether channel is a realtime admin channel
// ("realtime:admin:*"). These channels are broadcast-only and require an
// admin/instance_admin/service_role connection to subscribe or broadcast.
//
// The check is byte-prefix based: a channel shorter than the prefix cannot
// match, which avoids out-of-range slicing.
func isAdminChannel(channel string) bool {
	return hasChannelPrefix(channel, adminChannelPrefix)
}

// isFluxbaseChannel reports whether channel is a built-in fluxbase channel
// ("fluxbase:*"). These are broadcast-only and do not require a table.
func isFluxbaseChannel(channel string) bool {
	return hasChannelPrefix(channel, fluxbaseChannelPrefix)
}

// hasChannelPrefix is the shared prefix check: true iff channel is at least as
// long as prefix and its leading bytes equal prefix.
func hasChannelPrefix(channel, prefix string) bool {
	return len(channel) >= len(prefix) && channel[:len(prefix)] == prefix
}

// parseLogSubscription extracts the execution ID and type from a subscribe_logs
// ClientMessage. It tries, in order:
//  1. msg.Config as LogSubscriptionConfig ({ execution_id, type })
//  2. msg.Config as the legacy PostgresChangesConfig ({ schema, table }) —
//     mapped to (executionID, executionType)
//  3. msg.Payload as a map with "execution_id"/"type" string fields
//
// Returns empty strings when nothing could be parsed. Pure: no I/O.
func parseLogSubscription(msg ClientMessage) (executionID, executionType string) {
	// 1. New config format
	if len(msg.Config) > 0 {
		var logConfig LogSubscriptionConfig
		if err := json.Unmarshal(msg.Config, &logConfig); err == nil {
			executionID = logConfig.ExecutionID
			executionType = logConfig.Type
		}

		// 2. Legacy config format (schema/table fields repurposed)
		if executionID == "" {
			var pgConfig PostgresChangesConfig
			if err := json.Unmarshal(msg.Config, &pgConfig); err == nil {
				executionID = pgConfig.Schema
				executionType = pgConfig.Table
			}
		}
	}

	// 3. Fall back to payload map if config didn't provide an execution ID.
	if executionID == "" && len(msg.Payload) > 0 {
		var payload map[string]interface{}
		if err := json.Unmarshal(msg.Payload, &payload); err == nil {
			if v, ok := payload["execution_id"].(string); ok {
				executionID = v
			}
			if v, ok := payload["type"].(string); ok {
				executionType = v
			}
		}
	}

	return executionID, executionType
}

// parsePostgresChangesConfig extracts the event/schema/table/filter fields for
// a postgres_changes subscription. It prefers the new msg.Config format
// ({ event, schema, table, filter }) and falls back to the legacy top-level
// ClientMessage fields. Schema defaults to "public" and event to "*" when empty.
// Pure: no I/O.
func parsePostgresChangesConfig(msg ClientMessage) (event, schema, table, filter string) {
	if len(msg.Config) > 0 {
		var config PostgresChangesConfig
		if err := json.Unmarshal(msg.Config, &config); err == nil {
			event = config.Event
			schema = config.Schema
			table = config.Table
			filter = config.Filter
		}
	}
	// Fall back to legacy fields if config didn't parse a table.
	if table == "" {
		event = msg.Event
		schema = msg.Schema
		table = msg.Table
		filter = msg.Filter
	}
	if schema == "" {
		schema = defaultSubscribeSchema
	}
	if event == "" {
		event = defaultSubscribeEvent
	}
	return event, schema, table, filter
}
