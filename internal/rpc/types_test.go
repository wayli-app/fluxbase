package rpc

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// ExecutionStatus Tests
// =============================================================================

func TestExecutionStatus_Constants(t *testing.T) {
	t.Run("all status values are defined", func(t *testing.T) {
		assert.Equal(t, ExecutionStatus("pending"), StatusPending)
		assert.Equal(t, ExecutionStatus("running"), StatusRunning)
		assert.Equal(t, ExecutionStatus("completed"), StatusCompleted)
		assert.Equal(t, ExecutionStatus("failed"), StatusFailed)
		assert.Equal(t, ExecutionStatus("cancelled"), StatusCancelled)
		assert.Equal(t, ExecutionStatus("timeout"), StatusTimeout)
	})

	t.Run("status values are distinct", func(t *testing.T) {
		statuses := []ExecutionStatus{
			StatusPending, StatusRunning, StatusCompleted,
			StatusFailed, StatusCancelled, StatusTimeout,
		}

		seen := make(map[ExecutionStatus]bool)
		for _, status := range statuses {
			assert.False(t, seen[status], "Duplicate status: %s", status)
			seen[status] = true
		}
	})

	t.Run("can use status as string", func(t *testing.T) {
		assert.Equal(t, "completed", string(StatusCompleted))
		assert.Equal(t, "failed", string(StatusFailed))
	})
}

// =============================================================================
// Procedure Tests
// =============================================================================

func TestProcedure_Struct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		now := time.Now()
		schedule := "0 * * * *"
		createdBy := "user123"

		proc := Procedure{
			ID:                      "proc-1",
			Name:                    "get_users",
			Namespace:               "public",
			Description:             "Fetches all users",
			SQLQuery:                "SELECT * FROM users",
			OriginalCode:            "-- original\nSELECT * FROM users",
			InputSchema:             json.RawMessage(`{"id": "uuid"}`),
			OutputSchema:            json.RawMessage(`{"name": "string"}`),
			AllowedTables:           []string{"users", "profiles"},
			AllowedSchemas:          []string{"public", "auth"},
			MaxExecutionTimeSeconds: 30,
			RequireRoles:            []string{"admin"},
			IsPublic:                true,
			DisableExecutionLogs:    false,
			Schedule:                &schedule,
			Enabled:                 true,
			Version:                 2,
			Source:                  "filesystem",
			CreatedBy:               &createdBy,
			CreatedAt:               now,
			UpdatedAt:               now,
		}

		assert.Equal(t, "proc-1", proc.ID)
		assert.Equal(t, "get_users", proc.Name)
		assert.Equal(t, "public", proc.Namespace)
		assert.Equal(t, "Fetches all users", proc.Description)
		assert.Equal(t, "SELECT * FROM users", proc.SQLQuery)
		assert.Equal(t, []string{"users", "profiles"}, proc.AllowedTables)
		assert.True(t, proc.IsPublic)
		assert.Equal(t, "0 * * * *", *proc.Schedule)
	})

	t.Run("zero value procedure", func(t *testing.T) {
		var proc Procedure

		assert.Empty(t, proc.ID)
		assert.Empty(t, proc.Name)
		assert.False(t, proc.IsPublic)
		assert.False(t, proc.Enabled)
		assert.Equal(t, 0, proc.Version)
		assert.Nil(t, proc.Schedule)
	})
}

