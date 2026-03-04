package kiro

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
)

// unsupportedTools lists tool names that should be filtered out
var unsupportedTools = map[string]bool{
	"web_search": true,
	"websearch":  true,
}

// ConvertAnthropicToCodeWhisperer converts an Anthropic Messages API request to CodeWhisperer format
func ConvertAnthropicToCodeWhisperer(anthropicReq AnthropicRequest) (*CodeWhispererRequest, error) {
	if len(anthropicReq.Messages) == 0 {
		return nil, fmt.Errorf("no messages in request")
	}

	// Determine model ID
	modelID, ok := ModelMap[anthropicReq.Model]
	if !ok {
		modelID = "auto"
	}

	// Extract system prompt
	systemPrompt := extractSystemPrompt(anthropicReq.System)

	// Detect context compression
	isCompression := detectContextCompression(anthropicReq.Messages)

	// Build thinking prefix if enabled
	thinkingPrefix := ""
	if anthropicReq.Thinking != nil && anthropicReq.Thinking.Type == "enabled" {
		budgetTokens := anthropicReq.Thinking.BudgetTokens
		if budgetTokens > 24576 {
			budgetTokens = 24576
		}
		if budgetTokens <= 0 {
			budgetTokens = 10000
		}
		thinkingPrefix = fmt.Sprintf("<thinking_mode>enabled</thinking_mode><max_thinking_length>%d</max_thinking_length>", budgetTokens)
	}

	// Pre-process messages: merge trailing consecutive user messages
	messages := mergeTrailingUserMessages(anthropicReq.Messages)

	// Build history from all messages except the last
	history := buildHistory(messages, modelID)

	// Inject system prompt as user→assistant pair at the beginning of history
	if systemPrompt != "" {
		systemContent := systemPrompt
		if thinkingPrefix != "" {
			systemContent = thinkingPrefix + "\n\n" + systemContent
		}
		systemUserMsg := HistoryUserMessage{
			UserInputMessage: UserInputMessage{
				Content: systemContent,
				Origin:  "AI_EDITOR",
			},
		}
		systemAssistantMsg := HistoryAssistantMessage{
			AssistantResponseMessage: AssistantResponseMessage{
				MessageID: uuid.New().String(),
				Content:   "I will follow these instructions.",
			},
		}
		// Prepend system pair to history
		newHistory := make([]any, 0, len(history)+2)
		newHistory = append(newHistory, systemUserMsg, systemAssistantMsg)
		newHistory = append(newHistory, history...)
		history = newHistory
	} else if thinkingPrefix != "" {
		// No system prompt but thinking enabled - inject thinking config as system pair
		systemUserMsg := HistoryUserMessage{
			UserInputMessage: UserInputMessage{
				Content: thinkingPrefix,
				Origin:  "AI_EDITOR",
			},
		}
		systemAssistantMsg := HistoryAssistantMessage{
			AssistantResponseMessage: AssistantResponseMessage{
				MessageID: uuid.New().String(),
				Content:   "I will follow these instructions.",
			},
		}
		newHistory := make([]any, 0, len(history)+2)
		newHistory = append(newHistory, systemUserMsg, systemAssistantMsg)
		newHistory = append(newHistory, history...)
		history = newHistory
	}

	// Ensure history has proper user→assistant pairing (no orphan user messages)
	history = ensureHistoryPairing(history)

	// Build the current message content from the last message
	lastMsg := messages[len(messages)-1]

	var currentText string
	var images []CodeWhispererImage
	var err error

	if lastMsg.Role == "assistant" {
		// Messages end with assistant - add "continue" as current message
		// The assistant message is already in history
		currentText = "continue"
	} else {
		currentText, images, err = processMessageContent(lastMsg.Content)
		if err != nil {
			return nil, fmt.Errorf("process last message content: %w", err)
		}
	}

	// Convert tool results from the last message if it's a user message with tool results
	var toolResults []ToolResult
	if !isCompression {
		toolResults = convertToolResults(messages)
	}

	// Convert tools (filter unsupported ones)
	var tools []CodeWhispererTool
	if len(anthropicReq.Tools) > 0 && !isCompression {
		tools = convertAnthropicToolsToCodeWhisperer(filterUnsupportedTools(anthropicReq.Tools))
	}

	chatTriggerType := determineChatTriggerType(anthropicReq)

	cwReq := &CodeWhispererRequest{
		ConversationState: ConversationState{
			ChatTriggerType:     chatTriggerType,
			ConversationID:      uuid.New().String(),
			AgentContinuationId: uuid.New().String(),
			AgentTaskType:       "vibe",
			CurrentMessage: CurrentMessage{
				UserInputMessage: UserInputMessage{
					Content: currentText,
					ModelID: modelID,
					Origin:  "AI_EDITOR",
					Images:  images,
				},
			},
			History: history,
		},
	}

	// Attach tools and tool results to the user input message context
	if len(tools) > 0 || len(toolResults) > 0 {
		cwReq.ConversationState.CurrentMessage.UserInputMessage.UserInputMessageContext = &UserInputContext{
			Tools:       tools,
			ToolResults: toolResults,
		}
	}

	// Pass inference config (max_tokens, temperature, top_p)
	if anthropicReq.MaxTokens > 0 || anthropicReq.Temperature != nil || anthropicReq.TopP != nil {
		cwReq.InferenceConfig = &InferenceConfig{}
		if anthropicReq.MaxTokens > 0 {
			cwReq.InferenceConfig.MaxTokens = anthropicReq.MaxTokens
		}
		if anthropicReq.Temperature != nil {
			cwReq.InferenceConfig.Temperature = *anthropicReq.Temperature
		}
		if anthropicReq.TopP != nil {
			cwReq.InferenceConfig.TopP = *anthropicReq.TopP
		}
	}

	return cwReq, nil
}

