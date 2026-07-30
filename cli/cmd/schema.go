package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

// schemaCmd manages an application's own schema declaratively (the app-developer
// counterpart to 'internal-schema'). It is the opt-in alternative to imperative
// 'migrations sync': an app developer commits a desired-state schema file (e.g.
// public.sql) and syncs it; Fluxbase stores the content and applies the diff via
// pgschema. A given (namespace, schema) should use one mode, not both.
var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Manage declarative app schema",
	Long: `Manage your application's own database schema declaratively.

This is the app-developer counterpart to 'internal-schema' (which manages Fluxbase's
internal schemas). Commit a desired-state SQL file (e.g. fluxbase/schema/public.sql)
and sync it: Fluxbase stores the content and applies the diff to the live database via
pgschema, reconciling drift on every sync.

This is an opt-in alternative to imperative 'migrations sync'. Choose ONE mode per
(namespace, schema): declarative (this command) OR imperative (migrations sync). Do
not run both against the same schema.

Requires database.declarative_app_schema.enabled=true on the Fluxbase server for
startup auto-apply; sync/plan/apply work on demand regardless.

Examples:
  fluxbase schema sync --dir fluxbase/schema --namespace wayli
  fluxbase schema sync --dir fluxbase/schema --namespace wayli --no-apply
  fluxbase schema status --namespace wayli
  fluxbase schema plan --namespace wayli
  fluxbase schema validate --namespace wayli`,
}

var (
	schemaNamespace   string
	schemaName        string
	schemaSyncDir     string
	schemaNoApply     bool
	schemaAllowDest   bool
	schemaFailOnDrift bool
)

var schemaSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync declarative app schema content",
	Long: `Read a desired-state schema file and sync it to Fluxbase.

Reads <dir>/<schema>.sql (default <dir>/public.sql), computes a fingerprint, and
stores the content. If the content changed (or --apply is set), Fluxbase applies the
diff to the live database immediately. Re-syncing unchanged content is a no-op
(unless --apply). Drift introduced outside Fluxbase is reconciled on every sync.

Examples:
  fluxbase schema sync --dir fluxbase/schema --namespace wayli
  fluxbase schema sync --dir fluxbase/schema --namespace wayli --schema public
  fluxbase schema sync --dir fluxbase/schema --namespace wayli --no-apply`,
	PreRunE: requireAuth,
	RunE:    runSchemaSync,
}

var schemaStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show app schema status",
	Long: `Show the status of declarative app schema(s).

With --namespace, shows a single schema's stored fingerprint and last-applied state.
Without it, lists all stored app schemas.

Examples:
  fluxbase schema status
  fluxbase schema status --namespace wayli`,
	PreRunE: requireAuth,
	RunE:    runSchemaStatus,
}

var schemaPlanCmd = &cobra.Command{
	Use:   "plan",
	Short: "Show pending app schema changes",
	Long: `Compare stored schema content to the live database and show what would change.
Does not modify the database.

Example:
  fluxbase schema plan --namespace wayli`,
	PreRunE: requireAuth,
	RunE:    runSchemaPlan,
}

var schemaValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Check for app schema drift",
	Long: `Validate that the live database matches the stored schema content.
Useful for CI/CD pipelines. Use --fail-on-drift to exit non-zero on drift.

Example:
  fluxbase schema validate --namespace wayli --fail-on-drift`,
	PreRunE: requireAuth,
	RunE:    runSchemaValidate,
}

var schemaApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply stored app schema",
	Long: `Apply the already-stored schema content for a namespace (reconcile drift).

Example:
  fluxbase schema apply --namespace wayli`,
	PreRunE: requireAuth,
	RunE:    runSchemaApply,
}

