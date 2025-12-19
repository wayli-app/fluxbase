package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var realtimeCmd = &cobra.Command{
	Use:     "realtime",
	Aliases: []string{"rt"},
	Short:   "Realtime operations",
	Long:    `View realtime stats and broadcast messages.`,
}

var (
	rtMessage string
	rtEvent   string
)

var realtimeStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show realtime statistics",
	Long: `Display realtime connection and subscription statistics.

Examples:
  fluxbase realtime stats`,
	PreRunE: requireAuth,
	RunE:    runRealtimeStats,
}

var realtimeBroadcastCmd = &cobra.Command{
	Use:   "broadcast [channel]",
	Short: "Broadcast a message to a channel",
	Long: `Broadcast a message to a realtime channel.

Examples:
  fluxbase realtime broadcast my-channel --message '{"type": "notification", "text": "Hello!"}'
  fluxbase realtime broadcast updates --event custom_event --message '{"data": "value"}'`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runRealtimeBroadcast,
}

func init() {
	// Broadcast flags
	realtimeBroadcastCmd.Flags().StringVar(&rtMessage, "message", "", "JSON message to broadcast (required)")
	realtimeBroadcastCmd.Flags().StringVar(&rtEvent, "event", "broadcast", "Event type")
	_ = realtimeBroadcastCmd.MarkFlagRequired("message")

	realtimeCmd.AddCommand(realtimeStatsCmd)
	realtimeCmd.AddCommand(realtimeBroadcastCmd)
}

func runRealtimeStats(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var stats map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/realtime/stats", nil, &stats); err != nil {
		return err
	}

	formatter := GetFormatter()
	return formatter.Print(stats)
}

func runRealtimeBroadcast(cmd *cobra.Command, args []string) error {
	channel := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	body := map[string]interface{}{
		"channel": channel,
		"event":   rtEvent,
		"payload": rtMessage,
	}

	if err := apiClient.DoPost(ctx, "/api/v1/realtime/broadcast", body, nil); err != nil {
		return err
	}

	fmt.Printf("Message broadcast to channel '%s'.\n", channel)
	return nil
}