// extractSystemPrompt extracts the system prompt from the raw JSON
func extractSystemPrompt(systemRaw json.RawMessage) string {
	if len(systemRaw) == 0 {
		return ""
	}

	// Try as string first
	var systemStr string
	if err := json.Unmarshal(systemRaw, &systemStr); err == nil {
		return systemStr
	}

	// Try as array of system message blocks
	var blocks []AnthropicSystemMessage
	if err := json.Unmarshal(systemRaw, &blocks); err == nil {
		var parts []string
		for _, block := range blocks {
			if block.Type == "text" && block.Text != "" {
				parts = append(parts, block.Text)
			}
		}
		return strings.Join(parts, "\n\n")
	}

	return ""
}

// processMessageContent extracts text and images from message content
func processMessageContent(content any) (string, []CodeWhispererImage, error) {
	if content == nil {
		return "", nil, nil
	}

	// String content
	if str, ok := content.(string); ok {
		return str, nil, nil
	}

	// Array content (content blocks)
	contentBytes, err := json.Marshal(content)
	if err != nil {
		return "", nil, fmt.Errorf("marshal content: %w", err)
	}

	var blocks []ContentBlock
	if err := json.Unmarshal(contentBytes, &blocks); err != nil {
		return "", nil, fmt.Errorf("unmarshal content blocks: %w", err)
	}

	var textParts []string
	var images []CodeWhispererImage

	for _, block := range blocks {
		switch block.Type {
		case "text":
			if block.Text != "" {
				textParts = append(textParts, block.Text)
			}
		case "image":
			if block.Source != nil {
				images = append(images, CodeWhispererImage{
					Format: mapMediaTypeToFormat(block.Source.MediaType),
					Source: CodeWhispererImageSource{
						Bytes: block.Source.Data,
					},
				})
			}
		case "tool_use":
			// Tool use blocks in user messages are handled separately
		case "tool_result":
			// Tool results are handled by convertToolResults
			if block.Content != nil {
				text := extractToolResultText(block.Content)
				if text != "" {
					textParts = append(textParts, text)
				}
			}
		}
	}

	return strings.Join(textParts, "\n"), images, nil
}

// mapMediaTypeToFormat converts MIME type to CodeWhisperer image format
func mapMediaTypeToFormat(mediaType string) string {
	switch mediaType {
	case "image/png":
		return "png"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	default:
		return "jpeg"
	}
}

// extractToolResultText extracts text from tool result content
func extractToolResultText(content any) string {
	if str, ok := content.(string); ok {
		return str
	}

	contentBytes, err := json.Marshal(content)
	if err != nil {
		return ""
	}

	var blocks []ContentBlock
	if err := json.Unmarshal(contentBytes, &blocks); err != nil {
		return ""
	}

	var parts []string
	for _, block := range blocks {
		if block.Type == "text" && block.Text != "" {
			parts = append(parts, block.Text)
		}
	}
	return strings.Join(parts, "\n")
}

// filterUnsupportedTools removes unsupported tools (web_search, websearch)
func filterUnsupportedTools(tools []AnthropicTool) []AnthropicTool {
	var filtered []AnthropicTool
	for _, tool := range tools {
		if !unsupportedTools[tool.Name] {
			filtered = append(filtered, tool)
		}
	}
	return filtered
}

