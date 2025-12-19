package cmd

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/fluxbase-eu/fluxbase/cli/output"
)

var logsCmd = &cobra.Command{
	Use:     "logs",
	Aliases: []string{"log"},
	Short:   "View and query logs",
	Long:    `Query, view, and stream logs from the central logging system.`,
}

var (
	logsCategory       string
	logsCustomCategory string
	logsLevel          string
	logsComponent      string
	logsRequestID      string
	logsUserID         string
	logsSearch         string
	logsSince          string
	logsUntil          string
	logsLimit          int
	logsTail           int
	logsFollow         bool
	logsSortAsc        bool
)

var logsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List logs with filters",
	Long: `List logs from the central logging system with various filters.

Examples:
  fluxbase logs list
  fluxbase logs list --category system --level error
  fluxbase logs list --since 1h --search "database"
  fluxbase logs list --category execution --limit 50
  fluxbase logs list --user-id abc123 -o json`,
	PreRunE: requireAuth,
	RunE:    runLogsList,
}

var logsTailCmd = &cobra.Command{
	Use:   "tail",
	Short: "Tail logs in real-time",
	Long: `Continuously display new log entries as they arrive.

Examples:
  fluxbase logs tail
  fluxbase logs tail --category security
  fluxbase logs tail --level error
  fluxbase logs tail --category system --component auth`,
	PreRunE: requireAuth,
	RunE:    runLogsTail,
}

var logsStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show log statistics",
	Long: `Display statistics about the logging system.

Examples:
  fluxbase logs stats
  fluxbase logs stats -o json`,
	PreRunE: requireAuth,
	RunE:    runLogsStats,
}

var logsExecutionCmd = &cobra.Command{
	Use:   "execution [id]",
	Short: "View logs for a specific execution",
	Long: `View logs for a specific function, job, or RPC execution.

Examples:
  fluxbase logs execution abc123-def456
  fluxbase logs execution abc123-def456 -o json
  fluxbase logs execution abc123-def456 --follow`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runLogsExecution,
}

func init() {
	// List flags
	logsListCmd.Flags().StringVar(&logsCategory, "category", "", "Filter by category (system, http, security, execution, ai, custom)")
	logsListCmd.Flags().StringVar(&logsCustomCategory, "custom-category", "", "Filter by custom category name (requires --category=custom)")
	logsListCmd.Flags().StringVar(&logsLevel, "level", "", "Filter by level (debug, info, warn, error)")
	logsListCmd.Flags().StringVar(&logsComponent, "component", "", "Filter by component name")
	logsListCmd.Flags().StringVar(&logsRequestID, "request-id", "", "Filter by request ID")
	logsListCmd.Flags().StringVar(&logsUserID, "user-id", "", "Filter by user ID")
	logsListCmd.Flags().StringVar(&logsSearch, "search", "", "Full-text search in message")
	logsListCmd.Flags().StringVar(&logsSince, "since", "", "Show logs since time (e.g., 1h, 30m, 2024-01-15T10:00:00Z)")
	logsListCmd.Flags().StringVar(&logsUntil, "until", "", "Show logs until time")
	logsListCmd.Flags().IntVar(&logsLimit, "limit", 100, "Maximum number of entries to return")
	logsListCmd.Flags().BoolVar(&logsSortAsc, "asc", false, "Sort by timestamp ascending (oldest first)")

	// Tail flags
	logsTailCmd.Flags().StringVar(&logsCategory, "category", "", "Filter by category")
	logsTailCmd.Flags().StringVar(&logsLevel, "level", "", "Filter by level")
	logsTailCmd.Flags().StringVar(&logsComponent, "component", "", "Filter by component")
	logsTailCmd.Flags().IntVar(&logsTail, "lines", 20, "Number of initial lines to show")

	// Execution flags
	logsExecutionCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output (stream new entries)")
	logsExecutionCmd.Flags().IntVar(&logsTail, "tail", 0, "Show only last N lines")

	logsCmd.AddCommand(logsListCmd)
	logsCmd.AddCommand(logsTailCmd)
	logsCmd.AddCommand(logsStatsCmd)
	logsCmd.AddCommand(logsExecutionCmd)
}

func runLogsList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := buildLogsQuery()

	var result struct {
		Entries    []map[string]interface{} `json:"entries"`
		TotalCount int64                    `json:"total_count"`
		HasMore    bool                     `json:"has_more"`
	}

	if err := apiClient.DoGet(ctx, "/api/v1/admin/logs", query, &result); err != nil {
		return err
	}

	if len(result.Entries) == 0 {
		fmt.Println("No logs found.")
		return nil
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		printLogsTable(result.Entries, formatter)
	} else {
		if err := formatter.Print(result); err != nil {
			return err
		}
	}

	if result.HasMore {
		fmt.Printf("\nShowing %d of %d total entries. Use --limit to see more.\n", len(result.Entries), result.TotalCount)
	}

	return nil
}

