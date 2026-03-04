// Package glm provides constants and helpers for GLM (智谱AI/ChatGLM) API integration.
package glm

// DefaultBaseURL is the default base URL for the GLM API.
const DefaultBaseURL = "https://open.bigmodel.cn/api/paas"

// DefaultModels is the list of supported GLM models.
var DefaultModels = []string{
	"glm-4",
	"glm-4-plus",
	"glm-4-flash",
	"glm-4v",
	"glm-4-air",
	"glm-4-airx",
	"glm-4-long",
}

// ImageGenerationModels are GLM models that support image generation.
var ImageGenerationModels = []string{
	"cogview-4",
	"cogview-4-250304",
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
