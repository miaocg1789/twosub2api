package kiro

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// StreamProcessor processes CodeWhisperer EventStream responses and converts to Anthropic SSE
type StreamProcessor struct{}

// NewStreamProcessor creates a new StreamProcessor
func NewStreamProcessor() *StreamProcessor {
	return &StreamProcessor{}
}

// ProcessStream reads a CodeWhisperer EventStream response and writes Anthropic SSE events
func (sp *StreamProcessor) ProcessStream(c *gin.Context, resp *http.Response, messageID string, inputTokens int, model string, stream bool) (*StreamResult, error) {
	if messageID == "" {
		messageID = "msg_" + uuid.New().String()
	}

	parser := NewRobustEventStreamParser()
	stateManager := newSSEStateManager(messageID, model, inputTokens)
	stopManager := newStopReasonManager()
	toolManager := newToolLifecycleManager()

	buf := make([]byte, 32*1024)
	var firstTokenSent bool

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			parser.AddData(buf[:n])

			messages, parseErr := parser.GetMessages()
			if parseErr != nil {
				log.Printf("[kiro] EventStream parse error: %v", parseErr)
			}

			for _, msg := range messages {
				events := processEventStreamMessage(msg, stateManager, stopManager, toolManager)
				if stream {
					for _, event := range events {
						if !firstTokenSent {
							firstTokenSent = true
						}
						writeSSEEvent(c, event)
					}
				}
			}
		}

		if readErr != nil {
			if readErr != io.EOF {
				log.Printf("[kiro] Stream read error: %v", readErr)
			}
			break
		}
	}

	// Finalize
	result := &StreamResult{
		Content:      stateManager.getContent(),
		StopReason:   stopManager.getStopReason(),
		InputTokens:  stateManager.inputTokens,
		OutputTokens: stateManager.getOutputTokens(),
		Model:        model,
		ToolCalls:    stateManager.toolCalls,
		ThinkingContent: stateManager.getThinkingContent(),
	}

	if stream {
		// Send final events
		finalEvents := stateManager.finalize(stopManager.getStopReason())
		for _, event := range finalEvents {
			writeSSEEvent(c, event)
		}
	}

	return result, nil
}

// BuildNonStreamResponse builds a complete Anthropic JSON response from stream result
func BuildNonStreamResponse(result *StreamResult, messageID string) map[string]any {
	if messageID == "" {
		messageID = "msg_" + uuid.New().String()
	}

	content := []map[string]any{}

	// Add thinking block if present
	if result.ThinkingContent != "" {
		content = append(content, map[string]any{
			"type":     "thinking",
			"thinking": result.ThinkingContent,
		})
	}

	// Add text block if present
	if result.Content != "" {
		content = append(content, map[string]any{
			"type": "text",
			"text": result.Content,
		})
	}

	// Add tool_use blocks
	for _, tc := range result.ToolCalls {
		var inputObj any
		if tc.Input != "" {
			if err := json.Unmarshal([]byte(tc.Input), &inputObj); err != nil {
				inputObj = map[string]any{}
			}
		} else {
			inputObj = map[string]any{}
		}
		content = append(content, map[string]any{
			"type":  "tool_use",
			"id":    tc.ID,
			"name":  tc.Name,
			"input": inputObj,
		})
	}

	stopReason := result.StopReason
	if stopReason == "" {
		stopReason = "end_turn"
	}

	outputTokens := result.OutputTokens
	if outputTokens < 1 {
		outputTokens = 1
	}
	inputTokens := result.InputTokens
	if inputTokens < 1 {
		inputTokens = 1
	}

	return map[string]any{
		"id":            messageID,
		"type":          "message",
		"role":          "assistant",
		"content":       content,
		"model":         result.Model,
		"stop_reason":   stopReason,
		"stop_sequence": nil,
		"usage": map[string]any{
			"input_tokens":  inputTokens,
			"output_tokens": outputTokens,
		},
	}
}

// --- SSE State Manager ---

type sseStateManager struct {
	messageID        string
	model            string
	inputTokens      int
	outputTokens     int
	toolOutputTokens int
	content          strings.Builder
	thinkingContent  strings.Builder
	blockIndex       int
	inBlock          bool
	inThinkingBlock  bool
	started          bool
	toolCalls        []ToolCall
	// thinking tag detection state
	thinkingBuf    strings.Builder
	inThinkingTag  bool
}

