package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// handleMessage handles a user message
func (h *ChatHandler) handleMessage(ctx context.Context, chatCtx *ChatContext, msg *ClientMessage) {
	start := time.Now()

	state := chatCtx.Conversations[msg.ConversationID]
	if state == nil {
		h.sendError(chatCtx, msg.ConversationID, "NO_SESSION", "No active chat session")
		return
	}

	chatbot := chatCtx.ActiveChatbot
	if chatbot == nil {
		h.sendError(chatCtx, msg.ConversationID, "NO_CHATBOT", "No active chatbot")
		return
	}

	// Resolve template variables in chatbot annotation values (e.g., http-allowed-domains)
	if err := h.ResolveChatbotTemplates(ctx, chatbot, chatCtx.UserID); err != nil {
		log.Warn().Err(err).Str("chatbot", chatbot.Name).Msg("Failed to resolve chatbot templates")
		// Continue with unresolved values - don't fail the request
	}

	// Determine user identifier for rate limiting
	userIdentifier := "anonymous"
	if chatCtx.UserID != nil {
		userIdentifier = *chatCtx.UserID
	}

	// Check per-minute rate limit
	if !h.limiter.CheckRateLimit(chatbot.ID, userIdentifier, chatbot.RateLimitPerMinute) {
		h.sendError(chatCtx, msg.ConversationID, "RATE_LIMITED", "Rate limit exceeded. Please try again later.")
		return
	}

	// Check daily request limit
	if !h.limiter.CheckDailyRequestLimit(chatbot.ID, userIdentifier, chatbot.DailyRequestLimit) {
		h.sendError(chatCtx, msg.ConversationID, "DAILY_LIMIT", "Daily request limit exceeded.")
		return
	}

	// Check turn limit
	if state.TurnCount >= chatbot.MaxConversationTurns {
		h.sendError(chatCtx, msg.ConversationID, "TURN_LIMIT", "Conversation turn limit reached")
		return
	}

	// Send thinking progress
	h.sendProgress(chatCtx, msg.ConversationID, "thinking", "Thinking...")

	// Build system prompt with schema
	userID := ""
	if chatCtx.UserID != nil {
		userID = *chatCtx.UserID
	}

	systemPrompt, err := h.schemaBuilder.BuildSystemPrompt(ctx, chatbot, userID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to build system prompt")
		h.sendError(chatCtx, msg.ConversationID, "PROMPT_ERROR", "Failed to build prompt")
		return
	}

	// Build the dynamic per-turn context (user ID, current time, RAG) as a
	// separate message so the static system prompt stays byte-identical across
	// turns — enabling provider-side prompt caching (OpenAI automatic, Anthropic
	// explicit). Order matters: static system first, dynamic context second,
	// history third, user message last.
	var dynamicContext strings.Builder
	fmt.Fprintf(&dynamicContext, "Current user ID: %s\n", userID)
	fmt.Fprintf(&dynamicContext, "Current date and time: %s\n", time.Now().UTC().Format("Monday, January 2, 2006 at 3:04 PM MST"))

	// Retrieve RAG context if available (with user isolation) and append to
	// the dynamic context (NOT the static system prompt).
	if h.ragService != nil {
		ragOpts := RetrieveContextOptions{
			ChatbotID: chatbot.ID,
			Query:     msg.Content,
			UserID:    userID,
		}
		if chatbot.RAGMaxChunks > 0 {
			ragOpts.MaxChunks = chatbot.RAGMaxChunks
		}
		if chatbot.RAGSimilarityThreshold > 0 {
			ragOpts.Threshold = chatbot.RAGSimilarityThreshold
		}
		// Graph boost: chatbot annotation wins; fall back to global config.
		// Both default to 0 (off), preserving existing behavior.
		if chatbot.RAGGraphBoostWeight > 0 {
			ragOpts.GraphBoostWeight = chatbot.RAGGraphBoostWeight
		} else if h.config != nil && h.config.RAGGraphBoostWeight > 0 {
			ragOpts.GraphBoostWeight = h.config.RAGGraphBoostWeight
		}
		ragSection, err := h.ragService.RetrieveContext(ctx, ragOpts)
		if err != nil {
			log.Warn().Err(err).Str("chatbot_id", chatbot.ID).Msg("Failed to retrieve RAG context")
			// Continue without RAG - don't fail the request
		} else if ragSection != nil && ragSection.FormattedContext != "" {
			dynamicContext.WriteString("\n\n")
			dynamicContext.WriteString(ragSection.FormattedContext)
			log.Debug().
				Str("chatbot_id", chatbot.ID).
				Str("conversation_id", msg.ConversationID).
				Int("rag_section_len", len(ragSection.FormattedContext)).
				Msg("RAG context added to dynamic context message")
		}
	}

	// Build messages for LLM. The static system prompt is message[0]; the
	// dynamic context (user ID, time, RAG) is message[1] as a second system
	// message so providers can cache message[0] but not message[1].
	messages := []Message{
		{Role: RoleSystem, Content: systemPrompt},
		{Role: RoleSystem, Content: dynamicContext.String()},
	}

	// Add conversation history
	messages = append(messages, state.Messages...)

	// Add user message
	userMsg := Message{Role: RoleUser, Content: msg.Content}
	messages = append(messages, userMsg)

	// Pre-check daily token budget with a rough estimate of the upcoming request.
	// Real usage is reconciled after the request via AddTokenUsage; this guard
	// only stops obviously-over-budget requests from starting.
	if chatbot.DailyTokenBudget > 0 {
		estimatedInputChars := len(systemPrompt) + dynamicContext.Len() + len(msg.Content)
		for _, m := range state.Messages {
			estimatedInputChars += len(m.Content)
		}
		// ponytail: chars/4 estimate is a coarse lower bound for English text;
		// upgrade to a real tokenizer if budget edge cases matter.
		estimatedTokens := estimatedInputChars / 4
		if chatbot.MaxTokens > 0 {
			estimatedTokens += chatbot.MaxTokens
		}
		if !h.limiter.CheckDailyTokenBudget(chatbot.ID, userIdentifier, chatbot.DailyTokenBudget, estimatedTokens) {
			h.sendError(chatCtx, msg.ConversationID, "TOKEN_BUDGET", "Daily token budget exceeded.")
			return
		}
	}

	// Get provider
	provider, err := h.getProvider(ctx, chatbot)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get provider")
		h.sendError(chatCtx, msg.ConversationID, "PROVIDER_ERROR", "AI provider not available")
		return
	}

	// Save user message to conversation
	_ = h.conversations.AddMessage(ctx, msg.ConversationID, userMsg, 0, 0)

	// ── Supervisor mode branch ────────────────────────────────────────────
	// When the chatbot's reasoning mode is "supervisor" (the default for new
	// chatbots), the turn is handled by the multi-agent graph in
	// supervisor_graph.go. On any failure (provider down, parse error, etc.),
	// we fall back to the legacy ReAct loop below so a supervisor bug never
	// breaks the chatbot entirely.
	if chatbot.ReasoningMode == "supervisor" && h.canRunSupervisor(chatbot) {
		supervisorRan, supervisorErr := h.runSupervisorTurn(ctx, chatCtx, chatbot, msg, userID, provider)
		if supervisorRan {
			if supervisorErr != nil {
				// Supervisor completed but reported an error (e.g., a node
				// failed mid-graph). Surface as a soft warning and continue
				// to the legacy loop as a safety net.
				log.Warn().Err(supervisorErr).
					Str("chatbot", chatbot.Name).
					Str("conversation_id", msg.ConversationID).
					Msg("Supervisor turn reported an error; falling back to ReAct loop")
			} else {
				return
			}
		}
		// supervisorRan == false means we couldn't build a graph for this
		// chatbot (e.g., no provider). Fall through to legacy path.
	}

	// Tool calling loop - continue until AI generates content without tool calls
	var totalUsage UsageStats
	var accumulatedQueryResults []QueryResult // Accumulate query results for persistence
	// matchedIntentRules accumulates rules that fire across the turn (both
	// pre-tool-selection and post-tool-call). Surfaced in the done event so
	// apps can observe and tune their rules (Ask 5).
	var matchedIntentRules []MatchedIntentRule
	maxIterations := chatbot.MaxToolIterations
	if maxIterations <= 0 {
		maxIterations = 5
	}

	// Track consecutive tool validation failures to detect stubborn LLM behavior
	var lastFailedTool string
	var consecutiveFailures int
	const maxConsecutiveFailures = 2

	// Track whether think tool has been called (for enforcing ReAct pattern)
	hasUsedThink := false
	thinkRequired := chatbot.ReasoningMode == "react" || chatbot.ReasoningMode == "strict"

	for iteration := 0; iteration < maxIterations; iteration++ {
		// Determine forbidden tools based on user message and intent rules
		var forbiddenTools []string
		if len(chatbot.IntentRules) > 0 {
			intentValidator := NewIntentValidator(chatbot.IntentRules, chatbot.RequiredColumns, chatbot.DefaultTable)
			forbiddenTools = intentValidator.GetForbiddenTools(msg.Content)
			// Capture rule matches on the first iteration only (the user
			// message doesn't change between iterations of the tool loop).
			if iteration == 0 {
				matchedIntentRules = append(matchedIntentRules, intentValidator.GetMatchedRules(msg.Content)...)
			}
			if len(forbiddenTools) > 0 {
				log.Debug().
					Strs("forbidden_tools", forbiddenTools).
					Str("user_message", msg.Content).
					Msg("Filtering out forbidden tools based on intent rules")
			}
		}

		// Helper to check if a tool is forbidden
		isToolForbidden := func(toolName string) bool {
			for _, ft := range forbiddenTools {
				if ft == toolName {
					return true
				}
			}
			return false
		}

		// Helper to check if think tool is available
		hasThinkTool := func(tools []Tool) bool {
			for _, t := range tools {
				if t.Function.Name == "think" {
					return true
				}
			}
			return false
		}

		// Build tools list based on chatbot configuration
		var tools []Tool

		// Add MCP tools if configured (includes execute_sql as MCP tool now)
		if chatbot.HasMCPTools() && h.mcpExecutor != nil {
			mcpToolDefs := h.mcpExecutor.GetAvailableTools(chatbot)
			for _, def := range mcpToolDefs {
				// Skip forbidden tools
				if isToolForbidden(def.Name) {
					continue
				}
				tools = append(tools, Tool{
					Type:     "function",
					Function: ToolFunction(def),
				})
			}
		} else if !isToolForbidden("execute_sql") {
			// Fallback: add legacy execute_sql if no MCP tools configured
			tools = append(tools, ExecuteSQLTool)
		}

		// Enforce ReAct pattern: require think tool before other tools
		// If reasoning mode is react/strict and think hasn't been used yet,
		// only allow the think tool on the first iteration
		if thinkRequired && !hasUsedThink && iteration == 0 && hasThinkTool(tools) {
			// Filter to only include think tool
			var thinkOnlyTools []Tool
			for _, t := range tools {
				if t.Function.Name == "think" {
					thinkOnlyTools = append(thinkOnlyTools, t)
					break
				}
			}
			if len(thinkOnlyTools) > 0 {
				tools = thinkOnlyTools
				log.Debug().
					Str("chatbot", chatbot.Name).
					Msg("ReAct mode: restricting to think tool only on first iteration")
			}
		}

		log.Debug().
			Str("chatbot", chatbot.Name).
			Int("total_tools", len(tools)).
			Bool("think_required", thinkRequired).
			Bool("has_used_think", hasUsedThink).
			Msg("Tools available for chatbot")

		// Create chat request
		chatReq := &ChatRequest{
			Messages:    messages,
			Model:       chatbot.Model,
			MaxTokens:   chatbot.MaxTokens,
			Temperature: chatbot.Temperature,
			Tools:       tools,
			Stream:      true,
		}

		// Track response for this iteration
		var responseContent strings.Builder
		var pendingToolCalls []ToolCall

		// Stream callback
		callback := func(event StreamEvent) error {
			switch event.Type {
			case "content":
				responseContent.WriteString(event.Delta)
				h.send(chatCtx, ServerMessage{
					Type:           "content",
					ConversationID: msg.ConversationID,
					Delta:          event.Delta,
				})

			case "tool_call":
				// Collect tool calls to execute after streaming completes
				if event.ToolCall != nil {
					toolName := event.ToolCall.FunctionName
					// Accept legacy tools, MCP tools, or any tool requested by the model
					pendingToolCalls = append(pendingToolCalls, ToolCall{
						ID:   event.ToolCall.ID,
						Type: "function",
						Function: FunctionCall{
							Name:      toolName,
							Arguments: event.ToolCall.ArgumentsDelta,
						},
					})
				}

			case "done":
				if event.Usage != nil {
					totalUsage.PromptTokens += event.Usage.PromptTokens
					totalUsage.CompletionTokens += event.Usage.CompletionTokens
					totalUsage.TotalTokens += event.Usage.TotalTokens
					totalUsage.CachedTokens += event.Usage.CachedTokens
				}
			}
			return nil
		}

		// Stream the response
		h.sendProgress(chatCtx, msg.ConversationID, "generating", "Generating response...")

		providerStart := time.Now()
		providerName := provider.Name()
		if err := provider.ChatStream(ctx, chatReq, callback); err != nil {
			log.Error().Err(err).Msg("Chat stream error")
			if h.metrics != nil {
				h.metrics.RecordAIProviderRequest(providerName, "error", time.Since(providerStart))
			}
			h.sendError(chatCtx, msg.ConversationID, "STREAM_ERROR", "Error generating response")

			if h.metrics != nil {
				h.metrics.RecordAIChatRequest(chatbot.Name, "error", time.Since(start))
			}
			return
		}
		if h.metrics != nil {
			h.metrics.RecordAIProviderRequest(providerName, "success", time.Since(providerStart))
		}

		// If no tool calls, we're done
		if len(pendingToolCalls) == 0 {
			// Save assistant message with accumulated query results
			assistantMsg := Message{
				Role:         RoleAssistant,
				Content:      responseContent.String(),
				QueryResults: accumulatedQueryResults,
			}
			_ = h.conversations.AddMessage(ctx, msg.ConversationID, assistantMsg, totalUsage.PromptTokens, totalUsage.CompletionTokens)
			break
		}

		// Add assistant message with tool calls to conversation
		assistantMsg := Message{
			Role:      RoleAssistant,
			Content:   responseContent.String(),
			ToolCalls: pendingToolCalls,
		}
		messages = append(messages, assistantMsg)

		// Execute each tool call and add results
		for _, tc := range pendingToolCalls {
			toolName := tc.Function.Name

			// Track if think tool was used (for ReAct pattern)
			if toolName == "think" {
				hasUsedThink = true
			}

			// Validate tool call against intent rules (requiredTool/forbiddenTool)
			if len(chatbot.IntentRules) > 0 {
				intentValidator := NewIntentValidator(chatbot.IntentRules, chatbot.RequiredColumns, chatbot.DefaultTable)
				toolValidation := intentValidator.ValidateToolCall(msg.Content, toolName)
				// Surface tool/table rules that fire on this call. GetMatchedRules
				// already captured all matching rules once at iteration 0; here
				// we only add ones with tool/table constraints we haven't seen.
				matchedIntentRules = append(matchedIntentRules, toolValidation.MatchedRules...)

				log.Debug().
					Int("intent_rules_count", len(chatbot.IntentRules)).
					Str("tool", toolName).
					Str("user_message", msg.Content).
					Bool("valid", toolValidation.Valid).
					Strs("matched_keywords", toolValidation.MatchedKeywords).
					Msg("Tool validation check")

				if !toolValidation.Valid {
					// Track consecutive failures for the same tool
					if toolName == lastFailedTool {
						consecutiveFailures++
					} else {
						lastFailedTool = toolName
						consecutiveFailures = 1
					}

					// If the same tool fails too many times, break the loop
					if consecutiveFailures >= maxConsecutiveFailures {
						log.Warn().
							Str("tool", toolName).
							Int("failures", consecutiveFailures).
							Msg("Breaking loop due to repeated tool validation failures")

						h.send(chatCtx, ServerMessage{
							Type:  "error",
							Error: "Unable to process this request - the AI kept trying to use a tool that isn't allowed for this type of query. Please rephrase your question.",
						})
						return
					}

					// Build list of alternative tools (exclude the forbidden one)
					var alternativeTools []string
					for _, t := range chatbot.MCPTools {
						if t != toolName {
							alternativeTools = append(alternativeTools, t)
						}
					}

					errMsg := fmt.Sprintf("TOOL NOT ALLOWED: %s. %s Available tools: %s. Please use one of these tools instead.",
						strings.Join(toolValidation.Errors, "; "),
						strings.Join(toolValidation.Suggestions, " "),
						strings.Join(alternativeTools, ", "))

					log.Debug().
						Strs("errors", toolValidation.Errors).
						Strs("alternative_tools", alternativeTools).
						Str("error_message", errMsg).
						Msg("Tool validation failed, returning error to LLM")
					toolMsg := Message{
						Role:       RoleTool,
						Content:    errMsg,
						ToolCallID: tc.ID,
						Name:       toolName,
					}
					messages = append(messages, toolMsg)
					continue // Skip execution, let AI retry with correct tool
				}
			}

			toolResult, queryResult := h.executeToolCall(ctx, chatCtx, msg.ConversationID, chatbot, &tc, userID, msg.Content)

			// Accumulate successful query results for persistence
			if queryResult != nil {
				accumulatedQueryResults = append(accumulatedQueryResults, *queryResult)
			}

			// Add tool result message
			toolMsg := Message{
				Role:       RoleTool,
				Content:    toolResult,
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
			}
			messages = append(messages, toolMsg)
		}

		// Continue loop to get AI's response to tool results
		log.Debug().
			Int("iteration", iteration+1).
			Int("tool_calls", len(pendingToolCalls)).
			Msg("Processed tool calls, continuing conversation")
	}

	// Track token usage for daily budget enforcement
	if chatbot.DailyTokenBudget > 0 {
		userIdentifier := "anonymous"
		if chatCtx.UserID != nil {
			userIdentifier = *chatCtx.UserID
		}
		// ponytail: cached input tokens are billed at 0x (free) toward the
		// daily budget — they cost the provider ~0.1x (Anthropic) to ~0.5x
		// (OpenAI), so the user shouldn't foot the full bill. Effective spend
		// = fresh input + completion. CachedTokens is non-negative by
		// construction; the floor is a defensive guard.
		effective := totalUsage.PromptTokens - totalUsage.CachedTokens + totalUsage.CompletionTokens
		if effective < 0 {
			effective = 0
		}
		h.limiter.AddTokenUsage(chatbot.ID, userIdentifier, effective)
	}

	// Build the per-user daily quota snapshot for the done event (Ask 2).
	// Surfaced when the chatbot has either a daily request limit or a token
	// budget configured; omitted otherwise (preserves old wire shape).
	var dailyQuota *DailyQuotaSnapshot
	if chatbot.DailyRequestLimit > 0 || chatbot.DailyTokenBudget > 0 {
		usage := h.limiter.GetDailyUsage(chatbot.ID, userIdentifier, chatbot.DailyRequestLimit, chatbot.DailyTokenBudget)
		dailyQuota = &DailyQuotaSnapshot{
			Requests: Quota{Used: usage.RequestsUsed, Limit: usage.RequestsLimit},
			Tokens:   Quota{Used: usage.TokensUsed, Limit: usage.TokensLimit},
			ResetsAt: usage.ResetsAt.UTC().Format(time.RFC3339),
		}
	}

	// Send completion
	h.send(chatCtx, ServerMessage{
		Type:               "done",
		ConversationID:     msg.ConversationID,
		Usage:              &totalUsage,
		MatchedIntentRules: matchedIntentRules,
		DailyQuota:         dailyQuota,
	})

	// Record metrics
	if h.metrics != nil {
		h.metrics.RecordAIChatRequest(chatbot.Name, "success", time.Since(start))
		h.metrics.RecordAITokens(chatbot.Name, totalUsage.PromptTokens, totalUsage.CompletionTokens)
	}

	log.Debug().
		Str("conversation_id", msg.ConversationID).
		Int("prompt_tokens", totalUsage.PromptTokens).
		Int("completion_tokens", totalUsage.CompletionTokens).
		Msg("Message processed")
}
