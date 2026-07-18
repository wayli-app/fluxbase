package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
)

// SQLAgent is the specialist that investigates factual data questions by
// running SQL queries. It runs an internal tool-calling loop bounded by
// the chatbot's MaxToolIterations.
//
// The agent reuses the existing Executor for actual SQL execution (same
// RLS context, same validation, same row caps) — no SQL execution logic is
// duplicated. The only new code is the per-agent prompt and the loop glue.
type SQLAgent struct {
	deps *AgentDeps
}

// NewSQLAgent constructs a SQLAgent bound to the given deps.
func NewSQLAgent(deps *AgentDeps) *SQLAgent {
	return &SQLAgent{deps: deps}
}

// Name implements Node.
func (a *SQLAgent) Name() string { return "sql" }

// Run executes the SQL agent's investigation loop.
func (a *SQLAgent) Run(ctx context.Context, state *State) error {
	chatbot := a.deps.Chatbot
	provider := a.deps.Provider
	if provider == nil {
		return fmt.Errorf("sql agent: no provider")
	}
	if a.deps.SQLExecutor == nil {
		return fmt.Errorf("sql agent: no executor configured")
	}

	// Emit transition event so the client can show "Investigating with SQL..."
	if a.deps.Sender != nil {
		a.deps.Sender.SendAgentTransition(ctx, a.deps.ConversationID, AgentTransition{To: "sql"})
	}

	// Build per-turn messages. Schema description goes in the dynamic system
	// message so the static prompt stays byte-stable for caching.
	systemPrompt := BuildSQLAgentPrompt(chatbot)
	schemaDesc, err := a.buildSchemaDescription(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("sql agent: failed to build schema description")
		schemaDesc = ""
	}
	dynamicContext := BuildDynamicContextForAgent(chatbot, a.deps.UserID, "sql", a.deps.PageProfile)
	if schemaDesc != "" {
		dynamicContext += "\n## Schema\n\n" + schemaDesc + "\n"
	}
	// Page-profile table whitelist overrides chatbot's global allowed_tables
	// for this turn.
	allowedTables := a.deps.PageProfile.ResolvedTables(chatbot.AllowedTables)
	if len(allowedTables) > 0 {
		dynamicContext += fmt.Sprintf("\nTables in scope for this turn: %s\n", strings.Join(allowedTables, ", "))
	}

	// Initial message list — supervisor may have provided a sub-question
	// nudge via state; we use the user's original message by default.
	userMsg := state.UserMessage()
	messages := []Message{
		{Role: RoleSystem, Content: systemPrompt},
		{Role: RoleSystem, Content: dynamicContext},
		// Seed with recent conversation history so the agent has context
		// (last 4 messages keeps tokens bounded).
	}
	messages = append(messages, tailMessages(state.ConversationHistory(), 4)...)
	messages = append(messages, Message{Role: RoleUser, Content: userMsg})

	// Internal tool loop. Bounded by MaxToolIterations (default 5).
	maxIters := chatbot.MaxToolIterations
	if maxIters <= 0 {
		maxIters = 5
	}

	model := chatbot.Model
	if override, ok := chatbot.SupervisorAgentModels["sql"]; ok && override != "" {
		model = override
	}

	tools := []Tool{ExecuteSQLTool}

	// Get the supervisor plan's MinToolCalls (when investigative) so we can
	// prevent premature final answers.
	minToolCalls := 0
	if plan, _ := state.Get(SupervisorPlanKey); plan != nil {
		if p, ok := plan.(*SupervisorPlan); ok && p.IsInvestigative {
			minToolCalls = p.MinToolCalls
		}
	}

	toolCallCount := 0
	var finalContent string

	for iter := 0; iter < maxIters; iter++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		req := &ChatRequest{
			Messages:    messages,
			Model:       model,
			MaxTokens:   chatbot.MaxTokens,
			Temperature: chatbot.Temperature,
			Tools:       tools,
			Stream:      false,
		}

		resp, err := provider.Chat(ctx, req)
		if err != nil {
			return fmt.Errorf("sql agent: provider call failed: %w", err)
		}
		if resp == nil || len(resp.Choices) == 0 {
			return fmt.Errorf("sql agent: empty provider response")
		}
		if resp.Usage != nil {
			state.AddUsage(*resp.Usage)
		}

		msg := resp.Choices[0].Message

		// Emit pre-tool reasoning as a thought-process chunk. The model's
		// Content here is its explanation of what it's about to do ("Let
		// me check the orders table first because..."). Suppressed on the
		// wire when show_reasoning=false via chatHandlerSender.
		if msg.Content != "" && a.deps.Sender != nil {
			a.deps.Sender.SendAgentThought(ctx, a.deps.ConversationID, AgentThought{
				Agent: "sql",
				Kind:  "reasoning",
				Delta: msg.Content,
			})
		}

		if len(msg.ToolCalls) == 0 {
			// Budget gate: if the supervisor said this turn is investigative
			// with a min-tool-call floor and we haven't hit it yet, push
			// back once and let the LLM try again. Prevents premature
			// "I don't know" or hallucinated answers.
			if toolCallCount < minToolCalls && iter < maxIters-1 {
				messages = append(messages, Message{Role: RoleAssistant, Content: msg.Content})
				messages = append(messages, Message{
					Role:    RoleUser,
					Content: "You haven't finished investigating yet. Run at least one execute_sql query to verify before answering.",
				})
				continue
			}
			finalContent = msg.Content
			break
		}

		// Append assistant tool-call message and execute each tool
		messages = append(messages, msg)
		for _, tc := range msg.ToolCalls {
			// Emit a tool_call thought so clients can render "calling
			// execute_sql with args..." before the result arrives.
			if a.deps.Sender != nil {
				a.deps.Sender.SendAgentThought(ctx, a.deps.ConversationID, AgentThought{
					Agent:    "sql",
					Kind:     "tool_call",
					ToolName: tc.Function.Name,
					ToolArgs: json.RawMessage(tc.Function.Arguments),
				})
			}
			resultStr, queryResult := a.executeSQL(ctx, tc, chatbot, allowedTables)
			toolCallCount++
			if queryResult != nil {
				state.AppendToolResult(*queryResult)
				// Short tool_result summary in the thought stream. Full
				// structured data still flows via the query_result event
				// emitted by executeSQL.
				if a.deps.Sender != nil {
					a.deps.Sender.SendAgentThought(ctx, a.deps.ConversationID, AgentThought{
						Agent: "sql",
						Kind:  "tool_result",
						Delta: queryResult.Summary,
					})
				}
			}
			messages = append(messages, Message{
				Role:       RoleTool,
				Content:    resultStr,
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
			})
		}
	}

	if finalContent == "" {
		finalContent = "I wasn't able to complete my investigation. Please try rephrasing your question."
	}

	state.AppendAgentOutput("sql", finalContent)
	state.SetFinalResponse(finalContent)
	return nil
}

