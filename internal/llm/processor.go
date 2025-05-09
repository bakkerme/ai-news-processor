package llm

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/bakkerme/ai-news-processor/internal/customerrors"
	"github.com/bakkerme/ai-news-processor/internal/http"
	"github.com/bakkerme/ai-news-processor/internal/openai"
	"github.com/bakkerme/ai-news-processor/internal/persona"
	"github.com/bakkerme/ai-news-processor/internal/prompts"
	"github.com/bakkerme/ai-news-processor/internal/rss"

	"github.com/bakkerme/ai-news-processor/internal/models"
)

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

// processEntryWithRetry processes a single entry with retry support
func processEntryWithRetry(client openai.OpenAIClient, systemPrompt string, entry rss.Entry, config EntryProcessConfig) (models.Item, error) {
	entryString := entry.String(true)

	processFn := func() (models.Item, error) {
		// Process the entry
		results := make(chan customerrors.ErrorString, 1)
		chatCompletionForEntrySummary(client, systemPrompt, []string{entryString}, nil, results)
		result := <-results
		close(results)

		if result.Err != nil {
			return models.Item{}, fmt.Errorf("could not process value from LLM: %w", result.Err)
		}

		processedValue := client.PreprocessJSON(result.Value)

		item, err := llmResponseToItems(processedValue)
		if err != nil {
			return models.Item{}, fmt.Errorf("could not convert llm output to json. %s: %w", processedValue, err)
		}

		return item, nil
	}

	return withRetry(processFn, "entry", config)
}

// processImageWithRetry processes an image with retry support
func processImageWithRetry(imageClient openai.OpenAIClient, entry rss.Entry, imagePrompt string, config EntryProcessConfig) (string, error) {
	if len(entry.ImageURLs) == 0 {
		return "", nil // No image to process
	}

	imgURL := entry.ImageURLs[0].String()
	dataURI := http.FetchImageAsBase64(imgURL)
	if dataURI == "" {
		return "", fmt.Errorf("could not fetch image from URL: %s", imgURL)
	}

	processFn := func() (string, error) {
		// Process the image
		return chatCompletionImageSummary(imageClient, imagePrompt, []string{dataURI})
	}

	return withRetry(processFn, "image", config)
}

// withRetry is a generic retry function that takes a processing function and retries it with exponential backoff
func withRetry[T any](processFn func() (T, error), processType string, config EntryProcessConfig) (T, error) {
	var result T
	var lastErr error

	// Retry logic with exponential backoff
	backoff := config.InitialBackoff
	for attempt := 0; attempt < config.MaxRetries; attempt++ {
		if attempt > 0 {
			fmt.Printf("Retrying %s processing (attempt %d/%d) after error: %v\n", processType, attempt+1, config.MaxRetries, lastErr)
			time.Sleep(backoff)
			backoff = time.Duration(float64(backoff) * config.BackoffFactor)
			if backoff > config.MaxBackoff {
				backoff = config.MaxBackoff
			}
		}

		// Run the processing function
		var err error
		result, err = processFn()
		if err == nil {
			return result, nil // Success!
		}
		lastErr = err
	}

	// If we're here, we've exhausted all retry attempts
	var zero T
	return zero, fmt.Errorf("failed to process %s after %d attempts: %w", processType, config.MaxRetries, lastErr)
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