func TestProcedure_ToSummary(t *testing.T) {
	t.Run("converts all fields correctly", func(t *testing.T) {
		now := time.Now()
		schedule := "*/5 * * * *"

		proc := Procedure{
			ID:                      "proc-123",
			Name:                    "test_procedure",
			Namespace:               "api",
			Description:             "Test procedure for testing",
			SQLQuery:                "SELECT 1",
			OriginalCode:            "-- test\nSELECT 1",
			InputSchema:             json.RawMessage(`{"a": "b"}`),
			OutputSchema:            json.RawMessage(`{"c": "d"}`),
			AllowedTables:           []string{"table1", "table2"},
			AllowedSchemas:          []string{"public"},
			MaxExecutionTimeSeconds: 60,
			RequireRoles:            []string{"admin", "editor"},
			IsPublic:                true,
			DisableExecutionLogs:    true,
			Schedule:                &schedule,
			Enabled:                 true,
			Version:                 5,
			Source:                  "api",
			CreatedAt:               now,
			UpdatedAt:               now.Add(time.Hour),
		}

		summary := proc.ToSummary()

		// Verify all fields are copied
		assert.Equal(t, proc.ID, summary.ID)
		assert.Equal(t, proc.Name, summary.Name)
		assert.Equal(t, proc.Namespace, summary.Namespace)
		assert.Equal(t, proc.Description, summary.Description)
		assert.Equal(t, proc.AllowedTables, summary.AllowedTables)
		assert.Equal(t, proc.AllowedSchemas, summary.AllowedSchemas)
		assert.Equal(t, proc.MaxExecutionTimeSeconds, summary.MaxExecutionTimeSeconds)
		assert.Equal(t, proc.RequireRoles, summary.RequireRoles)
		assert.Equal(t, proc.IsPublic, summary.IsPublic)
		assert.Equal(t, proc.DisableExecutionLogs, summary.DisableExecutionLogs)
		assert.Equal(t, proc.Schedule, summary.Schedule)
		assert.Equal(t, proc.Enabled, summary.Enabled)
		assert.Equal(t, proc.Version, summary.Version)
		assert.Equal(t, proc.Source, summary.Source)
		assert.Equal(t, proc.CreatedAt, summary.CreatedAt)
		assert.Equal(t, proc.UpdatedAt, summary.UpdatedAt)
	})

	t.Run("summary does not include SQLQuery", func(t *testing.T) {
		proc := Procedure{
			Name:     "test",
			SQLQuery: "SELECT * FROM sensitive_data",
		}

		summary := proc.ToSummary()

		// Summary should not expose SQL
		assert.Equal(t, "test", summary.Name)
		// ProcedureSummary doesn't have SQLQuery field
	})

	t.Run("handles nil schedule", func(t *testing.T) {
		proc := Procedure{
			Name:     "test",
			Schedule: nil,
		}

		summary := proc.ToSummary()
		assert.Nil(t, summary.Schedule)
	})

	t.Run("handles empty slices", func(t *testing.T) {
		proc := Procedure{
			Name:           "test",
			AllowedTables:  []string{},
			AllowedSchemas: []string{},
			RequireRoles:   []string{},
		}

		summary := proc.ToSummary()

		assert.Empty(t, summary.AllowedTables)
		assert.Empty(t, summary.AllowedSchemas)
		assert.Empty(t, summary.RequireRoles)
	})

	t.Run("handles nil slices", func(t *testing.T) {
		proc := Procedure{
			Name:           "test",
			AllowedTables:  nil,
			AllowedSchemas: nil,
			RequireRoles:   nil,
		}

		summary := proc.ToSummary()

		assert.Nil(t, summary.AllowedTables)
		assert.Nil(t, summary.AllowedSchemas)
		assert.Nil(t, summary.RequireRoles)
	})
}

// =============================================================================
// Execution Tests
// =============================================================================

func TestExecution_Struct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		now := time.Now()
		procID := "proc-1"
		userID := "user-1"
		userRole := "admin"
		userEmail := "admin@example.com"
		errorMsg := "connection failed"
		rows := 10
		duration := 150

		exec := Execution{
			ID:            "exec-1",
			ProcedureID:   &procID,
			ProcedureName: "get_users",
			Namespace:     "public",
			Status:        StatusCompleted,
			InputParams:   json.RawMessage(`{"id": 1}`),
			Result:        json.RawMessage(`[{"name": "John"}]`),
			ErrorMessage:  &errorMsg,
			RowsReturned:  &rows,
			DurationMs:    &duration,
			UserID:        &userID,
			UserRole:      &userRole,
			UserEmail:     &userEmail,
			IsAsync:       false,
			CreatedAt:     now,
			StartedAt:     &now,
			CompletedAt:   &now,
		}

		assert.Equal(t, "exec-1", exec.ID)
		assert.Equal(t, "get_users", exec.ProcedureName)
		assert.Equal(t, StatusCompleted, exec.Status)
		assert.Equal(t, 10, *exec.RowsReturned)
		assert.Equal(t, 150, *exec.DurationMs)
	})

	t.Run("zero value execution", func(t *testing.T) {
		var exec Execution

		assert.Empty(t, exec.ID)
		assert.Nil(t, exec.ProcedureID)
		assert.Empty(t, exec.Status)
		assert.False(t, exec.IsAsync)
	})
}

// =============================================================================
// CallerContext Tests
// =============================================================================

func TestCallerContext_Struct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		ctx := CallerContext{
			UserID: "user-123",
			Role:   "authenticated",
			Email:  "user@example.com",
			Metadata: map[string]interface{}{
				"ip":         "192.168.1.1",
				"user_agent": "Chrome/120",
			},
		}

		assert.Equal(t, "user-123", ctx.UserID)
		assert.Equal(t, "authenticated", ctx.Role)
		assert.Equal(t, "user@example.com", ctx.Email)
		assert.Equal(t, "192.168.1.1", ctx.Metadata["ip"])
	})

	t.Run("zero value context", func(t *testing.T) {
		var ctx CallerContext

		assert.Empty(t, ctx.UserID)
		assert.Empty(t, ctx.Role)
		assert.Nil(t, ctx.Metadata)
	})
}

