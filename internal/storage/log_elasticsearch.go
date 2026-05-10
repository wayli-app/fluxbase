package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	elasticsearch8 "github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/elastic/go-elasticsearch/v8/esutil"

	elasticsearch9 "github.com/elastic/go-elasticsearch/v9"
	esapi9 "github.com/elastic/go-elasticsearch/v9/esapi"
	esutil9 "github.com/elastic/go-elasticsearch/v9/esutil"

	"github.com/google/uuid"
)

// ElasticsearchLogStorage implements LogStorage using Elasticsearch.
type ElasticsearchLogStorage struct {
	clientV8 *elasticsearch8.Client
	clientV9 *elasticsearch9.Client
	index    string
	username string
	password string
	version  int // ES version (8 or 9)
}

// newElasticsearchLogStorage creates a new Elasticsearch-backed log storage.
func newElasticsearchLogStorage(cfg LogStorageConfig) (*ElasticsearchLogStorage, error) {
	// Default URLs to localhost if not provided
	urls := cfg.ElasticsearchURLs
	if len(urls) == 0 {
		urls = []string{"http://localhost:9200"}
	}

	// Default index name if not provided
	index := cfg.ElasticsearchIndex
	if index == "" {
		index = "fluxbase-logs"
	}

	// Default version to 9 if not provided
	version := cfg.ElasticsearchVersion
	if version == 0 {
		version = 9
	}

	// Validate version
	if version != 8 && version != 9 {
		return nil, fmt.Errorf("unsupported elasticsearch version: %d (must be 8 or 9)", version)
	}

	storage := &ElasticsearchLogStorage{
		index:    index,
		username: cfg.ElasticsearchUsername,
		password: cfg.ElasticsearchPassword,
		version:  version,
	}

	if version == 8 {
		esCfg := elasticsearch8.Config{
			Addresses: urls,
			Username:  cfg.ElasticsearchUsername,
			Password:  cfg.ElasticsearchPassword,
		}
		client, err := elasticsearch8.NewClient(esCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create elasticsearch v8 client: %w", err)
		}
		storage.clientV8 = client
	} else {
		esCfg := elasticsearch9.Config{
			Addresses: urls,
			Username:  cfg.ElasticsearchUsername,
			Password:  cfg.ElasticsearchPassword,
		}
		client, err := elasticsearch9.NewClient(esCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create elasticsearch v9 client: %w", err)
		}
		storage.clientV9 = client
	}

	return storage, nil
}

// Name returns the backend identifier.
func (s *ElasticsearchLogStorage) Name() string {
	return "elasticsearch"
}

// Write writes a batch of log entries to Elasticsearch using the bulk API.
func (s *ElasticsearchLogStorage) Write(ctx context.Context, entries []*LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	if s.version == 9 {
		return s.writeV9(ctx, entries)
	}
	return s.writeV8(ctx, entries)
}

// writeV8 writes entries using the v8 client.
func (s *ElasticsearchLogStorage) writeV8(ctx context.Context, entries []*LogEntry) error {
	bi, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Client:        s.clientV8,
		NumWorkers:    1,
		FlushBytes:    5e+6, // 5MB
		FlushInterval: time.Second,
	})
	if err != nil {
		return fmt.Errorf("failed to create bulk indexer: %w", err)
	}

	for _, entry := range entries {
		if entry.ID == uuid.Nil {
			entry.ID = uuid.New()
		}
		if entry.Timestamp.IsZero() {
			entry.Timestamp = time.Now()
		}

		doc := s.toDocument(entry)
		data, err := json.Marshal(doc)
		if err != nil {
			_ = bi.Close(ctx)
			return fmt.Errorf("failed to marshal log entry: %w", err)
		}

		err = bi.Add(ctx, esutil.BulkIndexerItem{
			Action:     "index",
			Index:      s.index,
			DocumentID: entry.ID.String(),
			Body:       bytes.NewReader(data),
		})
		if err != nil {
			_ = bi.Close(ctx)
			return fmt.Errorf("failed to add entry to bulk indexer: %w", err)
		}
	}

	if err := bi.Close(ctx); err != nil {
		return fmt.Errorf("bulk indexer close failed: %w", err)
	}
	return nil
}