func newSSEStateManager(messageID, model string, inputTokens int) *sseStateManager {
	return &sseStateManager{
		messageID:   messageID,
		model:       model,
		inputTokens: inputTokens,
	}
}

func (sm *sseStateManager) getContent() string {
	return sm.content.String()
}

func (sm *sseStateManager) getThinkingContent() string {
	return sm.thinkingContent.String()
}

func (sm *sseStateManager) getOutputTokens() int {
	// If upstream provided output tokens, use that plus tool tokens
	if sm.outputTokens > 0 {
		total := sm.outputTokens + sm.toolOutputTokens
		if total < 1 {
			total = 1
		}
		return total
	}
	// Estimate output tokens from content length if not set by upstream
	textContent := sm.content.String()
	thinkingContent := sm.thinkingContent.String()
	estimated := estimateTokens(textContent) + estimateTokens(thinkingContent) + sm.toolOutputTokens
	if estimated < 1 {
		estimated = 1
	}
	return estimated
}

func (sm *sseStateManager) setOutputTokens(tokens int) {
	sm.outputTokens = tokens
}

func (sm *sseStateManager) addToolOutputTokens(inputLen int) {
	sm.toolOutputTokens += (inputLen + 3) / 4
}

func (sm *sseStateManager) ensureStarted() []SSEEvent {
	if sm.started {
		return nil
	}
	sm.started = true

	return []SSEEvent{
		{
			Event: "message_start",
			Data: mustJSON(map[string]any{
				"type": "message_start",
				"message": map[string]any{
					"id":            sm.messageID,
					"type":          "message",
					"role":          "assistant",
					"content":       []any{},
					"model":         sm.model,
					"stop_reason":   nil,
					"stop_sequence": nil,
					"usage": map[string]any{
						"input_tokens":  sm.inputTokens,
						"output_tokens": 0,
					},
				},
			}),
		},
	}
}

func (sm *sseStateManager) startThinkingBlock() []SSEEvent {
	events := sm.ensureStarted()
	if !sm.inThinkingBlock {
		sm.inThinkingBlock = true
		sm.inBlock = true
		events = append(events, SSEEvent{
			Event: "content_block_start",
			Data: mustJSON(map[string]any{
				"type":  "content_block_start",
				"index": sm.blockIndex,
				"content_block": map[string]any{
					"type":     "thinking",
					"thinking": "",
				},
			}),
		})
	}
	return events
}

func (sm *sseStateManager) addThinkingDelta(text string) []SSEEvent {
	events := sm.startThinkingBlock()
	sm.thinkingContent.WriteString(text)
	events = append(events, SSEEvent{
		Event: "content_block_delta",
		Data: mustJSON(map[string]any{
			"type":  "content_block_delta",
			"index": sm.blockIndex,
			"delta": map[string]any{
				"type":     "thinking_delta",
				"thinking": text,
			},
		}),
	})
	return events
}

func (sm *sseStateManager) stopThinkingBlock() []SSEEvent {
	if !sm.inThinkingBlock {
		return nil
	}
	sm.inThinkingBlock = false
	sm.inBlock = false
	events := []SSEEvent{
		{
			Event: "content_block_stop",
			Data: mustJSON(map[string]any{
				"type":  "content_block_stop",
				"index": sm.blockIndex,
			}),
		},
	}
	sm.blockIndex++
	return events
}

func (sm *sseStateManager) startTextBlock() []SSEEvent {
	events := sm.ensureStarted()
	// Close thinking block if open
	if sm.inThinkingBlock {
		events = append(events, sm.stopThinkingBlock()...)
	}
	if !sm.inBlock {
		sm.inBlock = true
		events = append(events, SSEEvent{
			Event: "content_block_start",
			Data: mustJSON(map[string]any{
				"type":  "content_block_start",
				"index": sm.blockIndex,
				"content_block": map[string]any{
					"type": "text",
					"text": "",
				},
			}),
		})
	}
	return events
}

func (sm *sseStateManager) addTextDelta(text string) []SSEEvent {
	events := sm.startTextBlock()
	sm.content.WriteString(text)
	events = append(events, SSEEvent{
		Event: "content_block_delta",
		Data: mustJSON(map[string]any{
			"type":  "content_block_delta",
			"index": sm.blockIndex,
			"delta": map[string]any{
				"type": "text_delta",
				"text": text,
			},
		}),
	})
	return events
}