func runLogsTail(cmd *cobra.Command, args []string) error {
	// First, get initial logs
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := buildLogsQuery()
	query.Set("limit", fmt.Sprintf("%d", logsTail))
	query.Set("sort_asc", "false") // Most recent first

	var result struct {
		Entries []map[string]interface{} `json:"entries"`
	}

	if err := apiClient.DoGet(ctx, "/api/v1/admin/logs", query, &result); err != nil {
		return err
	}

	formatter := GetFormatter()

	// Print initial logs (in reverse order so oldest is first)
	if len(result.Entries) > 0 {
		for i := len(result.Entries) - 1; i >= 0; i-- {
			printLogEntry(result.Entries[i], formatter)
		}
	}

	// Poll for new logs
	fmt.Println("\n--- Waiting for new logs (Ctrl+C to exit) ---")

	var lastTimestamp string
	if len(result.Entries) > 0 {
		lastTimestamp = getStringValue(result.Entries[0], "timestamp")
	} else {
		lastTimestamp = time.Now().UTC().Format(time.RFC3339)
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		pollCtx, pollCancel := context.WithTimeout(context.Background(), 10*time.Second)

		pollQuery := buildLogsQuery()
		pollQuery.Set("start_time", lastTimestamp)
		pollQuery.Set("sort_asc", "true")
		pollQuery.Set("limit", "100")

		var pollResult struct {
			Entries []map[string]interface{} `json:"entries"`
		}

		if err := apiClient.DoGet(pollCtx, "/api/v1/admin/logs", pollQuery, &pollResult); err != nil {
			pollCancel()
			// Don't fail on poll errors, just continue
			continue
		}
		pollCancel()

		for _, entry := range pollResult.Entries {
			ts := getStringValue(entry, "timestamp")
			if ts > lastTimestamp {
				printLogEntry(entry, formatter)
				lastTimestamp = ts
			}
		}
	}

	return nil
}

func runLogsStats(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var stats map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/admin/logs/stats", nil, &stats); err != nil {
		return err
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		fmt.Println("Log Statistics")
		fmt.Println("==============")
		fmt.Printf("Total Entries: %v\n\n", stats["total_entries"])

		if byCategory, ok := stats["entries_by_category"].(map[string]interface{}); ok {
			fmt.Println("By Category:")
			for cat, count := range byCategory {
				fmt.Printf("  %-12s %v\n", cat+":", count)
			}
		}
		fmt.Println()

		if byLevel, ok := stats["entries_by_level"].(map[string]interface{}); ok {
			fmt.Println("By Level:")
			for level, count := range byLevel {
				fmt.Printf("  %-12s %v\n", level+":", count)
			}
		}
		fmt.Println()

		if oldest, ok := stats["oldest_entry"].(string); ok && oldest != "" {
			fmt.Printf("Oldest Entry: %s\n", oldest)
		}
		if newest, ok := stats["newest_entry"].(string); ok && newest != "" {
			fmt.Printf("Newest Entry: %s\n", newest)
		}
	} else {
		if err := formatter.Print(stats); err != nil {
			return err
		}
	}

	return nil
}

func runLogsExecution(cmd *cobra.Command, args []string) error {
	executionID := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := url.Values{}
	if logsTail > 0 {
		query.Set("after_line", fmt.Sprintf("%d", -logsTail))
	}

	var result struct {
		Entries []map[string]interface{} `json:"entries"`
		Count   int                      `json:"count"`
	}

	path := fmt.Sprintf("/api/v1/admin/logs/executions/%s", url.PathEscape(executionID))
	if err := apiClient.DoGet(ctx, path, query, &result); err != nil {
		return err
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		if len(result.Entries) == 0 {
			fmt.Println("No logs found for this execution.")
			return nil
		}

		for _, entry := range result.Entries {
			printExecutionLogEntry(entry)
		}
	} else {
		if err := formatter.Print(result); err != nil {
			return err
		}
	}

	if logsFollow {
		fmt.Println("\n--- Following logs (Ctrl+C to exit) ---")

		lastLine := 0
		if len(result.Entries) > 0 {
			lastLine = getIntValue(result.Entries[len(result.Entries)-1], "line_number")
		}

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			pollCtx, pollCancel := context.WithTimeout(context.Background(), 10*time.Second)

			pollQuery := url.Values{}
			pollQuery.Set("after_line", fmt.Sprintf("%d", lastLine))

			var pollResult struct {
				Entries []map[string]interface{} `json:"entries"`
			}

			if err := apiClient.DoGet(pollCtx, path, pollQuery, &pollResult); err != nil {
				pollCancel()
				continue
			}
			pollCancel()

			for _, entry := range pollResult.Entries {
				lineNum := getIntValue(entry, "line_number")
				if lineNum > lastLine {
					printExecutionLogEntry(entry)
					lastLine = lineNum
				}
			}
		}
	}

	return nil
}