// writeV9 writes entries using the v9 client.
func (s *ElasticsearchLogStorage) writeV9(ctx context.Context, entries []*LogEntry) error {
	bi, err := esutil9.NewBulkIndexer(esutil9.BulkIndexerConfig{
		Client:        s.clientV9,
		NumWorkers:    1,
		FlushBytes:    5e+6, // 5MB
		FlushInterval: time.Second,
	})
	if err != nil {
		return fmt.Errorf("failed to create bulk indexer: %w", err)
	}

	for _, entry := range entries {
		if entry.ID == uuid.Nil {
			entry.ID = uuid.New()
		}
		if entry.Timestamp.IsZero() {
			entry.Timestamp = time.Now()
		}

		doc := s.toDocument(entry)
		data, err := json.Marshal(doc)
		if err != nil {
			_ = bi.Close(ctx)
			return fmt.Errorf("failed to marshal log entry: %w", err)
		}

		err = bi.Add(ctx, esutil9.BulkIndexerItem{
			Action:     "index",
			Index:      s.index,
			DocumentID: entry.ID.String(),
			Body:       bytes.NewReader(data),
		})
		if err != nil {
			_ = bi.Close(ctx)
			return fmt.Errorf("failed to add entry to bulk indexer: %w", err)
		}
	}

	if err := bi.Close(ctx); err != nil {
		return fmt.Errorf("bulk indexer close failed: %w", err)
	}
	return nil
}

