package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
)

// executeToolCall executes a tool call and returns:
// - the result as a string for the AI
// - the QueryResult for persistence (nil if query failed or not a SQL query)
func (h *ChatHandler) executeToolCall(ctx context.Context, chatCtx *ChatContext, conversationID string, chatbot *Chatbot, toolCall *ToolCall, userID string, userMessage string) (string, *QueryResult) {
	toolName := toolCall.Function.Name

	// Check if this is an MCP tool
	if chatbot.HasMCPTools() && h.mcpExecutor != nil && IsToolAllowed(toolName, chatbot.MCPTools) {
		return h.executeMCPTool(ctx, chatCtx, conversationID, chatbot, toolCall)
	}

	// Dispatch based on tool name for legacy tools
	switch toolName {
	case "execute_sql":
		return h.executeSQLTool(ctx, chatCtx, conversationID, chatbot, toolCall, userID, userMessage)
	default:
		return fmt.Sprintf("Error: Unknown tool '%s'", toolName), nil
	}
}

// executeSQLTool handles the execute_sql tool call
func (h *ChatHandler) executeSQLTool(ctx context.Context, chatCtx *ChatContext, conversationID string, chatbot *Chatbot, toolCall *ToolCall, userID string, userMessage string) (string, *QueryResult) {
	// Parse arguments
	var args struct {
		SQL         string `json:"sql"`
		Description string `json:"description"`
	}

	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		log.Error().Err(err).Str("args", toolCall.Function.Arguments).Msg("Failed to parse SQL tool call arguments")
		return fmt.Sprintf("Error: Failed to parse tool arguments: %v", err), nil
	}

	// Intent validation (before execution)
	if len(chatbot.IntentRules) > 0 || len(chatbot.RequiredColumns) > 0 {
		intentValidator := NewIntentValidator(
			chatbot.IntentRules,
			chatbot.RequiredColumns,
			chatbot.DefaultTable,
		)

		// Pre-validate SQL to get tables accessed
		preValidator := NewSQLValidator(chatbot.AllowedSchemas, chatbot.AllowedTables, chatbot.AllowedOperations)
		preResult := preValidator.Validate(args.SQL)

		// Validate intent matches query
		intentResult := intentValidator.ValidateIntent(userMessage, args.SQL, preResult.TablesAccessed)
		if !intentResult.Valid {
			errMsg := fmt.Sprintf("Intent validation failed: %s", strings.Join(intentResult.Errors, "; "))
			if len(intentResult.Suggestions) > 0 {
				errMsg += fmt.Sprintf(" Suggestions: %s", strings.Join(intentResult.Suggestions, "; "))
			}
			log.Warn().
				Str("user_message", userMessage).
				Str("sql", args.SQL).
				Strs("errors", intentResult.Errors).
				Msg("Intent validation failed")
			return errMsg, nil
		}

		// Validate required columns
		colResult := intentValidator.ValidateRequiredColumns(args.SQL, preResult.TablesAccessed)
		if !colResult.Valid {
			errMsg := fmt.Sprintf("Required columns missing: %s", strings.Join(colResult.Errors, "; "))
			if len(colResult.Suggestions) > 0 {
				errMsg += fmt.Sprintf(" Suggestions: %s", strings.Join(colResult.Suggestions, "; "))
			}
			log.Warn().
				Str("sql", args.SQL).
				Strs("errors", colResult.Errors).
				Msg("Required columns validation failed")
			return errMsg, nil
		}
	}

	// Send progress
	h.sendProgress(chatCtx, conversationID, "querying", fmt.Sprintf("Executing: %s", args.Description))

	// Execute SQL
	execReq := &ExecuteRequest{
		ChatbotName:       chatbot.Name,
		ChatbotID:         chatbot.ID,
		ConversationID:    conversationID,
		UserID:            userID,
		Role:              chatCtx.Role,
		Claims:            chatCtx.Claims,
		SQL:               args.SQL,
		Description:       args.Description,
		AllowedSchemas:    chatbot.AllowedSchemas,
		AllowedTables:     chatbot.AllowedTables,
		AllowedOperations: chatbot.AllowedOperations,
	}

	result, err := h.executor.Execute(ctx, execReq)
	if err != nil {
		log.Error().Err(err).Msg("SQL execution error")
		return FormatErrorForLLM(err, args.SQL, "execute_sql"), nil
	}

	// Log to audit (unless execution logs are disabled)
	if !chatbot.DisableExecutionLogs {
		_ = h.auditLogger.LogFromExecuteResult(
			ctx,
			chatbot.ID, conversationID, "", userID,
			args.SQL, result,
			chatCtx.Role, chatCtx.IPAddress, chatCtx.UserAgent,
		)

		// Log to central logging service
		if h.loggingService != nil {
			h.loggingService.LogAI(ctx, map[string]any{
				"tool":            "execute_sql",
				"chatbot_id":      chatbot.ID,
				"conversation_id": conversationID,
				"success":         result.Success,
				"rows_returned":   result.RowCount,
				"tables":          result.TablesAccessed,
				"duration_ms":     result.DurationMs,
			}, "", userID)
		}
	}

	// Send result to client for display
	h.send(chatCtx, ServerMessage{
		Type:           "query_result",
		ConversationID: conversationID,
		Query:          args.SQL,
		Summary:        result.Summary,
		RowCount:       result.RowCount,
		Data:           result.Rows,
	})

	// Return result as string for AI to interpret
	if !result.Success {
		return FormatErrorForLLM(fmt.Errorf("%s", result.Error), args.SQL, "execute_sql"), nil
	}

	// Build QueryResult for persistence
	queryResult := &QueryResult{
		Query:    args.SQL,
		Summary:  result.Summary,
		RowCount: result.RowCount,
		Data:     result.Rows,
	}

	// Format result for AI - include summary and sample data
	resultStr := fmt.Sprintf("Query executed successfully. %s\n", result.Summary)
	if len(result.Rows) > 0 {
		// Include first few rows as JSON for context
		maxRows := 5
		if len(result.Rows) < maxRows {
			maxRows = len(result.Rows)
		}
		sampleData, _ := json.Marshal(result.Rows[:maxRows])
		resultStr += fmt.Sprintf("Sample data (first %d rows): %s", maxRows, string(sampleData))
	}

	return resultStr, queryResult
}

