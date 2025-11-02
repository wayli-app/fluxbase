package performance

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// BenchmarkWebhookDelivery benchmarks webhook delivery
func BenchmarkWebhookDelivery(b *testing.B) {
	service := newMockWebhookService()

	webhook := &mockWebhook{
		id:  "webhook1",
		url: "https://example.com/webhook",
	}

	payload := map[string]interface{}{
		"event": "INSERT",
		"data":  "test",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.deliver(webhook, payload)
	}
}

// BenchmarkWebhookWithRetry benchmarks webhook delivery with retries
func BenchmarkWebhookWithRetry(b *testing.B) {
	service := newMockWebhookService()

	webhook := &mockWebhook{
		id:         "webhook1",
		url:        "https://example.com/webhook",
		maxRetries: 3,
	}

	payload := map[string]interface{}{
		"event": "INSERT",
		"data":  "test",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.deliverWithRetry(webhook, payload)
	}
}

// TestWebhookThroughput tests webhook delivery throughput
func TestWebhookThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	service := newMockWebhookService()

	webhook := &mockWebhook{
		id:  "webhook1",
		url: "https://example.com/webhook",
	}

	numDeliveries := 1000
	start := time.Now()

	for i := 0; i < numDeliveries; i++ {
		payload := map[string]interface{}{
			"event": "INSERT",
			"id":    i,
		}
		_ = service.deliver(webhook, payload)
	}

	duration := time.Since(start)
	dps := float64(numDeliveries) / duration.Seconds()

	t.Logf("Delivered %d webhooks in %v", numDeliveries, duration)
	t.Logf("Throughput: %.2f deliveries/second", dps)

	// Assert at least 100 deliveries per second
	if dps < 100 {
		t.Errorf("Webhook throughput too low: %.2f DPS", dps)
	}
}

// TestConcurrentWebhookDelivery tests concurrent webhook delivery
func TestConcurrentWebhookDelivery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	service := newMockWebhookService()

	// Create 10 different webhooks
	webhooks := make([]*mockWebhook, 10)
	for i := 0; i < 10; i++ {
		webhooks[i] = &mockWebhook{
			id:  fmt.Sprintf("webhook%d", i),
			url: fmt.Sprintf("https://example.com/webhook%d", i),
		}
	}

	numDeliveriesPerWebhook := 100
	var wg sync.WaitGroup
	var totalDelivered atomic.Int64

	start := time.Now()

	// Deliver concurrently to all webhooks
	for _, webhook := range webhooks {
		wg.Add(1)
		go func(wh *mockWebhook) {
			defer wg.Done()

			for i := 0; i < numDeliveriesPerWebhook; i++ {
				payload := map[string]interface{}{
					"event": "UPDATE",
					"id":    i,
				}
				_ = service.deliver(wh, payload)
				totalDelivered.Add(1)
			}
		}(webhook)
	}

	wg.Wait()
	duration := time.Since(start)

	total := totalDelivered.Load()
	dps := float64(total) / duration.Seconds()

	t.Logf("Delivered %d webhooks concurrently in %v", total, duration)
	t.Logf("Throughput: %.2f deliveries/second", dps)

	if duration > 10*time.Second {
		t.Errorf("Concurrent delivery too slow: %v", duration)
	}
}

// TestWebhookRetryMechanism tests webhook retry performance
func TestWebhookRetryMechanism(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	service := newMockWebhookService()

	webhook := &mockWebhook{
		id:         "webhook1",
		url:        "https://example.com/webhook",
		maxRetries: 3,
		failCount:  2, // Fail first 2 attempts
	}

	payload := map[string]interface{}{
		"event": "INSERT",
		"data":  "test",
	}

	start := time.Now()
	err := service.deliverWithRetry(webhook, payload)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Delivery failed after retries: %v", err)
	}

	t.Logf("Delivery succeeded after retries in %v", duration)

	// Should succeed on 3rd attempt (after 2 failures)
	// With exponential backoff: 1s + 2s = 3s minimum
	if duration < 3*time.Second {
		t.Errorf("Retry timing incorrect: %v (expected >= 3s)", duration)
	}
}