// =============================================================================
// InvokeRequest Tests
// =============================================================================

func TestInvokeRequest_Struct(t *testing.T) {
	t.Run("with params and async", func(t *testing.T) {
		req := InvokeRequest{
			Params: map[string]interface{}{
				"user_id": "123",
				"limit":   10,
			},
			Async: true,
		}

		assert.Equal(t, "123", req.Params["user_id"])
		assert.Equal(t, 10, req.Params["limit"])
		assert.True(t, req.Async)
	})

	t.Run("zero value request", func(t *testing.T) {
		var req InvokeRequest

		assert.Nil(t, req.Params)
		assert.False(t, req.Async)
	})
}

// =============================================================================
// InvokeResponse Tests
// =============================================================================

func TestInvokeResponse_Struct(t *testing.T) {
	t.Run("successful response", func(t *testing.T) {
		rows := 5
		duration := 25

		resp := InvokeResponse{
			ExecutionID:  "exec-123",
			Status:       StatusCompleted,
			Result:       json.RawMessage(`[{"id": 1}]`),
			RowsReturned: &rows,
			DurationMs:   &duration,
			Error:        nil,
		}

		assert.Equal(t, "exec-123", resp.ExecutionID)
		assert.Equal(t, StatusCompleted, resp.Status)
		assert.Equal(t, 5, *resp.RowsReturned)
		assert.Nil(t, resp.Error)
	})

	t.Run("error response", func(t *testing.T) {
		errMsg := "permission denied"

		resp := InvokeResponse{
			ExecutionID: "exec-456",
			Status:      StatusFailed,
			Error:       &errMsg,
		}

		assert.Equal(t, StatusFailed, resp.Status)
		assert.Equal(t, "permission denied", *resp.Error)
		assert.Nil(t, resp.Result)
	})
}

// =============================================================================
// Annotations Tests
// =============================================================================

func TestAnnotations_Struct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		schedule := "0 0 * * *"

		ann := Annotations{
			Name:                 "test_procedure",
			Description:          "A test procedure",
			InputSchema:          map[string]string{"id": "uuid"},
			OutputSchema:         map[string]string{"name": "string"},
			AllowedTables:        []string{"users"},
			AllowedSchemas:       []string{"public", "auth"},
			MaxExecutionTime:     60 * time.Second,
			RequireRoles:         []string{"admin"},
			IsPublic:             true,
			DisableExecutionLogs: true,
			Version:              3,
			Schedule:             &schedule,
		}

		assert.Equal(t, "test_procedure", ann.Name)
		assert.Equal(t, "uuid", ann.InputSchema["id"])
		assert.Equal(t, 60*time.Second, ann.MaxExecutionTime)
		assert.True(t, ann.IsPublic)
		assert.Equal(t, 3, ann.Version)
	})
}

// =============================================================================
// Sync Types Tests
// =============================================================================

func TestProcedureSpec_Struct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		spec := ProcedureSpec{
			Name:        "my_procedure",
			Code:        "SELECT * FROM users",
			Description: "Gets users",
			Enabled:     true,
		}

		assert.Equal(t, "my_procedure", spec.Name)
		assert.Equal(t, "SELECT * FROM users", spec.Code)
		assert.True(t, spec.Enabled)
	})
}

func TestSyncRequest_Struct(t *testing.T) {
	t.Run("with procedures", func(t *testing.T) {
		req := SyncRequest{
			Namespace: "api",
			Procedures: []ProcedureSpec{
				{Name: "proc1", Code: "SELECT 1"},
				{Name: "proc2", Code: "SELECT 2"},
			},
			Options: SyncOptions{
				DeleteMissing: true,
				DryRun:        false,
			},
		}

		assert.Equal(t, "api", req.Namespace)
		assert.Len(t, req.Procedures, 2)
		assert.True(t, req.Options.DeleteMissing)
	})
}

func TestSyncResult_Struct(t *testing.T) {
	t.Run("successful sync", func(t *testing.T) {
		result := SyncResult{
			Message:   "Sync completed successfully",
			Namespace: "api",
			Summary: SyncSummary{
				Created:   2,
				Updated:   1,
				Deleted:   0,
				Unchanged: 5,
				Errors:    0,
			},
			Details: SyncDetails{
				Created:   []string{"new_proc"},
				Updated:   []string{"existing_proc"},
				Unchanged: []string{"stable1", "stable2"},
			},
			DryRun: false,
		}

		assert.Equal(t, 2, result.Summary.Created)
		assert.Len(t, result.Details.Created, 1)
		assert.False(t, result.DryRun)
	})

	t.Run("sync with errors", func(t *testing.T) {
		result := SyncResult{
			Message:   "Sync completed with errors",
			Namespace: "api",
			Summary: SyncSummary{
				Created: 0,
				Errors:  2,
			},
			Errors: []SyncError{
				{Procedure: "bad_proc1", Error: "syntax error"},
				{Procedure: "bad_proc2", Error: "invalid schema"},
			},
		}

		assert.Len(t, result.Errors, 2)
		assert.Equal(t, "bad_proc1", result.Errors[0].Procedure)
	})
}

