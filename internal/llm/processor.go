package llm

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bakkerme/ai-news-processor/internal/customerrors"
	"github.com/bakkerme/ai-news-processor/internal/openai"
	"github.com/bakkerme/ai-news-processor/internal/persona"
	"github.com/bakkerme/ai-news-processor/internal/prompts"
	"github.com/bakkerme/ai-news-processor/internal/rss"
	"github.com/invopop/jsonschema"

	"github.com/bakkerme/ai-news-processor/internal/models"
)

// imageResult holds both the result and the index of the entry it belongs to
type imageResult struct {
	result   customerrors.ErrorString
	entryIdx int
}

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

// ChatCompletionForEntrySummary sends a ChatCompletion to get summaries for RSS entries
func ChatCompletionForEntrySummary(client openai.OpenAIClient, systemPrompt string, userPrompts []string, imageURLs []string, results chan customerrors.ErrorString) {
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

// ChatCompletionForFeedSummary sends a ChatCompletion to get a summary for an entire feed
func ChatCompletionForFeedSummary(client openai.OpenAIClient, systemPrompt string, userPrompts []string, results chan customerrors.ErrorString) {
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

// ChatCompletionImageSummary sends a ChatCompletion to get descriptions for images
func ChatCompletionImageSummary(client openai.OpenAIClient, systemPrompt string, imageURLs []string) (string, error) {
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

// ParseSummaryResponse parses a JSON string into a SummaryResponse
func ParseSummaryResponse(jsonStr string) (*models.SummaryResponse, error) {
	var summary models.SummaryResponse
	err := json.Unmarshal([]byte(jsonStr), &summary)
	if err != nil {
		return nil, err
	}
	return &summary, nil
}

// fetchImageAsBase64 fetches an image from a URL and returns it as a base64-encoded data URI
// Returns an empty string if any errors occur
func fetchImageAsBase64(imageURL string) string {
	// Set a timeout for the HTTP client
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Make the HTTP request
	resp, err := client.Get(imageURL)
	if err != nil {
		fmt.Printf("Error fetching image %s: %v\n", imageURL, err)
		return ""
	}
	defer resp.Body.Close()

	// Check if response status is OK
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error fetching image %s: status code %d\n", imageURL, resp.StatusCode)
		return ""
	}

	// Read the image data
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading image data %s: %v\n", imageURL, err)
		return ""
	}

	// Determine content type
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		// Try to guess from URL extension
		if strings.HasSuffix(strings.ToLower(imageURL), ".jpg") || strings.HasSuffix(strings.ToLower(imageURL), ".jpeg") {
			contentType = "image/jpeg"
		} else if strings.HasSuffix(strings.ToLower(imageURL), ".png") {
			contentType = "image/png"
		} else if strings.HasSuffix(strings.ToLower(imageURL), ".gif") {
			contentType = "image/gif"
		} else if strings.HasSuffix(strings.ToLower(imageURL), ".webp") {
			contentType = "image/webp"
		} else {
			contentType = "image/jpeg" // Default assumption
		}
	}

	// Encode the image data as base64
	base64Encoded := base64.StdEncoding.EncodeToString(imageData)

	// Create the data URI
	dataURI := fmt.Sprintf("data:%s;base64,%s", contentType, base64Encoded)

	return dataURI
}

// processEntryWithRetry processes a single entry with retry support
func processEntryWithRetry(client openai.OpenAIClient, systemPrompt string, entry rss.Entry, config EntryProcessConfig) (models.Item, error) {
	entryString := entry.String(true)
	var lastErr error

	// Retry logic with exponential backoff
	backoff := config.InitialBackoff
	for attempt := 0; attempt < config.MaxRetries; attempt++ {
		if attempt > 0 {
			fmt.Printf("Retrying entry (attempt %d/%d) after error: %v\n", attempt+1, config.MaxRetries, lastErr)
			time.Sleep(backoff)
			backoff = time.Duration(float64(backoff) * config.BackoffFactor)
			if backoff > config.MaxBackoff {
				backoff = config.MaxBackoff
			}
		}

		// Process the entry
		results := make(chan customerrors.ErrorString, 1)
		ChatCompletionForEntrySummary(client, systemPrompt, []string{entryString}, nil, results)
		result := <-results
		close(results)

		if result.Err != nil {
			lastErr = fmt.Errorf("could not process value from LLM: %w", result.Err)
			continue // Try again
		}

		processedValue := client.PreprocessJSON(result.Value)

		item, err := llmResponseToItems(processedValue)
		if err != nil {
			lastErr = fmt.Errorf("could not convert llm output to json. %s: %w", processedValue, err)
			continue // Try again
		}

		return item, nil // Success!
	}

	// If we're here, we've exhausted all retry attempts
	return models.Item{}, fmt.Errorf("failed to process entry after %d attempts: %w", config.MaxRetries, lastErr)
}

