package llm

import (
	"time"

	"github.com/bakkerme/ai-news-processor/internal/customerrors"
	"github.com/bakkerme/ai-news-processor/internal/models"
	"github.com/bakkerme/ai-news-processor/internal/openai"
	"github.com/invopop/jsonschema"
)

// EntryProcessConfig holds the configuration for processing entries
type EntryProcessConfig struct {
	// Maximum number of retry attempts for failed entry processing
	MaxRetries int
	// Initial backoff duration between retries
	InitialBackoff time.Duration
	// Maximum backoff duration between retries
	MaxBackoff time.Duration
	// Backoff multiplier for each retry attempt
	BackoffFactor float64
}

// DefaultEntryProcessConfig provides default values for entry processing
var DefaultEntryProcessConfig = EntryProcessConfig{
	MaxRetries:     3,
	InitialBackoff: 2 * time.Second,
	MaxBackoff:     30 * time.Second,
	BackoffFactor:  2.0,
}

// Generate the JSON schema at initialization time
var ItemResponseSchema = GenerateSchema[[]models.Item]()
var SummaryResponseSchema = GenerateSchema[models.SummaryResponse]()

// GenerateSchema creates a JSON schema for the given type
func GenerateSchema[T any]() interface{} {
	// Structured Outputs uses a subset of JSON schema
	// These flags are necessary to comply with the subset
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)
	return schema
}

// chatCompletionForEntrySummary sends a ChatCompletion to get summaries for RSS entries
func chatCompletionForEntrySummary(client openai.OpenAIClient, systemPrompt string, userPrompts []string, imageURLs []string, results chan customerrors.ErrorString) {
	// Schema parameters commented for future reference:
	// Schema: ItemResponseSchema
	// Name: "post_item"
	// Description: "an object representing a post"
	client.ChatCompletion(
		systemPrompt,
		userPrompts,
		imageURLs,
		nil, // Schema parameters currently disabled
		results,
	)
}

// chatCompletionForFeedSummary sends a ChatCompletion to get a summary for an entire feed
func chatCompletionForFeedSummary(client openai.OpenAIClient, systemPrompt string, userPrompts []string, results chan customerrors.ErrorString) {
	// Feed summaries don't include images directly
	// Schema parameters commented for future reference:
	// Schema: SummaryResponseSchema
	// Name: "summary"
	// Description: "a summary of multiple AI news items"
	client.ChatCompletion(
		systemPrompt,
		userPrompts,
		[]string{}, // No images for feed summaries
		nil,        // Schema parameters currently disabled
		results,
	)
}

// chatCompletionImageSummary sends a ChatCompletion to get descriptions for images
func chatCompletionImageSummary(client openai.OpenAIClient, systemPrompt string, imageURLs []string) (string, error) {
	results := make(chan customerrors.ErrorString, 1)

	// Empty userPrompt as the image is the content
	// No schema parameters needed for image analysis
	client.ChatCompletion(
		systemPrompt,
		[]string{}, // No additional text prompt, just let the model analyze the images
		imageURLs,
		nil, // Schema parameters not needed for image analysis
		results,
	)

	result := <-results
	close(results)

	if result.Err != nil {
		return "", result.Err
	}

	return result.Value, nil
}