func (sm *sseStateManager) stopCurrentBlock() []SSEEvent {
	if sm.inThinkingBlock {
		return sm.stopThinkingBlock()
	}
	if !sm.inBlock {
		return nil
	}
	sm.inBlock = false
	events := []SSEEvent{
		{
			Event: "content_block_stop",
			Data: mustJSON(map[string]any{
				"type":  "content_block_stop",
				"index": sm.blockIndex,
			}),
		},
	}
	sm.blockIndex++
	return events
}

func (sm *sseStateManager) startToolUseBlock(toolCall ToolCall) []SSEEvent {
	events := sm.stopCurrentBlock()
	events = append(events, sm.ensureStarted()...)
	sm.inBlock = true
	sm.toolCalls = append(sm.toolCalls, toolCall)

	events = append(events, SSEEvent{
		Event: "content_block_start",
		Data: mustJSON(map[string]any{
			"type":  "content_block_start",
			"index": sm.blockIndex,
			"content_block": map[string]any{
				"type":  "tool_use",
				"id":    toolCall.ID,
				"name":  toolCall.Name,
				"input": map[string]any{},
			},
		}),
	})
	return events
}

func (sm *sseStateManager) addToolInputDelta(input string) []SSEEvent {
	return []SSEEvent{
		{
			Event: "content_block_delta",
			Data: mustJSON(map[string]any{
				"type":  "content_block_delta",
				"index": sm.blockIndex,
				"delta": map[string]any{
					"type":         "input_json_delta",
					"partial_json": input,
				},
			}),
		},
	}
}

func (sm *sseStateManager) finalize(stopReason string) []SSEEvent {
	var events []SSEEvent

	// Ensure we've started
	events = append(events, sm.ensureStarted()...)

	// Close any open block
	events = append(events, sm.stopCurrentBlock()...)

	if stopReason == "" {
		stopReason = "end_turn"
	}

	// message_delta
	events = append(events, SSEEvent{
		Event: "message_delta",
		Data: mustJSON(map[string]any{
			"type": "message_delta",
			"delta": map[string]any{
				"stop_reason":   stopReason,
				"stop_sequence": nil,
			},
			"usage": map[string]any{
				"input_tokens":  sm.inputTokens,
				"output_tokens": sm.getOutputTokens(),
			},
		}),
	})

	// message_stop
	events = append(events, SSEEvent{
		Event: "message_stop",
		Data:  mustJSON(map[string]any{"type": "message_stop"}),
	})

	return events
}

// --- Stop Reason Manager ---

type stopReasonManager struct {
	stopReason string
	hasToolUse bool
}

func newStopReasonManager() *stopReasonManager {
	return &stopReasonManager{}
}

func (m *stopReasonManager) setToolUse() {
	m.hasToolUse = true
}

func (m *stopReasonManager) setStopReason(reason string) {
	m.stopReason = reason
}

func (m *stopReasonManager) getStopReason() string {
	if m.hasToolUse {
		return "tool_use"
	}
	if m.stopReason != "" {
		return m.stopReason
	}
	return "end_turn"
}

// --- Tool Lifecycle Manager ---

type toolLifecycleManager struct {
	activeTools  map[string]*ToolCall
	inputBuffers map[string]*strings.Builder
}

func newToolLifecycleManager() *toolLifecycleManager {
	return &toolLifecycleManager{
		activeTools:  make(map[string]*ToolCall),
		inputBuffers: make(map[string]*strings.Builder),
	}
}

func (m *toolLifecycleManager) startTool(id, name string) {
	m.activeTools[id] = &ToolCall{ID: id, Name: name}
	m.inputBuffers[id] = &strings.Builder{}
}

func (m *toolLifecycleManager) appendInput(id, input string) {
	if buf, ok := m.inputBuffers[id]; ok {
		buf.WriteString(input)
	}
}

func (m *toolLifecycleManager) finishTool(id string) string {
	if buf, ok := m.inputBuffers[id]; ok {
		input := buf.String()
		delete(m.activeTools, id)
		delete(m.inputBuffers, id)
		return input
	}
	return ""
}

// --- Event Processing ---

