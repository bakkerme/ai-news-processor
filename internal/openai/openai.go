package openai

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bakkerme/ai-news-processor/internal/customerrors"
	"github.com/bakkerme/ai-news-processor/internal/http/retry"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// OpenAIClient defines the interface for interacting with an OpenAI-compatible API
type OpenAIClient interface {
	// ChatCompletion performs a general-purpose chat completion request
	// systemPrompt: The system prompt to use
	// userPrompts: A list of user messages to send
	// imageURLs: Optional list of image URLs to include in the prompt
	// schema: Optional JSON schema for response formatting (can be nil)
	// schemaName: Name for the schema when provided
	// schemaDescription: Description for the schema when provided
	// returns: Channel that will receive the response or error
	ChatCompletion(
		systemPrompt string,
		userPrompts []string,
		imageURLs []string,
		schema interface{},
		schemaName string,
		schemaDescription string,
		results chan customerrors.ErrorString,
	)

	// SetRetryConfig updates the retry behavior configuration
	SetRetryConfig(config retry.RetryConfig)

	// PreprocessJSON cleans up JSON responses from the API
	PreprocessJSON(response string) string
}

// DefaultOpenAIRetryConfig provides sensible default values for OpenAI retry behavior
var DefaultOpenAIRetryConfig = retry.RetryConfig{
	MaxRetries:      5,
	InitialBackoff:  1 * time.Second,
	MaxBackoff:      30 * time.Second,
	BackoffFactor:   2.0,
	MaxTotalTimeout: 30 * time.Minute, // LLM calls can take a while
}

type Client struct {
	client *openai.Client
	model  string
	retry  retry.RetryConfig
}

// New creates a new OpenAI client
func New(baseURL, key, model string) *Client {
	client := openai.NewClient(
		option.WithAPIKey(key),
		option.WithBaseURL(baseURL),
		option.WithJSONSet("cache_set", true),
	)
	return &Client{
		client: &client,
		model:  model,
		retry:  DefaultOpenAIRetryConfig,
	}
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

// ChatCompletion sends a request to the OpenAI API with the given prompts, optional images, and schema
func (c *Client) ChatCompletion(
	systemPrompt string,
	userPrompts []string,
	imageURLs []string,
	schema interface{},
	schemaName string,
	schemaDescription string,
	results chan customerrors.ErrorString,
) {
	// Prepare messages array
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPrompt),
	}

	// If we have image URLs, create a message with multi-modal content
	if len(imageURLs) > 0 {
		// Build image content parts
		contentParts := []openai.ChatCompletionContentPartUnionParam{}

		// First, add a text part if we have userPrompts
		if len(userPrompts) > 0 {
			textPart := openai.TextContentPart(userPrompts[0]) // First prompt as the text part
			contentParts = append(contentParts, textPart)
		}

		// Then add all the image parts
		for _, imgURL := range imageURLs {
			if imgURL != "" { // Basic validation
				imageParam := openai.ChatCompletionContentPartImageImageURLParam{
					URL: imgURL,
					// Optional: Detail: openai.String("auto"), // Can be "low", "high", or "auto"
				}
				imagePart := openai.ImageContentPart(imageParam)
				contentParts = append(contentParts, imagePart)
			}
		}

		// Create a user message with the multi-modal content parts
		messages = append(messages, openai.UserMessage(contentParts))

		// If there are additional prompts (beyond the first one), add them separately
		if len(userPrompts) > 1 {
			// Join the remaining prompts and add as a separate message
			messages = append(messages, openai.UserMessage(strings.Join(userPrompts[1:], "\n")))
		}
	} else {
		// No images, just add text prompts as usual
		messages = append(messages, openai.UserMessage(strings.Join(userPrompts, "\n")))
	}

	params := openai.ChatCompletionNewParams{
		Model:    c.model,
		Messages: messages,
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

	shouldRetry := func(err error) bool {
		return isModelLoadingError(err)
	}

	ChatCompletionFn := func(ctx context.Context) (*openai.ChatCompletion, error) {
		return c.client.Chat.Completions.New(ctx, params)
	}

	resp, err := retry.RetryWithBackoff(context.Background(), c.retry, ChatCompletionFn, shouldRetry)

	if err != nil {
		var errMsg string
		if isModelLoadingError(err) {
			errMsg = fmt.Sprintf("model failed to load after retries: %v", err)
		} else {
			errMsg = fmt.Sprintf("error during API call: %v", err)
		}

		results <- customerrors.ErrorString{
			Value: "",
			Err:   errors.New(errMsg),
		}
		return
	}

	if len(resp.Choices) == 0 {
		results <- customerrors.ErrorString{
			Value: "",
			Err:   fmt.Errorf("empty response from llm"),
		}
		return
	}

	results <- customerrors.ErrorString{
		Value: resp.Choices[0].Message.Content,
		Err:   nil,
	}
}

// PreprocessJSON extracts JSON content from the API response
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

// SetRetryConfig updates the retry configuration
func (c *Client) SetRetryConfig(config retry.RetryConfig) {
	c.retry = config
}
