package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/fluxbase-eu/fluxbase/cli/annotations"
	"github.com/fluxbase-eu/fluxbase/cli/bundler"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync all Fluxbase resources",
	Long: `Sync all resource types (functions, jobs, rpc, chatbots, migrations) from a directory.

Automatically detects resource subdirectories and syncs each one.

Examples:
  fluxbase sync                           # Auto-detect from ./fluxbase/ or current dir
  fluxbase sync --dir ./src               # Specify root directory
  fluxbase sync --namespace production    # Apply namespace to all
  fluxbase sync --dry-run                 # Preview all changes`,
	PreRunE: requireAuth,
	RunE:    runSync,
}

var (
	syncRootDir   string
	syncNamespace string
	syncDryRun    bool
	syncKeep      bool
)

func init() {
	syncCmd.Flags().StringVar(&syncRootDir, "dir", "", "Root directory (default: ./fluxbase or current dir)")
	syncCmd.Flags().StringVar(&syncNamespace, "namespace", "default", "Target namespace for all resources")
	syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "Preview changes without applying")
	syncCmd.Flags().BoolVar(&syncKeep, "keep", false, "Keep items not present in directory")
}

// detectResourceDir finds the directory for a resource type
// Assumes user is either in the right directory or one level above (with fluxbase/ subfolder)
func detectResourceDir(resourceType string, explicitDir string, defaultDir string) (string, error) {
	// If explicit dir matches the default, treat as "not specified" for auto-detection
	if explicitDir != "" && explicitDir != defaultDir {
		// User specified explicit directory
		if _, err := os.Stat(explicitDir); os.IsNotExist(err) {
			return "", fmt.Errorf("directory not found: %s", explicitDir)
		}
		return explicitDir, nil
	}

	// Check if we're in the resource directory itself (e.g., current dir IS "functions")
	cwd, _ := os.Getwd()
	if filepath.Base(cwd) == resourceType {
		return ".", nil
	}

	// Auto-detection: look for fluxbase/<resource>/ first, then ./<resource>/
	candidates := []string{
		filepath.Join("fluxbase", resourceType), // ./fluxbase/functions/
		resourceType,                            // ./functions/
	}

	for _, dir := range candidates {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir, nil
		}
	}

	return "", fmt.Errorf("no %s directory found (looked for ./fluxbase/%s/ and ./%s/)",
		resourceType, resourceType, resourceType)
}

func runSync(cmd *cobra.Command, args []string) error {
	// Detect root directory
	rootDir := syncRootDir
	if rootDir == "" {
		if info, err := os.Stat("fluxbase"); err == nil && info.IsDir() {
			rootDir = "fluxbase"
		} else {
			rootDir = "."
		}
	}

	fmt.Printf("Syncing Fluxbase resources from %s...\n\n", rootDir)

	// Resource types in sync order (dependencies first)
	type resourceSync struct {
		name    string
		dirName string
		syncFn  func(ctx context.Context, dir, namespace string, dryRun, deleteMissing bool) error
	}

	resources := []resourceSync{
		{"RPC procedures", "rpc", syncRPCFromDir},
		{"Migrations", "migrations", syncMigrationsFromDir},
		{"Functions", "functions", syncFunctionsFromDir},
		{"Jobs", "jobs", syncJobsFromDir},
		{"Chatbots", "chatbots", syncChatbotsFromDir},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Increase HTTP client timeout for sync operations (large bundled payloads)
	originalTimeout := apiClient.HTTPClient.Timeout
	apiClient.HTTPClient.Timeout = 5 * time.Minute
	defer func() { apiClient.HTTPClient.Timeout = originalTimeout }()

	foundAny := false
	for _, res := range resources {
		dir := filepath.Join(rootDir, res.dirName)
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			foundAny = true
			fmt.Printf("Syncing %s from %s...\n", res.name, dir)
			if err := res.syncFn(ctx, dir, syncNamespace, syncDryRun, !syncKeep); err != nil {
				return fmt.Errorf("failed to sync %s: %w", res.name, err)
			}
			fmt.Println()
		}
	}

	if !foundAny {
		return fmt.Errorf("no resource directories found in %s (expected: rpc/, migrations/, functions/, jobs/, chatbots/)", rootDir)
	}

	fmt.Println("Sync completed successfully.")
	return nil
}