// =============================================================================
// ListExecutionsOptions Tests
// =============================================================================

func TestListExecutionsOptions_Struct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		opts := ListExecutionsOptions{
			Namespace:     "api",
			ProcedureName: "get_users",
			Status:        StatusCompleted,
			UserID:        "user-1",
			Limit:         50,
			Offset:        100,
		}

		assert.Equal(t, "api", opts.Namespace)
		assert.Equal(t, "get_users", opts.ProcedureName)
		assert.Equal(t, StatusCompleted, opts.Status)
		assert.Equal(t, 50, opts.Limit)
	})

	t.Run("zero value options", func(t *testing.T) {
		var opts ListExecutionsOptions

		assert.Empty(t, opts.Namespace)
		assert.Empty(t, opts.Status)
		assert.Equal(t, 0, opts.Limit)
	})
}

// =============================================================================
// JSON Serialization Tests
// =============================================================================

func TestProcedure_JSON(t *testing.T) {
	t.Run("serializes to JSON", func(t *testing.T) {
		proc := Procedure{
			ID:       "test-id",
			Name:     "test_proc",
			IsPublic: true,
		}

		data, err := json.Marshal(proc)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"id":"test-id"`)
		assert.Contains(t, string(data), `"name":"test_proc"`)
		assert.Contains(t, string(data), `"is_public":true`)
	})

	t.Run("deserializes from JSON", func(t *testing.T) {
		jsonData := `{"id":"proc-1","name":"get_users","is_public":true,"version":2}`

		var proc Procedure
		err := json.Unmarshal([]byte(jsonData), &proc)
		require.NoError(t, err)

		assert.Equal(t, "proc-1", proc.ID)
		assert.Equal(t, "get_users", proc.Name)
		assert.True(t, proc.IsPublic)
		assert.Equal(t, 2, proc.Version)
	})

	t.Run("omits empty optional fields", func(t *testing.T) {
		proc := Procedure{
			ID:   "test-id",
			Name: "test",
		}

		data, err := json.Marshal(proc)
		require.NoError(t, err)

		// Description has omitempty
		assert.NotContains(t, string(data), "description")
	})
}

func TestInvokeResponse_JSON(t *testing.T) {
	t.Run("serializes successful response", func(t *testing.T) {
		rows := 5
		resp := InvokeResponse{
			ExecutionID:  "exec-1",
			Status:       StatusCompleted,
			Result:       json.RawMessage(`[{"name":"John"}]`),
			RowsReturned: &rows,
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"execution_id":"exec-1"`)
		assert.Contains(t, string(data), `"status":"completed"`)
		assert.Contains(t, string(data), `"rows_returned":5`)
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkProcedure_ToSummary(b *testing.B) {
	schedule := "0 * * * *"
	proc := Procedure{
		ID:                      "proc-1",
		Name:                    "benchmark_proc",
		Namespace:               "api",
		Description:             "A procedure for benchmarking",
		AllowedTables:           []string{"users", "orders", "products"},
		AllowedSchemas:          []string{"public"},
		MaxExecutionTimeSeconds: 30,
		RequireRoles:            []string{"admin", "editor"},
		IsPublic:                true,
		Version:                 5,
		Schedule:                &schedule,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = proc.ToSummary()
	}
}

func BenchmarkProcedure_Marshal(b *testing.B) {
	proc := Procedure{
		ID:          "proc-1",
		Name:        "benchmark_proc",
		Namespace:   "api",
		SQLQuery:    "SELECT * FROM users WHERE id = $1",
		IsPublic:    true,
		Version:     1,
		InputSchema: json.RawMessage(`{"id": "uuid"}`),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(proc)
	}
}

func BenchmarkProcedure_Unmarshal(b *testing.B) {
	data := []byte(`{"id":"proc-1","name":"test","namespace":"api","sql_query":"SELECT 1","is_public":true,"version":2}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var proc Procedure
		_ = json.Unmarshal(data, &proc)
	}
}

func BenchmarkInvokeResponse_Marshal(b *testing.B) {
	rows := 100
	duration := 50
	resp := InvokeResponse{
		ExecutionID:  "exec-1",
		Status:       StatusCompleted,
		Result:       json.RawMessage(`[{"id":1,"name":"test"}]`),
		RowsReturned: &rows,
		DurationMs:   &duration,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(resp)
	}
}
