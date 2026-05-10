package realtime

import (
	"time"
)

// enrichJobWithETA computes ETA fields for job queue events and adds them to the record.
// Shared by both Listener and ListenerPool — see listener.go and listener_pool.go for duplication context.
func enrichJobWithETA(event *ChangeEvent) {
	progressData, ok := event.Record["progress"].(map[string]interface{})
	if !ok || progressData == nil {
		return
	}

	var percent int
	var message string
	var etaSeconds *int

	if p, ok := progressData["percent"].(float64); ok {
		percent = int(p)
	}
	if m, ok := progressData["message"].(string); ok {
		message = m
	}
	if e, ok := progressData["estimated_seconds_left"].(float64); ok {
		eta := int(e)
		etaSeconds = &eta
	}

	status, _ := event.Record["status"].(string)
	startedAtStr, _ := event.Record["started_at"].(string)

	if etaSeconds == nil && status == "running" && percent > 0 && percent < 100 {
		if startedAt, err := time.Parse(time.RFC3339, startedAtStr); err == nil {
			elapsed := time.Since(startedAt).Seconds()
			if elapsed > 0 {
				remainingPercent := float64(100 - percent)
				eta := int((elapsed / float64(percent)) * remainingPercent)
				etaSeconds = &eta
			}
		}
	}

	event.Record["progress_percent"] = percent
	if message != "" {
		event.Record["progress_message"] = message
	}
	if etaSeconds != nil {
		event.Record["estimated_seconds_left"] = *etaSeconds
	}
}