// convertAnthropicToolsToCodeWhisperer converts Anthropic tools to CodeWhisperer format
func convertAnthropicToolsToCodeWhisperer(tools []AnthropicTool) []CodeWhispererTool {
	var cwTools []CodeWhispererTool
	for _, tool := range tools {
		desc := tool.Description
		if len(desc) > MaxToolDescriptionLength {
			desc = desc[:MaxToolDescriptionLength] + "..."
		}
		cwTools = append(cwTools, CodeWhispererTool{
			ToolSpec: ToolSpecification{
				Name:        shortenToolName(tool.Name),
				Description: desc,
				InputSchema: InputSchema{
					JSON: tool.InputSchema,
				},
			},
		})
	}
	return cwTools
}

// shortenToolName truncates tool names to MaxToolNameLength (64 chars).
// For MCP tools (mcp__server__tool), it tries to remove the middle segment first.
func shortenToolName(name string) string {
	if len(name) <= MaxToolNameLength {
		return name
	}
	// MCP tools: mcp__server__tool -> mcp__tool
	if strings.HasPrefix(name, "mcp__") {
		lastIdx := strings.LastIndex(name, "__")
		if lastIdx > 5 {
			shortened := "mcp__" + name[lastIdx+2:]
			if len(shortened) <= MaxToolNameLength {
				return shortened
			}
		}
	}
	return name[:MaxToolNameLength]
}

// convertToolResults extracts tool results from the message sequence
func convertToolResults(messages []AnthropicRequestMessage) []ToolResult {
	if len(messages) == 0 {
		return nil
	}

	lastMsg := messages[len(messages)-1]
	if lastMsg.Role != "user" {
		return nil
	}

	contentBytes, err := json.Marshal(lastMsg.Content)
	if err != nil {
		return nil
	}

	var blocks []ContentBlock
	if err := json.Unmarshal(contentBytes, &blocks); err != nil {
		return nil
	}

	var results []ToolResult
	for _, block := range blocks {
		if block.Type == "tool_result" && block.ToolUseID != "" {
			text := ""
			if block.Content != nil {
				text = extractToolResultText(block.Content)
			}
			status := "SUCCESS"
			if block.IsError != nil && *block.IsError {
				status = "ERROR"
			}
			results = append(results, ToolResult{
				ToolUseID: block.ToolUseID,
				Content: []ToolResultContent{
					{Text: text},
				},
				Status: status,
			})
		}
	}

	return results
}

// mergeTrailingUserMessages merges consecutive trailing user messages into one
func mergeTrailingUserMessages(messages []AnthropicRequestMessage) []AnthropicRequestMessage {
	if len(messages) <= 1 {
		return messages
	}

	// Find how many trailing user messages there are
	trailingCount := 0
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			trailingCount++
		} else {
			break
		}
	}

	if trailingCount <= 1 {
		return messages
	}

	// Merge trailing user messages
	startIdx := len(messages) - trailingCount
	var mergedParts []string
	var mergedBlocks []ContentBlock

	hasBlocks := false
	for i := startIdx; i < len(messages); i++ {
		msg := messages[i]
		if str, ok := msg.Content.(string); ok {
			mergedParts = append(mergedParts, str)
		} else {
			hasBlocks = true
			contentBytes, err := json.Marshal(msg.Content)
			if err != nil {
				continue
			}
			var blocks []ContentBlock
			if err := json.Unmarshal(contentBytes, &blocks); err != nil {
				continue
			}
			mergedBlocks = append(mergedBlocks, blocks...)
		}
	}

	result := make([]AnthropicRequestMessage, startIdx+1)
	copy(result, messages[:startIdx])

	if hasBlocks {
		// If any message had structured content, convert all text to blocks and merge
		for _, text := range mergedParts {
			mergedBlocks = append([]ContentBlock{{Type: "text", Text: text}}, mergedBlocks...)
		}
		result[startIdx] = AnthropicRequestMessage{
			Role:    "user",
			Content: mergedBlocks,
		}
	} else {
		result[startIdx] = AnthropicRequestMessage{
			Role:    "user",
			Content: strings.Join(mergedParts, "\n"),
		}
	}

	return result
}

