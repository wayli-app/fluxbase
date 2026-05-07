package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/nimbleflux/fluxbase/cli/output"
)

var aiMetricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Get AI usage metrics",
	Long: `Get aggregated AI usage metrics including requests, tokens, and error rates.

Examples:
  fluxbase ai metrics
  fluxbase ai metrics -o json`,
	PreRunE: requireAuth,
	RunE:    runAIMetrics,
}

func init() {
	aiProvidersCmd.AddCommand(aiMetricsCmd)
}

func runAIMetrics(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var metrics map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/admin/ai/metrics", nil, &metrics); err != nil {
		return err
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		fmt.Printf("Total Requests:         %v\n", metrics["total_requests"])
		fmt.Printf("Total Tokens:           %v\n", metrics["total_tokens"])
		fmt.Printf("  Prompt Tokens:        %v\n", metrics["total_prompt_tokens"])
		fmt.Printf("  Completion Tokens:    %v\n", metrics["total_completion_tokens"])
		fmt.Printf("Active Conversations:   %v\n", metrics["active_conversations"])
		fmt.Printf("Total Conversations:    %v\n", metrics["total_conversations"])
		fmt.Printf("Error Rate:             %v\n", metrics["error_rate"])
		fmt.Printf("Avg Response Time:      %v ms\n", metrics["avg_response_time_ms"])
	} else {
		if err := formatter.Print(metrics); err != nil {
			return err
		}
	}

	return nil
}
