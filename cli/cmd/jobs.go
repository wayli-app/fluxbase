package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/fluxbase-eu/fluxbase/cli/annotations"
	"github.com/fluxbase-eu/fluxbase/cli/bundler"
	"github.com/fluxbase-eu/fluxbase/cli/output"
)

var jobsCmd = &cobra.Command{
	Use:     "jobs",
	Aliases: []string{"job"},
	Short:   "Manage background jobs",
	Long:    `Submit, monitor, and manage background jobs.`,
}

var (
	jobNamespace   string
	jobPayload     string
	jobPayloadFile string
	jobPriority    int
	jobSchedule    string
	jobTail        int
	jobFollow      bool
	jobSyncDir     string
	jobDryRun      bool
	jobKeep        bool
)

var jobsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List job functions",
	Long: `List all job functions.

Examples:
  fluxbase jobs list
  fluxbase jobs list --namespace production`,
	PreRunE: requireAuth,
	RunE:    runJobsList,
}

var jobsSubmitCmd = &cobra.Command{
	Use:   "submit [name]",
	Short: "Submit a job for execution",
	Long: `Submit a background job for execution.

Examples:
  fluxbase jobs submit my-job
  fluxbase jobs submit my-job --payload '{"key": "value"}'
  fluxbase jobs submit my-job --file ./payload.json`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runJobsSubmit,
}

var jobsStatusCmd = &cobra.Command{
	Use:   "status [id]",
	Short: "Get job execution status",
	Long: `Get the status of a job execution.

Examples:
  fluxbase jobs status abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runJobsStatus,
}

var jobsCancelCmd = &cobra.Command{
	Use:   "cancel [id]",
	Short: "Cancel a running job",
	Long: `Cancel a pending or running job.

Examples:
  fluxbase jobs cancel abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runJobsCancel,
}

var jobsRetryCmd = &cobra.Command{
	Use:   "retry [id]",
	Short: "Retry a failed job",
	Long: `Retry a failed job execution.

Examples:
  fluxbase jobs retry abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runJobsRetry,
}

var jobsLogsCmd = &cobra.Command{
	Use:   "logs [id]",
	Short: "View job execution logs",
	Long: `View logs for a job execution.

Examples:
  fluxbase jobs logs abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runJobsLogs,
}

var jobsStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show job queue statistics",
	Long: `Display statistics about the job queue.

Examples:
  fluxbase jobs stats`,
	PreRunE: requireAuth,
	RunE:    runJobsStats,
}

var jobsSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync job functions from a directory",
	Long: `Sync job functions from a directory to the server.

Examples:
  fluxbase jobs sync --dir ./jobs
  fluxbase jobs sync --dir ./jobs --namespace production`,
	PreRunE: requireAuth,
	RunE:    runJobsSync,
}

func init() {
	// List flags
	jobsListCmd.Flags().StringVar(&jobNamespace, "namespace", "", "Filter by namespace")

	// Submit flags
	jobsSubmitCmd.Flags().StringVar(&jobPayload, "payload", "", "JSON payload for the job")
	jobsSubmitCmd.Flags().StringVar(&jobPayloadFile, "file", "", "File containing JSON payload")
	jobsSubmitCmd.Flags().IntVar(&jobPriority, "priority", 0, "Job priority (higher = more priority)")
	jobsSubmitCmd.Flags().StringVar(&jobSchedule, "schedule", "", "Schedule job for later (RFC3339 format)")

	// Logs flags
	jobsLogsCmd.Flags().IntVar(&jobTail, "tail", 50, "Number of log lines to show")
	jobsLogsCmd.Flags().BoolVar(&jobFollow, "follow", false, "Follow log output")

	// Sync flags
	jobsSyncCmd.Flags().StringVar(&jobSyncDir, "dir", "./jobs", "Directory containing job functions")
	jobsSyncCmd.Flags().StringVar(&jobNamespace, "namespace", "default", "Target namespace")
	jobsSyncCmd.Flags().BoolVar(&jobDryRun, "dry-run", false, "Preview changes without applying")
	jobsSyncCmd.Flags().BoolVar(&jobKeep, "keep", false, "Keep jobs not present in directory")

	jobsCmd.AddCommand(jobsListCmd)
	jobsCmd.AddCommand(jobsSubmitCmd)
	jobsCmd.AddCommand(jobsStatusCmd)
	jobsCmd.AddCommand(jobsCancelCmd)
	jobsCmd.AddCommand(jobsRetryCmd)
	jobsCmd.AddCommand(jobsLogsCmd)
	jobsCmd.AddCommand(jobsStatsCmd)
	jobsCmd.AddCommand(jobsSyncCmd)
}

