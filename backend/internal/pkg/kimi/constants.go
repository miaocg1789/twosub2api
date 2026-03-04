// Package kimi provides constants and helpers for Kimi API integration.
package kimi

// DefaultBaseURL is the default base URL for the Kimi API.
const DefaultBaseURL = "https://api.kimi.com/coding/v1"

// OAuthBaseURL is the base URL for Kimi OAuth.
const OAuthBaseURL = "https://auth.kimi.com"

// DefaultModels is the list of supported Kimi models (OAuth).
var DefaultModels = []string{
	"kimi-k2",
	"kimi-k2-thinking",
	"kimi-k2.5",
}

// DefaultModelIDs returns the default model ID list.
func DefaultModelIDs() []string {
	return DefaultModels
}
