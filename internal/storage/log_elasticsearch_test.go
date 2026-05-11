package storage

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func esTestServer(fn http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fn(w, r)
	}))
}

func countBulkItems(r *http.Request) int {
	body, _ := io.ReadAll(r.Body)
	lineCount := 0
	for _, b := range body {
		if b == '\n' {
			lineCount++
		}
	}
	if len(body) > 0 && body[len(body)-1] != '\n' {
		lineCount++
	}
	return lineCount / 2
}

func writeBulkResponse(w http.ResponseWriter, itemCount int) {
	items := make([]map[string]interface{}, itemCount)
	for i := 0; i < itemCount; i++ {
		items[i] = map[string]interface{}{
			"index": map[string]interface{}{
				"_index": "fluxbase-logs",
				"_id":    fmt.Sprintf("doc-%d", i),
				"status": 201,
				"result": "created",
			},
		}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"took":   0,
		"errors": false,
		"items":  items,
	})
}

func writeSearchResponse(w http.ResponseWriter, total int64, hits []map[string]interface{}) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"took":      1,
		"timed_out": false,
		"hits": map[string]interface{}{
			"total": map[string]interface{}{
				"value":    total,
				"relation": "eq",
			},
			"max_score": 1.0,
			"hits":      hits,
		},
	})
}

func writeDeleteResponse(w http.ResponseWriter, deleted int64) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"took":    10,
		"deleted": deleted,
	})
}

func writeStatsResponse(w http.ResponseWriter, total int64, catBuckets []map[string]interface{}, levelBuckets []map[string]interface{}, minTS, maxTS string) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"took": 1,
		"hits": map[string]interface{}{
			"total": map[string]interface{}{
				"value":    total,
				"relation": "eq",
			},
		},
		"aggregations": map[string]interface{}{
			"categories": map[string]interface{}{
				"buckets": catBuckets,
			},
			"levels": map[string]interface{}{
				"buckets": levelBuckets,
			},
			"min_timestamp": map[string]interface{}{
				"value_as_string": minTS,
			},
			"max_timestamp": map[string]interface{}{
				"value_as_string": maxTS,
			},
		},
	})
}

func entryToSource(entry *LogEntry) map[string]interface{} {
	return map[string]interface{}{
		"id":         entry.ID.String(),
		"@timestamp": entry.Timestamp.Format(time.RFC3339Nano),
		"category":   string(entry.Category),
		"level":      string(entry.Level),
		"message":    entry.Message,
		"component":  entry.Component,
	}
}

func newESStorageWithServer(t *testing.T, version int, fn http.HandlerFunc) (*ElasticsearchLogStorage, *httptest.Server) {
	t.Helper()
	server := esTestServer(fn)
	cfg := LogStorageConfig{
		ElasticsearchURLs:    []string{server.URL},
		ElasticsearchIndex:   "fluxbase-logs",
		ElasticsearchVersion: version,
	}
	storage, err := newElasticsearchLogStorage(cfg)
	require.NoError(t, err)
	require.NotNil(t, storage)
	return storage, server
}

// =============================================================================
// Constructor Tests
// =============================================================================

func TestNewElasticsearchLogStorage_DefaultValues(t *testing.T) {
	t.Run("defaults URLs to localhost:9200", func(t *testing.T) {
		cfg := LogStorageConfig{ElasticsearchVersion: 9}
		storage, err := newElasticsearchLogStorage(cfg)
		require.NoError(t, err)
		require.NotNil(t, storage)
		assert.Equal(t, "fluxbase-logs", storage.index)
		assert.Equal(t, 9, storage.version)
		assert.NotNil(t, storage.clientV9)
	})

	t.Run("defaults version to 9 when zero", func(t *testing.T) {
		cfg := LogStorageConfig{ElasticsearchVersion: 0}
		storage, err := newElasticsearchLogStorage(cfg)
		require.NoError(t, err)
		assert.Equal(t, 9, storage.version)
	})
}

func TestNewElasticsearchLogStorage_Version8(t *testing.T) {
	t.Run("creates v8 client when version is 8", func(t *testing.T) {
		cfg := LogStorageConfig{
			ElasticsearchURLs:    []string{"http://localhost:9200"},
			ElasticsearchVersion: 8,
		}
		storage, err := newElasticsearchLogStorage(cfg)
		require.NoError(t, err)
		require.NotNil(t, storage)
		assert.Equal(t, 8, storage.version)
		assert.NotNil(t, storage.clientV8)
		assert.Nil(t, storage.clientV9)
	})
}

func TestNewElasticsearchLogStorage_Version9(t *testing.T) {
	t.Run("creates v9 client when version is 9", func(t *testing.T) {
		cfg := LogStorageConfig{
			ElasticsearchURLs:    []string{"http://localhost:9200"},
			ElasticsearchVersion: 9,
		}
		storage, err := newElasticsearchLogStorage(cfg)
		require.NoError(t, err)
		require.NotNil(t, storage)
		assert.Equal(t, 9, storage.version)
		assert.NotNil(t, storage.clientV9)
		assert.Nil(t, storage.clientV8)
	})
}

func TestNewElasticsearchLogStorage_InvalidVersion(t *testing.T) {
	t.Run("rejects version 7", func(t *testing.T) {
		cfg := LogStorageConfig{
			ElasticsearchURLs:    []string{"http://localhost:9200"},
			ElasticsearchVersion: 7,
		}
		storage, err := newElasticsearchLogStorage(cfg)
		assert.Nil(t, storage)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported elasticsearch version: 7")
	})

	t.Run("rejects version 10", func(t *testing.T) {
		cfg := LogStorageConfig{
			ElasticsearchURLs:    []string{"http://localhost:9200"},
			ElasticsearchVersion: 10,
		}
		storage, err := newElasticsearchLogStorage(cfg)
		assert.Nil(t, storage)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported elasticsearch version: 10")
	})

	t.Run("rejects version 1", func(t *testing.T) {
		cfg := LogStorageConfig{
			ElasticsearchURLs:    []string{"http://localhost:9200"},
			ElasticsearchVersion: 1,
		}
		storage, err := newElasticsearchLogStorage(cfg)
		assert.Nil(t, storage)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported elasticsearch version: 1")
	})
}

func TestNewElasticsearchLogStorage_CustomIndex(t *testing.T) {
	t.Run("uses custom index name", func(t *testing.T) {
		cfg := LogStorageConfig{
			ElasticsearchIndex:   "my-custom-logs",
			ElasticsearchVersion: 9,
		}
		storage, err := newElasticsearchLogStorage(cfg)
		require.NoError(t, err)
		assert.Equal(t, "my-custom-logs", storage.index)
	})

	t.Run("defaults index to fluxbase-logs when empty", func(t *testing.T) {
		cfg := LogStorageConfig{
			ElasticsearchVersion: 9,
		}
		storage, err := newElasticsearchLogStorage(cfg)
		require.NoError(t, err)
		assert.Equal(t, "fluxbase-logs", storage.index)
	})
}

