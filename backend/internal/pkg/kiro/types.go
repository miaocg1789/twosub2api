package kiro

import "encoding/json"

// --- Anthropic Request Types ---

// AnthropicRequest represents an incoming Anthropic Messages API request
type AnthropicRequest struct {
	Model       string                   `json:"model"`
	MaxTokens   int                      `json:"max_tokens"`
	Messages    []AnthropicRequestMessage `json:"messages"`
	System      json.RawMessage          `json:"system,omitempty"`
	Stream      bool                     `json:"stream"`
	Temperature *float64                 `json:"temperature,omitempty"`
	TopP        *float64                 `json:"top_p,omitempty"`
	TopK        *int                     `json:"top_k,omitempty"`
	StopSequences []string              `json:"stop_sequences,omitempty"`
	Tools       []AnthropicTool          `json:"tools,omitempty"`
	ToolChoice  *ToolChoice              `json:"tool_choice,omitempty"`
	Thinking    *ThinkingConfig          `json:"thinking,omitempty"`
	Metadata    map[string]any           `json:"metadata,omitempty"`
}

// ThinkingConfig represents thinking/extended thinking configuration
type ThinkingConfig struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens,omitempty"`
}

// AnthropicRequestMessage represents a message in the Anthropic request
type AnthropicRequestMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

// AnthropicSystemMessage represents a system message block
type AnthropicSystemMessage struct {
	Type         string          `json:"type"`
	Text         string          `json:"text,omitempty"`
	CacheControl json.RawMessage `json:"cache_control,omitempty"`
}

// ContentBlock represents a content block in a message
type ContentBlock struct {
	Type      string       `json:"type"`
	Text      string       `json:"text,omitempty"`
	Source    *ImageSource `json:"source,omitempty"`
	ID        string       `json:"id,omitempty"`
	Name      string       `json:"name,omitempty"`
	Input     any          `json:"input,omitempty"`
	ToolUseID string       `json:"tool_use_id,omitempty"`
	Content   any          `json:"content,omitempty"`
	IsError   *bool        `json:"is_error,omitempty"`
}

// ImageSource represents an image source in a content block
type ImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// AnthropicTool represents a tool definition in the Anthropic format
type AnthropicTool struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	InputSchema any    `json:"input_schema"`
}

// ToolChoice represents tool choice configuration
type ToolChoice struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

// --- CodeWhisperer Request Types ---

// CodeWhispererRequest represents the request to CodeWhisperer API
type CodeWhispererRequest struct {
	ConversationState    ConversationState `json:"conversationState"`
	ProfileArn           string            `json:"profileArn,omitempty"`
	InferenceConfig      *InferenceConfig  `json:"inferenceConfig,omitempty"`
}