// syncRPCFromDir syncs RPC procedures from a directory
func syncRPCFromDir(ctx context.Context, dir, namespace string, dryRun, deleteMissing bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	var procedures []map[string]interface{}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		content, err := os.ReadFile(filepath.Join(dir, name)) //nolint:gosec
		if err != nil {
			fmt.Printf("  Warning: failed to read %s: %v\n", name, err)
			continue
		}

		procName := strings.TrimSuffix(name, ".sql")
		procedures = append(procedures, map[string]interface{}{
			"name": procName,
			"code": string(content),
		})
	}

	if len(procedures) == 0 {
		fmt.Println("  No SQL files found.")
		return nil
	}

	if dryRun {
		fmt.Println("  Dry run - would sync:")
		for _, proc := range procedures {
			fmt.Printf("    - %s\n", proc["name"])
		}
		return nil
	}

	body := map[string]interface{}{
		"namespace":  namespace,
		"procedures": procedures,
		"options": map[string]interface{}{
			"delete_missing": deleteMissing,
		},
	}

	var result map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/admin/rpc/sync", body, &result); err != nil {
		return err
	}

	printSyncSummary(result, "procedures")
	return nil
}

// syncMigrationsFromDir syncs migrations from a directory
func syncMigrationsFromDir(ctx context.Context, dir, namespace string, dryRun, _ bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

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

		content, err := os.ReadFile(filepath.Join(dir, name)) //nolint:gosec
		if err != nil {
			fmt.Printf("  Warning: failed to read %s: %v\n", name, err)
			continue
		}

		if migrations[migName] == nil {
			migrations[migName] = make(map[string]string)
		}
		migrations[migName][sqlType] = string(content)
	}

	if len(migrations) == 0 {
		fmt.Println("  No migration files found.")
		return nil
	}

	if dryRun {
		fmt.Println("  Dry run - would sync:")
		for name := range migrations {
			fmt.Printf("    - %s\n", name)
		}
		return nil
	}

	// Get sorted list of migration names (important for sequential application)
	migNames := make([]string, 0, len(migrations))
	for name := range migrations {
		migNames = append(migNames, name)
	}
	sort.Strings(migNames)

	// Build migrations array in sorted order
	var migList []map[string]interface{}
	for _, name := range migNames {
		sqls := migrations[name]
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
		"namespace":  namespace,
		"migrations": migList,
		"options": map[string]interface{}{
			"auto_apply": true,
		},
	}

	var result map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/admin/migrations/sync", body, &result); err != nil {
		return err
	}

	printSyncSummary(result, "migrations")
	return nil
}