// buildSchemaDescription returns the schema description for the chatbot's
// allowed tables, falling back to an empty string on error.
func (a *SQLAgent) buildSchemaDescription(ctx context.Context) (string, error) {
	if a.deps.SchemaBuilder == nil {
		return "", nil
	}
	return a.deps.SchemaBuilder.BuildSchemaDescription(ctx, a.deps.Chatbot.AllowedSchemas, a.deps.Chatbot.AllowedTables)
}

// executeSQL runs one SQL query via the existing Executor and returns the
// formatted result string (for the LLM) and the QueryResult (for state).
func (a *SQLAgent) executeSQL(ctx context.Context, tc ToolCall, chatbot *Chatbot, allowedTables []string) (string, *QueryResult) {
	var args struct {
		SQL         string `json:"sql"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		return fmt.Sprintf("Error: failed to parse tool arguments: %v", err), nil
	}

	if a.deps.Sender != nil {
		a.deps.Sender.SendProgress(ctx, a.deps.ConversationID, "querying", fmt.Sprintf("Executing: %s", args.Description))
	}

	execReq := &ExecuteRequest{
		ChatbotName:       chatbot.Name,
		ChatbotID:         chatbot.ID,
		ConversationID:    a.deps.ConversationID,
		UserID:            a.deps.UserID,
		Role:              a.deps.Role,
		Claims:            a.deps.Claims,
		SQL:               args.SQL,
		Description:       args.Description,
		AllowedSchemas:    chatbot.AllowedSchemas,
		AllowedTables:     allowedTables,
		AllowedOperations: chatbot.AllowedOperations,
	}

	result, err := a.deps.SQLExecutor.Execute(ctx, execReq)
	if err != nil {
		return FormatErrorForLLM(err, args.SQL, "execute_sql"), nil
	}

	if a.deps.Sender != nil {
		a.deps.Sender.SendQueryResult(ctx, a.deps.ConversationID, QueryResult{
			Query:    args.SQL,
			Summary:  result.Summary,
			RowCount: result.RowCount,
			Data:     result.Rows,
		})
	}

	if !result.Success {
		return FormatErrorForLLM(fmt.Errorf("%s", result.Error), args.SQL, "execute_sql"), nil
	}

	queryResult := &QueryResult{
		Query:    args.SQL,
		Summary:  result.Summary,
		RowCount: result.RowCount,
		Data:     result.Rows,
	}

	// Format for LLM consumption: summary + first 5 rows
	resultStr := fmt.Sprintf("Query executed successfully. %s\n", result.Summary)
	if len(result.Rows) > 0 {
		maxRows := 5
		if len(result.Rows) < maxRows {
			maxRows = len(result.Rows)
		}
		sampleData, _ := json.Marshal(result.Rows[:maxRows])
		resultStr += fmt.Sprintf("Sample data (first %d rows): %s", maxRows, string(sampleData))
	}

	// ponytail: intentionally not running intent/required-columns validation
	// here — the SQL agent's prompt already enforces good behavior, and the
	// executor's SQLValidator runs the actual safety checks. Intent rules
	// remain enforced on the legacy react path for backwards compat.

	return resultStr, queryResult
}

// tailMessages returns the last n messages from a conversation history.
// Used to seed an agent's context window without unbounded token cost.
func tailMessages(msgs []Message, n int) []Message {
	if len(msgs) <= n {
		return msgs
	}
	return msgs[len(msgs)-n:]
}
