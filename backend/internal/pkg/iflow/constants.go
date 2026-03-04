package iflow

// DefaultBaseURL is the default base URL for the iFlow API.
const DefaultBaseURL = "https://apis.iflow.cn/v1"

// OAuthBaseURL is the iFlow OAuth base URL.
const OAuthBaseURL = "https://iflow.cn"

// DefaultModels is the list of supported iFlow models.
var DefaultModels = []string{
	"tstars2.0",
	"qwen3-coder-plus",
	"qwen3-max",
	"qwen3-vl-plus",
	"qwen3-max-preview",
	"kimi-k2-0905",
	"kimi-k2",
	"kimi-k2-thinking",
	"kimi-k2.5",
	"deepseek-v3",
	"deepseek-v3.1",
	"deepseek-v3.2",
	"deepseek-r1",
	"minimax-m2",
	"minimax-m2.1",
	"minimax-m2.5",
	"glm-4.6",
	"glm-4.7",
	"glm-5",
	"iflow-rome",
}

// DefaultModelIDs returns the default model ID list.
func DefaultModelIDs() []string {
	return DefaultModels
}
