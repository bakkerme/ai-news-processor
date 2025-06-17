package prompts

import (
	"github.com/bakkerme/ai-news-processor/models"
)

// GetRealItemJSONExample generates a JSON example using the actual models.Item struct
func GetRealItemJSONExample() (string, error) {
	generator := &JSONExampleGenerator{}
	return generator.GenerateJSONExampleCompact(models.ItemSubset{})
}

// GetRealSummaryResponseJSONExample generates a JSON example using the actual models.SummaryResponse struct
func GetRealSummaryResponseJSONExample() (string, error) {
	generator := &JSONExampleGenerator{}
	return generator.GenerateJSONExampleCompact(models.SummaryResponse{})
}

// GetRealKeyDevelopmentJSONExample generates a JSON example using the actual models.KeyDevelopment struct
func GetRealKeyDevelopmentJSONExample() (string, error) {
	generator := &JSONExampleGenerator{}
	return generator.GenerateJSONExampleCompact(models.KeyDevelopment{})
}
