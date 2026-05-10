package tenantdb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// BackfillTenantIDToDefault assigns NULL tenant_id rows to the default tenant.
// This handles the upgrade path from pre-multi-tenant Fluxbase where all data
// was created without tenant context.
func BackfillTenantIDToDefault(pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var defaultTenantID string
	err := pool.QueryRow(
		ctx,
		"SELECT id::text FROM platform.tenants WHERE is_default = true AND deleted_at IS NULL LIMIT 1",
	).Scan(&defaultTenantID)
	if err != nil {
		if err.Error() == "no rows in result set" || strings.Contains(err.Error(), "no rows") {
			log.Debug().Msg("No default tenant found, skipping tenant_id backfill")
			return nil
		}
		return fmt.Errorf("failed to get default tenant: %w", err)
	}

	tenantIDDedupTables := map[string][]string{
		"auth.users":               {"email"},
		"functions.edge_functions": {"name, namespace"},
		"storage.buckets":          {"name"},
		"branching.branches":       {"name", "slug"},
		"branching.github_config":  {"repository"},
		"mcp.custom_tools":         {"name, namespace"},
		"mcp.custom_resources":     {"uri, namespace"},
	}

	tables := []string{
		"auth.users",
		"auth.client_keys",
		"auth.impersonation_sessions",
		"auth.webhooks",
		"auth.webhook_deliveries",
		"auth.webhook_events",
		"auth.saml_providers",
		"auth.sessions",
		"auth.oauth_links",
		"auth.oauth_tokens",
		"auth.mfa_factors",
		"auth.saml_sessions",
		"auth.magic_links",
		"auth.otp_codes",
		"auth.email_verification_tokens",
		"auth.password_reset_tokens",
		"auth.two_factor_setups",
		"auth.two_factor_recovery_attempts",
		"auth.oauth_logout_states",
		"auth.mcp_oauth_clients",
		"auth.mcp_oauth_codes",
		"auth.mcp_oauth_tokens",
		"auth.client_key_usage",
		"auth.service_key_revocations",
		"functions.edge_functions",
		"functions.edge_executions",
		"functions.edge_files",
		"functions.edge_triggers",
		"functions.secrets",
		"functions.secret_versions",
		"functions.shared_modules",
		"functions.function_dependencies",
		"jobs.functions",
		"jobs.function_files",
		"jobs.workers",
		"jobs.queue",
		"ai.knowledge_bases",
		"ai.knowledge_base_permissions",
		"ai.documents",
		"ai.document_permissions",
		"ai.chunks",
		"ai.entities",
		"ai.document_entities",
		"ai.entity_relationships",
		"ai.providers",
		"ai.chatbots",
		"ai.chatbot_knowledge_bases",
		"ai.conversations",
		"ai.messages",
		"ai.query_audit_log",
		"ai.retrieval_log",
		"ai.table_export_sync_configs",
		"ai.user_chatbot_usage",
		"ai.user_provider_preferences",
		"ai.user_quotas",
		"rpc.procedures",
		"rpc.executions",
		"realtime.schema_registry",
		"storage.buckets",
		"storage.objects",
		"storage.chunked_upload_sessions",
		"storage.object_permissions",
		"branching.branches",
		"branching.activity_log",
		"branching.branch_access",
		"branching.github_config",
		"branching.migration_history",
		"branching.seed_execution_log",
		"logging.entries",
		"logging.entries_ai",
		"logging.entries_custom",
		"logging.entries_execution",
		"logging.entries_http",
		"logging.entries_security",
		"logging.entries_system",
		"mcp.custom_resources",
		"mcp.custom_tools",
		"platform.invitation_tokens",
	}

	var totalBackfilled int
	for _, table := range tables {
		if dedupSets, needsDedup := tenantIDDedupTables[table]; needsDedup {
			for _, dedupCols := range dedupSets {
				dedupQuery := fmt.Sprintf(
					"DELETE FROM %s WHERE id IN (SELECT id FROM (SELECT id, row_number() OVER (PARTITION BY %s ORDER BY created_at DESC) AS rn FROM %s WHERE tenant_id IS NULL) sub WHERE rn > 1)",
					table, dedupCols, table,
				)
				if dedupResult, err := pool.Exec(ctx, dedupQuery); err != nil {
					log.Warn().Err(err).Str("table", table).Str("cols", dedupCols).Msg("Failed to dedup NULL-tenant rows before backfill")
				} else if n := dedupResult.RowsAffected(); n > 0 {
					log.Info().Str("table", table).Str("cols", dedupCols).Int64("duplicates_removed", n).Msg("Removed duplicate NULL-tenant rows before backfill")
				}

				conflictQuery := fmt.Sprintf(
					"DELETE FROM %s WHERE id IN (SELECT n.id FROM %s n WHERE n.tenant_id IS NULL AND EXISTS (SELECT 1 FROM %s e WHERE e.tenant_id = $1 AND (%s)))",
					table, table, table, buildJoinCondition(dedupCols, "n", "e"),
				)
				if conflictResult, err := pool.Exec(ctx, conflictQuery, defaultTenantID); err != nil {
					log.Warn().Err(err).Str("table", table).Str("cols", dedupCols).Msg("Failed to remove NULL-tenant rows conflicting with existing tenant rows")
				} else if n := conflictResult.RowsAffected(); n > 0 {
					log.Info().Str("table", table).Str("cols", dedupCols).Int64("conflicts_removed", n).Msg("Removed NULL-tenant rows that would conflict with existing tenant rows")
				}
			}
		}

		result, err := pool.Exec(
			ctx,
			fmt.Sprintf("UPDATE %s SET tenant_id = $1::uuid WHERE tenant_id IS NULL", table),
			defaultTenantID,
		)
		if err != nil {
			log.Warn().Err(err).Str("table", table).Msg("Failed to backfill tenant_id")
			continue
		}
		if n := result.RowsAffected(); n > 0 {
			log.Info().Str("table", table).Int64("rows", n).Msg("Backfilled tenant_id to default tenant")
			totalBackfilled += int(n)
		}
	}

	if totalBackfilled > 0 {
		log.Info().Int("total_rows", totalBackfilled).Msg("Tenant_id backfill complete")
	} else {
		log.Debug().Msg("No NULL tenant_id rows found to backfill")
	}

	return nil
}

func buildJoinCondition(cols, leftAlias, rightAlias string) string {
	parts := strings.Split(cols, ",")
	conditions := make([]string, len(parts))
	for i, col := range parts {
		col = strings.TrimSpace(col)
		conditions[i] = fmt.Sprintf("%s.%s = %s.%s", leftAlias, col, rightAlias, col)
	}
	return strings.Join(conditions, " AND ")
}
