package llm

import (
	"fmt"

	"github.com/bakkerme/ai-news-processor/internal/customerrors"
	"github.com/bakkerme/ai-news-processor/internal/openai"
	"github.com/bakkerme/ai-news-processor/internal/persona"
	"github.com/bakkerme/ai-news-processor/internal/prompts"
	"github.com/bakkerme/ai-news-processor/internal/rss"
	"github.com/bakkerme/ai-news-processor/models"
)

// GenerateSummary creates a summary for a set of relevant RSS entries
func GenerateSummary(client openai.OpenAIClient, entries []rss.Entry, p persona.Persona) (*models.SummaryResponse, error) {
	fmt.Println("Generating summary of relevant items")

	// Create input for summary
	summaryInputs := make([]string, len(entries))
	for i, entry := range entries {
		summaryInputs[i] = entry.String(true)
	}

	summaryChannel := make(chan customerrors.ErrorString, 1)
	summaryPrompt, err := prompts.ComposeSummaryPrompt(p)
	if err != nil {
		return nil, fmt.Errorf("could not compose summary prompt for persona %s: %w", p.Name, err)
	}

	go chatCompletionForFeedSummary(client, summaryPrompt, summaryInputs, summaryChannel)

	summaryResult := <-summaryChannel
	if summaryResult.Err != nil {
		return nil, fmt.Errorf("could not generate summary: %w", summaryResult.Err)
	}

	processedSummary := client.PreprocessJSON(summaryResult.Value)
	summary, err := models.UnmarshalSummaryResponseJSON([]byte(processedSummary))
	if err != nil {
		return nil, fmt.Errorf("could not parse summary response: %w", err)
	}

	return summary, nil
}