// executeMCPTool handles MCP tool execution for chatbots with MCP tools configured
func (h *ChatHandler) executeMCPTool(ctx context.Context, chatCtx *ChatContext, conversationID string, chatbot *Chatbot, toolCall *ToolCall) (string, *QueryResult) {
	toolName := toolCall.Function.Name

	// Parse tool arguments
	var args map[string]any
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		log.Error().Err(err).Str("tool", toolName).Str("args", toolCall.Function.Arguments).Msg("Failed to parse MCP tool arguments")
		return fmt.Sprintf("Error: Failed to parse tool arguments: %v", err), nil
	}

	// Send progress to client
	progressMsg := fmt.Sprintf("Executing %s...", toolName)
	if tableName, ok := args["table"].(string); ok && tableName != "" {
		progressMsg = fmt.Sprintf("Executing %s on %s...", toolName, tableName)
	}
	h.sendProgress(chatCtx, conversationID, "executing", progressMsg)

	// Execute the MCP tool
	result, err := h.mcpExecutor.ExecuteTool(ctx, toolName, args, chatCtx, chatbot)
	if err != nil {
		log.Error().Err(err).Str("tool", toolName).Msg("MCP tool execution error")
		return fmt.Sprintf("Error executing %s: %v", toolName, err), nil
	}

	if result.IsError {
		log.Warn().Str("tool", toolName).Str("error", result.Content).Msg("MCP tool returned error")
		// Surface the error to the client so they don't see silent execution.
		// Without this, the only feedback was the LLM's interpretation of the
		// error string returned below.
		h.send(chatCtx, ServerMessage{
			Type:           "tool_result",
			ConversationID: conversationID,
			Message:        toolName,
			Error:          result.Content,
		})
		return fmt.Sprintf("Error: %s", result.Content), nil
	}

	// Log successful execution
	log.Debug().
		Str("chatbot", chatbot.Name).
		Str("tool", toolName).
		Int("result_length", len(result.Content)).
		Msg("MCP tool executed successfully")

	// Parse query results for data-returning tools
	var queryResult *QueryResult
	switch toolName {
	case "query_table":
		queryResult = h.parseMCPQueryResult(toolName, args, result.Content)
	case "execute_sql":
		queryResult = h.parseMCPExecuteSQLResult(args, result.Content)
	}

	// Build server message. When the tool returned structured query data, emit
	// the canonical `query_result` event type so clients (including old SDKs
	// that only handled `query_result`) receive it uniformly across the legacy
	// and MCP execution paths. Otherwise emit `tool_result`.
	serverMsg := ServerMessage{
		ConversationID: conversationID,
		Message:        toolName,
	}

	// Add structured fields whenever we parsed a QueryResult (execute_sql and
	// query_table). The previous gate restricted this to execute_sql only,
	// which left query_table results parsed-for-persistence but unstreamed.
	if queryResult != nil {
		serverMsg.Type = "query_result"
		serverMsg.Query = queryResult.Query
		serverMsg.Summary = queryResult.Summary
		serverMsg.RowCount = queryResult.RowCount
		serverMsg.Data = queryResult.Data
	} else {
		serverMsg.Type = "tool_result"
		serverMsg.Data = []map[string]any{{"tool": toolName, "result": result.Content}}
	}

	// Suppress think/reasoning tool results from client when ShowReasoning is false
	if toolName == "think" && !chatbot.ShowReasoning {
		return result.Content, queryResult
	}

	h.send(chatCtx, serverMsg)

	return result.Content, queryResult
}