// Query retrieves logs matching the given options using Elasticsearch query DSL.
func (s *ElasticsearchLogStorage) Query(ctx context.Context, opts LogQueryOptions) (*LogQueryResult, error) {
	// Build query DSL
	query := s.buildQuery(opts)

	// Build search request
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, fmt.Errorf("failed to encode query: %w", err)
	}

	// Set pagination
	size := opts.Limit
	if size <= 0 {
		size = 100
	}
	from := opts.Offset
	if from < 0 {
		from = 0
	}

	// Build sort order
	sortField := "@timestamp"
	sortOrder := "desc"
	if opts.SortAsc {
		sortOrder = "asc"
	}

	// Execute search based on version
	var body io.ReadCloser
	if s.version == 9 {
		req := esapi9.SearchRequest{
			Index: []string{s.index},
			Body:  &buf,
			Size:  &size,
			From:  &from,
			Sort:  []string{sortField + ":" + sortOrder},
		}
		res, err := req.Do(ctx, s.clientV9)
		if err != nil {
			return nil, fmt.Errorf("failed to execute search: %w", err)
		}
		defer func() { _ = res.Body.Close() }()
		if res.IsError() {
			return nil, fmt.Errorf("elasticsearch search error: %s", res.String())
		}
		body = res.Body
	} else {
		req := esapi.SearchRequest{
			Index: []string{s.index},
			Body:  &buf,
			Size:  &size,
			From:  &from,
			Sort:  []string{sortField + ":" + sortOrder},
		}
		res, err := req.Do(ctx, s.clientV8)
		if err != nil {
			return nil, fmt.Errorf("failed to execute search: %w", err)
		}
		defer func() { _ = res.Body.Close() }()
		if res.IsError() {
			return nil, fmt.Errorf("elasticsearch search error: %s", res.String())
		}
		body = res.Body
	}

	// Parse response
	var searchResponse struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source *LogEntry `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	// Extract entries
	entries := make([]*LogEntry, 0, len(searchResponse.Hits.Hits))
	for _, hit := range searchResponse.Hits.Hits {
		if hit.Source != nil {
			entries = append(entries, hit.Source)
		}
	}

	return &LogQueryResult{
		Entries:    entries,
		TotalCount: searchResponse.Hits.Total.Value,
		HasMore:    int64(from+len(entries)) < searchResponse.Hits.Total.Value,
	}, nil
}

// GetExecutionLogs retrieves logs for a specific execution, ordered by line number.
func (s *ElasticsearchLogStorage) GetExecutionLogs(ctx context.Context, executionID string, afterLine int) ([]*LogEntry, error) {
	// Build query for execution logs
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"term": map[string]interface{}{
							"execution_id": executionID,
						},
					},
					{
						"range": map[string]interface{}{
							"line_number": map[string]interface{}{
								"gt": afterLine,
							},
						},
					},
				},
			},
		},
		"sort": []map[string]interface{}{
			{
				"line_number": map[string]interface{}{
					"order": "asc",
				},
			},
		},
	}

	// Execute search
	return s.executeSearch(ctx, query, 0, 10000)
}

// Delete removes logs matching the given options using delete by query.
func (s *ElasticsearchLogStorage) Delete(ctx context.Context, opts LogQueryOptions) (int64, error) {
	// Build query
	query := s.buildQuery(opts)

	// Check if at least one filter is set
	if _, hasFilter := query["query"]; !hasFilter {
		return 0, fmt.Errorf("delete requires at least one filter condition")
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return 0, fmt.Errorf("failed to encode delete query: %w", err)
	}

	// Execute delete by query based on version
	var body io.ReadCloser
	if s.version == 9 {
		req := esapi9.DeleteByQueryRequest{
			Index: []string{s.index},
			Body:  &buf,
		}
		res, err := req.Do(ctx, s.clientV9)
		if err != nil {
			return 0, fmt.Errorf("failed to execute delete: %w", err)
		}
		defer func() { _ = res.Body.Close() }()
		if res.IsError() {
			return 0, fmt.Errorf("elasticsearch delete error: %s", res.String())
		}
		body = res.Body
	} else {
		req := esapi.DeleteByQueryRequest{
			Index: []string{s.index},
			Body:  &buf,
		}
		res, err := req.Do(ctx, s.clientV8)
		if err != nil {
			return 0, fmt.Errorf("failed to execute delete: %w", err)
		}
		defer func() { _ = res.Body.Close() }()
		if res.IsError() {
			return 0, fmt.Errorf("elasticsearch delete error: %s", res.String())
		}
		body = res.Body
	}

	// Parse response
	var deleteResponse struct {
		Deleted int64 `json:"deleted"`
	}

	if err := json.NewDecoder(body).Decode(&deleteResponse); err != nil {
		return 0, fmt.Errorf("failed to decode delete response: %w", err)
	}

	return deleteResponse.Deleted, nil
}

// Stats returns statistics about stored logs using aggregations.
func (s *ElasticsearchLogStorage) Stats(ctx context.Context) (*LogStats, error) {
	stats := &LogStats{
		EntriesByCategory: make(map[LogCategory]int64),
		EntriesByLevel:    make(map[LogLevel]int64),
	}

	// Build aggregation query
	query := map[string]interface{}{
		"size": 0,
		"aggs": map[string]interface{}{
			"categories": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "category",
				},
			},
			"levels": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "level",
				},
			},
			"min_timestamp": map[string]interface{}{
				"min": map[string]interface{}{
					"field": "@timestamp",
				},
			},
			"max_timestamp": map[string]interface{}{
				"max": map[string]interface{}{
					"field": "@timestamp",
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, fmt.Errorf("failed to encode stats query: %w", err)
	}

	// Execute search based on version
	var body io.ReadCloser
	if s.version == 9 {
		req := esapi9.SearchRequest{
			Index: []string{s.index},
			Body:  &buf,
		}
		res, err := req.Do(ctx, s.clientV9)
		if err != nil {
			return nil, fmt.Errorf("failed to execute stats query: %w", err)
		}
		defer func() { _ = res.Body.Close() }()
		if res.IsError() {
			return nil, fmt.Errorf("elasticsearch stats error: %s", res.String())
		}
		body = res.Body
	} else {
		req := esapi.SearchRequest{
			Index: []string{s.index},
			Body:  &buf,
		}
		res, err := req.Do(ctx, s.clientV8)
		if err != nil {
			return nil, fmt.Errorf("failed to execute stats query: %w", err)
		}
		defer func() { _ = res.Body.Close() }()
		if res.IsError() {
			return nil, fmt.Errorf("elasticsearch stats error: %s", res.String())
		}
		body = res.Body
	}

	// Parse response
	var searchResponse struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
		} `json:"hits"`
		Aggregations struct {
			Categories struct {
				Buckets []struct {
					Key      string `json:"key"`
					DocCount int64  `json:"doc_count"`
				} `json:"buckets"`
			} `json:"categories"`
			Levels struct {
				Buckets []struct {
					Key      string `json:"key"`
					DocCount int64  `json:"doc_count"`
				} `json:"buckets"`
			} `json:"levels"`
			MinTimestamp struct {
				ValueAsString string `json:"value_as_string"`
			} `json:"min_timestamp"`
			MaxTimestamp struct {
				ValueAsString string `json:"value_as_string"`
			} `json:"max_timestamp"`
		} `json:"aggregations"`
	}

	if err := json.NewDecoder(body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("failed to decode stats response: %w", err)
	}

	stats.TotalEntries = searchResponse.Hits.Total.Value

	// Parse category counts
	for _, bucket := range searchResponse.Aggregations.Categories.Buckets {
		stats.EntriesByCategory[LogCategory(bucket.Key)] = bucket.DocCount
	}

	// Parse level counts
	for _, bucket := range searchResponse.Aggregations.Levels.Buckets {
		stats.EntriesByLevel[LogLevel(bucket.Key)] = bucket.DocCount
	}

	// Parse timestamp range
	if searchResponse.Aggregations.MinTimestamp.ValueAsString != "" {
		if t, err := time.Parse(time.RFC3339Nano, searchResponse.Aggregations.MinTimestamp.ValueAsString); err == nil {
			stats.OldestEntry = &t
		}
	}
	if searchResponse.Aggregations.MaxTimestamp.ValueAsString != "" {
		if t, err := time.Parse(time.RFC3339Nano, searchResponse.Aggregations.MaxTimestamp.ValueAsString); err == nil {
			stats.NewestEntry = &t
		}
	}

	return stats, nil
}

// Health checks if the Elasticsearch cluster is operational.
func (s *ElasticsearchLogStorage) Health(ctx context.Context) error {
	// Ping the cluster based on version
	if s.version == 9 {
		req := esapi9.PingRequest{}
		res, err := req.Do(ctx, s.clientV9)
		if err != nil {
			return fmt.Errorf("elasticsearch ping failed: %w", err)
		}
		defer func() { _ = res.Body.Close() }()
		if res.IsError() {
			return fmt.Errorf("elasticsearch health check failed: %s", res.String())
		}
	} else {
		req := esapi.PingRequest{}
		res, err := req.Do(ctx, s.clientV8)
		if err != nil {
			return fmt.Errorf("elasticsearch ping failed: %w", err)
		}
		defer func() { _ = res.Body.Close() }()
		if res.IsError() {
			return fmt.Errorf("elasticsearch health check failed: %s", res.String())
		}
	}

	return nil
}

// Close releases resources.
func (s *ElasticsearchLogStorage) Close() error {
	// Elasticsearch client doesn't need explicit closing
	return nil
}

// toDocument converts a LogEntry to Elasticsearch document format.
func (s *ElasticsearchLogStorage) toDocument(entry *LogEntry) map[string]interface{} {
	doc := map[string]interface{}{
		"@timestamp":      entry.Timestamp.Format(time.RFC3339Nano),
		"id":              entry.ID.String(),
		"category":        string(entry.Category),
		"level":           string(entry.Level),
		"message":         entry.Message,
		"custom_category": entry.CustomCategory,
		"request_id":      entry.RequestID,
		"trace_id":        entry.TraceID,
		"component":       entry.Component,
		"user_id":         entry.UserID,
		"ip_address":      entry.IPAddress,
		"fields":          entry.Fields,
		"execution_id":    entry.ExecutionID,
		"line_number":     entry.LineNumber,
	}
	return doc
}

// buildQuery converts LogQueryOptions to Elasticsearch query DSL.
func (s *ElasticsearchLogStorage) buildQuery(opts LogQueryOptions) map[string]interface{} {
	var mustClauses []map[string]interface{}
	var filterClauses []map[string]interface{}
	var shouldClauses []map[string]interface{}

	// Category filter
	if opts.Category != "" {
		filterClauses = append(filterClauses, map[string]interface{}{
			"term": map[string]interface{}{
				"category": string(opts.Category),
			},
		})
	}

	// Custom category filter
	if opts.CustomCategory != "" {
		filterClauses = append(filterClauses, map[string]interface{}{
			"term": map[string]interface{}{
				"custom_category": opts.CustomCategory,
			},
		})
	}

	// Level filter
	if len(opts.Levels) > 0 {
		levels := make([]interface{}, len(opts.Levels))
		for i, level := range opts.Levels {
			levels[i] = string(level)
		}
		filterClauses = append(filterClauses, map[string]interface{}{
			"terms": map[string]interface{}{
				"level": levels,
			},
		})
	}

	// Component filter
	if opts.Component != "" {
		filterClauses = append(filterClauses, map[string]interface{}{
			"term": map[string]interface{}{
				"component": opts.Component,
			},
		})
	}

	// Request ID filter
	if opts.RequestID != "" {
		filterClauses = append(filterClauses, map[string]interface{}{
			"term": map[string]interface{}{
				"request_id": opts.RequestID,
			},
		})
	}

	// Trace ID filter
	if opts.TraceID != "" {
		filterClauses = append(filterClauses, map[string]interface{}{
			"term": map[string]interface{}{
				"trace_id": opts.TraceID,
			},
		})
	}

	// User ID filter
	if opts.UserID != "" {
		filterClauses = append(filterClauses, map[string]interface{}{
			"term": map[string]interface{}{
				"user_id": opts.UserID,
			},
		})
	}

	// Execution ID filter
	if opts.ExecutionID != "" {
		filterClauses = append(filterClauses, map[string]interface{}{
			"term": map[string]interface{}{
				"execution_id": opts.ExecutionID,
			},
		})
	}

	// Execution type filter (stored in fields)
	if opts.ExecutionType != "" {
		filterClauses = append(filterClauses, map[string]interface{}{
			"term": map[string]interface{}{
				"fields.execution_type": opts.ExecutionType,
			},
		})
	}

	// Time range filters
	var rangeClauses map[string]interface{}
	if !opts.StartTime.IsZero() || !opts.EndTime.IsZero() {
		rangeClauses = make(map[string]interface{})
		if !opts.StartTime.IsZero() {
			rangeClauses["gte"] = opts.StartTime.Format(time.RFC3339Nano)
		}
		if !opts.EndTime.IsZero() {
			rangeClauses["lte"] = opts.EndTime.Format(time.RFC3339Nano)
		}
		filterClauses = append(filterClauses, map[string]interface{}{
			"range": map[string]interface{}{
				"@timestamp": rangeClauses,
			},
		})
	}

	// After line filter
	if opts.AfterLine > 0 {
		filterClauses = append(filterClauses, map[string]interface{}{
			"range": map[string]interface{}{
				"line_number": map[string]interface{}{
					"gt": opts.AfterLine,
				},
			},
		})
	}

	// Full-text search
	if opts.Search != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"query_string": map[string]interface{}{
				"query":            "*" + opts.Search + "*",
				"fields":           []string{"message^2", "fields.*"},
				"analyze_wildcard": true,
			},
		})
	}

	// Hide static assets filter
	if opts.HideStaticAssets {
		// Exclude HTTP logs where the path ends with a static asset extension
		staticAssetExtensions := []string{
			".js", ".mjs", ".ts", ".jsx", ".tsx",
			".css",
			".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".ico", ".avif",
			".woff", ".woff2", ".ttf", ".eot", ".otf",
			".map",
		}

		var wildcardPatterns []string
		for _, ext := range staticAssetExtensions {
			wildcardPatterns = append(wildcardPatterns, "*"+ext)
		}

		shouldClauses = append(shouldClauses, map[string]interface{}{
			"bool": map[string]interface{}{
				"must_not": []map[string]interface{}{
					{
						"bool": map[string]interface{}{
							"must": []map[string]interface{}{
								{
									"term": map[string]interface{}{
										"category": "http",
									},
								},
								{
									"wildcard": map[string]interface{}{
										"fields.path": strings.Join(wildcardPatterns, " "),
									},
								},
							},
						},
					},
				},
			},
		})
	}

	// Build the bool query
	query := map[string]interface{}{}
	boolQuery := make(map[string]interface{})

	if len(mustClauses) > 0 {
		boolQuery["must"] = mustClauses
	}
	if len(filterClauses) > 0 {
		boolQuery["filter"] = filterClauses
	}
	if len(shouldClauses) > 0 {
		boolQuery["should"] = shouldClauses
		boolQuery["minimum_should_match"] = 1
	}

	if len(boolQuery) > 0 {
		query["query"] = map[string]interface{}{
			"bool": boolQuery,
		}
	}

	return query
}

// executeSearch executes a search query and returns the entries.
func (s *ElasticsearchLogStorage) executeSearch(ctx context.Context, query map[string]interface{}, from, size int) ([]*LogEntry, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, fmt.Errorf("failed to encode query: %w", err)
	}

	// Execute search based on version
	var body io.ReadCloser
	if s.version == 9 {
		req := esapi9.SearchRequest{
			Index: []string{s.index},
			Body:  &buf,
			From:  &from,
			Size:  &size,
		}
		res, err := req.Do(ctx, s.clientV9)
		if err != nil {
			return nil, fmt.Errorf("failed to execute search: %w", err)
		}
		defer func() { _ = res.Body.Close() }()
		if res.IsError() {
			return nil, fmt.Errorf("elasticsearch search error: %s", res.String())
		}
		body = res.Body
	} else {
		req := esapi.SearchRequest{
			Index: []string{s.index},
			Body:  &buf,
			From:  &from,
			Size:  &size,
		}
		res, err := req.Do(ctx, s.clientV8)
		if err != nil {
			return nil, fmt.Errorf("failed to execute search: %w", err)
		}
		defer func() { _ = res.Body.Close() }()
		if res.IsError() {
			return nil, fmt.Errorf("elasticsearch search error: %s", res.String())
		}
		body = res.Body
	}

	// Parse response
	var searchResponse struct {
		Hits struct {
			Hits []struct {
				Source *LogEntry `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	// Extract entries
	entries := make([]*LogEntry, 0, len(searchResponse.Hits.Hits))
	for _, hit := range searchResponse.Hits.Hits {
		if hit.Source != nil {
			entries = append(entries, hit.Source)
		}
	}

	return entries, nil
}

