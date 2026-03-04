// Package deepseek provides constants and helpers for DeepSeek API integration.
package deepseek

// DefaultBaseURL is the default base URL for the DeepSeek API.
const DefaultBaseURL = "https://api.deepseek.com"

// DefaultModels is the list of supported DeepSeek models.
var DefaultModels = []string{
	"deepseek-chat",
	"deepseek-coder",
	"deepseek-reasoner",
	"deepseek-v3",
	"deepseek-v3-0324",
	"deepseek-r1",
	"deepseek-r1-0528",
}

// DefaultModelIDs returns the default model ID list.
func DefaultModelIDs() []string {
	ids := make([]string, len(DefaultModels))
	copy(ids, DefaultModels)
	return ids
}