func TestNewElasticsearchLogStorage_CustomURLs(t *testing.T) {
	t.Run("accepts multiple URLs", func(t *testing.T) {
		cfg := LogStorageConfig{
			ElasticsearchURLs:    []string{"http://es1:9200", "http://es2:9200"},
			ElasticsearchVersion: 9,
		}
		storage, err := newElasticsearchLogStorage(cfg)
		require.NoError(t, err)
		require.NotNil(t, storage)
	})
}

func TestNewElasticsearchLogStorage_WithAuth(t *testing.T) {
	t.Run("stores credentials", func(t *testing.T) {
		cfg := LogStorageConfig{
			ElasticsearchUsername: "elastic",
			ElasticsearchPassword: "changeme",
			ElasticsearchVersion:  9,
		}
		storage, err := newElasticsearchLogStorage(cfg)
		require.NoError(t, err)
		assert.Equal(t, "elastic", storage.username)
		assert.Equal(t, "changeme", storage.password)
	})
}

// =============================================================================
// Name Tests
// =============================================================================

func TestElasticsearchLogStorage_Name(t *testing.T) {
	t.Run("returns elasticsearch", func(t *testing.T) {
		storage := &ElasticsearchLogStorage{version: 9}
		assert.Equal(t, "elasticsearch", storage.Name())
	})
}

// =============================================================================
// Close Tests
// =============================================================================

func TestElasticsearchLogStorage_Close(t *testing.T) {
	t.Run("returns nil", func(t *testing.T) {
		storage := &ElasticsearchLogStorage{version: 9}
		err := storage.Close()
		assert.NoError(t, err)
	})
}

// =============================================================================
// OpenSearch Constructor Tests
// =============================================================================

func TestNewOpenSearchLogStorage_DefaultValues(t *testing.T) {
	t.Run("creates with default localhost URLs", func(t *testing.T) {
		cfg := LogStorageConfig{}
		storage, err := newOpenSearchLogStorage(cfg)
		require.NoError(t, err)
		require.NotNil(t, storage)
		assert.Equal(t, "opensearch", storage.Name())
	})
}

func TestNewOpenSearchLogStorage_WithConfig(t *testing.T) {
	t.Run("maps OpenSearch config to Elasticsearch config", func(t *testing.T) {
		server := esTestServer(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		})
		defer server.Close()

		cfg := LogStorageConfig{
			OpenSearchURLs:     []string{server.URL},
			OpenSearchUsername: "admin",
			OpenSearchPassword: "admin",
			OpenSearchIndex:    "os-logs",
		}
		storage, err := newOpenSearchLogStorage(cfg)
		require.NoError(t, err)
		require.NotNil(t, storage)
		assert.Equal(t, "opensearch", storage.Name())
		assert.Equal(t, "os-logs", storage.ElasticsearchLogStorage.index)
	})
}

func TestOpenSearchLogStorage_Name(t *testing.T) {
	t.Run("returns opensearch", func(t *testing.T) {
		storage := &OpenSearchLogStorage{ElasticsearchLogStorage: &ElasticsearchLogStorage{version: 9}}
		assert.Equal(t, "opensearch", storage.Name())
	})
}

// =============================================================================
// Write Tests
// =============================================================================

func TestElasticsearchLogStorage_Write_EmptyBatch(t *testing.T) {
	t.Run("returns nil for empty entries", func(t *testing.T) {
		storage := &ElasticsearchLogStorage{version: 9, index: "test-logs"}
		err := storage.Write(context.Background(), []*LogEntry{})
		assert.NoError(t, err)
	})

	t.Run("returns nil for nil entries", func(t *testing.T) {
		storage := &ElasticsearchLogStorage{version: 9, index: "test-logs"}
		err := storage.Write(context.Background(), nil)
		assert.NoError(t, err)
	})
}

