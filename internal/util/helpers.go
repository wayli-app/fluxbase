package util

import (
	"fmt"

	"github.com/google/uuid"
)

func ValueOr[T any](ptr *T, defaultVal T) T {
	if ptr != nil {
		return *ptr
	}
	return defaultVal
}

func ToString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	if uid, ok := v.(*uuid.UUID); ok {
		if uid == nil {
			return ""
		}
		return uid.String()
	}
	return fmt.Sprintf("%v", v)
}

func TruncateString(s string, maxLen int) string {
	// Clamp nonsensical (negative) maxLen to 0 so a caller passing a computed
	// length can't trigger a slice-bounds panic. With maxLen 0 an empty string
	// returns "" and a non-empty string returns "...".
	if maxLen < 0 {
		maxLen = 0
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
