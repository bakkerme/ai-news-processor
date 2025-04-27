package openai

import (
	"context"
	"encoding/json"

	// "fmt"
	"fmt"
	"strings"

	"github.com/bakkerme/ai-news-processor/internal/common"
	"github.com/invopop/jsonschema"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type Client struct {
	client *openai.Client
	model  string
}

func New(baseURL, key, model string) *Client {
	client := openai.NewClient(
		option.WithAPIKey(key),
		option.WithBaseURL(baseURL),
		option.WithJSONSet("cache_set", true),
	)
	return &Client{client: &client, model: model}
}

// Generate the JSON schema at initialization time
var ItemResponseSchema = GenerateSchema[[]common.Item]()
var SummaryResponseSchema = GenerateSchema[common.SummaryResponse]()

func (c *Client) QueryForEntrySummary(systemPrompt string, userPrompts []string, results chan common.ErrorString) {
	c.QueryWithSchema(
		systemPrompt,
		userPrompts,
		ItemResponseSchema,
		"post_item",
		"an object representing a post",
		results,
	)
}

func (c *Client) QueryForFeedSummary(systemPrompt string, userPrompts []string, results chan common.ErrorString) {
	c.QueryWithSchema(
		systemPrompt,
		userPrompts,
		SummaryResponseSchema,
		"summary",
		"a summary of multiple AI news items",
		results,
	)
}

func (c *Client) QueryWithSchema(
	systemPrompt string,
	userPrompts []string,
	schema interface{},
	schemaName string,
	schemaDescription string,
	results chan common.ErrorString,
) {
	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        schemaName,
		Description: openai.String(schemaDescription),
		Schema:      schema,
		Strict:      openai.Bool(true),
	}

	userPrompt := strings.Join(userPrompts, "\n")

	resp, err := c.client.Chat.Completions.New(
		context.Background(),
		openai.ChatCompletionNewParams{
			Model: c.model,
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage(systemPrompt),
				openai.UserMessage(userPrompt),
			},
			ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
				OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{JSONSchema: schemaParam},
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

	if len(resp.Choices) == 0 {
		results <- common.ErrorString{
			Value: "",
			Err:   fmt.Errorf("empty response from llm"),
		}
		return
	}

	results <- common.ErrorString{
		Value: resp.Choices[0].Message.Content,
		Err:   nil,
	}
}

func (c *Client) ParseSummaryResponse(jsonStr string) (*common.SummaryResponse, error) {
	var summary common.SummaryResponse
	err := json.Unmarshal([]byte(jsonStr), &summary)
	if err != nil {
		return nil, fmt.Errorf("failed to parse summary response: %w", err)
	}

	return &summary, nil
}

func (c *Client) PreprocessJSON(response string) string {
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
