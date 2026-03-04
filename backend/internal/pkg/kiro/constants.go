package kiro

// KiroVersion is the Kiro IDE version to use in User-Agent headers
const KiroVersion = "0.8.0"

// KiroEndpoint represents a CodeWhisperer/AmazonQ endpoint configuration
type KiroEndpoint struct {
	URL       string
	Origin    string
	AmzTarget string
	Name      string
}

// KiroEndpoints defines the dual endpoints with automatic failover
var KiroEndpoints = []KiroEndpoint{
	{
		URL:       "https://codewhisperer.us-east-1.amazonaws.com/generateAssistantResponse",
		Origin:    "AI_EDITOR",
		AmzTarget: "AmazonCodeWhispererStreamingService.GenerateAssistantResponse",
		Name:      "CodeWhisperer",
	},
	{
		URL:       "https://q.us-east-1.amazonaws.com/generateAssistantResponse",
		Origin:    "CLI",
		AmzTarget: "AmazonQDeveloperStreamingService.SendMessage",
		Name:      "AmazonQ",
	},
}

// CodeWhisperer API URL (primary, kept for backward compatibility)
const CodeWhispererURL = "https://codewhisperer.us-east-1.amazonaws.com/generateAssistantResponse"

// CodeWhisperer REST API base URL (for usage/models queries)
const CodeWhispererRestAPIBase = "https://codewhisperer.us-east-1.amazonaws.com"

// Auth types
const (
	AuthTypeSocial = "Social"
	AuthTypeIdC    = "IdC"
)

// ModelMap maps Anthropic model names to CodeWhisperer/Kiro model IDs
var ModelMap = map[string]string{
	// Opus 4.6
	"claude-opus-4-6":   "claude-opus-4.6",
	"claude-opus-4.6":   "claude-opus-4.6",
	// Opus 4.5
	"claude-opus-4-5":          "claude-opus-4.5",
	"claude-opus-4-5-20251101": "claude-opus-4.5",
	"claude-opus-4.5":          "claude-opus-4.5",
	// Sonnet 4.6
	"claude-sonnet-4-6": "claude-sonnet-4.6",
	"claude-sonnet-4.6": "claude-sonnet-4.6",
	// Sonnet 4.5
	"claude-sonnet-4-5":          "claude-sonnet-4.5",
	"claude-sonnet-4-5-20250929": "claude-sonnet-4.5",
	"claude-sonnet-4.5":          "claude-sonnet-4.5",
	// Sonnet 4
	"claude-sonnet-4-20250514": "claude-sonnet-4",
	"claude-sonnet-4":          "claude-sonnet-4",
	// Sonnet 3.7
	"claude-3-7-sonnet-20250219": "CLAUDE_3_7_SONNET_20250219_V1_0",
	// Haiku 4.5
	"claude-3-5-haiku-20241022": "auto",
	"claude-haiku-4-5-20251001": "auto",
	"claude-haiku-4-5":          "auto",
	"claude-haiku-4.5":          "auto",
	// Legacy model aliases (map to closest supported model)
	"claude-3-5-sonnet":  "claude-sonnet-4.5",
	"claude-3-opus":      "claude-sonnet-4.5",
	"claude-3-sonnet":    "claude-sonnet-4",
	"claude-3-haiku":     "auto",
	// GPT aliases
	"gpt-4":         "claude-sonnet-4.5",
	"gpt-4o":        "claude-sonnet-4.5",
	"gpt-4-turbo":   "claude-sonnet-4.5",
	"gpt-3.5-turbo": "claude-sonnet-4.5",
}

// DefaultModels is the list of canonical models shown in the UI.
// Date-suffix variants are still handled by ModelMap for API requests.
var DefaultModels = []string{
	"claude-opus-4-6",
	"claude-opus-4-5",
	"claude-sonnet-4-6",
	"claude-sonnet-4-5",
	"claude-sonnet-4",
	"claude-3-7-sonnet-20250219",
	"claude-haiku-4-5",
}

// DefaultTestModel is the default model for testing connections
const DefaultTestModel = "claude-haiku-4-5-20251001"

// CanonicalModelName maps model name variants to a canonical (display) name.
// Used by usage stats to group the same underlying model together.
var CanonicalModelName = map[string]string{
	"claude-opus-4-6":            "claude-opus-4-6",
	"claude-opus-4.6":            "claude-opus-4-6",
	"claude-opus-4-5":            "claude-opus-4-5",
	"claude-opus-4-5-20251101":   "claude-opus-4-5",
	"claude-opus-4.5":            "claude-opus-4-5",
	"claude-sonnet-4-6":          "claude-sonnet-4-6",
	"claude-sonnet-4.6":          "claude-sonnet-4-6",
	"claude-sonnet-4-5":          "claude-sonnet-4-5",
	"claude-sonnet-4-5-20250929": "claude-sonnet-4-5",
	"claude-sonnet-4.5":          "claude-sonnet-4-5",
	"claude-sonnet-4":            "claude-sonnet-4",
	"claude-sonnet-4-20250514":   "claude-sonnet-4",
	"claude-3-7-sonnet-20250219": "claude-3-7-sonnet",
	"claude-haiku-4-5":           "claude-haiku-4-5",
	"claude-haiku-4-5-20251001":  "claude-haiku-4-5",
	"claude-haiku-4.5":           "claude-haiku-4-5",
	"claude-3-5-haiku-20241022":  "claude-haiku-4-5",
}

// DisplayModelName maps canonical model names to human-friendly labels.
var DisplayModelName = map[string]string{
	"claude-opus-4-6":   "Opus 4.6",
	"claude-opus-4-5":   "Opus 4.5",
	"claude-sonnet-4-6": "Sonnet 4.6",
	"claude-sonnet-4-5": "Sonnet 4.5",
	"claude-sonnet-4":   "Sonnet 4",
	"claude-3-7-sonnet": "Sonnet 3.7",
	"claude-haiku-4-5":  "Haiku 4.5",
}

// MaxToolDescriptionLength is the max length for tool descriptions
const MaxToolDescriptionLength = 10000

// MaxToolNameLength is the max length for tool names (CodeWhisperer limit)
const MaxToolNameLength = 64

// EventStream constants
const (
	EventStreamMinMessageSize = 16
	EventStreamMaxMessageSize = 16 * 1024 * 1024
	ParserMaxErrors           = 50
)