// processImageWithRetry processes an image with retry support
func processImageWithRetry(imageClient openai.OpenAIClient, entry rss.Entry, imagePrompt string, config EntryProcessConfig) (string, error) {
	if len(entry.ImageURLs) == 0 {
		return "", nil // No image to process
	}

	imgURL := entry.ImageURLs[0].String()
	dataURI := fetchImageAsBase64(imgURL)
	if dataURI == "" {
		return "", fmt.Errorf("could not fetch image from URL: %s", imgURL)
	}

	var lastErr error

	// Retry logic with exponential backoff
	backoff := config.InitialBackoff
	for attempt := 0; attempt < config.MaxRetries; attempt++ {
		if attempt > 0 {
			fmt.Printf("Retrying image processing (attempt %d/%d) after error: %v\n", attempt+1, config.MaxRetries, lastErr)
			time.Sleep(backoff)
			backoff = time.Duration(float64(backoff) * config.BackoffFactor)
			if backoff > config.MaxBackoff {
				backoff = config.MaxBackoff
			}
		}

		// Process the image
		imageDescription, err := ChatCompletionImageSummary(imageClient, imagePrompt, []string{dataURI})
		if err != nil {
			lastErr = err
			continue // Try again
		}

		return imageDescription, nil // Success!
	}

	// If we're here, we've exhausted all retry attempts
	return "", fmt.Errorf("failed to process image after %d attempts: %w", config.MaxRetries, lastErr)
}

// ProcessEntries takes RSS entries, processes them through an LLM, and returns processed items
func ProcessEntries(client openai.OpenAIClient, imageClient openai.OpenAIClient, systemPrompt string, entries []rss.Entry, imageEnabled bool, persona persona.Persona, debugOutputBenchmark bool) ([]models.Item, []string, error) {
	var items []models.Item
	var benchmarkInputs []string
	var processingErrors []error

	// Use default configuration
	config := DefaultEntryProcessConfig

	// Process entries sequentially with retry support
	for i, entry := range entries {
		fmt.Printf("Processing entry %d\n", i)

		// Store the entry string for benchmarking if needed
		if debugOutputBenchmark {
			benchmarkInputs = append(benchmarkInputs, entry.String(true))
		}

		// Process images first if image processing is enabled
		if imageEnabled && len(entry.ImageURLs) > 0 {
			// Create the image prompt
			imagePrompt, err := prompts.ComposeImagePrompt(persona, entry.Title)
			if err != nil {
				fmt.Printf("Error creating image prompt: %v\n", err)
			} else {
				fmt.Printf("Processing image for entry %d\n", i)
				imageDescription, err := processImageWithRetry(imageClient, entry, imagePrompt, config)
				if err != nil {
					fmt.Printf("Error processing image for entry %d: %v\n", i, err)
				} else {
					entry.ImageDescription = imageDescription
					fmt.Printf("Image processing successful for entry %d\n", i)
				}
			}
		}

		// Process the entry text
		item, err := processEntryWithRetry(client, systemPrompt, entry, config)
		if err != nil {
			fmt.Printf("Error processing entry %d: %v\n", i, err)
			processingErrors = append(processingErrors, fmt.Errorf("entry %d: %w", i, err))
			continue
		}

		fmt.Printf("Processed item %d successfully\n", i)
		items = append(items, item)
	}

	// If all entries failed, return an error
	if len(items) == 0 && len(processingErrors) > 0 {
		return nil, benchmarkInputs, fmt.Errorf("all entries failed processing: %v", processingErrors[0])
	}

	// If some entries failed but we have some successes, just log the errors
	if len(processingErrors) > 0 {
		fmt.Printf("Warning: %d entries failed processing\n", len(processingErrors))
	}

	return items, benchmarkInputs, nil
}

// EnrichItems adds links from RSS entries to items based on item ID
func EnrichItems(items []models.Item, entries []rss.Entry) []models.Item {
	enrichedItems := make([]models.Item, len(items))
	copy(enrichedItems, items)

	for i, item := range enrichedItems {
		id := item.ID
		if id == "" {
			continue
		}

		entry := rss.FindEntryByID(id, entries)
		if entry == nil {
			fmt.Printf("could not find item with ID %s in RSS entry\n", id)
			continue
		}

		enrichedItems[i].Link = entry.Link.Href
	}

	return enrichedItems
}

// FilterRelevantItems filters items by relevance and non-empty ID
func FilterRelevantItems(items []models.Item) []models.Item {
	var relevantItems []models.Item
	for _, item := range items {
		if item.IsRelevant && item.ID != "" {
			relevantItems = append(relevantItems, item)
		}
	}
	return relevantItems
}

// llmResponseToItems converts a JSON LLM response to a slice of Items
func llmResponseToItems(jsonStr string) (models.Item, error) {
	var items models.Item
	err := json.Unmarshal([]byte(jsonStr), &items)
	if err != nil {
		return models.Item{}, fmt.Errorf("could not unmarshal llm response to items: %w", err)
	}
	return items, nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
