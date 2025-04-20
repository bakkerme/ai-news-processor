package openai

import (
	"context"
	"github.com/bakkerme/ai-news-processor/internal/common"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"strings"
)

type OpenAIClient struct {
	client *openai.Client
	model  string
}

func NewOpenAIClient(baseURL, key, model string) *OpenAIClient {
	client := openai.NewClient(
		option.WithAPIKey(key),
		option.WithBaseURL(baseURL),
		option.WithJSONSet("cache_set", true),
	)
	return &OpenAIClient{client: &client, model: model}
}

func (c *OpenAIClient) Query(systemPrompt string, userPrompts []string, results chan common.ErrorString) {
	userPrompt := strings.Join(userPrompts, "\n")

	resp, err := c.client.Chat.Completions.New(
		context.Background(),
		openai.ChatCompletionNewParams{
			Model: c.model,
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage(systemPrompt),
				openai.UserMessage(userPrompt),
			},
		},
	)

	if err != nil {
		results <- common.ErrorString{
			Value: "",
			Err:   err,
		}
		return
	}

	results <- common.ErrorString{
		Value: resp.Choices[0].Message.Content,
		Err:   nil,
	}
}

func (c *OpenAIClient) PreprocessJSON(response string) string {
	// Find the start and end markers
	startMarker := "```json"
	endMarker := "```"

	startIdx := strings.Index(response, startMarker)
	if startIdx == -1 {
		// If no start marker found, return the original string trimmed
		return strings.TrimSpace(response)
	}

	// Adjust start index to be after the marker
	startIdx += len(startMarker)

	endIdx := strings.Index(response[startIdx:], endMarker)
	if endIdx == -1 {
		// If no end marker found, return from start marker to end
		return strings.TrimSpace(response[startIdx:])
	}

	// Extract the content between markers
	jsonContent := response[startIdx : startIdx+endIdx]
	return strings.TrimSpace(jsonContent)
}
