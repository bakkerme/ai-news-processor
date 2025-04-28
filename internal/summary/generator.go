package summary

import (
	"fmt"

	"github.com/bakkerme/ai-news-processor/internal/common"
	"github.com/bakkerme/ai-news-processor/internal/openai"
	"github.com/bakkerme/ai-news-processor/internal/prompts"
	"github.com/bakkerme/ai-news-processor/internal/rss"
)

// Generate creates a summary for a set of relevant RSS entries
func Generate(client *openai.Client, entries []rss.Entry, persona common.Persona) (*common.SummaryResponse, error) {
	fmt.Println("Generating summary of relevant items")

	// Create input for summary
	summaryInputs := make([]string, len(entries))
	for i, entry := range entries {
		summaryInputs[i] = entry.String(false)
	}

	summaryChannel := make(chan common.ErrorString, 1)
	summaryPrompt, err := prompts.ComposeSummaryPrompt(persona)
	if err != nil {
		return nil, fmt.Errorf("could not compose summary prompt for persona %s: %w", persona.Name, err)
	}

	go client.QueryForFeedSummary(summaryPrompt, summaryInputs, summaryChannel)

	summaryResult := <-summaryChannel
	if summaryResult.Err != nil {
		return nil, fmt.Errorf("could not generate summary: %w", summaryResult.Err)
	}

	processedSummary := client.PreprocessJSON(summaryResult.Value)
	summary, err := client.ParseSummaryResponse(processedSummary)
	if err != nil {
		return nil, fmt.Errorf("could not parse summary response: %w", err)
	}

	return summary, nil
}
