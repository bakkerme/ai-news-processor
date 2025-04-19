package openai

import (
	"context"
	"github.com/bakkerme/ai-news-processor/common"
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
	return &OpenAIClient{client: &client}
}

func (c *OpenAIClient) Query(systemPrompt string, userPrompts []string, results chan common.ErrorString) {
	userPrompt := strings.Join(userPrompts, "\n")

	resp, err := c.client.Chat.Completions.New(
		context.Background(),
		openai.ChatCompletionNewParams{
			Model: c.model,
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.DeveloperMessage(systemPrompt),
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
	response = strings.ReplaceAll(response, "```json", " ")
	response = strings.ReplaceAll(response, "```", " ")
	return strings.Join(strings.Fields(response), " ")
}