// InferenceConfig holds inference parameters for the request
type InferenceConfig struct {
	MaxTokens   int     `json:"maxTokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	TopP        float64 `json:"topP,omitempty"`
}

// ConversationState holds the conversation state for CodeWhisperer
type ConversationState struct {
	ChatTriggerType       string              `json:"chatTriggerType"`
	ConversationID        string              `json:"conversationId,omitempty"`
	CurrentMessage        CurrentMessage      `json:"currentMessage"`
	History               []any               `json:"history,omitempty"`
	AgentContinuationId   string              `json:"agentContinuationId,omitempty"`
	AgentTaskType         string              `json:"agentTaskType,omitempty"`
}

// CurrentMessage represents the current message in the conversation
type CurrentMessage struct {
	UserInputMessage UserInputMessage `json:"userInputMessage"`
}

// UserInputMessage represents the user's input message
type UserInputMessage struct {
	Content                string              `json:"content"`
	ModelID                string              `json:"modelId,omitempty"`
	Origin                 string              `json:"origin"`
	UserInputMessageContext *UserInputContext   `json:"userInputMessageContext,omitempty"`
	Images                 []CodeWhispererImage `json:"images,omitempty"`
}

// UserInputContext holds context for the user input
type UserInputContext struct {
	Tools       []CodeWhispererTool `json:"tools,omitempty"`
	ToolResults []ToolResult        `json:"toolResults,omitempty"`
}

// CodeWhispererImage represents an image in CodeWhisperer format
type CodeWhispererImage struct {
	Format string              `json:"format"`
	Source CodeWhispererImageSource `json:"source"`
}

// CodeWhispererImageSource holds the bytes data for an image
type CodeWhispererImageSource struct {
	Bytes string `json:"bytes"`
}

// CodeWhispererTool represents a tool in CodeWhisperer format
type CodeWhispererTool struct {
	ToolSpec ToolSpecification `json:"toolSpec"`
}

// ToolSpecification represents a tool specification
type ToolSpecification struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

// InputSchema represents the input schema for a tool
type InputSchema struct {
	JSON any `json:"json"`
}

// --- History Types ---

// HistoryUserMessage represents a user message in history
type HistoryUserMessage struct {
	UserInputMessage UserInputMessage `json:"userInputMessage"`
}

// HistoryAssistantMessage represents an assistant message in history
type HistoryAssistantMessage struct {
	AssistantResponseMessage AssistantResponseMessage `json:"assistantResponseMessage"`
}

// AssistantResponseMessage represents the assistant's response
type AssistantResponseMessage struct {
	MessageID string `json:"messageId,omitempty"`
	Content   string `json:"content"`
	ToolUses  []ToolUseEntry `json:"toolUses,omitempty"`
}

// ToolResult represents a tool result
type ToolResult struct {
	ToolUseID string `json:"toolUseId"`
	Content   []ToolResultContent `json:"content"`
	Status    string `json:"status"`
}

// ToolResultContent represents content within a tool result
type ToolResultContent struct {
	Text string `json:"text,omitempty"`
}

// ToolUseEntry represents a tool use in the assistant response
type ToolUseEntry struct {
	ToolUseID string `json:"toolUseId"`
	Name      string `json:"name"`
	Input     any    `json:"input"`
}

// --- Auth Types ---

// RefreshRequest represents a Social token refresh request
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// RefreshResponse represents a Social token refresh response
type RefreshResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int    `json:"expiresIn"`
}

// IdcRefreshRequest represents an IdC token refresh request
type IdcRefreshRequest struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	GrantType    string `json:"grantType"`
	RefreshToken string `json:"refreshToken"`
}

// IdcRefreshResponse represents an IdC token refresh response
type IdcRefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// SSEEvent represents a Server-Sent Event to send to the client
type SSEEvent struct {
	Event string
	Data  string
}

// ToolUseEvent represents a tool use event from CodeWhisperer
type ToolUseEvent struct {
	ToolUseID string `json:"toolUseId,omitempty"`
	Name      string `json:"name,omitempty"`
	Input     any    `json:"input,omitempty"`
}

// UsageEvent represents token usage information
type UsageEvent struct {
	InputTokens  int `json:"inputTokens,omitempty"`
	OutputTokens int `json:"outputTokens,omitempty"`
}

// FullAssistantResponseEvent represents the full response event from CodeWhisperer
type FullAssistantResponseEvent struct {
	AssistantResponseEvent *AssistantResponseEvent `json:"assistantResponseEvent,omitempty"`
	SupplementaryWebLinksEvent *json.RawMessage   `json:"supplementaryWebLinksEvent,omitempty"`
}

// AssistantResponseEvent represents an assistant response event
type AssistantResponseEvent struct {
	Content              string          `json:"content,omitempty"`
	MessageID            string          `json:"messageId,omitempty"`
	FollowupPrompt       *FollowupPrompt `json:"followupPrompt,omitempty"`
	ToolUse              *ToolUseEvent   `json:"toolUse,omitempty"`
	Usage                *UsageEvent     `json:"usage,omitempty"`
	StopReason           string          `json:"stopReason,omitempty"`
}

// FollowupPrompt represents a followup prompt suggestion
type FollowupPrompt struct {
	Content  string `json:"content,omitempty"`
	UserIntent string `json:"userIntent,omitempty"`
}

// --- Tool Execution Types ---

// ToolExecution represents a tool execution event
type ToolExecution struct {
	ToolUseID string `json:"toolUseId"`
	Name      string `json:"name"`
	Input     string `json:"input"`
}

// ToolCall represents a tool call being built
type ToolCall struct {
	ID    string
	Name  string
	Input string
}

// ToolCallRequest represents a tool call request event from CodeWhisperer
type ToolCallRequest struct {
	ToolUseID string `json:"toolUseId"`
	Name      string `json:"name"`
	Input     any    `json:"input"`
}

// SessionInfo holds session information from CodeWhisperer
type SessionInfo struct {
	SessionID string
	MessageID string
}

// ParseResult holds the result of parsing an EventStream
type ParseResult struct {
	Content      string
	ToolCalls    []ToolCall
	StopReason   string
	InputTokens  int
	OutputTokens int
	MessageID    string
}

// StreamResult holds the result of stream processing
type StreamResult struct {
	Content         string
	StopReason      string
	InputTokens     int
	OutputTokens    int
	Model           string
	ToolCalls       []ToolCall
	ThinkingContent string
}

// --- CodeWhisperer REST API Response Types ---

// UsageLimitsResponse represents the response from GetUsageLimits API
type UsageLimitsResponse struct {
	UsageBreakdownList []UsageBreakdown  `json:"usageBreakdownList"`
	NextDateReset      json.Number       `json:"nextDateReset"`
	SubscriptionInfo   *SubscriptionInfo `json:"subscriptionInfo"`
	UserInfo           *KiroUserInfo     `json:"userInfo"`
}

// UsageBreakdown represents a single usage breakdown entry
type UsageBreakdown struct {
	ResourceType  string         `json:"resourceType"`
	CurrentUsage  float64        `json:"currentUsage"`
	UsageLimit    float64        `json:"usageLimit"`
	Currency      string         `json:"currency"`
	Unit          string         `json:"unit"`
	OverageRate   float64        `json:"overageRate"`
	FreeTrialInfo *FreeTrialInfo `json:"freeTrialInfo"`
	Bonuses       []BonusInfo    `json:"bonuses"`
}

// SubscriptionInfo represents subscription details
type SubscriptionInfo struct {
	SubscriptionName  string `json:"subscriptionName"`
	SubscriptionTitle string `json:"subscriptionTitle"`
	SubscriptionType  string `json:"subscriptionType"`
	Status            string `json:"status"`
	UpgradeCapability string `json:"upgradeCapability"`
}

// FreeTrialInfo represents free trial details
type FreeTrialInfo struct {
	CurrentUsage    float64     `json:"currentUsage"`
	UsageLimit      float64     `json:"usageLimit"`
	FreeTrialStatus string      `json:"freeTrialStatus"`
	FreeTrialExpiry json.Number `json:"freeTrialExpiry"`
}

// BonusInfo represents bonus usage details
type BonusInfo struct {
	BonusCode    string      `json:"bonusCode"`
	DisplayName  string      `json:"displayName"`
	CurrentUsage float64     `json:"currentUsage"`
	UsageLimit   float64     `json:"usageLimit"`
	ExpiresAt    json.Number `json:"expiresAt"`
	Status       string      `json:"status"`
}

// KiroUserInfo represents user info from the usage API
type KiroUserInfo struct {
	Email  string `json:"email"`
	UserId string `json:"userId"`
}

// ModelInfo represents a model returned by ListAvailableModels
type ModelInfo struct {
	ModelId        string   `json:"modelId"`
	ModelName      string   `json:"modelName"`
	Description    string   `json:"description"`
	InputTypes     []string `json:"supportedInputTypes"`
	RateMultiplier float64  `json:"rateMultiplier"`
	TokenLimits    *struct {
		MaxInputTokens  int `json:"maxInputTokens"`
		MaxOutputTokens int `json:"maxOutputTokens"`
	} `json:"tokenLimits"`
}