// syncFunctionsFromDir syncs functions from a directory
func syncFunctionsFromDir(ctx context.Context, dir, namespace string, dryRun, deleteMissing bool) error {
	// Check for _shared directory first
	sharedDir := filepath.Join(dir, "_shared")
	sharedModulesMap := make(map[string]string)
	if info, err := os.Stat(sharedDir); err == nil && info.IsDir() {
		sharedEntries, err := os.ReadDir(sharedDir)
		if err == nil {
			for _, entry := range sharedEntries {
				if entry.IsDir() {
					continue
				}
				name := entry.Name()
				if !strings.HasSuffix(name, ".ts") && !strings.HasSuffix(name, ".js") {
					continue
				}
				content, err := os.ReadFile(filepath.Join(sharedDir, name)) //nolint:gosec
				if err == nil {
					sharedModulesMap["_shared/"+name] = string(content)
				}
			}
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	var functions []map[string]interface{}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".ts") && !strings.HasSuffix(name, ".js") {
			continue
		}

		content, err := os.ReadFile(filepath.Join(dir, name)) //nolint:gosec
		if err != nil {
			fmt.Printf("  Warning: failed to read %s: %v\n", name, err)
			continue
		}

		fnName := strings.TrimSuffix(strings.TrimSuffix(name, ".ts"), ".js")

		// Parse @fluxbase: annotations BEFORE bundling (esbuild strips comments)
		fn := map[string]interface{}{
			"name": fnName,
			"code": string(content),
		}
		config := annotations.ParseFunctionAnnotations(string(content))
		annotations.ApplyFunctionConfig(fn, config)

		functions = append(functions, fn)
	}

	if len(functions) == 0 {
		fmt.Println("  No function files found.")
		return nil
	}

	if dryRun {
		fmt.Println("  Dry run - would sync:")
		for _, fn := range functions {
			fmt.Printf("    - %s\n", fn["name"])
		}
		return nil
	}

	// Try to bundle if needed
	needsBundling := false
	for _, fn := range functions {
		code := fn["code"].(string)
		if strings.Contains(code, "import ") {
			needsBundling = true
			break
		}
	}

	if needsBundling {
		b, err := bundler.NewBundler(dir)
		if err != nil {
			fmt.Println("  Note: Deno not available for local bundling. Server will bundle functions.")
		} else {
			for i, fn := range functions {
				code := fn["code"].(string)
				fnName := fn["name"].(string)

				if !b.NeedsBundle(code) {
					continue
				}

				if err := b.ValidateImports(code); err != nil {
					return fmt.Errorf("function %s: %w", fnName, err)
				}

				fmt.Printf("  Bundling %s...", fnName)
				result, err := b.Bundle(ctx, code, sharedModulesMap)
				if err != nil {
					fmt.Println()
					return fmt.Errorf("failed to bundle %s: %w", fnName, err)
				}
				fmt.Printf(" %s → %s\n", formatBytes(len(code)), formatBytes(len(result.BundledCode)))
				functions[i]["code"] = result.BundledCode
				functions[i]["is_bundled"] = true
			}
		}
	}

	body := map[string]interface{}{
		"namespace": namespace,
		"functions": functions,
		"options": map[string]interface{}{
			"delete_missing": deleteMissing,
		},
	}

	var result map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/admin/functions/sync", body, &result); err != nil {
		return err
	}

	printSyncSummary(result, "functions")
	return nil
}

// syncJobsFromDir syncs jobs from a directory
func syncJobsFromDir(ctx context.Context, dir, namespace string, dryRun, deleteMissing bool) error {
	// Check for _shared directory first
	sharedDir := filepath.Join(dir, "_shared")
	sharedModulesMap := make(map[string]string)
	if info, err := os.Stat(sharedDir); err == nil && info.IsDir() {
		_ = filepath.WalkDir(sharedDir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			name := d.Name()
			if !strings.HasSuffix(name, ".ts") && !strings.HasSuffix(name, ".js") &&
				!strings.HasSuffix(name, ".json") && !strings.HasSuffix(name, ".geojson") {
				return nil
			}
			content, err := os.ReadFile(path) //nolint:gosec
			if err != nil {
				return nil
			}
			relPath, _ := filepath.Rel(dir, path)
			relPath = filepath.ToSlash(relPath)
			sharedModulesMap[relPath] = string(content)
			return nil
		})
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	var jobs []map[string]interface{}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".ts") && !strings.HasSuffix(name, ".js") {
			continue
		}

		content, err := os.ReadFile(filepath.Join(dir, name)) //nolint:gosec
		if err != nil {
			fmt.Printf("  Warning: failed to read %s: %v\n", name, err)
			continue
		}

		jobName := strings.TrimSuffix(strings.TrimSuffix(name, ".ts"), ".js")

		// Parse @fluxbase: annotations BEFORE bundling (esbuild strips comments)
		job := map[string]interface{}{
			"name": jobName,
			"code": string(content),
		}
		jobConfig := annotations.ParseJobAnnotations(string(content))
		annotations.ApplyJobConfig(job, jobConfig)

		jobs = append(jobs, job)
	}

	if len(jobs) == 0 {
		fmt.Println("  No job files found.")
		return nil
	}

	if dryRun {
		fmt.Println("  Dry run - would sync:")
		for _, job := range jobs {
			fmt.Printf("    - %s\n", job["name"])
		}
		return nil
	}

	// Try to bundle if needed
	needsBundling := false
	for _, job := range jobs {
		code := job["code"].(string)
		if strings.Contains(code, "import ") {
			needsBundling = true
			break
		}
	}

	if needsBundling {
		b, err := bundler.NewBundler(dir)
		if err != nil {
			fmt.Println("  Note: Deno not available for local bundling. Server will bundle jobs.")
		} else {
			for i, job := range jobs {
				code := job["code"].(string)
				jobName := job["name"].(string)

				if !b.NeedsBundle(code) {
					continue
				}

				if err := b.ValidateImports(code); err != nil {
					return fmt.Errorf("job %s: %w", jobName, err)
				}

				fmt.Printf("  Bundling %s...", jobName)
				result, err := b.Bundle(ctx, code, sharedModulesMap)
				if err != nil {
					fmt.Println()
					return fmt.Errorf("failed to bundle %s: %w", jobName, err)
				}
				fmt.Printf(" %s → %s\n", formatBytes(len(code)), formatBytes(len(result.BundledCode)))
				jobs[i]["code"] = result.BundledCode
				jobs[i]["is_bundled"] = true
			}
		}
	}

	body := map[string]interface{}{
		"namespace": namespace,
		"jobs":      jobs,
		"options": map[string]interface{}{
			"delete_missing": deleteMissing,
		},
	}

	var result map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/admin/jobs/sync", body, &result); err != nil {
		return err
	}

	printSyncSummary(result, "jobs")
	return nil
}