func init() {
	rootCmd.AddCommand(schemaCmd)
	schemaCmd.AddCommand(schemaSyncCmd)
	schemaCmd.AddCommand(schemaStatusCmd)
	schemaCmd.AddCommand(schemaPlanCmd)
	schemaCmd.AddCommand(schemaValidateCmd)
	schemaCmd.AddCommand(schemaApplyCmd)

	// Persistent flags shared across schema subcommands.
	schemaCmd.PersistentFlags().StringVar(&schemaNamespace, "namespace", "", "App namespace (e.g. wayli)")
	schemaCmd.PersistentFlags().StringVar(&schemaName, "schema", "public", "PostgreSQL schema name (default public)")

	// sync flags
	schemaSyncCmd.Flags().StringVar(&schemaSyncDir, "dir", "", "Directory containing the schema file (default ./schema)")
	schemaSyncCmd.Flags().BoolVar(&schemaNoApply, "no-apply", false, "Store content only; do not apply")
	schemaSyncCmd.Flags().BoolVar(&schemaAllowDest, "allow-destructive", false, "Permit destructive changes during apply")

	// validate flags
	schemaValidateCmd.Flags().BoolVar(&schemaFailOnDrift, "fail-on-drift", false, "Exit non-zero if drift is detected")
}

func runSchemaSync(cmd *cobra.Command, args []string) error {
	if schemaNamespace == "" {
		return fmt.Errorf("--namespace is required")
	}

	// Resolve schema file path: <dir>/<schema>.sql
	dir, err := detectResourceDir("schema", schemaSyncDir, "./schema")
	if err != nil {
		return err
	}
	schemaFile := filepath.Join(dir, effectiveSchemaName(schemaName)+".sql")

	content, err := os.ReadFile(schemaFile) //nolint:gosec // CLI reads a user-provided path
	if err != nil {
		return fmt.Errorf("failed to read schema file %s: %w", schemaFile, err)
	}
	if len(content) == 0 {
		return fmt.Errorf("schema file %s is empty", schemaFile)
	}

	// Optional .pgschemaignore in the same directory; suppresses diffs on objects
	// pgschema shouldn't manage (e.g. extension-member objects). Sent alongside the
	// schema so Fluxbase writes it next to the temp file and pgschema discovers it.
	ignoreFile := filepath.Join(dir, ".pgschemaignore")
	ignoreContent, _ := os.ReadFile(ignoreFile) //nolint:gosec // optional file

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	body := map[string]interface{}{
		"namespace":      schemaNamespace,
		"schema":         effectiveSchemaName(schemaName),
		"content":        string(content),
		"ignore_content": string(ignoreContent),
		"no_apply":       schemaNoApply,
		"apply":          !schemaNoApply,
	}
	_ = schemaAllowDest // destructive allowance is a server-side config; flagged through for future use

	var result map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/admin/app-schema/sync", body, &result); err != nil {
		return err
	}

	changed, _ := result["changed"].(bool)
	fingerprint, _ := result["fingerprint"].(string)
	applied, _ := result["applied"].(bool)

	if changed {
		fmt.Printf("Schema updated: %s/%s\n", schemaNamespace, effectiveSchemaName(schemaName))
	} else {
		fmt.Printf("Schema unchanged: %s/%s\n", schemaNamespace, effectiveSchemaName(schemaName))
	}
	if len(fingerprint) >= 12 {
		fmt.Printf("Fingerprint: %s\n", fingerprint[:12])
	}
	if applied {
		changes, _ := result["changes"].(float64)
		duration, _ := result["duration"].(string)
		fmt.Printf("Applied %d change(s) in %s\n", int(changes), duration)
	} else if schemaNoApply {
		fmt.Println("Stored only (--no-apply).")
	}
	return nil
}

func runSchemaStatus(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := url.Values{}
	if schemaNamespace != "" {
		query.Set("namespace", schemaNamespace)
	}
	query.Set("schema", effectiveSchemaName(schemaName))

	var result map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/admin/app-schema/status", query, &result); err != nil {
		return err
	}

	enabled, _ := result["enabled"].(bool)
	fmt.Println("App Schema Status:")
	fmt.Printf("  Feature enabled: %v\n", enabled)
	fmt.Println()

	if status, ok := result["status"].(map[string]interface{}); ok {
		printAppSchemaStatus(status)
	} else if schemas, ok := result["schemas"].([]interface{}); ok {
		if len(schemas) == 0 {
			fmt.Println("No app schemas stored.")
			return nil
		}
		for _, s := range schemas {
			if m, ok := s.(map[string]interface{}); ok {
				printAppSchemaStatus(m)
			}
		}
	}
	return nil
}