// buildLogsQuery builds query parameters from flags
func buildLogsQuery() url.Values {
	query := url.Values{}

	if logsCategory != "" {
		query.Set("category", logsCategory)
	}
	if logsCustomCategory != "" {
		query.Set("custom_category", logsCustomCategory)
	}
	if logsLevel != "" {
		query.Set("level", logsLevel)
	}
	if logsComponent != "" {
		query.Set("component", logsComponent)
	}
	if logsRequestID != "" {
		query.Set("request_id", logsRequestID)
	}
	if logsUserID != "" {
		query.Set("user_id", logsUserID)
	}
	if logsSearch != "" {
		query.Set("search", logsSearch)
	}
	if logsSince != "" {
		startTime := parseTimeArg(logsSince)
		if !startTime.IsZero() {
			query.Set("start_time", startTime.Format(time.RFC3339))
		}
	}
	if logsUntil != "" {
		endTime := parseTimeArg(logsUntil)
		if !endTime.IsZero() {
			query.Set("end_time", endTime.Format(time.RFC3339))
		}
	}
	if logsLimit > 0 {
		query.Set("limit", fmt.Sprintf("%d", logsLimit))
	}
	if logsSortAsc {
		query.Set("sort_asc", "true")
	}

	return query
}

// parseTimeArg parses a time argument that can be a duration (1h, 30m) or RFC3339 timestamp
func parseTimeArg(arg string) time.Time {
	// Try parsing as duration first
	if d, err := time.ParseDuration(arg); err == nil {
		return time.Now().Add(-d)
	}

	// Try parsing as RFC3339
	if t, err := time.Parse(time.RFC3339, arg); err == nil {
		return t
	}

	// Try parsing as date only
	if t, err := time.Parse("2006-01-02", arg); err == nil {
		return t
	}

	return time.Time{}
}

// printLogsTable prints logs in table format
func printLogsTable(entries []map[string]interface{}, formatter *output.Formatter) {
	data := output.TableData{
		Headers: []string{"TIME", "LEVEL", "CATEGORY", "COMPONENT", "MESSAGE"},
		Rows:    make([][]string, len(entries)),
	}

	for i, entry := range entries {
		timestamp := getStringValue(entry, "timestamp")
		if t, err := time.Parse(time.RFC3339Nano, timestamp); err == nil {
			timestamp = t.Local().Format("15:04:05")
		}

		level := strings.ToUpper(getStringValue(entry, "level"))
		category := getStringValue(entry, "category")
		component := getStringValue(entry, "component")
		message := getStringValue(entry, "message")

		// Truncate message if too long
		if len(message) > 80 {
			message = message[:77] + "..."
		}

		data.Rows[i] = []string{timestamp, level, category, component, message}
	}

	formatter.PrintTable(data)
}

// printLogEntry prints a single log entry in a readable format
func printLogEntry(entry map[string]interface{}, formatter *output.Formatter) {
	timestamp := getStringValue(entry, "timestamp")
	if t, err := time.Parse(time.RFC3339Nano, timestamp); err == nil {
		timestamp = t.Local().Format("2006-01-02 15:04:05")
	}

	level := strings.ToUpper(getStringValue(entry, "level"))
	category := getStringValue(entry, "category")
	component := getStringValue(entry, "component")
	message := getStringValue(entry, "message")

	levelColor := getLevelColor(level)

	if component != "" {
		fmt.Printf("%s [%s%s\033[0m] [%s] [%s] %s\n", timestamp, levelColor, level, category, component, message)
	} else {
		fmt.Printf("%s [%s%s\033[0m] [%s] %s\n", timestamp, levelColor, level, category, message)
	}
}

// printExecutionLogEntry prints a single execution log entry
func printExecutionLogEntry(entry map[string]interface{}) {
	lineNum := getIntValue(entry, "line_number")
	level := strings.ToUpper(getStringValue(entry, "level"))
	message := getStringValue(entry, "message")
	timestamp := getStringValue(entry, "timestamp")

	if t, err := time.Parse(time.RFC3339Nano, timestamp); err == nil {
		timestamp = t.Local().Format("15:04:05.000")
	}

	levelColor := getLevelColor(level)
	fmt.Printf("%s %4d [%s%s\033[0m] %s\n", timestamp, lineNum, levelColor, level, message)
}

// getLevelColor returns ANSI color code for log level
func getLevelColor(level string) string {
	switch strings.ToUpper(level) {
	case "DEBUG", "TRACE":
		return "\033[36m" // Cyan
	case "INFO":
		return "\033[32m" // Green
	case "WARN", "WARNING":
		return "\033[33m" // Yellow
	case "ERROR", "FATAL", "PANIC":
		return "\033[31m" // Red
	default:
		return ""
	}
}
