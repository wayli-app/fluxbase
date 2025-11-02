package performance

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// BenchmarkRESTAPISequential benchmarks sequential REST API requests
func BenchmarkRESTAPISequential(b *testing.B) {
	client := &mockRESTClient{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.get(fmt.Sprintf("/api/users/%d", i))
	}
}

// BenchmarkRESTAPIConcurrent benchmarks concurrent REST API requests
func BenchmarkRESTAPIConcurrent(b *testing.B) {
	client := &mockRESTClient{}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_ = client.get(fmt.Sprintf("/api/users/%d", i))
			i++
		}
	})
}

// BenchmarkRESTAPIBulkInsert benchmarks bulk insert operations
func BenchmarkRESTAPIBulkInsert(b *testing.B) {
	client := &mockRESTClient{}

	// Prepare bulk data
	records := make([]map[string]interface{}, 100)
	for i := 0; i < 100; i++ {
		records[i] = map[string]interface{}{
			"id":   i,
			"name": fmt.Sprintf("User %d", i),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.bulkInsert("/api/users", records)
	}
}

// BenchmarkRESTAPIWithFiltering benchmarks API requests with complex filters
func BenchmarkRESTAPIWithFiltering(b *testing.B) {
	client := &mockRESTClient{}

	filters := map[string]interface{}{
		"status": "active",
		"age":    ">25",
		"role":   "in.(admin,user)",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.getWithFilters("/api/users", filters)
	}
}

// BenchmarkRESTAPIPagination benchmarks paginated requests
func BenchmarkRESTAPIPagination(b *testing.B) {
	client := &mockRESTClient{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		page := (i % 10) + 1
		_ = client.getWithPagination("/api/users", page, 25)
	}
}

// TestConcurrent100Users simulates 100 concurrent users
func TestConcurrent100Users(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	client := &mockRESTClient{}
	numUsers := 100
	requestsPerUser := 10

	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()
			for j := 0; j < requestsPerUser; j++ {
				_ = client.get(fmt.Sprintf("/api/users/%d", userID))
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	totalRequests := numUsers * requestsPerUser
	qps := float64(totalRequests) / duration.Seconds()

	t.Logf("Completed %d requests in %v", totalRequests, duration)
	t.Logf("Throughput: %.2f requests/second", qps)
	t.Logf("Average response time: %v", duration/time.Duration(totalRequests))

	// Assert reasonable performance
	if duration > 30*time.Second {
		t.Errorf("Performance degraded: took %v for %d requests", duration, totalRequests)
	}
}

// TestBulkInsert1000Records tests bulk insert of 1000 records
func TestBulkInsert1000Records(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	client := &mockRESTClient{}

	// Prepare 1000 records
	records := make([]map[string]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		records[i] = map[string]interface{}{
			"id":    i,
			"name":  fmt.Sprintf("User %d", i),
			"email": fmt.Sprintf("user%d@example.com", i),
		}
	}

	start := time.Now()
	err := client.bulkInsert("/api/users", records)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Bulk insert failed: %v", err)
	}

	t.Logf("Bulk inserted 1000 records in %v", duration)
	t.Logf("Average: %v per record", duration/1000)

	// Assert completes within 5 seconds
	if duration > 5*time.Second {
		t.Errorf("Bulk insert too slow: took %v", duration)
	}
}

// TestHighThroughputRead tests high-throughput read operations
func TestHighThroughputRead(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	client := &mockRESTClient{}
	numRequests := 10000

	start := time.Now()

	for i := 0; i < numRequests; i++ {
		_ = client.get(fmt.Sprintf("/api/users/%d", i%100))
	}

	duration := time.Since(start)
	qps := float64(numRequests) / duration.Seconds()

	t.Logf("Completed %d read requests in %v", numRequests, duration)
	t.Logf("Throughput: %.2f requests/second", qps)

	// Assert at least 1000 QPS
	if qps < 1000 {
		t.Errorf("Throughput too low: %.2f QPS", qps)
	}
}

// mockRESTClient simulates a REST API client
type mockRESTClient struct{}

func (c *mockRESTClient) get(path string) error {
	// Simulate 1ms latency
	time.Sleep(1 * time.Millisecond)
	return nil
}

func (c *mockRESTClient) getWithFilters(path string, filters map[string]interface{}) error {
	// Simulate 2ms latency for filtered queries
	time.Sleep(2 * time.Millisecond)
	return nil
}

func (c *mockRESTClient) getWithPagination(path string, page, pageSize int) error {
	// Simulate 1.5ms latency for paginated queries
	time.Sleep(1500 * time.Microsecond)
	return nil
}

func (c *mockRESTClient) bulkInsert(path string, records []map[string]interface{}) error {
	// Simulate 1ms per 10 records
	time.Sleep(time.Duration(len(records)/10) * time.Millisecond)
	return nil
}