func runSchemaPlan(cmd *cobra.Command, args []string) error {
	if schemaNamespace == "" {
		return fmt.Errorf("--namespace is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	body := map[string]interface{}{
		"namespace": schemaNamespace,
		"schema":    effectiveSchemaName(schemaName),
	}
	var result map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/admin/app-schema/plan", body, &result); err != nil {
		return err
	}

	plan, ok := result["plan"].(map[string]interface{})
	if !ok {
		fmt.Println("No plan returned.")
		return nil
	}

	changes, _ := plan["changes"].([]interface{})
	if len(changes) == 0 {
		fmt.Println("No changes detected - database matches stored schema content.")
		return nil
	}

	fmt.Printf("Found %d pending change(s):\n\n", len(changes))
	for i, change := range changes {
		c, ok := change.(map[string]interface{})
		if !ok {
			continue
		}
		destructive := ""
		if d, _ := c["destructive"].(bool); d {
			destructive = " [DESTRUCTIVE]"
		}
		fmt.Printf("  %d. %s %s.%s (%s)%s\n",
			i+1, c["type"], c["schema"], c["name"], c["object_type"], destructive)
	}

	if summary, ok := plan["summary"].(map[string]interface{}); ok {
		fmt.Println()
		fmt.Println("Summary:")
		fmt.Printf("  Total changes: %v\n", summary["total_changes"])
		fmt.Printf("  Destructive:   %v\n", summary["destructive_count"])
	}
	return nil
}

func runSchemaValidate(cmd *cobra.Command, args []string) error {
	if schemaNamespace == "" {
		return fmt.Errorf("--namespace is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	query := url.Values{}
	query.Set("namespace", schemaNamespace)
	query.Set("schema", effectiveSchemaName(schemaName))

	var result map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/admin/app-schema/validate", query, &result); err != nil {
		return err
	}

	valid, _ := result["valid"].(bool)
	if valid {
		fmt.Println("App schema is valid - no drift detected.")
		return nil
	}

	drifts, _ := result["drifts"].([]interface{})
	fmt.Printf("App schema drift detected (%d change(s)):\n", len(drifts))
	for _, drift := range drifts {
		d, ok := drift.(map[string]interface{})
		if !ok {
			continue
		}
		destructive := ""
		if desc, _ := d["destructive"].(bool); desc {
			destructive = " [DESTRUCTIVE]"
		}
		fmt.Printf("  - %s %s.%s (%s)%s\n", d["type"], d["schema"], d["name"], d["object_type"], destructive)
	}

	if schemaFailOnDrift {
		return fmt.Errorf("app schema drift detected")
	}
	return nil
}

func runSchemaApply(cmd *cobra.Command, args []string) error {
	if schemaNamespace == "" {
		return fmt.Errorf("--namespace is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	body := map[string]interface{}{
		"namespace": schemaNamespace,
		"schema":    effectiveSchemaName(schemaName),
	}
	var result map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/admin/app-schema/apply", body, &result); err != nil {
		return err
	}

	applied, _ := result["applied"].(float64)
	duration, _ := result["duration"].(string)
	fmt.Printf("Applied %d change(s) in %s\n", int(applied), duration)
	return nil
}

// effectiveSchemaName defaults an empty --schema to "public".
func effectiveSchemaName(s string) string {
	if s == "" {
		return "public"
	}
	return s
}

// printAppSchemaStatus prints a single status record.
func printAppSchemaStatus(m map[string]interface{}) {
	ns, _ := m["namespace"].(string)
	schema, _ := m["schema_name"].(string)
	if schema == "" {
		schema, _ = m["schema"].(string)
	}
	fmt.Printf("  %s/%s\n", ns, schema)
	if fp, _ := m["schema_fingerprint"].(string); len(fp) >= 12 {
		fmt.Printf("    Fingerprint:     %s\n", fp[:12])
	}
	if has, _ := m["has_stored_schema"].(bool); has {
		fmt.Printf("    Stored:          yes\n")
	} else if hasFalse, present := m["has_stored_schema"]; present && hasFalse == false {
		fmt.Printf("    Stored:          no\n")
	}
	if last, _ := m["last_applied_fingerprint"].(string); len(last) >= 12 {
		fmt.Printf("    Last applied:    %s\n", last[:12])
	}
}