func processEventStreamMessage(msg EventStreamMessage, sm *sseStateManager, stopMgr *stopReasonManager, toolMgr *toolLifecycleManager) []SSEEvent {
	messageType := msg.GetHeaderString(":message-type")
	eventType := msg.GetHeaderString(":event-type")

	if messageType == MessageTypeException {
		exceptionType := msg.GetHeaderString(":exception-type")
		log.Printf("[kiro] EventStream exception: type=%s, payload=%s", exceptionType, string(msg.Payload))

		// Handle ContentLengthExceededException
		if exceptionType == "ContentLengthExceededException" {
			stopMgr.setStopReason("max_tokens")
		}
		return nil
	}

	if messageType != MessageTypeEvent {
		return nil
	}

	switch eventType {
	case "assistantResponseEvent":
		return handleAssistantResponseEvent(msg.Payload, sm, stopMgr, toolMgr)
	case "reasoningContentEvent":
		return handleReasoningContentEvent(msg.Payload, sm)
	case "messageMetadataEvent", "metadataEvent":
		return handleMetadataEvent(msg.Payload, sm)
	case "contextUsageEvent":
		return handleContextUsageEvent(msg.Payload, sm)
	case "meteringEvent":
		handleMeteringEvent(msg.Payload)
		return nil
	case "supplementaryWebLinksEvent":
		// Ignore supplementary web links
		return nil
	default:
		// Try to parse as assistant response event anyway
		if len(msg.Payload) > 0 {
			return handleAssistantResponseEvent(msg.Payload, sm, stopMgr, toolMgr)
		}
		return nil
	}
}

// thinkingState tracks the state of inline thinking tag detection
// (embedded in sseStateManager, not global)

func handleAssistantResponseEvent(payload []byte, sm *sseStateManager, stopMgr *stopReasonManager, toolMgr *toolLifecycleManager) []SSEEvent {
	if len(payload) == 0 {
		return nil
	}

	// EventStream binary format: event type is in headers, payload is the direct content.
	var event AssistantResponseEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		log.Printf("[kiro] Failed to parse assistant response event: %v", err)
		return nil
	}

	var allEvents []SSEEvent

	// Handle content/completion text with thinking tag detection
	if event.Content != "" {
		events := processContentWithThinking(event.Content, sm)
		allEvents = append(allEvents, events...)
	}

	// Handle tool calls
	if event.ToolUse != nil {
		toolID := event.ToolUse.ToolUseID
		if toolID == "" {
			toolID = "toolu_" + uuid.New().String()
		}

		toolMgr.startTool(toolID, event.ToolUse.Name)
		stopMgr.setToolUse()

		tc := ToolCall{
			ID:   toolID,
			Name: event.ToolUse.Name,
		}
		events := sm.startToolUseBlock(tc)
		allEvents = append(allEvents, events...)

		// Send tool input
		if event.ToolUse.Input != nil {
			inputBytes, err := json.Marshal(event.ToolUse.Input)
			if err == nil && len(inputBytes) > 0 && string(inputBytes) != "null" {
				toolMgr.appendInput(toolID, string(inputBytes))
				events := sm.addToolInputDelta(string(inputBytes))
				allEvents = append(allEvents, events...)
				// Count tool input towards output tokens
				sm.addToolOutputTokens(len(inputBytes))
			}
		}

		// Close tool block
		toolMgr.finishTool(toolID)
		events = sm.stopCurrentBlock()
		allEvents = append(allEvents, events...)
	}

	// Handle token usage
	if event.Usage != nil {
		if event.Usage.OutputTokens > 0 {
			sm.setOutputTokens(event.Usage.OutputTokens)
		}
	}

	// Handle stop reason
	if event.StopReason != "" {
		stopMgr.setStopReason(event.StopReason)
	}

	return allEvents
}

