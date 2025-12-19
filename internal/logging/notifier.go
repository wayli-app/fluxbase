package logging

import (
	"context"
	"encoding/json"

	"github.com/fluxbase-eu/fluxbase/internal/pubsub"
	"github.com/fluxbase-eu/fluxbase/internal/storage"
)

// LogChannel is the PubSub channel for execution log notifications.
const LogChannel = "fluxbase:logs"

// AllLogsChannel is the PubSub channel for all log notifications (admin streaming).
const AllLogsChannel = "fluxbase:all_logs"

// CategoryChannelPrefix is the prefix for category-specific log channels.
const CategoryChannelPrefix = "fluxbase:logs:"

// CategoryChannel returns the PubSub channel for a specific log category.
func CategoryChannel(category string) string {
	return CategoryChannelPrefix + category
}

// PubSubNotifier sends log notifications via PubSub for realtime streaming.
type PubSubNotifier struct {
	pubsub  pubsub.PubSub
	channel string
}

// NewPubSubNotifier creates a new PubSub-based log notifier.
func NewPubSubNotifier(ps pubsub.PubSub, channel string) *PubSubNotifier {
	if channel == "" {
		channel = LogChannel
	}
	return &PubSubNotifier{
		pubsub:  ps,
		channel: channel,
	}
}

// Notify sends a notification for a log entry.
// Execution logs are sent to the execution-specific channel.
// All logs are sent to the all-logs channel for admin streaming.
func (n *PubSubNotifier) Notify(ctx context.Context, entry *storage.LogEntry) error {
	// Always publish to all-logs channel for admin streaming
	if err := n.notifyAllLogs(ctx, entry); err != nil {
		// Log error but continue - don't fail the main operation
		_ = err
	}

	// Also publish execution logs to the execution-specific channel for backwards compatibility
	if entry.Category == storage.LogCategoryExecution && entry.ExecutionID != "" {
		return n.notifyExecutionLog(ctx, entry)
	}

	return nil
}

// notifyExecutionLog sends an execution log event to the execution logs channel.
func (n *PubSubNotifier) notifyExecutionLog(ctx context.Context, entry *storage.LogEntry) error {
	event := storage.ExecutionLogEvent{
		ExecutionID:   entry.ExecutionID,
		ExecutionType: entry.ExecutionType,
		LineNumber:    entry.LineNumber,
		Level:         entry.Level,
		Message:       entry.Message,
		Timestamp:     entry.Timestamp,
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return n.pubsub.Publish(ctx, n.channel, payload)
}

// notifyAllLogs sends a log stream event to both the all-logs channel and the category-specific channel.
func (n *PubSubNotifier) notifyAllLogs(ctx context.Context, entry *storage.LogEntry) error {
	event := storage.LogStreamEvent{
		ID:             entry.ID.String(),
		Timestamp:      entry.Timestamp,
		Category:       entry.Category,
		Level:          entry.Level,
		Message:        entry.Message,
		CustomCategory: entry.CustomCategory,
		RequestID:      entry.RequestID,
		TraceID:        entry.TraceID,
		Component:      entry.Component,
		UserID:         entry.UserID,
		IPAddress:      entry.IPAddress,
		Fields:         entry.Fields,
		ExecutionID:    entry.ExecutionID,
		ExecutionType:  entry.ExecutionType,
		LineNumber:     entry.LineNumber,
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// Publish to all-logs channel for admin dashboard
	if err := n.pubsub.Publish(ctx, AllLogsChannel, payload); err != nil {
		return err
	}

	// Also publish to category-specific channel for targeted subscriptions
	categoryChannel := CategoryChannel(string(entry.Category))
	return n.pubsub.Publish(ctx, categoryChannel, payload)
}

// NotifyBatch sends notifications for a batch of log entries.
// Only execution logs are sent.
func (n *PubSubNotifier) NotifyBatch(ctx context.Context, entries []*storage.LogEntry) error {
	for _, entry := range entries {
		if err := n.Notify(ctx, entry); err != nil {
			// Log error but continue with other entries
			continue
		}
	}
	return nil
}