// syncChatbotsFromDir syncs chatbots from a directory
func syncChatbotsFromDir(ctx context.Context, dir, namespace string, dryRun, deleteMissing bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	var chatbots []map[string]interface{}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}

		content, err := os.ReadFile(filepath.Join(dir, name)) //nolint:gosec
		if err != nil {
			fmt.Printf("  Warning: failed to read %s: %v\n", name, err)
			continue
		}

		cbName := strings.TrimSuffix(strings.TrimSuffix(name, ".yaml"), ".yml")
		chatbots = append(chatbots, map[string]interface{}{
			"name": cbName,
			"code": string(content),
		})
	}

	if len(chatbots) == 0 {
		fmt.Println("  No chatbot YAML files found.")
		return nil
	}

	if dryRun {
		fmt.Println("  Dry run - would sync:")
		for _, cb := range chatbots {
			fmt.Printf("    - %s\n", cb["name"])
		}
		return nil
	}

	body := map[string]interface{}{
		"namespace": namespace,
		"chatbots":  chatbots,
		"options": map[string]interface{}{
			"delete_missing": deleteMissing,
		},
	}

	var result map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/admin/ai/chatbots/sync", body, &result); err != nil {
		return err
	}

	printSyncSummary(result, "chatbots")
	return nil
}

// printSyncSummary prints a standardized sync result summary
func printSyncSummary(result map[string]interface{}, resourceType string) {
	if summary, ok := result["summary"].(map[string]interface{}); ok {
		created := getIntValue(summary, "created")
		updated := getIntValue(summary, "updated")
		deleted := getIntValue(summary, "deleted")
		unchanged := getIntValue(summary, "unchanged")
		errors := getIntValue(summary, "errors")

		fmt.Printf("  %d created, %d updated, %d deleted, %d unchanged",
			created, updated, deleted, unchanged)
		if errors > 0 {
			fmt.Printf(", %d errors", errors)
		}
		fmt.Println()
	}

	// Print any errors from "errors" field (legacy format)
	if errs, ok := result["errors"].([]interface{}); ok && len(errs) > 0 {
		fmt.Println("  Errors:")
		for _, e := range errs {
			switch err := e.(type) {
			case string:
				// Simple string error
				fmt.Printf("    - %s\n", err)
			case map[string]interface{}:
				// Object with name/error fields
				name := getStringValue(err, "name")
				if name == "" {
					name = getStringValue(err, "procedure")
				}
				errMsg := getStringValue(err, "error")
				if name != "" && errMsg != "" {
					fmt.Printf("    - %s: %s\n", name, errMsg)
				} else if errMsg != "" {
					fmt.Printf("    - %s\n", errMsg)
				} else {
					fmt.Printf("    - %v\n", err)
				}
			default:
				fmt.Printf("    - %v\n", e)
			}
		}
	}

	// Print any errors from "details.errors" field (migrations sync format)
	if details, ok := result["details"].(map[string]interface{}); ok {
		if errs, ok := details["errors"].([]interface{}); ok && len(errs) > 0 {
			fmt.Println("  Errors:")
			for _, e := range errs {
				if errStr, ok := e.(string); ok {
					fmt.Printf("    - %s\n", errStr)
				} else {
					fmt.Printf("    - %v\n", e)
				}
			}
		}
	}
}

// Ensure url package is used (for compatibility with other sync commands)
var _ = url.PathEscape