// processContentWithThinking processes content text, detecting inline <thinking> tags
// and splitting into thinking deltas vs text deltas
func processContentWithThinking(content string, sm *sseStateManager) []SSEEvent {
	var allEvents []SSEEvent

	// Buffer content and look for thinking tags
	sm.thinkingBuf.WriteString(content)
	buffered := sm.thinkingBuf.String()

	for len(buffered) > 0 {
		if sm.inThinkingTag {
			// Looking for </thinking>
			closeIdx := strings.Index(buffered, "</thinking>")
			if closeIdx == -1 {
				// Haven't found close tag yet, emit as thinking
				events := sm.addThinkingDelta(buffered)
				allEvents = append(allEvents, events...)
				buffered = ""
			} else {
				// Found close tag - check if it's followed by \n\n
				afterClose := buffered[closeIdx+len("</thinking>"):]
				if strings.HasPrefix(afterClose, "\n\n") {
					// Genuine thinking close
					thinkingText := buffered[:closeIdx]
					if thinkingText != "" {
						events := sm.addThinkingDelta(thinkingText)
						allEvents = append(allEvents, events...)
					}
					// Close thinking block
					events := sm.stopThinkingBlock()
					allEvents = append(allEvents, events...)
					sm.inThinkingTag = false
					buffered = afterClose[2:] // skip \n\n
				} else if len(afterClose) < 2 {
					// Not enough data to determine, wait for more
					events := sm.addThinkingDelta(buffered[:closeIdx])
					allEvents = append(allEvents, events...)
					buffered = buffered[closeIdx:]
					break
				} else {
					// Not a genuine close, emit as thinking content
					events := sm.addThinkingDelta(buffered[:closeIdx+len("</thinking>")])
					allEvents = append(allEvents, events...)
					buffered = afterClose
				}
			}
		} else {
			// Looking for <thinking>
			openIdx := strings.Index(buffered, "<thinking>")
			if openIdx == -1 {
				// No thinking tag, emit as text
				events := sm.addTextDelta(buffered)
				allEvents = append(allEvents, events...)
				buffered = ""
			} else {
				// Check if this is a genuine thinking tag (not quoted)
				before := buffered[:openIdx]
				isQuoted := false
				if len(before) > 0 {
					lastChar := before[len(before)-1]
					if lastChar == '"' || lastChar == '\'' || lastChar == '`' {
						isQuoted = true
					}
				}
				if isQuoted {
					// Not a genuine tag, emit everything up to and including the tag as text
					end := openIdx + len("<thinking>")
					events := sm.addTextDelta(buffered[:end])
					allEvents = append(allEvents, events...)
					buffered = buffered[end:]
				} else {
					// Genuine thinking open
					if before != "" {
						events := sm.addTextDelta(before)
						allEvents = append(allEvents, events...)
					}
					sm.inThinkingTag = true
					buffered = buffered[openIdx+len("<thinking>"):]
					// Skip leading newline after <thinking>
					buffered = strings.TrimLeft(buffered, "\n")
				}
			}
		}
	}

	sm.thinkingBuf.Reset()
	if buffered != "" {
		sm.thinkingBuf.WriteString(buffered)
	}

	return allEvents
}

// handleReasoningContentEvent handles thinking/reasoning content from CodeWhisperer
func handleReasoningContentEvent(payload []byte, sm *sseStateManager) []SSEEvent {
	if len(payload) == 0 {
		return nil
	}

	var event struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil
	}

	if event.Text == "" {
		return nil
	}

	// Emit reasoning content as thinking delta
	return sm.addThinkingDelta(event.Text)
}

// handleMetadataEvent handles token usage metadata from CodeWhisperer
func handleMetadataEvent(payload []byte, sm *sseStateManager) []SSEEvent {
	if len(payload) == 0 {
		return nil
	}

	var event struct {
		TokenUsage *struct {
			OutputTokens          int `json:"outputTokens"`
			UncachedInputTokens   int `json:"uncachedInputTokens"`
			CacheReadInputTokens  int `json:"cacheReadInputTokens"`
			CacheWriteInputTokens int `json:"cacheWriteInputTokens"`
		} `json:"tokenUsage"`
	}
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil
	}

	if event.TokenUsage != nil {
		if event.TokenUsage.OutputTokens > 0 {
			sm.setOutputTokens(event.TokenUsage.OutputTokens)
		}
		totalInput := event.TokenUsage.UncachedInputTokens +
			event.TokenUsage.CacheReadInputTokens +
			event.TokenUsage.CacheWriteInputTokens
		if totalInput > 0 {
			sm.inputTokens = totalInput
		}
	}

	return nil
}

