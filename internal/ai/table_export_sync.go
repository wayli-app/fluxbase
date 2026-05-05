package ai

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/database"
)

// SyncMode defines how table exports are kept in sync
type SyncMode string

const (
	SyncModeManual    SyncMode = "manual"    // Manual re-export only
	SyncModeAutomatic SyncMode = "automatic" // Auto-sync via webhooks
)

// SyncStatus defines the status of the last sync operation
type SyncStatus string

const (
	SyncStatusSuccess SyncStatus = "success"
	SyncStatusFailed  SyncStatus = "failed"
	SyncStatusPending SyncStatus = "pending"
)

// TableExportSyncConfig stores sync configuration for an exported table
type TableExportSyncConfig struct {
	ID                 string     `json:"id"`
	KnowledgeBaseID    string     `json:"knowledge_base_id"`
	SchemaName         string     `json:"schema_name"`
	TableName          string     `json:"table_name"`
	Columns            []string   `json:"columns,omitempty"`
	SyncMode           SyncMode   `json:"sync_mode"`
	SyncOnInsert       bool       `json:"sync_on_insert"`
	SyncOnUpdate       bool       `json:"sync_on_update"`
	SyncOnDelete       bool       `json:"sync_on_delete"`
	DebounceSeconds    int        `json:"debounce_seconds"`
	IncludeForeignKeys bool       `json:"include_foreign_keys"`
	IncludeIndexes     bool       `json:"include_indexes"`
	LastSyncAt         *time.Time `json:"last_sync_at,omitempty"`
	LastSyncStatus     string     `json:"last_sync_status,omitempty"`
	LastSyncError      string     `json:"last_sync_error,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// CreateTableExportSyncConfig is the request for creating a sync config
type CreateTableExportSyncConfig struct {
	KnowledgeBaseID    string   `json:"knowledge_base_id"`
	SchemaName         string   `json:"schema_name"`
	TableName          string   `json:"table_name"`
	Columns            []string `json:"columns,omitempty"`
	SyncMode           SyncMode `json:"sync_mode"`
	SyncOnInsert       bool     `json:"sync_on_insert"`
	SyncOnUpdate       bool     `json:"sync_on_update"`
	SyncOnDelete       bool     `json:"sync_on_delete"`
	DebounceSeconds    int      `json:"debounce_seconds"`
	IncludeForeignKeys bool     `json:"include_foreign_keys"`
	IncludeIndexes     bool     `json:"include_indexes"`
	ExportNow          bool     `json:"export_now"` // Trigger initial export on creation
}

// UpdateTableExportSyncConfig is the request for updating a sync config
type UpdateTableExportSyncConfig struct {
	Columns            *[]string `json:"columns,omitempty"`
	SyncMode           *SyncMode `json:"sync_mode,omitempty"`
	SyncOnInsert       *bool     `json:"sync_on_insert,omitempty"`
	SyncOnUpdate       *bool     `json:"sync_on_update,omitempty"`
	SyncOnDelete       *bool     `json:"sync_on_delete,omitempty"`
	DebounceSeconds    *int      `json:"debounce_seconds,omitempty"`
	IncludeForeignKeys *bool     `json:"include_foreign_keys,omitempty"`
	IncludeIndexes     *bool     `json:"include_indexes,omitempty"`
}

// TableExportSyncService manages sync configurations and manual triggers for exported tables
type TableExportSyncService struct {
	database.TenantAware
	exporter *TableExporter
	storage  *KnowledgeBaseStorage
}

// NewTableExportSyncService creates a new sync service
func NewTableExportSyncService(
	db *database.Connection,
	exporter *TableExporter,
	storage *KnowledgeBaseStorage,
) *TableExportSyncService {
	return &TableExportSyncService{
		TenantAware: database.TenantAware{DB: db},
		exporter:    exporter,
		storage:     storage,
	}
}

// CreateSyncConfig creates a new sync configuration
func (s *TableExportSyncService) CreateSyncConfig(ctx context.Context, config *CreateTableExportSyncConfig) (*TableExportSyncConfig, error) {
	// Set defaults
	if config.SyncMode == "" {
		config.SyncMode = SyncModeManual
	}
	if config.DebounceSeconds == 0 {
		config.DebounceSeconds = 60
	}

	id := uuid.New().String()
	now := time.Now()

	query := `
		INSERT INTO ai.table_export_sync_configs (
			id, knowledge_base_id, schema_name, table_name, columns,
			sync_mode, sync_on_insert, sync_on_update, sync_on_delete,
			debounce_seconds, include_foreign_keys, include_indexes,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING created_at, updated_at
	`

	var columns interface{} = nil
	if len(config.Columns) > 0 {
		columns = config.Columns
	}

	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx, query,
			id, config.KnowledgeBaseID, config.SchemaName, config.TableName, columns,
			config.SyncMode, config.SyncOnInsert, config.SyncOnUpdate, config.SyncOnDelete,
			config.DebounceSeconds, config.IncludeForeignKeys, config.IncludeIndexes,
			now, now,
		).Scan(&now, &now)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create sync config: %w", err)
	}

	result := &TableExportSyncConfig{
		ID:                 id,
		KnowledgeBaseID:    config.KnowledgeBaseID,
		SchemaName:         config.SchemaName,
		TableName:          config.TableName,
		Columns:            config.Columns,
		SyncMode:           config.SyncMode,
		SyncOnInsert:       config.SyncOnInsert,
		SyncOnUpdate:       config.SyncOnUpdate,
		SyncOnDelete:       config.SyncOnDelete,
		DebounceSeconds:    config.DebounceSeconds,
		IncludeForeignKeys: config.IncludeForeignKeys,
		IncludeIndexes:     config.IncludeIndexes,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	// Trigger initial export if requested
	if config.ExportNow {
		_, err := s.TriggerSync(ctx, id)
		if err != nil {
			log.Warn().Err(err).Str("sync_id", id).Msg("Failed to trigger initial export")
		}
	}

	return result, nil
}

// GetSyncConfig retrieves a sync configuration by ID
func (s *TableExportSyncService) GetSyncConfig(ctx context.Context, id string) (*TableExportSyncConfig, error) {
	query := `
		SELECT id, knowledge_base_id, schema_name, table_name, columns,
			   sync_mode, sync_on_insert, sync_on_update, sync_on_delete,
			   debounce_seconds, include_foreign_keys, include_indexes,
			   last_sync_at, last_sync_status, last_sync_error,
			   created_at, updated_at
		FROM ai.table_export_sync_configs
		WHERE id = $1
	`

	config := &TableExportSyncConfig{}
	var columns []byte
	var lastSyncAt *time.Time
	var lastSyncStatus, lastSyncError *string

	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, id).Scan(
			&config.ID, &config.KnowledgeBaseID, &config.SchemaName, &config.TableName, &columns,
			&config.SyncMode, &config.SyncOnInsert, &config.SyncOnUpdate, &config.SyncOnDelete,
			&config.DebounceSeconds, &config.IncludeForeignKeys, &config.IncludeIndexes,
			&lastSyncAt, &lastSyncStatus, &lastSyncError,
			&config.CreatedAt, &config.UpdatedAt,
		)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("sync config not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get sync config: %w", err)
	}

	if columns != nil {
		config.Columns = parsePostgresArray(string(columns))
	}
	config.LastSyncAt = lastSyncAt
	if lastSyncStatus != nil {
		config.LastSyncStatus = *lastSyncStatus
	}
	if lastSyncError != nil {
		config.LastSyncError = *lastSyncError
	}

	return config, nil
}

// GetSyncConfigsByKnowledgeBase lists sync configs for a knowledge base
func (s *TableExportSyncService) GetSyncConfigsByKnowledgeBase(ctx context.Context, knowledgeBaseID string) ([]TableExportSyncConfig, error) {
	query := `
		SELECT id, knowledge_base_id, schema_name, table_name, columns,
			   sync_mode, sync_on_insert, sync_on_update, sync_on_delete,
			   debounce_seconds, include_foreign_keys, include_indexes,
			   last_sync_at, last_sync_status, last_sync_error,
			   created_at, updated_at
		FROM ai.table_export_sync_configs
		WHERE knowledge_base_id = $1
		ORDER BY created_at DESC
	`

	var configs []TableExportSyncConfig

	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, knowledgeBaseID)
		if err != nil {
			return fmt.Errorf("failed to list sync configs: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			config := TableExportSyncConfig{}
			var columns []byte
			var lastSyncAt *time.Time
			var lastSyncStatus, lastSyncError *string

			err := rows.Scan(
				&config.ID, &config.KnowledgeBaseID, &config.SchemaName, &config.TableName, &columns,
				&config.SyncMode, &config.SyncOnInsert, &config.SyncOnUpdate, &config.SyncOnDelete,
				&config.DebounceSeconds, &config.IncludeForeignKeys, &config.IncludeIndexes,
				&lastSyncAt, &lastSyncStatus, &lastSyncError,
				&config.CreatedAt, &config.UpdatedAt,
			)
			if err != nil {
				return fmt.Errorf("failed to scan sync config: %w", err)
			}

			if columns != nil {
				config.Columns = parsePostgresArray(string(columns))
			}
			config.LastSyncAt = lastSyncAt
			if lastSyncStatus != nil {
				config.LastSyncStatus = *lastSyncStatus
			}
			if lastSyncError != nil {
				config.LastSyncError = *lastSyncError
			}

			configs = append(configs, config)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return configs, nil
}

// UpdateSyncConfig updates an existing sync configuration
func (s *TableExportSyncService) UpdateSyncConfig(ctx context.Context, id string, updates UpdateTableExportSyncConfig) (*TableExportSyncConfig, error) {
	query := `UPDATE ai.table_export_sync_configs SET updated_at = NOW()`
	args := []interface{}{id}
	argNum := 2

	if updates.Columns != nil {
		query += fmt.Sprintf(", columns = $%d", argNum)
		if len(*updates.Columns) == 0 {
			args = append(args, nil)
		} else {
			args = append(args, *updates.Columns)
		}
		argNum++
	}
	if updates.SyncMode != nil {
		query += fmt.Sprintf(", sync_mode = $%d", argNum)
		args = append(args, *updates.SyncMode)
		argNum++
	}
	if updates.SyncOnInsert != nil {
		query += fmt.Sprintf(", sync_on_insert = $%d", argNum)
		args = append(args, *updates.SyncOnInsert)
		argNum++
	}
	if updates.SyncOnUpdate != nil {
		query += fmt.Sprintf(", sync_on_update = $%d", argNum)
		args = append(args, *updates.SyncOnUpdate)
		argNum++
	}
	if updates.SyncOnDelete != nil {
		query += fmt.Sprintf(", sync_on_delete = $%d", argNum)
		args = append(args, *updates.SyncOnDelete)
		argNum++
	}
	if updates.DebounceSeconds != nil {
		query += fmt.Sprintf(", debounce_seconds = $%d", argNum)
		args = append(args, *updates.DebounceSeconds)
		argNum++
	}
	if updates.IncludeForeignKeys != nil {
		query += fmt.Sprintf(", include_foreign_keys = $%d", argNum)
		args = append(args, *updates.IncludeForeignKeys)
		argNum++
	}
	if updates.IncludeIndexes != nil {
		query += fmt.Sprintf(", include_indexes = $%d", argNum)
		args = append(args, *updates.IncludeIndexes)
	}

	query += " WHERE id = $1 RETURNING id"

	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, args...).Scan(&id)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update sync config: %w", err)
	}

	return s.GetSyncConfig(ctx, id)
}

// DeleteSyncConfig deletes a sync configuration
func (s *TableExportSyncService) DeleteSyncConfig(ctx context.Context, id string) error {
	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `DELETE FROM ai.table_export_sync_configs WHERE id = $1`
		_, err := tx.Exec(ctx, query, id)
		if err != nil {
			return fmt.Errorf("failed to delete sync config: %w", err)
		}
		return nil
	})
}

// TriggerSync manually triggers a sync for a config
func (s *TableExportSyncService) TriggerSync(ctx context.Context, id string) (*ExportTableResult, error) {
	config, err := s.GetSyncConfig(ctx, id)
	if err != nil {
		return nil, err
	}

	req := ExportTableRequest{
		KnowledgeBaseID:    config.KnowledgeBaseID,
		Schema:             config.SchemaName,
		Table:              config.TableName,
		Columns:            config.Columns,
		IncludeForeignKeys: config.IncludeForeignKeys,
		IncludeIndexes:     config.IncludeIndexes,
	}

	result, err := s.exporter.ExportTable(ctx, req)
	if err != nil {
		// Update sync status to failed
		if updateErr := s.updateSyncStatus(ctx, id, SyncStatusFailed, err.Error()); updateErr != nil {
			log.Error().Err(updateErr).Msg("Failed to update sync status")
		}
		return nil, err
	}

	// Update sync status to success
	if updateErr := s.updateSyncStatus(ctx, id, SyncStatusSuccess, ""); updateErr != nil {
		log.Error().Err(updateErr).Msg("Failed to update sync status")
	}
	return result, nil
}

// updateSyncStatus updates the sync status after a sync operation
func (s *TableExportSyncService) updateSyncStatus(ctx context.Context, id string, status SyncStatus, errMsg string) error {
	query := `
		UPDATE ai.table_export_sync_configs
		SET last_sync_at = NOW(),
			last_sync_status = $2,
			last_sync_error = $3,
			updated_at = NOW()
		WHERE id = $1
	`
	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, id, string(status), errMsg)
		return err
	})
}

// parsePostgresArray parses a PostgreSQL array string into a Go slice
func parsePostgresArray(s string) []string {
	if s == "{}" || s == "" || s == "NULL" {
		return nil
	}

	// Remove surrounding braces
	s = s[1 : len(s)-1]

	// Simple split by comma (doesn't handle quoted strings with commas)
	var result []string
	current := ""
	inQuote := false

	for _, r := range s {
		switch r {
		case '"':
			inQuote = !inQuote
		case ',':
			if !inQuote {
				if current != "" {
					result = append(result, current)
				}
				current = ""
				continue
			}
			fallthrough
		default:
			current += string(r)
		}
	}
	if current != "" {
		result = append(result, current)
	}

	return result
}
