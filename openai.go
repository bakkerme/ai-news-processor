package main

import (
	"context"
	"fmt"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"os"
)

type OpenAIClient struct {
	client *openai.Client
	model  string
}

func NewOpenAIClient(baseURL, model string) *OpenAIClient {
	client := openai.NewClient(
		option.WithAPIKey(os.Getenv("FEATHERLESS_API_KEY")),
		option.WithBaseURL(baseURL),
		option.WithJSONSet("cache_set", true),
	)
	return &OpenAIClient{client: &client}
}

func (c *OpenAIClient) Query(systemPrompt, userPrompt string, results chan string) {
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
		results <- fmt.Sprintf("Error: %v", err)
		return
	}

	results <- resp.Choices[0].Message.Content
}