// buildHistory builds conversation history from messages (excluding the last one)
func buildHistory(messages []AnthropicRequestMessage, modelId string) []any {
	if len(messages) <= 1 {
		return nil
	}

	var history []any
	// Process all messages except the last one
	for i := 0; i < len(messages)-1; i++ {
		msg := messages[i]
		switch msg.Role {
		case "user":
			text, images, err := processMessageContent(msg.Content)
			if err != nil {
				log.Printf("[kiro] warning: failed to process history user message: %v", err)
				continue
			}

			// Extract tool results from this user message for context
			var toolResults []ToolResult
			if contentBytes, err := json.Marshal(msg.Content); err == nil {
				var blocks []ContentBlock
				if err := json.Unmarshal(contentBytes, &blocks); err == nil {
					for _, block := range blocks {
						if block.Type == "tool_result" && block.ToolUseID != "" {
							trText := ""
							if block.Content != nil {
								trText = extractToolResultText(block.Content)
							}
							status := "SUCCESS"
							if block.IsError != nil && *block.IsError {
								status = "ERROR"
							}
							toolResults = append(toolResults, ToolResult{
								ToolUseID: block.ToolUseID,
								Content:   []ToolResultContent{{Text: trText}},
								Status:    status,
							})
						}
					}
				}
			}

			userMsg := HistoryUserMessage{
				UserInputMessage: UserInputMessage{
					Content: text,
					Origin:  "AI_EDITOR",
					Images:  images,
				},
			}
			if len(toolResults) > 0 {
				userMsg.UserInputMessage.UserInputMessageContext = &UserInputContext{
					ToolResults: toolResults,
				}
			}
			history = append(history, userMsg)

		case "assistant":
			text, _, err := processMessageContent(msg.Content)
			if err != nil {
				log.Printf("[kiro] warning: failed to process history assistant message: %v", err)
				continue
			}

			// Extract tool uses from assistant message
			toolUses := extractToolUses(msg.Content)

			assistantMsg := HistoryAssistantMessage{
				AssistantResponseMessage: AssistantResponseMessage{
					MessageID: uuid.New().String(),
					Content:   text,
					ToolUses:  toolUses,
				},
			}
			history = append(history, assistantMsg)
		}
	}

	return history
}

// ensureHistoryPairing ensures history has proper user→assistant pairing.
// Orphan user messages get a synthetic "OK" assistant response.
func ensureHistoryPairing(history []any) []any {
	if len(history) == 0 {
		return history
	}

	var result []any
	for i := 0; i < len(history); i++ {
		result = append(result, history[i])

		// Check if this is a user message
		if _, isUser := history[i].(HistoryUserMessage); isUser {
			// Check if next message is an assistant message
			hasAssistantNext := false
			if i+1 < len(history) {
				if _, isAssistant := history[i+1].(HistoryAssistantMessage); isAssistant {
					hasAssistantNext = true
				}
			}
			if !hasAssistantNext {
				// Insert synthetic assistant response
				result = append(result, HistoryAssistantMessage{
					AssistantResponseMessage: AssistantResponseMessage{
						MessageID: uuid.New().String(),
						Content:   "OK",
					},
				})
			}
		}
	}

	return result
}

// extractToolUses extracts tool use entries from assistant message content
func extractToolUses(content any) []ToolUseEntry {
	if content == nil {
		return nil
	}

	contentBytes, err := json.Marshal(content)
	if err != nil {
		return nil
	}

	var blocks []ContentBlock
	if err := json.Unmarshal(contentBytes, &blocks); err != nil {
		return nil
	}

	var toolUses []ToolUseEntry
	for _, block := range blocks {
		if block.Type == "tool_use" && block.ID != "" {
			inputStr := ""
			if block.Input != nil {
				inputBytes, err := json.Marshal(block.Input)
				if err == nil {
					inputStr = string(inputBytes)
				}
			}
			toolUses = append(toolUses, ToolUseEntry{
				ToolUseID: block.ID,
				Name:      block.Name,
				Input:     inputStr,
			})
		}
	}

	return toolUses
}

// determineChatTriggerType determines the chat trigger type based on the request
func determineChatTriggerType(req AnthropicRequest) string {
	// When tools exist and tool_choice specifies "any" or "tool" → AUTO
	if len(req.Tools) > 0 && req.ToolChoice != nil {
		switch req.ToolChoice.Type {
		case "any", "tool":
			return "AUTO"
		}
	}

	// Check if the last message contains tool results → AUTO
	if len(req.Messages) > 0 {
		lastMsg := req.Messages[len(req.Messages)-1]
		if lastMsg.Role == "user" {
			contentBytes, err := json.Marshal(lastMsg.Content)
			if err == nil {
				var blocks []ContentBlock
				if err := json.Unmarshal(contentBytes, &blocks); err == nil {
					for _, block := range blocks {
						if block.Type == "tool_result" {
							return "AUTO"
						}
					}
				}
			}
		}
	}

	return "MANUAL"
}

// detectContextCompression checks if the last user message indicates context compression
func detectContextCompression(messages []AnthropicRequestMessage) bool {
	if len(messages) == 0 {
		return false
	}

	lastMsg := messages[len(messages)-1]
	if lastMsg.Role != "user" {
		return false
	}

	text, _, _ := processMessageContent(lastMsg.Content)
	lower := strings.ToLower(text)
	return strings.Contains(lower, "compress the conversation history") ||
		strings.Contains(lower, "context compression system")
}
