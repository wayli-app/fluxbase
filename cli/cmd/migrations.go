package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/fluxbase-eu/fluxbase/cli/output"
)

var migrationsCmd = &cobra.Command{
	Use:     "migrations",
	Aliases: []string{"migration", "migrate"},
	Short:   "Manage database migrations",
	Long:    `Create, apply, and manage database migrations.`,
}

var (
	migNamespace string
	migUpSQL     string
	migDownSQL   string
	migSyncDir   string
	migNoApply bool
	migDryRun    bool
)

var migrationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all migrations",
	Long: `List all database migrations.

Examples:
  fluxbase migrations list
  fluxbase migrations list --namespace default`,
	PreRunE: requireAuth,
	RunE:    runMigrationsList,
}

var migrationsGetCmd = &cobra.Command{
	Use:   "get [name]",
	Short: "Get migration details",
	Long: `Get details of a specific migration.

Examples:
  fluxbase migrations get 001_create_users`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runMigrationsGet,
}

var migrationsCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new migration",
	Long: `Create a new database migration.

Examples:
  fluxbase migrations create create_users --up-sql "CREATE TABLE users..." --down-sql "DROP TABLE users"`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runMigrationsCreate,
}

var migrationsApplyCmd = &cobra.Command{
	Use:   "apply [name]",
	Short: "Apply a migration",
	Long: `Apply a specific migration.

Examples:
  fluxbase migrations apply 001_create_users`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runMigrationsApply,
}

var migrationsRollbackCmd = &cobra.Command{
	Use:   "rollback [name]",
	Short: "Rollback a migration",
	Long: `Rollback a specific migration.

Examples:
  fluxbase migrations rollback 001_create_users`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runMigrationsRollback,
}

var migrationsApplyPendingCmd = &cobra.Command{
	Use:   "apply-pending",
	Short: "Apply all pending migrations",
	Long: `Apply all pending migrations in order.

Examples:
  fluxbase migrations apply-pending`,
	PreRunE: requireAuth,
	RunE:    runMigrationsApplyPending,
}

var migrationsSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync migrations from a directory",
	Long: `Sync database migrations from a directory.

Migration files should follow the naming convention:
  001_migration_name.up.sql
  001_migration_name.down.sql

Examples:
  fluxbase migrations sync --dir ./migrations
  fluxbase migrations sync --dir ./migrations --no-apply`,
	PreRunE: requireAuth,
	RunE:    runMigrationsSync,
}

func init() {
	// List flags
	migrationsListCmd.Flags().StringVar(&migNamespace, "namespace", "", "Filter by namespace")

	// Create flags
	migrationsCreateCmd.Flags().StringVar(&migUpSQL, "up-sql", "", "Up migration SQL")
	migrationsCreateCmd.Flags().StringVar(&migDownSQL, "down-sql", "", "Down migration SQL")
	migrationsCreateCmd.Flags().StringVar(&migNamespace, "namespace", "default", "Migration namespace")

	// Sync flags
	migrationsSyncCmd.Flags().StringVar(&migSyncDir, "dir", "./migrations", "Directory containing migration files")
	migrationsSyncCmd.Flags().StringVar(&migNamespace, "namespace", "default", "Target namespace")
	migrationsSyncCmd.Flags().BoolVar(&migNoApply, "no-apply", false, "Do not apply migrations after sync")
	migrationsSyncCmd.Flags().BoolVar(&migDryRun, "dry-run", false, "Preview changes without applying")

	migrationsCmd.AddCommand(migrationsListCmd)
	migrationsCmd.AddCommand(migrationsGetCmd)
	migrationsCmd.AddCommand(migrationsCreateCmd)
	migrationsCmd.AddCommand(migrationsApplyCmd)
	migrationsCmd.AddCommand(migrationsRollbackCmd)
	migrationsCmd.AddCommand(migrationsApplyPendingCmd)
	migrationsCmd.AddCommand(migrationsSyncCmd)
}

func runMigrationsList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := url.Values{}
	if migNamespace != "" {
		query.Set("namespace", migNamespace)
	}

	var migrations []map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/admin/migrations", query, &migrations); err != nil {
		return err
	}

	if len(migrations) == 0 {
		fmt.Println("No migrations found.")
		return nil
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		data := output.TableData{
			Headers: []string{"NAME", "NAMESPACE", "STATUS", "APPLIED AT"},
			Rows:    make([][]string, len(migrations)),
		}

		for i, mig := range migrations {
			name := getStringValue(mig, "name")
			namespace := getStringValue(mig, "namespace")
			appliedAt := getStringValue(mig, "applied_at")
			status := "pending"
			if appliedAt != "" {
				status = "applied"
			}

			data.Rows[i] = []string{name, namespace, status, appliedAt}
		}

		formatter.PrintTable(data)
	} else {
		if err := formatter.Print(migrations); err != nil {
			return err
		}
	}

	return nil
}