// parseMCPQueryResult attempts to parse MCP query results for persistence
func (h *ChatHandler) parseMCPQueryResult(toolName string, args map[string]any, resultContent string) *QueryResult {
	// Try to parse the result as JSON array
	var rows []map[string]any
	if err := json.Unmarshal([]byte(resultContent), &rows); err != nil {
		// Not valid JSON array, skip persistence
		return nil
	}

	// Build a description for the query
	tableName := ""
	if t, ok := args["table"].(string); ok {
		tableName = t
	}

	return &QueryResult{
		Query:    fmt.Sprintf("MCP %s on %s", toolName, tableName),
		Summary:  fmt.Sprintf("Query returned %d row(s)", len(rows)),
		RowCount: len(rows),
		Data:     rows,
	}
}

// parseMCPExecuteSQLResult parses execute_sql MCP tool results for persistence
func (h *ChatHandler) parseMCPExecuteSQLResult(args map[string]any, resultContent string) *QueryResult {
	// Parse the result JSON from the MCP tool
	var execResult struct {
		Success    bool             `json:"success"`
		RowCount   int              `json:"row_count"`
		Columns    []string         `json:"columns"`
		Rows       []map[string]any `json:"rows"`
		Summary    string           `json:"summary"`
		DurationMs int64            `json:"duration_ms"`
		Tables     []string         `json:"tables"`
	}

	if err := json.Unmarshal([]byte(resultContent), &execResult); err != nil {
		return nil
	}

	if !execResult.Success {
		return nil
	}

	// Extract SQL query from tool arguments
	sqlQuery := ""
	if sql, ok := args["sql"].(string); ok {
		sqlQuery = sql
	}

	return &QueryResult{
		Query:    sqlQuery,
		Summary:  execResult.Summary,
		RowCount: execResult.RowCount,
		Data:     execResult.Rows,
	}
}
