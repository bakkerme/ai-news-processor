package prompts

import (
	"github.com/bakkerme/ai-news-processor/models"
)

// GetRealItemJSONExample generates a JSON example using the actual models.Item struct
func GetRealItemJSONExample() (string, error) {
	generator := &JSONExampleGenerator{}
	return generator.GenerateJSONExampleCompact(models.ItemSubset{})
}

// GetRealItemJSONExampleForPersona generates a JSON example tailored to persona configuration
func GetRealItemJSONExampleForPersona(includeCommentSummary bool) (string, error) {
	generator := &JSONExampleGenerator{}

	// Create allowlist based on persona configuration
	allowlist := map[string]bool{
		"id":                  true,
		"title":               true,
		"overview":            true,
		"summary":             true,
		"relevanceToCriteria": true,
		"isRelevant":          true,
	}

	// Conditionally include comment summary
	if includeCommentSummary {
		allowlist["commentSummary"] = true
	}

	return generator.GenerateJSONExampleCompactWithAllowlist(models.ItemSubset{}, allowlist)
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
