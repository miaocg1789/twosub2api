// Package qwen provides constants and helpers for Qwen (通义千问) API integration.
package qwen

// DefaultBaseURL is the default base URL for the Qwen API (Alibaba DashScope compatible mode).
const DefaultBaseURL = "https://dashscope.aliyuncs.com/compatible-mode"

// DefaultModels is the list of supported Qwen models (API Key / DashScope).
var DefaultModels = []string{
	"qwen-turbo",
	"qwen-plus",
	"qwen-max",
	"qwen-long",
	"qwen-vl-plus",
	"qwen-vl-max",
	"qwen2.5-72b-instruct",
	"qwen2.5-32b-instruct",
	"qwen3-235b-a22b",
	"qwq-32b",
}

// OAuthModels is the list of supported Qwen models for OAuth accounts (chat.qwen.ai / portal.qwen.ai).
var OAuthModels = []string{
	"qwen3-coder-plus",
	"qwen3-coder-flash",
	"coder-model",
	"vision-model",
}

// OAuthBaseURL is the default base URL for Qwen OAuth accounts.
const OAuthBaseURL = "https://portal.qwen.ai/v1"

// ImageGenerationModels are Qwen models that support image generation via DashScope.
var ImageGenerationModels = []string{
	"wanx-v1",
	"wanx-style",
	"flux-schnell",
	"flux-dev",
	"stable-diffusion-xl",
	"stable-diffusion-3.5-large",
}

// DefaultModelIDs returns the default model ID list (including image generation models).
func DefaultModelIDs() []string {
	ids := make([]string, 0, len(DefaultModels)+len(ImageGenerationModels))
	ids = append(ids, DefaultModels...)
	ids = append(ids, ImageGenerationModels...)
	return ids
}

// IsImageGenerationModel checks if a model is an image generation model.
func IsImageGenerationModel(model string) bool {
	for _, m := range ImageGenerationModels {
		if m == model {
			return true
		}
	}
	return false
}