func runJobsList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := url.Values{}
	if jobNamespace != "" {
		query.Set("namespace", jobNamespace)
	}

	var jobs []map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/admin/jobs/functions", query, &jobs); err != nil {
		return err
	}

	if len(jobs) == 0 {
		fmt.Println("No job functions found.")
		return nil
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		data := output.TableData{
			Headers: []string{"NAME", "NAMESPACE", "ENABLED", "SCHEDULE", "TIMEOUT"},
			Rows:    make([][]string, len(jobs)),
		}

		for i, job := range jobs {
			name := getStringValue(job, "name")
			namespace := getStringValue(job, "namespace")
			enabled := fmt.Sprintf("%v", job["enabled"])
			schedule := getStringValue(job, "schedule")
			if schedule == "" {
				schedule = "-"
			}
			timeout := fmt.Sprintf("%vs", getIntValue(job, "timeout_seconds"))

			data.Rows[i] = []string{name, namespace, enabled, schedule, timeout}
		}

		formatter.PrintTable(data)
	} else {
		if err := formatter.Print(jobs); err != nil {
			return err
		}
	}

	return nil
}

func runJobsSubmit(cmd *cobra.Command, args []string) error {
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	body := map[string]interface{}{
		"job_name": name,
	}

	// Parse payload
	if jobPayloadFile != "" {
		data, err := os.ReadFile(jobPayloadFile) //nolint:gosec // CLI tool reads user-provided file path
		if err != nil {
			return fmt.Errorf("failed to read payload file: %w", err)
		}
		var payload interface{}
		if err := json.Unmarshal(data, &payload); err != nil {
			return fmt.Errorf("invalid JSON in payload file: %w", err)
		}
		body["payload"] = payload
	} else if jobPayload != "" {
		var payload interface{}
		if err := json.Unmarshal([]byte(jobPayload), &payload); err != nil {
			return fmt.Errorf("invalid JSON payload: %w", err)
		}
		body["payload"] = payload
	}

	if jobPriority != 0 {
		body["priority"] = jobPriority
	}

	if jobSchedule != "" {
		body["scheduled_at"] = jobSchedule
	}

	var result map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/jobs/submit", body, &result); err != nil {
		return err
	}

	jobID := getStringValue(result, "id")
	fmt.Printf("Job submitted successfully. ID: %s\n", jobID)

	return nil
}

func runJobsStatus(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var job map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/jobs/"+url.PathEscape(id), nil, &job); err != nil {
		return err
	}

	formatter := GetFormatter()
	return formatter.Print(job)
}

func runJobsCancel(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := apiClient.DoPost(ctx, "/api/v1/jobs/"+url.PathEscape(id)+"/cancel", nil, nil); err != nil {
		return err
	}

	fmt.Printf("Job '%s' cancelled.\n", id)
	return nil
}

func runJobsRetry(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var result map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/jobs/"+url.PathEscape(id)+"/retry", nil, &result); err != nil {
		return err
	}

	newID := getStringValue(result, "id")
	fmt.Printf("Job retry submitted. New ID: %s\n", newID)
	return nil
}

func runJobsLogs(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var logs []map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/admin/jobs/queue/"+url.PathEscape(id)+"/logs", nil, &logs); err != nil {
		return err
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		for _, log := range logs {
			timestamp := getStringValue(log, "timestamp")
			level := getStringValue(log, "level")
			message := getStringValue(log, "message")
			fmt.Printf("[%s] %s: %s\n", timestamp, level, message)
		}
	} else {
		if err := formatter.Print(logs); err != nil {
			return err
		}
	}

	return nil
}

func runJobsStats(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var stats map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/admin/jobs/stats", nil, &stats); err != nil {
		return err
	}

	formatter := GetFormatter()
	return formatter.Print(stats)
}

