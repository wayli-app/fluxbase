package ai

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChatbotLimiter_GetDailyUsage(t *testing.T) {
	limiter := NewChatbotLimiter()
	t.Cleanup(func() {
		// cleanupLoop goroutine runs forever; the limiter is local so it leaks,
		// but that's fine for a unit test.
		_ = limiter
	})

	chatbotID := "chatbot-1"
	userID := "user-1"

	t.Run("zero before any activity", func(t *testing.T) {
		usage := limiter.GetDailyUsage(chatbotID, userID, 500, 100_000)
		assert.Equal(t, 0, usage.RequestsUsed)
		assert.Equal(t, 0, usage.TokensUsed)
		assert.Equal(t, 500, usage.RequestsLimit)
		assert.Equal(t, 100_000, usage.TokensLimit)
		assert.False(t, usage.ResetsAt.IsZero())
	})

	t.Run("reflects request counter after CheckDailyRequestLimit", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			require.True(t, limiter.CheckDailyRequestLimit(chatbotID, userID, 500))
		}
		usage := limiter.GetDailyUsage(chatbotID, userID, 500, 100_000)
		assert.Equal(t, 3, usage.RequestsUsed)
	})

	t.Run("reflects token counter after AddTokenUsage", func(t *testing.T) {
		limiter.AddTokenUsage(chatbotID, userID, 1234)
		usage := limiter.GetDailyUsage(chatbotID, userID, 500, 100_000)
		assert.Equal(t, 1234, usage.TokensUsed)
	})

	t.Run("resets_at is the start of the next local day", func(t *testing.T) {
		usage := limiter.GetDailyUsage(chatbotID, userID, 500, 100_000)
		now := time.Now()
		expectedReset := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).
			Add(24 * time.Hour)
		assert.WithinDuration(t, expectedReset, usage.ResetsAt, time.Second)
	})
}