// TestWebhookBatchProcessing tests batch webhook processing
func TestWebhookBatchProcessing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	service := newMockWebhookService()

	webhook := &mockWebhook{
		id:  "webhook1",
		url: "https://example.com/webhook",
	}

	// Create batch of 100 payloads
	payloads := make([]map[string]interface{}, 100)
	for i := 0; i < 100; i++ {
		payloads[i] = map[string]interface{}{
			"event": "INSERT",
			"id":    i,
		}
	}

	start := time.Now()
	err := service.deliverBatch(webhook, payloads)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Batch delivery failed: %v", err)
	}

	t.Logf("Delivered batch of %d webhooks in %v", len(payloads), duration)
	t.Logf("Average: %v per webhook", duration/time.Duration(len(payloads)))

	// Batch should be more efficient than individual deliveries
	if duration > 5*time.Second {
		t.Errorf("Batch delivery too slow: %v", duration)
	}
}

// TestWebhookQueueProcessing tests queue-based webhook processing
func TestWebhookQueueProcessing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	service := newMockWebhookService()
	queue := newWebhookQueue(4) // 4 workers

	webhook := &mockWebhook{
		id:  "webhook1",
		url: "https://example.com/webhook",
	}

	numJobs := 1000
	start := time.Now()

	// Queue webhook deliveries
	for i := 0; i < numJobs; i++ {
		payload := map[string]interface{}{
			"event": "INSERT",
			"id":    i,
		}
		queue.enqueue(webhook, payload, service)
	}

	// Wait for queue to process all jobs
	queue.wait()
	duration := time.Since(start)

	t.Logf("Processed %d queued webhooks in %v", numJobs, duration)
	t.Logf("Throughput: %.2f webhooks/second", float64(numJobs)/duration.Seconds())

	if duration > 20*time.Second {
		t.Errorf("Queue processing too slow: %v", duration)
	}
}

// mockWebhookService simulates a webhook delivery service
type mockWebhookService struct{}

type mockWebhook struct {
	id         string
	url        string
	maxRetries int
	failCount  int
	attempts   int
}

func newMockWebhookService() *mockWebhookService {
	return &mockWebhookService{}
}

func (s *mockWebhookService) deliver(webhook *mockWebhook, payload map[string]interface{}) error {
	// Simulate HTTP request latency
	time.Sleep(10 * time.Millisecond)
	return nil
}

func (s *mockWebhookService) deliverWithRetry(webhook *mockWebhook, payload map[string]interface{}) error {
	for attempt := 0; attempt <= webhook.maxRetries; attempt++ {
		webhook.attempts++

		// Simulate failure for first N attempts
		if webhook.attempts <= webhook.failCount {
			// Exponential backoff
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			time.Sleep(backoff)
			continue
		}

		// Success
		return s.deliver(webhook, payload)
	}

	return fmt.Errorf("max retries exceeded")
}

func (s *mockWebhookService) deliverBatch(webhook *mockWebhook, payloads []map[string]interface{}) error {
	// Simulate batch delivery with reduced overhead
	time.Sleep(time.Duration(len(payloads)) * 5 * time.Millisecond)
	return nil
}

// webhookQueue implements a worker queue for webhook processing
type webhookQueue struct {
	jobs    chan *webhookJob
	workers int
	wg      sync.WaitGroup
}

type webhookJob struct {
	webhook *mockWebhook
	payload map[string]interface{}
	service *mockWebhookService
}

func newWebhookQueue(workers int) *webhookQueue {
	q := &webhookQueue{
		jobs:    make(chan *webhookJob, 100),
		workers: workers,
	}

	// Start workers
	for i := 0; i < workers; i++ {
		go q.worker()
	}

	return q
}

func (q *webhookQueue) worker() {
	for job := range q.jobs {
		_ = job.service.deliver(job.webhook, job.payload)
		q.wg.Done()
	}
}

func (q *webhookQueue) enqueue(webhook *mockWebhook, payload map[string]interface{}, service *mockWebhookService) {
	q.wg.Add(1)
	q.jobs <- &webhookJob{
		webhook: webhook,
		payload: payload,
		service: service,
	}
}

func (q *webhookQueue) wait() {
	q.wg.Wait()
	close(q.jobs)
}