// Compile-time check that ElasticsearchLogStorage implements LogStorage
var _ LogStorage = (*ElasticsearchLogStorage)(nil)

// OpenSearchLogStorage wraps ElasticsearchLogStorage for OpenSearch compatibility.
// OpenSearch 2.x is API-compatible with Elasticsearch 7.x, so we reuse the ES implementation.
type OpenSearchLogStorage struct {
	*ElasticsearchLogStorage
}

// newOpenSearchLogStorage creates a new OpenSearch-backed log storage.
func newOpenSearchLogStorage(cfg LogStorageConfig) (*OpenSearchLogStorage, error) {
	// Default URLs to localhost if not provided
	urls := cfg.OpenSearchURLs
	if len(urls) == 0 {
		urls = []string{"http://localhost:9200"}
	}

	// Create ES-compatible config
	esCfg := LogStorageConfig{
		ElasticsearchURLs:     urls,
		ElasticsearchUsername: cfg.OpenSearchUsername,
		ElasticsearchPassword: cfg.OpenSearchPassword,
		ElasticsearchIndex:    cfg.OpenSearchIndex,
	}

	// Create base Elasticsearch storage
	esStorage, err := newElasticsearchLogStorage(esCfg)
	if err != nil {
		return nil, err
	}

	return &OpenSearchLogStorage{
		ElasticsearchLogStorage: esStorage,
	}, nil
}

// Name returns the backend identifier.
func (s *OpenSearchLogStorage) Name() string {
	return "opensearch"
}

// Compile-time check that OpenSearchLogStorage implements LogStorage
var _ LogStorage = (*OpenSearchLogStorage)(nil)