func runJobsSync(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Increase HTTP client timeout for sync operations (large bundled payloads)
	originalTimeout := apiClient.HTTPClient.Timeout
	apiClient.HTTPClient.Timeout = 5 * time.Minute
	defer func() { apiClient.HTTPClient.Timeout = originalTimeout }()

	// Auto-detect directory if not explicitly specified
	dir, err := detectResourceDir("jobs", jobSyncDir, "./jobs")
	if err != nil {
		return err
	}
	jobSyncDir = dir

	// Check for _shared directory and sync shared modules first
	sharedDir := filepath.Join(jobSyncDir, "_shared")
	if info, err := os.Stat(sharedDir); err == nil && info.IsDir() {
		sharedEntries, err := os.ReadDir(sharedDir)
		if err == nil {
			sharedCount := 0
			for _, entry := range sharedEntries {
				if entry.IsDir() {
					continue
				}
				name := entry.Name()
				if !strings.HasSuffix(name, ".ts") && !strings.HasSuffix(name, ".js") {
					continue
				}

				content, err := os.ReadFile(filepath.Join(sharedDir, name)) //nolint:gosec // CLI tool reads user-provided file path
				if err != nil {
					fmt.Printf("Warning: failed to read shared module %s: %v\n", name, err)
					continue
				}

				modulePath := "_shared/" + name

				// Sync the shared module via API (upsert)
				moduleBody := map[string]interface{}{
					"module_path": modulePath,
					"content":     string(content),
				}

				err = apiClient.DoPost(ctx, "/api/v1/functions/shared/", moduleBody, nil)
				if err != nil {
					fmt.Printf("Warning: failed to sync shared module %s: %v\n", modulePath, err)
					continue
				}
				sharedCount++
			}
			if sharedCount > 0 {
				fmt.Printf("Synced %d shared modules.\n", sharedCount)
			}
		}
	}

	// Read jobs from directory
	entries, err := os.ReadDir(jobSyncDir)
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

		// Read file
		content, err := os.ReadFile(filepath.Join(jobSyncDir, name)) //nolint:gosec // CLI tool reads user-provided file path
		if err != nil {
			fmt.Printf("Warning: failed to read %s: %v\n", name, err)
			continue
		}

		// Remove extension for job name
		jobName := strings.TrimSuffix(strings.TrimSuffix(name, ".ts"), ".js")

		// Parse @fluxbase: annotations BEFORE bundling (esbuild strips comments)
		job := map[string]interface{}{
			"name": jobName,
			"code": string(content),
		}
		config := annotations.ParseJobAnnotations(string(content))
		annotations.ApplyJobConfig(job, config)

		jobs = append(jobs, job)
	}

	if len(jobs) == 0 {
		fmt.Println("No job functions found in directory.")
		return nil
	}

	if jobDryRun {
		fmt.Println("Dry run - would sync the following jobs:")
		for _, job := range jobs {
			fmt.Printf("  - %s\n", job["name"])
		}
		return nil
	}

	// Read shared modules for bundling (recursively, including JSON/GeoJSON)
	sharedModulesMap := make(map[string]string)
	if info, err := os.Stat(sharedDir); err == nil && info.IsDir() {
		err := filepath.WalkDir(sharedDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil // Skip files we can't access
			}
			if d.IsDir() {
				return nil // Continue into directories
			}

			name := d.Name()
			// Include TypeScript, JavaScript, JSON, and GeoJSON files
			if !strings.HasSuffix(name, ".ts") && !strings.HasSuffix(name, ".js") &&
				!strings.HasSuffix(name, ".json") && !strings.HasSuffix(name, ".geojson") {
				return nil
			}

			content, err := os.ReadFile(path) //nolint:gosec // CLI tool reads user-provided file path
			if err != nil {
				return nil // Skip files we can't read
			}

			// Calculate relative path from jobSyncDir (e.g., "_shared/services/utils.ts")
			relPath, err := filepath.Rel(jobSyncDir, path)
			if err != nil {
				return nil
			}
			// Normalize to forward slashes for consistent module paths
			relPath = filepath.ToSlash(relPath)

			sharedModulesMap[relPath] = string(content)
			return nil
		})
		if err != nil {
			fmt.Printf("Warning: error walking shared directory: %v\n", err)
		}
	}

	// Check if any job needs bundling
	needsBundling := false
	for _, job := range jobs {
		code := job["code"].(string)
		// Simple check for imports
		if strings.Contains(code, "import ") {
			needsBundling = true
			break
		}
	}

	// Bundle jobs that have imports (if Deno is available)
	if needsBundling {
		b, err := bundler.NewBundler(jobSyncDir)
		if err != nil {
			// Deno not available - send unbundled code, server will handle it
			fmt.Println("Note: Deno not available for local bundling. Server will bundle jobs.")
		} else {
			for i, job := range jobs {
				code := job["code"].(string)
				jobName := job["name"].(string)

				if !b.NeedsBundle(code) {
					continue // No imports, skip bundling
				}

				// Validate imports first
				if err := b.ValidateImports(code); err != nil {
					return fmt.Errorf("job %s: %w", jobName, err)
				}

				fmt.Printf("Bundling %s...", jobName)

				result, err := b.Bundle(ctx, code, sharedModulesMap)
				if err != nil {
					fmt.Println() // Complete the line
					return fmt.Errorf("failed to bundle %s: %w", jobName, err)
				}

				// Print size info
				originalSize := len(code)
				bundledSize := len(result.BundledCode)
				fmt.Printf(" %s → %s\n", formatBytes(originalSize), formatBytes(bundledSize))

				// Replace code with bundled code
				jobs[i]["code"] = result.BundledCode
				jobs[i]["is_bundled"] = true
			}
		}
	}

	// Call sync API
	body := map[string]interface{}{
		"namespace": jobNamespace,
		"jobs":      jobs,
		"options": map[string]interface{}{
			"delete_missing": !jobKeep,
		},
	}

	var result map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/admin/jobs/sync", body, &result); err != nil {
		return err
	}

	// Parse the nested summary response
	summary, ok := result["summary"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected response format from server")
	}

	created := getIntValue(summary, "created")
	updated := getIntValue(summary, "updated")
	deleted := getIntValue(summary, "deleted")
	errors := getIntValue(summary, "errors")

	fmt.Printf("Synced jobs to namespace '%s': %d created, %d updated, %d deleted.\n",
		jobNamespace, created, updated, deleted)
	if errors > 0 {
		fmt.Printf("Warning: %d errors occurred during sync.\n", errors)
		// Print error details if available
		if errorList, ok := result["errors"].([]interface{}); ok {
			for _, e := range errorList {
				if errMap, ok := e.(map[string]interface{}); ok {
					fmt.Printf("  - %s: %v\n", errMap["job"], errMap["error"])
				}
			}
		}
	}

	return nil
}
