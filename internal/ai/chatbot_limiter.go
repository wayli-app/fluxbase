package ai

import (
	"sync"
	"time"
)

type rateLimitEntry struct {
	count       int
	windowStart time.Time
}

type dailyUsageEntry struct {
	requestCount int
	tokenCount   int
	dayStart     time.Time
}

type ChatbotLimiter struct {
	mu         sync.Mutex
	rateLimits map[string]*rateLimitEntry
	dailyUsage map[string]*dailyUsageEntry
}

func NewChatbotLimiter() *ChatbotLimiter {
	cl := &ChatbotLimiter{
		rateLimits: make(map[string]*rateLimitEntry),
		dailyUsage: make(map[string]*dailyUsageEntry),
	}
	go cl.cleanupLoop()
	return cl
}

func (cl *ChatbotLimiter) key(chatbotID, userID string) string {
	return chatbotID + ":" + userID
}

func (cl *ChatbotLimiter) CheckRateLimit(chatbotID, userID string, limit int) bool {
	if limit <= 0 {
		return true
	}
	cl.mu.Lock()
	defer cl.mu.Unlock()

	k := cl.key(chatbotID, userID)
	now := time.Now()
	entry, ok := cl.rateLimits[k]
	if !ok || now.Sub(entry.windowStart) >= time.Minute {
		cl.rateLimits[k] = &rateLimitEntry{count: 1, windowStart: now}
		return true
	}
	if entry.count >= limit {
		return false
	}
	entry.count++
	return true
}

func (cl *ChatbotLimiter) CheckDailyRequestLimit(chatbotID, userID string, limit int) bool {
	if limit <= 0 {
		return true
	}
	cl.mu.Lock()
	defer cl.mu.Unlock()

	k := cl.key(chatbotID, userID)
	entry := cl.getOrCreateDaily(k)
	if entry.requestCount >= limit {
		return false
	}
	entry.requestCount++
	return true
}

func (cl *ChatbotLimiter) CheckDailyTokenBudget(chatbotID, userID string, budget, tokensToAdd int) bool {
	if budget <= 0 {
		return true
	}
	cl.mu.Lock()
	defer cl.mu.Unlock()

	k := cl.key(chatbotID, userID)
	entry := cl.getOrCreateDaily(k)
	return entry.tokenCount+tokensToAdd <= budget
}

func (cl *ChatbotLimiter) AddTokenUsage(chatbotID, userID string, tokens int) {
	if tokens <= 0 {
		return
	}
	cl.mu.Lock()
	defer cl.mu.Unlock()

	k := cl.key(chatbotID, userID)
	entry := cl.getOrCreateDaily(k)
	entry.tokenCount += tokens
}

func (cl *ChatbotLimiter) getOrCreateDaily(k string) *dailyUsageEntry {
	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	entry, ok := cl.dailyUsage[k]
	if !ok || entry.dayStart != dayStart {
		entry = &dailyUsageEntry{dayStart: dayStart}
		cl.dailyUsage[k] = entry
	}
	return entry
}

func (cl *ChatbotLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		cl.cleanup()
	}
}

func (cl *ChatbotLimiter) cleanup() {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	now := time.Now()
	for k, entry := range cl.rateLimits {
		if now.Sub(entry.windowStart) >= 2*time.Minute {
			delete(cl.rateLimits, k)
		}
	}
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	for k, entry := range cl.dailyUsage {
		if entry.dayStart.Before(dayStart) {
			delete(cl.dailyUsage, k)
		}
	}
}