func runMigrationsGet(cmd *cobra.Command, args []string) error {
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var migration map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/admin/migrations/"+url.PathEscape(name), nil, &migration); err != nil {
		return err
	}

	formatter := GetFormatter()
	return formatter.Print(migration)
}

func runMigrationsCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	body := map[string]interface{}{
		"name":      name,
		"namespace": migNamespace,
	}

	if migUpSQL != "" {
		body["up_sql"] = migUpSQL
	}
	if migDownSQL != "" {
		body["down_sql"] = migDownSQL
	}

	var result map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/admin/migrations", body, &result); err != nil {
		return err
	}

	fmt.Printf("Migration '%s' created.\n", name)
	return nil
}

func runMigrationsApply(cmd *cobra.Command, args []string) error {
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := apiClient.DoPost(ctx, "/api/v1/admin/migrations/"+url.PathEscape(name)+"/apply", nil, nil); err != nil {
		return err
	}

	fmt.Printf("Migration '%s' applied successfully.\n", name)
	return nil
}

func runMigrationsRollback(cmd *cobra.Command, args []string) error {
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := apiClient.DoPost(ctx, "/api/v1/admin/migrations/"+url.PathEscape(name)+"/rollback", nil, nil); err != nil {
		return err
	}

	fmt.Printf("Migration '%s' rolled back successfully.\n", name)
	return nil
}

func runMigrationsApplyPending(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	var result map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/admin/migrations/apply-pending", nil, &result); err != nil {
		return err
	}

	applied := getIntValue(result, "applied")
	fmt.Printf("Applied %d pending migrations.\n", applied)
	return nil
}

func runMigrationsSync(cmd *cobra.Command, args []string) error {
	// Auto-detect directory if not explicitly specified
	dir, err := detectResourceDir("migrations", migSyncDir, "./migrations")
	if err != nil {
		return err
	}
	migSyncDir = dir

	// Read migration files
	entries, err := os.ReadDir(migSyncDir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	// Group up and down files
	migrations := make(map[string]map[string]string)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		var migName string
		var sqlType string

		if strings.HasSuffix(name, ".up.sql") {
			migName = strings.TrimSuffix(name, ".up.sql")
			sqlType = "up"
		} else if strings.HasSuffix(name, ".down.sql") {
			migName = strings.TrimSuffix(name, ".down.sql")
			sqlType = "down"
		} else {
			continue
		}

		content, err := os.ReadFile(filepath.Join(migSyncDir, name)) //nolint:gosec // CLI tool reads user-provided file path
		if err != nil {
			fmt.Printf("Warning: failed to read %s: %v\n", name, err)
			continue
		}

		if migrations[migName] == nil {
			migrations[migName] = make(map[string]string)
		}
		migrations[migName][sqlType] = string(content)
	}

	if len(migrations) == 0 {
		fmt.Println("No migration files found.")
		return nil
	}

	if migDryRun {
		fmt.Println("Dry run - would sync the following migrations:")
		for name := range migrations {
			fmt.Printf("  - %s\n", name)
		}
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Build migrations array
	var migList []map[string]interface{}
	for name, sqls := range migrations {
		mig := map[string]interface{}{
			"name": name,
		}
		if up, ok := sqls["up"]; ok {
			mig["up_sql"] = up
		}
		if down, ok := sqls["down"]; ok {
			mig["down_sql"] = down
		}
		migList = append(migList, mig)
	}

	body := map[string]interface{}{
		"namespace":  migNamespace,
		"migrations": migList,
		"options": map[string]interface{}{
			"auto_apply": !migNoApply,
		},
	}

	var result map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/admin/migrations/sync", body, &result); err != nil {
		return err
	}

	// Parse the nested summary response
	summary, ok := result["summary"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected response format from server")
	}

	created := getIntValue(summary, "created")
	updated := getIntValue(summary, "updated")
	applied := getIntValue(summary, "applied")
	errors := getIntValue(summary, "errors")

	fmt.Printf("Synced migrations: %d created, %d updated.\n", created, updated)
	if !migNoApply && applied > 0 {
		fmt.Printf("Applied %d pending migrations.\n", applied)
	}
	if errors > 0 {
		fmt.Printf("Warning: %d errors occurred during sync.\n", errors)
		// Print error details if available
		if errorList, ok := result["errors"].([]interface{}); ok {
			for _, e := range errorList {
				fmt.Printf("  - %v\n", e)
			}
		}
	}

	return nil
}
