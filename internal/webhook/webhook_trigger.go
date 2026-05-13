package webhook

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/database"
)

// parseTableReference splits a table reference into schema and table name
// e.g.:
//   - "auth.users" -> ("auth", "users")
//   - "ai.documents" -> ("ai", "documents")
//   - "users" -> ("auth", "users") - defaults to auth schema for backward compatibility
//
// For AI schema tables, always use the full reference "ai.documents", "ai.chunks", etc.
func parseTableReference(tableRef string) (schema, table string) {
	if idx := strings.Index(tableRef, "."); idx > 0 {
		return tableRef[:idx], tableRef[idx+1:]
	}
	// Default to auth schema for backward compatibility
	// For AI schema tables, use explicit "ai.documents" format
	return "auth", tableRef
}

// ManageTriggersForWebhook ensures database triggers exist for all tables monitored by this webhook
func (s *WebhookService) ManageTriggersForWebhook(ctx context.Context, events []EventConfig) error {
	for _, event := range events {
		if event.Table == "*" {
			continue // Wildcard doesn't need specific trigger
		}
		schema, table := parseTableReference(event.Table)
		if err := s.incrementTableCount(ctx, schema, table); err != nil {
			return fmt.Errorf("failed to create trigger for %s.%s: %w", schema, table, err)
		}
	}
	return nil
}

// CleanupTriggersForWebhook decrements reference counts for monitored tables
func (s *WebhookService) CleanupTriggersForWebhook(ctx context.Context, events []EventConfig) error {
	for _, event := range events {
		if event.Table == "*" {
			continue
		}
		schema, table := parseTableReference(event.Table)
		if err := s.decrementTableCount(ctx, schema, table); err != nil {
			log.Error().Err(err).Str("schema", schema).Str("table", table).Msg("Failed to decrement table count")
		}
	}
	return nil
}

// incrementTableCount calls the database function to increment webhook count for a table
func (s *WebhookService) incrementTableCount(ctx context.Context, schema, table string) error {
	query := `SELECT auth.increment_webhook_table_count($1, $2)`
	return database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, schema, table)
		return err
	})
}

// decrementTableCount calls the database function to decrement webhook count for a table
func (s *WebhookService) decrementTableCount(ctx context.Context, schema, table string) error {
	query := `SELECT auth.decrement_webhook_table_count($1, $2)`
	return database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, schema, table)
		return err
	})
}
