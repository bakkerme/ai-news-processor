package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bakkerme/ai-news-processor/internal/common"
	"github.com/invopop/jsonschema"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// RetryConfig holds configuration for retry behavior
type RetryConfig struct {
	MaxRetries      int
	InitialBackoff  time.Duration
	MaxBackoff      time.Duration
	BackoffFactor   float64
	MaxTotalTimeout time.Duration
}

// DefaultRetryConfig provides sensible default values for retry behavior
var DefaultRetryConfig = RetryConfig{
	MaxRetries:      5,
	InitialBackoff:  1 * time.Second,
	MaxBackoff:      30 * time.Second,
	BackoffFactor:   2.0,
	MaxTotalTimeout: 2 * time.Minute,
}

type Client struct {
	client *openai.Client
	model  string
	retry  RetryConfig
}

func New(baseURL, key, model string) *Client {
	client := openai.NewClient(
		option.WithAPIKey(key),
		option.WithBaseURL(baseURL),
		option.WithJSONSet("cache_set", true),
	)
	return &Client{
		client: &client,
		model:  model,
		retry:  DefaultRetryConfig,
	}
}

// Generate the JSON schema at initialization time
var ItemResponseSchema = GenerateSchema[[]common.Item]()
var SummaryResponseSchema = GenerateSchema[common.SummaryResponse]()

func (c *Client) QueryForEntrySummary(systemPrompt string, userPrompts []string, results chan common.ErrorString) {
	c.Query(
		systemPrompt,
		userPrompts,
		ItemResponseSchema,
		"post_item",
		"an object representing a post",
		results,
	)
}

func (c *Client) QueryForFeedSummary(systemPrompt string, userPrompts []string, results chan common.ErrorString) {
	c.Query(
		systemPrompt,
		userPrompts,
		SummaryResponseSchema,
		"summary",
		"a summary of multiple AI news items",
		results,
	)
}

// isModelLoadingError checks if the error is specifically a 404 due to model loading
func isModelLoadingError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "404 Not Found") &&
		strings.Contains(errStr, "Failed to load model") &&
		strings.Contains(errStr, "Model does not exist")
}

func (c *Client) Query(
	systemPrompt string,
	userPrompts []string,
	schema interface{},
	schemaName string,
	schemaDescription string,
	results chan common.ErrorString,
) {
	params := openai.ChatCompletionNewParams{
		Model: c.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(strings.Join(userPrompts, "\n")),
		},
	}

	if schema != nil {
		schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
			Name:        schemaName,
			Description: openai.String(schemaDescription),
			Schema:      schema,
			Strict:      openai.Bool(true),
		}
		params.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{JSONSchema: schemaParam},
		}
	}

	var resp *openai.ChatCompletion
	var lastErr error
	currentBackoff := c.retry.InitialBackoff
	startTime := time.Now()

	for attempt := 0; attempt <= c.retry.MaxRetries; attempt++ {
		// Create a new context for each attempt
		ctx := context.Background()
		resp, lastErr = c.client.Chat.Completions.New(ctx, params)

		if lastErr == nil {
			break
		}

		// Only retry on specific model loading errors
		if isModelLoadingError(lastErr) {
			if attempt == c.retry.MaxRetries {
				break
			}

			// Check if we've exceeded total timeout
			if time.Since(startTime) > c.retry.MaxTotalTimeout {
				lastErr = fmt.Errorf("exceeded maximum total timeout of %v waiting for model to load: %w",
					c.retry.MaxTotalTimeout, lastErr)
				break
			}

			// Wait with exponential backoff
			time.Sleep(currentBackoff)

			// Calculate next backoff
			currentBackoff = time.Duration(float64(currentBackoff) * c.retry.BackoffFactor)
			if currentBackoff > c.retry.MaxBackoff {
				currentBackoff = c.retry.MaxBackoff
			}
			continue
		} else {
			// If it's not a model loading error, don't retry
			break
		}
	}

	if lastErr != nil {
		var errMsg string
		if isModelLoadingError(lastErr) {
			errMsg = fmt.Sprintf("model failed to load after %d retries over %v: %w",
				c.retry.MaxRetries, time.Since(startTime), lastErr)
		} else {
			errMsg = fmt.Sprintf("error during API call: %w", lastErr)
		}

		results <- common.ErrorString{
			Value: "",
			Err:   errors.New(errMsg),
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