func TestElasticsearchLogStorage_Write_SingleEntry_V9(t *testing.T) {
	t.Run("writes single entry via bulk API", func(t *testing.T) {
		var receivedMethod string
		var receivedPath string
		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			receivedMethod = r.Method
			receivedPath = r.URL.Path
			if strings.Contains(r.URL.Path, "_bulk") {
				itemCount := countBulkItems(r)
				writeBulkResponse(w, itemCount)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		entry := &LogEntry{
			ID:        uuid.New(),
			Timestamp: time.Now(),
			Category:  LogCategoryHTTP,
			Level:     LogLevelInfo,
			Message:   "Test log message",
		}
		err := storage.Write(context.Background(), []*LogEntry{entry})
		assert.NoError(t, err)
		assert.Equal(t, "POST", receivedMethod)
		assert.Contains(t, receivedPath, "_bulk")
	})
}

func TestElasticsearchLogStorage_Write_SingleEntry_V8(t *testing.T) {
	t.Run("writes single entry via v8 bulk API", func(t *testing.T) {
		storage, server := newESStorageWithServer(t, 8, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_bulk") {
				itemCount := countBulkItems(r)
				writeBulkResponse(w, itemCount)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		entry := &LogEntry{
			ID:        uuid.New(),
			Timestamp: time.Now(),
			Category:  LogCategorySystem,
			Level:     LogLevelDebug,
			Message:   "V8 test",
		}
		err := storage.Write(context.Background(), []*LogEntry{entry})
		assert.NoError(t, err)
	})
}

func TestElasticsearchLogStorage_Write_MultipleEntries(t *testing.T) {
	t.Run("writes multiple entries in single bulk request", func(t *testing.T) {
		var bulkItemCount int
		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_bulk") {
				bulkItemCount = countBulkItems(r)
				writeBulkResponse(w, bulkItemCount)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		entries := []*LogEntry{
			{ID: uuid.New(), Timestamp: time.Now(), Category: LogCategoryHTTP, Level: LogLevelInfo, Message: "Msg 1"},
			{ID: uuid.New(), Timestamp: time.Now(), Category: LogCategoryHTTP, Level: LogLevelWarn, Message: "Msg 2"},
			{ID: uuid.New(), Timestamp: time.Now(), Category: LogCategorySystem, Level: LogLevelError, Message: "Msg 3"},
		}
		err := storage.Write(context.Background(), entries)
		assert.NoError(t, err)
		assert.Equal(t, 3, bulkItemCount)
	})
}

func TestElasticsearchLogStorage_Write_AutoGenerateID(t *testing.T) {
	t.Run("generates UUID for nil entry ID", func(t *testing.T) {
		var generatedIDs []string
		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_bulk") {
				scanner := bufio.NewScanner(r.Body)
				for scanner.Scan() {
					line := scanner.Text()
					if strings.Contains(line, `"index"`) {
						var action map[string]json.RawMessage
						json.Unmarshal([]byte(line), &action)
						if idx, ok := action["index"]; ok {
							var idxData map[string]interface{}
							json.Unmarshal(idx, &idxData)
							if id, ok := idxData["_id"]; ok {
								generatedIDs = append(generatedIDs, fmt.Sprintf("%v", id))
							}
						}
					}
				}
				writeBulkResponse(w, 1)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		entry := &LogEntry{
			ID:        uuid.Nil,
			Timestamp: time.Now(),
			Category:  LogCategoryHTTP,
			Level:     LogLevelInfo,
			Message:   "Auto ID",
		}
		err := storage.Write(context.Background(), []*LogEntry{entry})
		assert.NoError(t, err)
		assert.Len(t, generatedIDs, 1)
		assert.NotEqual(t, uuid.Nil.String(), generatedIDs[0])
	})
}

func TestElasticsearchLogStorage_Write_AutoGenerateTimestamp(t *testing.T) {
	t.Run("generates timestamp for zero entry timestamp", func(t *testing.T) {
		entry := &LogEntry{
			ID:        uuid.New(),
			Timestamp: time.Time{},
			Category:  LogCategoryHTTP,
			Level:     LogLevelInfo,
			Message:   "Auto timestamp",
		}
		originalTimestamp := entry.Timestamp

		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_bulk") {
				writeBulkResponse(w, countBulkItems(r))
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		err := storage.Write(context.Background(), []*LogEntry{entry})
		assert.NoError(t, err)
		assert.True(t, originalTimestamp.IsZero())
		assert.False(t, entry.Timestamp.IsZero())
	})
}

func TestElasticsearchLogStorage_Write_BulkIndexFormat(t *testing.T) {
	t.Run("sends correct NDJSON bulk format", func(t *testing.T) {
		entryID := uuid.New()
		var actionLine map[string]interface{}
		var docLine map[string]interface{}

		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_bulk") {
				scanner := bufio.NewScanner(r.Body)
				lineIdx := 0
				for scanner.Scan() {
					line := scanner.Text()
					if line == "" {
						continue
					}
					if lineIdx == 0 {
						json.Unmarshal([]byte(line), &actionLine)
					} else if lineIdx == 1 {
						json.Unmarshal([]byte(line), &docLine)
					}
					lineIdx++
				}
				writeBulkResponse(w, 1)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		entry := &LogEntry{
			ID:        entryID,
			Timestamp: time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC),
			Category:  LogCategoryHTTP,
			Level:     LogLevelInfo,
			Message:   "Bulk format test",
		}
		err := storage.Write(context.Background(), []*LogEntry{entry})
		require.NoError(t, err)

		require.NotNil(t, actionLine)
		indexAction, ok := actionLine["index"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "fluxbase-logs", indexAction["_index"])
		assert.Equal(t, entryID.String(), indexAction["_id"])

		require.NotNil(t, docLine)
		assert.Equal(t, "http", docLine["category"])
		assert.Equal(t, "info", docLine["level"])
		assert.Equal(t, "Bulk format test", docLine["message"])
		assert.Contains(t, docLine, "@timestamp")
	})
}

func TestElasticsearchLogStorage_Write_ConnectionError(t *testing.T) {
	t.Run("handles unreachable server gracefully", func(t *testing.T) {
		cfg := LogStorageConfig{
			ElasticsearchURLs:    []string{"http://127.0.0.1:1"},
			ElasticsearchVersion: 9,
		}
		storage, err := newElasticsearchLogStorage(cfg)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		entry := &LogEntry{
			ID:        uuid.New(),
			Timestamp: time.Now(),
			Category:  LogCategoryHTTP,
			Level:     LogLevelInfo,
			Message:   "Connection test",
		}
		_ = storage.Write(ctx, []*LogEntry{entry})
	})
}

func TestElasticsearchLogStorage_Write_ServerError(t *testing.T) {
	t.Run("handles server error without panic", func(t *testing.T) {
		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_bulk") {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]interface{}{
						"type":   "server_error",
						"reason": "internal error",
					},
				})
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		entry := &LogEntry{
			ID:        uuid.New(),
			Timestamp: time.Now(),
			Category:  LogCategoryHTTP,
			Level:     LogLevelInfo,
			Message:   "Error test",
		}
		_ = storage.Write(ctx, []*LogEntry{entry})
	})
}

// =============================================================================
// Query Tests
// =============================================================================

func TestElasticsearchLogStorage_Query_NoResults(t *testing.T) {
	t.Run("returns empty result for no matches", func(t *testing.T) {
		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_search") {
				writeSearchResponse(w, 0, nil)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		result, err := storage.Query(context.Background(), LogQueryOptions{})
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.Entries)
		assert.Equal(t, int64(0), result.TotalCount)
		assert.False(t, result.HasMore)
	})
}

func TestElasticsearchLogStorage_Query_WithResults(t *testing.T) {
	t.Run("returns entries from search hits", func(t *testing.T) {
		entry1 := &LogEntry{
			ID:        uuid.New(),
			Timestamp: time.Now(),
			Category:  LogCategoryHTTP,
			Level:     LogLevelInfo,
			Message:   "Test message 1",
			Component: "api",
		}
		entry2 := &LogEntry{
			ID:        uuid.New(),
			Timestamp: time.Now(),
			Category:  LogCategorySystem,
			Level:     LogLevelError,
			Message:   "Test message 2",
		}

		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_search") {
				hits := []map[string]interface{}{
					{"_index": "fluxbase-logs", "_id": entry1.ID.String(), "_score": 1.0, "_source": entryToSource(entry1)},
					{"_index": "fluxbase-logs", "_id": entry2.ID.String(), "_score": 1.0, "_source": entryToSource(entry2)},
				}
				writeSearchResponse(w, 2, hits)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		result, err := storage.Query(context.Background(), LogQueryOptions{})
		require.NoError(t, err)
		assert.Len(t, result.Entries, 2)
		assert.Equal(t, int64(2), result.TotalCount)
	})
}