// handleContextUsageEvent handles context usage percentage events
func handleContextUsageEvent(payload []byte, sm *sseStateManager) []SSEEvent {
	if len(payload) == 0 {
		return nil
	}

	var event struct {
		ContextUsagePercentage float64 `json:"context_usage_percentage"`
	}
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil
	}

	if event.ContextUsagePercentage > 0 {
		// Calculate input_tokens from percentage: percentage * 200000 / 100
		inputTokens := int(event.ContextUsagePercentage * 200000 / 100)
		if inputTokens > 0 {
			sm.inputTokens = inputTokens
		}
	}

	return nil
}

// handleMeteringEvent logs credit usage from CodeWhisperer
func handleMeteringEvent(payload []byte) {
	if len(payload) == 0 {
		return
	}

	var event struct {
		Usage float64 `json:"usage"`
	}
	if err := json.Unmarshal(payload, &event); err != nil {
		return
	}

	if event.Usage > 0 {
		log.Printf("[kiro] Metering: %.4f credits used", event.Usage)
	}
}

// --- SSE Writing ---

func writeSSEEvent(c *gin.Context, event SSEEvent) {
	c.Writer.WriteString("event: " + event.Event + "\n")
	c.Writer.WriteString("data: " + event.Data + "\n\n")
	c.Writer.Flush()
}

// estimateTokens estimates the number of tokens in a text string.
// Chinese characters count as ~1.5 chars/token, other characters ~4 chars/token.
func estimateTokens(text string) int {
	if text == "" {
		return 0
	}
	var cjkCount, otherCount int
	for _, r := range text {
		if unicode.Is(unicode.Han, r) || unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Hangul, r) {
			cjkCount++
		} else {
			otherCount++
		}
	}
	// CJK: ~1.5 chars per token => multiply by 2/3
	cjkTokens := (cjkCount*2 + 2) / 3
	otherTokens := (otherCount + 3) / 4
	total := cjkTokens + otherTokens
	if total < 1 && (cjkCount+otherCount) > 0 {
		total = 1
	}
	return total
}

// EstimateInputTokens estimates the input token count from an AnthropicRequest.
// This is used as a fallback when CodeWhisperer doesn't return contextUsageEvent.
func EstimateInputTokens(req AnthropicRequest) int {
	total := 0

	// System messages
	if len(req.System) > 0 {
		// Try to parse as array of system messages
		var systemMsgs []AnthropicSystemMessage
		if err := json.Unmarshal(req.System, &systemMsgs); err == nil {
			for _, msg := range systemMsgs {
				total += estimateTokens(msg.Text)
			}
		} else {
			// Try as plain string
			var systemStr string
			if err := json.Unmarshal(req.System, &systemStr); err == nil {
				total += estimateTokens(systemStr)
			}
		}
	}

	// Messages
	for _, msg := range req.Messages {
		switch content := msg.Content.(type) {
		case string:
			total += estimateTokens(content)
		case []any:
			for _, block := range content {
				if blockMap, ok := block.(map[string]any); ok {
					if text, ok := blockMap["text"].(string); ok {
						total += estimateTokens(text)
					}
					// tool_result content
					if contentVal, ok := blockMap["content"]; ok {
						switch c := contentVal.(type) {
						case string:
							total += estimateTokens(c)
						case []any:
							for _, sub := range c {
								if subMap, ok := sub.(map[string]any); ok {
									if text, ok := subMap["text"].(string); ok {
										total += estimateTokens(text)
									}
								}
							}
						}
					}
					// tool_use input
					if input, ok := blockMap["input"]; ok {
						if inputBytes, err := json.Marshal(input); err == nil {
							total += estimateTokens(string(inputBytes))
						}
					}
				}
			}
		}
	}

	// Tools
	for _, tool := range req.Tools {
		total += estimateTokens(tool.Name)
		total += estimateTokens(tool.Description)
		if tool.InputSchema != nil {
			if schemaBytes, err := json.Marshal(tool.InputSchema); err == nil {
				total += estimateTokens(string(schemaBytes))
			}
		}
	}

	if total < 1 {
		total = 1
	}
	return total
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// --- Ping support ---

// StartPingTicker starts a goroutine that sends SSE ping events
func StartPingTicker(c *gin.Context, done <-chan struct{}) {
	ticker := time.NewTicker(15 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				writeSSEEvent(c, SSEEvent{
					Event: "ping",
					Data:  mustJSON(map[string]any{"type": "ping"}),
				})
			}
		}
	}()
}