func TestElasticsearchLogStorage_Query_V8(t *testing.T) {
	t.Run("queries using v8 client", func(t *testing.T) {
		entry := &LogEntry{
			ID:        uuid.New(),
			Timestamp: time.Now(),
			Category:  LogCategoryHTTP,
			Level:     LogLevelInfo,
			Message:   "V8 query test",
		}

		storage, server := newESStorageWithServer(t, 8, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_search") {
				hits := []map[string]interface{}{
					{"_index": "fluxbase-logs", "_id": entry.ID.String(), "_score": 1.0, "_source": entryToSource(entry)},
				}
				writeSearchResponse(w, 1, hits)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		result, err := storage.Query(context.Background(), LogQueryOptions{})
		require.NoError(t, err)
		assert.Len(t, result.Entries, 1)
	})
}

func TestElasticsearchLogStorage_Query_Pagination(t *testing.T) {
	t.Run("returns HasMore when offset plus entries less than total", func(t *testing.T) {
		entry := &LogEntry{
			ID:        uuid.New(),
			Timestamp: time.Now(),
			Category:  LogCategoryHTTP,
			Level:     LogLevelInfo,
			Message:   "Page item",
		}

		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_search") {
				hits := []map[string]interface{}{
					{"_index": "fluxbase-logs", "_id": entry.ID.String(), "_score": 1.0, "_source": entryToSource(entry)},
				}
				writeSearchResponse(w, 100, hits)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		opts := LogQueryOptions{Limit: 1, Offset: 0}
		result, err := storage.Query(context.Background(), opts)
		require.NoError(t, err)
		assert.True(t, result.HasMore)
		assert.Equal(t, int64(100), result.TotalCount)
	})

	t.Run("defaults limit to 100 when zero", func(t *testing.T) {
		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_search") {
				writeSearchResponse(w, 0, nil)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		result, err := storage.Query(context.Background(), LogQueryOptions{Limit: 0})
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("defaults offset to 0 when negative", func(t *testing.T) {
		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_search") {
				writeSearchResponse(w, 0, nil)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		result, err := storage.Query(context.Background(), LogQueryOptions{Offset: -5})
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestElasticsearchLogStorage_Query_HasMore(t *testing.T) {
	t.Run("sets HasMore true when more results exist", func(t *testing.T) {
		entry := &LogEntry{
			ID:        uuid.New(),
			Timestamp: time.Now(),
			Category:  LogCategoryHTTP,
			Level:     LogLevelInfo,
			Message:   "Test",
		}

		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_search") {
				hits := []map[string]interface{}{
					{"_index": "fluxbase-logs", "_id": entry.ID.String(), "_score": 1.0, "_source": entryToSource(entry)},
				}
				writeSearchResponse(w, 100, hits)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		result, err := storage.Query(context.Background(), LogQueryOptions{Limit: 1, Offset: 0})
		require.NoError(t, err)
		assert.True(t, result.HasMore)
		assert.Equal(t, int64(100), result.TotalCount)
	})

	t.Run("sets HasMore false when all results returned", func(t *testing.T) {
		entry := &LogEntry{
			ID:        uuid.New(),
			Timestamp: time.Now(),
			Category:  LogCategoryHTTP,
			Level:     LogLevelInfo,
			Message:   "Item",
		}

		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_search") {
				hits := []map[string]interface{}{
					{"_index": "fluxbase-logs", "_id": entry.ID.String(), "_score": 1.0, "_source": entryToSource(entry)},
				}
				writeSearchResponse(w, 1, hits)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		result, err := storage.Query(context.Background(), LogQueryOptions{Limit: 10, Offset: 0})
		require.NoError(t, err)
		assert.False(t, result.HasMore)
	})
}

func TestElasticsearchLogStorage_Query_SortAsc(t *testing.T) {
	t.Run("sends ascending sort order via URL parameter", func(t *testing.T) {
		var sortParam string
		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_search") {
				sortParam = r.URL.Query().Get("sort")
				writeSearchResponse(w, 0, nil)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		_, err := storage.Query(context.Background(), LogQueryOptions{SortAsc: true})
		require.NoError(t, err)
		assert.Contains(t, sortParam, "@timestamp")
		assert.Contains(t, sortParam, "asc")
	})

	t.Run("sends descending sort order by default via URL parameter", func(t *testing.T) {
		var sortParam string
		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_search") {
				sortParam = r.URL.Query().Get("sort")
				writeSearchResponse(w, 0, nil)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		_, err := storage.Query(context.Background(), LogQueryOptions{SortAsc: false})
		require.NoError(t, err)
		assert.Contains(t, sortParam, "@timestamp")
		assert.Contains(t, sortParam, "desc")
	})
}

func TestElasticsearchLogStorage_Query_SendsQueryDSL(t *testing.T) {
	t.Run("sends category filter in query body", func(t *testing.T) {
		var receivedBody string
		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_search") {
				body, _ := io.ReadAll(r.Body)
				receivedBody = string(body)
				writeSearchResponse(w, 0, nil)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		opts := LogQueryOptions{
			Category:  LogCategoryHTTP,
			Levels:    []LogLevel{LogLevelInfo, LogLevelWarn},
			Component: "api",
		}
		_, err := storage.Query(context.Background(), opts)
		require.NoError(t, err)

		assert.Contains(t, receivedBody, "http")
		assert.Contains(t, receivedBody, "info")
		assert.Contains(t, receivedBody, "warn")
		assert.Contains(t, receivedBody, "api")
	})
}

func TestElasticsearchLogStorage_Query_ServerError(t *testing.T) {
	t.Run("returns error when search returns 500", func(t *testing.T) {
		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_search") {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, `{"error":{"type":"search_error"}}`)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		_, err := storage.Query(context.Background(), LogQueryOptions{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "elasticsearch search error")
	})
}

func TestElasticsearchLogStorage_Query_MalformedResponse(t *testing.T) {
	t.Run("returns error for invalid JSON response", func(t *testing.T) {
		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_search") {
				fmt.Fprint(w, "not valid json")
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		_, err := storage.Query(context.Background(), LogQueryOptions{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode search response")
	})
}

func TestElasticsearchLogStorage_Query_ConnectionError(t *testing.T) {
	t.Run("returns error on connection failure", func(t *testing.T) {
		cfg := LogStorageConfig{
			ElasticsearchURLs:    []string{"http://127.0.0.1:1"},
			ElasticsearchVersion: 9,
		}
		storage, err := newElasticsearchLogStorage(cfg)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err = storage.Query(ctx, LogQueryOptions{})
		assert.Error(t, err)
	})
}

func TestElasticsearchLogStorage_Query_WithAuth(t *testing.T) {
	t.Run("sends auth credentials", func(t *testing.T) {
		var authUser, authPass string
		server := esTestServer(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_search") {
				authUser, authPass, _ = r.BasicAuth()
				writeSearchResponse(w, 0, nil)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		cfg := LogStorageConfig{
			ElasticsearchURLs:     []string{server.URL},
			ElasticsearchUsername: "elastic",
			ElasticsearchPassword: "s3cret",
			ElasticsearchVersion:  9,
		}
		storage, err := newElasticsearchLogStorage(cfg)
		require.NoError(t, err)

		_, err = storage.Query(context.Background(), LogQueryOptions{})
		require.NoError(t, err)
		assert.Equal(t, "elastic", authUser)
		assert.Equal(t, "s3cret", authPass)
	})
}

// =============================================================================
// GetExecutionLogs Tests
// =============================================================================

func TestElasticsearchLogStorage_GetExecutionLogs(t *testing.T) {
	t.Run("queries execution logs by execution_id", func(t *testing.T) {
		var receivedBody string
		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_search") {
				body, _ := io.ReadAll(r.Body)
				receivedBody = string(body)
				writeSearchResponse(w, 0, nil)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		_, err := storage.GetExecutionLogs(context.Background(), "exec-123", 0)
		require.NoError(t, err)
		assert.Contains(t, receivedBody, "exec-123")
		assert.Contains(t, receivedBody, "execution_id")
	})

	t.Run("filters by after_line", func(t *testing.T) {
		var receivedBody string
		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_search") {
				body, _ := io.ReadAll(r.Body)
				receivedBody = string(body)
				writeSearchResponse(w, 0, nil)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		_, err := storage.GetExecutionLogs(context.Background(), "exec-456", 50)
		require.NoError(t, err)
		assert.Contains(t, receivedBody, "line_number")
		assert.Contains(t, receivedBody, "50")
	})

	t.Run("sorts by line_number ascending", func(t *testing.T) {
		var receivedBody string
		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_search") {
				body, _ := io.ReadAll(r.Body)
				receivedBody = string(body)
				writeSearchResponse(w, 0, nil)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		_, err := storage.GetExecutionLogs(context.Background(), "exec-789", 0)
		require.NoError(t, err)
		assert.Contains(t, receivedBody, "line_number")
		assert.Contains(t, receivedBody, "asc")
	})

	t.Run("returns parsed entries", func(t *testing.T) {
		entry := &LogEntry{
			ID:          uuid.New(),
			Timestamp:   time.Now(),
			Category:    LogCategoryExecution,
			Level:       LogLevelInfo,
			Message:     "Line 10 output",
			ExecutionID: "exec-abc",
			LineNumber:  10,
		}

		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_search") {
				hits := []map[string]interface{}{
					{"_index": "fluxbase-logs", "_id": entry.ID.String(), "_source": entryToSource(entry)},
				}
				writeSearchResponse(w, 1, hits)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		entries, err := storage.GetExecutionLogs(context.Background(), "exec-abc", 5)
		require.NoError(t, err)
		assert.Len(t, entries, 1)
		assert.Equal(t, "Line 10 output", entries[0].Message)
	})
}

// =============================================================================
// Delete Tests
// =============================================================================

func TestElasticsearchLogStorage_Delete_WithFilter(t *testing.T) {
	t.Run("deletes entries matching filter", func(t *testing.T) {
		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_delete_by_query") {
				writeDeleteResponse(w, 42)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		opts := LogQueryOptions{Category: LogCategoryHTTP}
		deleted, err := storage.Delete(context.Background(), opts)
		require.NoError(t, err)
		assert.Equal(t, int64(42), deleted)
	})
}

func TestElasticsearchLogStorage_Delete_WithFilter_V8(t *testing.T) {
	t.Run("deletes entries via v8 client", func(t *testing.T) {
		storage, server := newESStorageWithServer(t, 8, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_delete_by_query") {
				writeDeleteResponse(w, 10)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		opts := LogQueryOptions{Category: LogCategorySystem}
		deleted, err := storage.Delete(context.Background(), opts)
		require.NoError(t, err)
		assert.Equal(t, int64(10), deleted)
	})
}

func TestElasticsearchLogStorage_Delete_NoFilter(t *testing.T) {
	t.Run("returns error when no filter is set", func(t *testing.T) {
		storage := &ElasticsearchLogStorage{version: 9, index: "test-logs"}
		_, err := storage.Delete(context.Background(), LogQueryOptions{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "delete requires at least one filter condition")
	})
}

func TestElasticsearchLogStorage_Delete_ServerError(t *testing.T) {
	t.Run("returns error when delete returns 500", func(t *testing.T) {
		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_delete_by_query") {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, `{"error":{"type":"delete_error"}}`)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		opts := LogQueryOptions{Category: LogCategoryHTTP}
		_, err := storage.Delete(context.Background(), opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "elasticsearch delete error")
	})
}

func TestElasticsearchLogStorage_Delete_MalformedResponse(t *testing.T) {
	t.Run("returns error for invalid JSON response", func(t *testing.T) {
		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_delete_by_query") {
				fmt.Fprint(w, "not json")
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		opts := LogQueryOptions{Category: LogCategoryHTTP}
		_, err := storage.Delete(context.Background(), opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode delete response")
	})
}

func TestElasticsearchLogStorage_Delete_ConnectionError(t *testing.T) {
	t.Run("returns error on connection failure", func(t *testing.T) {
		cfg := LogStorageConfig{
			ElasticsearchURLs:    []string{"http://127.0.0.1:1"},
			ElasticsearchVersion: 9,
		}
		storage, err := newElasticsearchLogStorage(cfg)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		opts := LogQueryOptions{Category: LogCategoryHTTP}
		_, err = storage.Delete(ctx, opts)
		assert.Error(t, err)
	})
}

// =============================================================================
// Stats Tests
// =============================================================================

func TestElasticsearchLogStorage_Stats(t *testing.T) {
	t.Run("returns stats from aggregation response", func(t *testing.T) {
		minTS := "2024-01-01T00:00:00.000Z"
		maxTS := "2024-06-15T12:30:00.000Z"

		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_search") {
				catBuckets := []map[string]interface{}{
					{"key": "http", "doc_count": 50},
					{"key": "system", "doc_count": 30},
				}
				levelBuckets := []map[string]interface{}{
					{"key": "info", "doc_count": 60},
					{"key": "warn", "doc_count": 20},
				}
				writeStatsResponse(w, 80, catBuckets, levelBuckets, minTS, maxTS)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		stats, err := storage.Stats(context.Background())
		require.NoError(t, err)
		assert.Equal(t, int64(80), stats.TotalEntries)
		assert.Equal(t, int64(50), stats.EntriesByCategory[LogCategoryHTTP])
		assert.Equal(t, int64(30), stats.EntriesByCategory[LogCategorySystem])
		assert.Equal(t, int64(60), stats.EntriesByLevel[LogLevelInfo])
		assert.Equal(t, int64(20), stats.EntriesByLevel[LogLevelWarn])
		assert.NotNil(t, stats.OldestEntry)
		assert.NotNil(t, stats.NewestEntry)
	})
}

func TestElasticsearchLogStorage_Stats_V8(t *testing.T) {
	t.Run("returns stats via v8 client", func(t *testing.T) {
		storage, server := newESStorageWithServer(t, 8, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_search") {
				writeStatsResponse(w, 10, nil, nil, "", "")
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		stats, err := storage.Stats(context.Background())
		require.NoError(t, err)
		assert.Equal(t, int64(10), stats.TotalEntries)
		assert.Nil(t, stats.OldestEntry)
		assert.Nil(t, stats.NewestEntry)
	})
}

func TestElasticsearchLogStorage_Stats_EmptyAggregations(t *testing.T) {
	t.Run("handles empty buckets gracefully", func(t *testing.T) {
		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_search") {
				writeStatsResponse(w, 0, nil, nil, "", "")
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		stats, err := storage.Stats(context.Background())
		require.NoError(t, err)
		assert.Equal(t, int64(0), stats.TotalEntries)
		assert.Empty(t, stats.EntriesByCategory)
		assert.Empty(t, stats.EntriesByLevel)
		assert.Nil(t, stats.OldestEntry)
		assert.Nil(t, stats.NewestEntry)
	})
}

func TestElasticsearchLogStorage_Stats_ServerError(t *testing.T) {
	t.Run("returns error when stats returns 500", func(t *testing.T) {
		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_search") {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, `{"error":{"type":"search_error"}}`)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		_, err := storage.Stats(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "elasticsearch stats error")
	})
}

func TestElasticsearchLogStorage_Stats_MalformedResponse(t *testing.T) {
	t.Run("returns error for invalid JSON", func(t *testing.T) {
		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_search") {
				fmt.Fprint(w, "not json")
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		_, err := storage.Stats(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode stats response")
	})
}

func TestElasticsearchLogStorage_Stats_ConnectionError(t *testing.T) {
	t.Run("returns error on connection failure", func(t *testing.T) {
		cfg := LogStorageConfig{
			ElasticsearchURLs:    []string{"http://127.0.0.1:1"},
			ElasticsearchVersion: 9,
		}
		storage, err := newElasticsearchLogStorage(cfg)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err = storage.Stats(ctx)
		assert.Error(t, err)
	})
}

// =============================================================================
// Health Tests
// =============================================================================

func TestElasticsearchLogStorage_Health_Healthy(t *testing.T) {
	t.Run("returns nil when cluster is healthy", func(t *testing.T) {
		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "HEAD" {
				w.WriteHeader(200)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := storage.Health(ctx)
		assert.NoError(t, err)
	})
}

func TestElasticsearchLogStorage_Health_Healthy_V8(t *testing.T) {
	t.Run("returns nil when cluster is healthy via v8", func(t *testing.T) {
		storage, server := newESStorageWithServer(t, 8, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "HEAD" {
				w.WriteHeader(200)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := storage.Health(ctx)
		assert.NoError(t, err)
	})
}

func TestElasticsearchLogStorage_Health_Unhealthy(t *testing.T) {
	t.Run("returns error when cluster returns 500", func(t *testing.T) {
		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "HEAD" {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := storage.Health(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "elasticsearch health check failed")
	})
}

func TestElasticsearchLogStorage_Health_ConnectionError(t *testing.T) {
	t.Run("returns error on connection failure", func(t *testing.T) {
		cfg := LogStorageConfig{
			ElasticsearchURLs:    []string{"http://127.0.0.1:1"},
			ElasticsearchVersion: 9,
		}
		storage, err := newElasticsearchLogStorage(cfg)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = storage.Health(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "elasticsearch ping failed")
	})
}

// =============================================================================
// buildQuery Tests (unit, no HTTP)
// =============================================================================

func TestElasticsearchLogStorage_buildQuery_Empty(t *testing.T) {
	t.Run("returns empty query for no options", func(t *testing.T) {
		storage := &ElasticsearchLogStorage{index: "test-logs", version: 9}
		query := storage.buildQuery(LogQueryOptions{})
		assert.Empty(t, query)
	})
}

func TestElasticsearchLogStorage_buildQuery_Category(t *testing.T) {
	t.Run("adds category term filter", func(t *testing.T) {
		storage := &ElasticsearchLogStorage{index: "test-logs", version: 9}
		query := storage.buildQuery(LogQueryOptions{Category: LogCategoryHTTP})
		assert.Contains(t, query, "query")

		queryJSON, err := json.Marshal(query)
		require.NoError(t, err)
		assert.Contains(t, string(queryJSON), `"category"`)
		assert.Contains(t, string(queryJSON), `"http"`)
	})
}

func TestElasticsearchLogStorage_buildQuery_CustomCategory(t *testing.T) {
	t.Run("adds custom_category term filter", func(t *testing.T) {
		storage := &ElasticsearchLogStorage{index: "test-logs", version: 9}
		query := storage.buildQuery(LogQueryOptions{CustomCategory: "my_category"})

		queryJSON, err := json.Marshal(query)
		require.NoError(t, err)
		assert.Contains(t, string(queryJSON), `"custom_category"`)
		assert.Contains(t, string(queryJSON), `"my_category"`)
	})
}

func TestElasticsearchLogStorage_buildQuery_Levels(t *testing.T) {
	t.Run("adds single level filter", func(t *testing.T) {
		storage := &ElasticsearchLogStorage{index: "test-logs", version: 9}
		query := storage.buildQuery(LogQueryOptions{Levels: []LogLevel{LogLevelInfo}})

		queryJSON, err := json.Marshal(query)
		require.NoError(t, err)
		assert.Contains(t, string(queryJSON), `"level"`)
		assert.Contains(t, string(queryJSON), `"info"`)
	})

	t.Run("adds multiple levels filter", func(t *testing.T) {
		storage := &ElasticsearchLogStorage{index: "test-logs", version: 9}
		query := storage.buildQuery(LogQueryOptions{Levels: []LogLevel{LogLevelInfo, LogLevelWarn, LogLevelError}})

		queryJSON, err := json.Marshal(query)
		require.NoError(t, err)
		assert.Contains(t, string(queryJSON), `"level"`)
		assert.Contains(t, string(queryJSON), `"info"`)
		assert.Contains(t, string(queryJSON), `"warn"`)
		assert.Contains(t, string(queryJSON), `"error"`)
		assert.Contains(t, string(queryJSON), `"terms"`)
	})
}

func TestElasticsearchLogStorage_buildQuery_Component(t *testing.T) {
	t.Run("adds component term filter", func(t *testing.T) {
		storage := &ElasticsearchLogStorage{index: "test-logs", version: 9}
		query := storage.buildQuery(LogQueryOptions{Component: "auth"})

		queryJSON, err := json.Marshal(query)
		require.NoError(t, err)
		assert.Contains(t, string(queryJSON), `"component"`)
		assert.Contains(t, string(queryJSON), `"auth"`)
	})
}

func TestElasticsearchLogStorage_buildQuery_RequestID(t *testing.T) {
	t.Run("adds request_id term filter", func(t *testing.T) {
		storage := &ElasticsearchLogStorage{index: "test-logs", version: 9}
		query := storage.buildQuery(LogQueryOptions{RequestID: "req-123"})

		queryJSON, err := json.Marshal(query)
		require.NoError(t, err)
		assert.Contains(t, string(queryJSON), `"request_id"`)
		assert.Contains(t, string(queryJSON), `"req-123"`)
	})
}

func TestElasticsearchLogStorage_buildQuery_TraceID(t *testing.T) {
	t.Run("adds trace_id term filter", func(t *testing.T) {
		storage := &ElasticsearchLogStorage{index: "test-logs", version: 9}
		query := storage.buildQuery(LogQueryOptions{TraceID: "trace-456"})

		queryJSON, err := json.Marshal(query)
		require.NoError(t, err)
		assert.Contains(t, string(queryJSON), `"trace_id"`)
		assert.Contains(t, string(queryJSON), `"trace-456"`)
	})
}

func TestElasticsearchLogStorage_buildQuery_UserID(t *testing.T) {
	t.Run("adds user_id term filter", func(t *testing.T) {
		storage := &ElasticsearchLogStorage{index: "test-logs", version: 9}
		query := storage.buildQuery(LogQueryOptions{UserID: "user-789"})

		queryJSON, err := json.Marshal(query)
		require.NoError(t, err)
		assert.Contains(t, string(queryJSON), `"user_id"`)
		assert.Contains(t, string(queryJSON), `"user-789"`)
	})
}

func TestElasticsearchLogStorage_buildQuery_ExecutionID(t *testing.T) {
	t.Run("adds execution_id term filter", func(t *testing.T) {
		storage := &ElasticsearchLogStorage{index: "test-logs", version: 9}
		query := storage.buildQuery(LogQueryOptions{ExecutionID: "exec-abc"})

		queryJSON, err := json.Marshal(query)
		require.NoError(t, err)
		assert.Contains(t, string(queryJSON), `"execution_id"`)
		assert.Contains(t, string(queryJSON), `"exec-abc"`)
	})
}

func TestElasticsearchLogStorage_buildQuery_ExecutionType(t *testing.T) {
	t.Run("adds execution_type field filter", func(t *testing.T) {
		storage := &ElasticsearchLogStorage{index: "test-logs", version: 9}
		query := storage.buildQuery(LogQueryOptions{ExecutionType: "function"})

		queryJSON, err := json.Marshal(query)
		require.NoError(t, err)
		assert.Contains(t, string(queryJSON), `"fields.execution_type"`)
		assert.Contains(t, string(queryJSON), `"function"`)
	})
}

func TestElasticsearchLogStorage_buildQuery_TimeRange(t *testing.T) {
	t.Run("adds start time range filter", func(t *testing.T) {
		storage := &ElasticsearchLogStorage{index: "test-logs", version: 9}
		startTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		query := storage.buildQuery(LogQueryOptions{StartTime: startTime})

		queryJSON, err := json.Marshal(query)
		require.NoError(t, err)
		assert.Contains(t, string(queryJSON), `"@timestamp"`)
		assert.Contains(t, string(queryJSON), `"gte"`)
		assert.Contains(t, string(queryJSON), "2024-01-01")
	})

	t.Run("adds end time range filter", func(t *testing.T) {
		storage := &ElasticsearchLogStorage{index: "test-logs", version: 9}
		endTime := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
		query := storage.buildQuery(LogQueryOptions{EndTime: endTime})

		queryJSON, err := json.Marshal(query)
		require.NoError(t, err)
		assert.Contains(t, string(queryJSON), `"@timestamp"`)
		assert.Contains(t, string(queryJSON), `"lte"`)
		assert.Contains(t, string(queryJSON), "2024-12-31")
	})

	t.Run("adds both start and end time range", func(t *testing.T) {
		storage := &ElasticsearchLogStorage{index: "test-logs", version: 9}
		startTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		endTime := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
		query := storage.buildQuery(LogQueryOptions{StartTime: startTime, EndTime: endTime})

		queryJSON, err := json.Marshal(query)
		require.NoError(t, err)
		assert.Contains(t, string(queryJSON), `"gte"`)
		assert.Contains(t, string(queryJSON), `"lte"`)
	})
}

func TestElasticsearchLogStorage_buildQuery_AfterLine(t *testing.T) {
	t.Run("adds line_number range filter", func(t *testing.T) {
		storage := &ElasticsearchLogStorage{index: "test-logs", version: 9}
		query := storage.buildQuery(LogQueryOptions{AfterLine: 50})

		queryJSON, err := json.Marshal(query)
		require.NoError(t, err)
		assert.Contains(t, string(queryJSON), `"line_number"`)
		assert.Contains(t, string(queryJSON), `"gt"`)
	})
}

func TestElasticsearchLogStorage_buildQuery_Search(t *testing.T) {
	t.Run("adds query_string for full-text search", func(t *testing.T) {
		storage := &ElasticsearchLogStorage{index: "test-logs", version: 9}
		query := storage.buildQuery(LogQueryOptions{Search: "error message"})

		queryJSON, err := json.Marshal(query)
		require.NoError(t, err)
		assert.Contains(t, string(queryJSON), `"query_string"`)
		assert.Contains(t, string(queryJSON), `"*error message*"`)
		assert.Contains(t, string(queryJSON), `"message^2"`)
		assert.Contains(t, string(queryJSON), `"fields.*"`)
		assert.Contains(t, string(queryJSON), `"analyze_wildcard"`)
	})
}

func TestElasticsearchLogStorage_buildQuery_HideStaticAssets(t *testing.T) {
	t.Run("adds should clause with static asset exclusion", func(t *testing.T) {
		storage := &ElasticsearchLogStorage{index: "test-logs", version: 9}
		query := storage.buildQuery(LogQueryOptions{HideStaticAssets: true})

		queryJSON, err := json.Marshal(query)
		require.NoError(t, err)
		assert.Contains(t, string(queryJSON), `"should"`)
		assert.Contains(t, string(queryJSON), `"minimum_should_match"`)
		assert.Contains(t, string(queryJSON), "*.js")
		assert.Contains(t, string(queryJSON), "*.css")
		assert.Contains(t, string(queryJSON), "*.png")
		assert.Contains(t, string(queryJSON), "*.svg")
	})
}

func TestElasticsearchLogStorage_buildQuery_CombinedFilters(t *testing.T) {
	t.Run("combines multiple filter types", func(t *testing.T) {
		storage := &ElasticsearchLogStorage{index: "test-logs", version: 9}
		opts := LogQueryOptions{
			Category:      LogCategoryHTTP,
			Levels:        []LogLevel{LogLevelInfo, LogLevelWarn},
			Component:     "api",
			RequestID:     "req-1",
			TraceID:       "trace-1",
			UserID:        "user-1",
			ExecutionID:   "exec-1",
			ExecutionType: "function",
			StartTime:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndTime:       time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
			Search:        "test query",
		}
		query := storage.buildQuery(opts)

		require.Contains(t, query, "query")
		queryJSON, err := json.Marshal(query)
		require.NoError(t, err)

		s := string(queryJSON)
		assert.Contains(t, s, `"category"`)
		assert.Contains(t, s, `"http"`)
		assert.Contains(t, s, `"level"`)
		assert.Contains(t, s, `"component"`)
		assert.Contains(t, s, `"api"`)
		assert.Contains(t, s, `"request_id"`)
		assert.Contains(t, s, `"trace_id"`)
		assert.Contains(t, s, `"user_id"`)
		assert.Contains(t, s, `"execution_id"`)
		assert.Contains(t, s, `"fields.execution_type"`)
		assert.Contains(t, s, `"@timestamp"`)
		assert.Contains(t, s, `"query_string"`)
	})
}

// =============================================================================
// toDocument Tests (unit, no HTTP)
// =============================================================================

func TestElasticsearchLogStorage_toDocument_AllFields(t *testing.T) {
	t.Run("converts all fields to document", func(t *testing.T) {
		storage := &ElasticsearchLogStorage{index: "test-logs", version: 9}
		entryID := uuid.New()
		ts := time.Date(2024, 6, 15, 10, 30, 0, 123456789, time.UTC)
		entry := &LogEntry{
			ID:             entryID,
			Timestamp:      ts,
			Category:       LogCategoryHTTP,
			Level:          LogLevelInfo,
			Message:        "Test message",
			CustomCategory: "custom_cat",
			RequestID:      "req-1",
			TraceID:        "trace-1",
			Component:      "api",
			UserID:         "user-1",
			IPAddress:      "192.168.1.1",
			Fields:         map[string]any{"key": "value"},
			ExecutionID:    "exec-1",
			LineNumber:     42,
		}

		doc := storage.toDocument(entry)

		assert.Equal(t, ts.Format(time.RFC3339Nano), doc["@timestamp"])
		assert.Equal(t, entryID.String(), doc["id"])
		assert.Equal(t, "http", doc["category"])
		assert.Equal(t, "info", doc["level"])
		assert.Equal(t, "Test message", doc["message"])
		assert.Equal(t, "custom_cat", doc["custom_category"])
		assert.Equal(t, "req-1", doc["request_id"])
		assert.Equal(t, "trace-1", doc["trace_id"])
		assert.Equal(t, "api", doc["component"])
		assert.Equal(t, "user-1", doc["user_id"])
		assert.Equal(t, "192.168.1.1", doc["ip_address"])
		assert.Equal(t, "exec-1", doc["execution_id"])
		assert.Equal(t, 42, doc["line_number"])
		assert.NotNil(t, doc["fields"])
	})
}

func TestElasticsearchLogStorage_toDocument_MinimalFields(t *testing.T) {
	t.Run("handles entry with minimal fields", func(t *testing.T) {
		storage := &ElasticsearchLogStorage{index: "test-logs", version: 9}
		entryID := uuid.New()
		entry := &LogEntry{
			ID:        entryID,
			Timestamp: time.Now(),
			Category:  LogCategorySystem,
			Level:     LogLevelDebug,
			Message:   "Simple log",
		}

		doc := storage.toDocument(entry)

		assert.Equal(t, entryID.String(), doc["id"])
		assert.Equal(t, "system", doc["category"])
		assert.Equal(t, "debug", doc["level"])
		assert.Equal(t, "Simple log", doc["message"])
		assert.Equal(t, "", doc["custom_category"])
		assert.Equal(t, "", doc["request_id"])
		assert.Equal(t, "", doc["component"])
	})
}

// =============================================================================
// Index Name Tests
// =============================================================================

func TestElasticsearchLogStorage_IndexName(t *testing.T) {
	t.Run("uses configured index in bulk requests", func(t *testing.T) {
		var actionLine map[string]interface{}
		server := esTestServer(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "_bulk") {
				scanner := bufio.NewScanner(r.Body)
				if scanner.Scan() {
					json.Unmarshal([]byte(scanner.Text()), &actionLine)
				}
				writeBulkResponse(w, countBulkItems(r))
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		cfg := LogStorageConfig{
			ElasticsearchURLs:    []string{server.URL},
			ElasticsearchIndex:   "custom-index-name",
			ElasticsearchVersion: 9,
		}
		storage, err := newElasticsearchLogStorage(cfg)
		require.NoError(t, err)

		entry := &LogEntry{
			ID:        uuid.New(),
			Timestamp: time.Now(),
			Category:  LogCategoryHTTP,
			Level:     LogLevelInfo,
			Message:   "Index test",
		}
		err = storage.Write(context.Background(), []*LogEntry{entry})
		require.NoError(t, err)

		require.NotNil(t, actionLine)
		indexAction, ok := actionLine["index"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "custom-index-name", indexAction["_index"])
	})
}

func TestElasticsearchLogStorage_SearchUsesCorrectIndex(t *testing.T) {
	t.Run("searches against configured index", func(t *testing.T) {
		var requestedPath string
		server := esTestServer(func(w http.ResponseWriter, r *http.Request) {
			requestedPath = r.URL.Path
			if strings.Contains(r.URL.Path, "_search") {
				writeSearchResponse(w, 0, nil)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		cfg := LogStorageConfig{
			ElasticsearchURLs:    []string{server.URL},
			ElasticsearchIndex:   "my-app-logs",
			ElasticsearchVersion: 9,
		}
		storage, err := newElasticsearchLogStorage(cfg)
		require.NoError(t, err)

		_, err = storage.Query(context.Background(), LogQueryOptions{})
		require.NoError(t, err)
		assert.Contains(t, requestedPath, "my-app-logs")
	})
}

// =============================================================================
// Context Cancellation Tests
// =============================================================================

func TestElasticsearchLogStorage_Write_CancelledContext(t *testing.T) {
	t.Run("returns error when context is cancelled", func(t *testing.T) {
		storage, server := newESStorageWithServer(t, 9, func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			if strings.Contains(r.URL.Path, "_bulk") {
				writeBulkResponse(w, 0)
				return
			}
			w.WriteHeader(200)
		})
		defer server.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		entry := &LogEntry{
			ID:        uuid.New(),
			Timestamp: time.Now(),
			Category:  LogCategoryHTTP,
			Level:     LogLevelInfo,
			Message:   "Cancelled context test",
		}
		err := storage.Write(ctx, []*LogEntry{entry})
		assert.Error(t, err)
	})
}

func TestElasticsearchLogStorage_Query_CancelledContext(t *testing.T) {
	t.Run("returns error when context is cancelled", func(t *testing.T) {
		cfg := LogStorageConfig{
			ElasticsearchURLs:    []string{"http://127.0.0.1:1"},
			ElasticsearchVersion: 9,
		}
		storage, err := newElasticsearchLogStorage(cfg)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err = storage.Query(ctx, LogQueryOptions{})
		assert.Error(t, err)
	})
}

// =============================================================================
// LogStorage Interface Compliance Tests
// =============================================================================

func TestElasticsearchLogStorage_ImplementsLogStorage(t *testing.T) {
	t.Run("implements LogStorage interface", func(t *testing.T) {
		var _ LogStorage = (*ElasticsearchLogStorage)(nil)
	})
}

func TestOpenSearchLogStorage_ImplementsLogStorage(t *testing.T) {
	t.Run("implements LogStorage interface", func(t *testing.T) {
		var _ LogStorage = (*OpenSearchLogStorage)(nil)
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkElasticsearchLogStorage_buildQuery_Simple(b *testing.B) {
	storage := &ElasticsearchLogStorage{index: "test-logs", version: 9}
	opts := LogQueryOptions{
		Category: LogCategoryHTTP,
		Levels:   []LogLevel{LogLevelInfo},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = storage.buildQuery(opts)
	}
}

func BenchmarkElasticsearchLogStorage_buildQuery_Complex(b *testing.B) {
	storage := &ElasticsearchLogStorage{index: "test-logs", version: 9}
	opts := LogQueryOptions{
		Category:         LogCategoryHTTP,
		Levels:           []LogLevel{LogLevelInfo, LogLevelWarn, LogLevelError},
		Component:        "auth",
		UserID:           uuid.New().String(),
		ExecutionID:      "exec-1",
		ExecutionType:    "function",
		StartTime:        time.Now().Add(-24 * time.Hour),
		EndTime:          time.Now(),
		Search:           "failed login",
		HideStaticAssets: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = storage.buildQuery(opts)
	}
}

func BenchmarkElasticsearchLogStorage_toDocument(b *testing.B) {
	storage := &ElasticsearchLogStorage{index: "test-logs", version: 9}
	entry := &LogEntry{
		ID:        uuid.New(),
		Timestamp: time.Now(),
		Category:  LogCategoryHTTP,
		Level:     LogLevelInfo,
		Message:   "Benchmark log message with some content",
		Component: "api",
		Fields: map[string]any{
			"status_code": 200,
			"path":        "/api/test",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = storage.toDocument(entry)
	}
}
