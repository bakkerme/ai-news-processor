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
func ChatCompletionImageSummary(client openai.OpenAIClient, systemPrompt string, imageURLs []string, results chan customerrors.ErrorString) {
	// Empty userPrompt as the image is the content
	// No schema parameters needed for image analysis
	client.ChatCompletion(
		systemPrompt,
		[]string{}, // No additional text prompt, just let the model analyze the images
		imageURLs,
		nil, // Schema parameters not needed for image analysis
		results,
	)
	close(results)
}

// ParseSummaryResponse parses a JSON string into a SummaryResponse
func ParseSummaryResponse(jsonStr string) (*models.SummaryResponse, error) {
	var summary models.SummaryResponse
	err := json.Unmarshal([]byte(jsonStr), &summary)
	if err != nil {
		return nil, fmt.Errorf("failed to parse summary response: %w", err)
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

// ProcessEntries takes RSS entries, processes them through an LLM in batches, and returns processed items
func ProcessEntries(client openai.OpenAIClient, imageClient openai.OpenAIClient, systemPrompt string, entries []rss.Entry, batchSize int, imageEnabled bool, persona persona.Persona, debugOutputBenchmark bool) ([]models.Item, []string, error) {
	var items []models.Item
	var benchmarkInputs []string

	// Process images first if image processing is enabled
	if imageEnabled {
		completionChannel := make(chan imageResult, len(entries))
		entriesWithImageDescriptionCount := 0

		for i := range entries {
			// Get the URL, ensure it has a scheme
			entry := entries[i]
			if len(entry.ImageURLs) == 0 {
				continue
			}

			imgURL := entry.ImageURLs[0].String()

			// Fetch and convert to base64
			fmt.Printf("Processing image %s\n", imgURL)
			dataURI := fetchImageAsBase64(imgURL)

			// Create the image prompt
			imagePrompt, err := prompts.ComposeImagePrompt(persona, entry.Title)
			if err != nil {
				fmt.Printf("Error creating image prompt: %v\n", err)
				continue
			}

			// Create a closure to capture the current index
			entryIndex := i
			go func(idx int) {
				resultChan := make(chan customerrors.ErrorString, 1)
				ChatCompletionImageSummary(imageClient, imagePrompt, []string{dataURI}, resultChan)
				// Wait for the result from the channel
				result, ok := <-resultChan
				if !ok {
					// Channel was closed without a result
					completionChannel <- imageResult{
						result: customerrors.ErrorString{
							Err:   fmt.Errorf("image processing channel closed without result"),
							Value: "",
						},
						entryIdx: idx,
					}
					return
				}
				completionChannel <- imageResult{result: result, entryIdx: idx}
			}(entryIndex)

			entriesWithImageDescriptionCount++
		}

		fmt.Printf("Waiting for %d image descriptions\n", entriesWithImageDescriptionCount)
		// Process results and assign them to the correct entries
		for i := 0; i < entriesWithImageDescriptionCount; i++ {
			result := <-completionChannel
			if result.result.Err != nil {
				fmt.Printf("Error processing images for entry %d: %v\n", result.entryIdx, result.result.Err)
			} else {
				fmt.Printf("Got result for entry %d: %s\n", result.entryIdx, result.result.Value)
				// Store the image description in the correct entry
				entries[result.entryIdx].ImageDescription = result.result.Value
				fmt.Printf("Image processing successful for entry %d\n", result.entryIdx)
			}
		}
		close(completionChannel)
	}

	completionChannel := make(chan customerrors.ErrorString, len(entries))
	batchCounter := 0

	// Process entries in batches
	for i := 0; i < len(entries); i += batchSize {
		batch := entries[i:min(i+batchSize, len(entries))]
		fmt.Printf("Sending item %d\n", i)

		batchStrings := make([]string, len(batch))
		for j, entry := range batch {
			batchStrings[j] = entry.String(false)
		}

		// Store inputs for benchmarking
		if debugOutputBenchmark {
			benchmarkInputs = append(benchmarkInputs, batchStrings...)
		}

		go ChatCompletionForEntrySummary(client, systemPrompt, batchStrings, nil, completionChannel)
		batchCounter++
	}

	// Process results from all batches
	for i := 0; i < batchCounter; i++ {
		fmt.Printf("Waiting for item %d\n", i)
		result := <-completionChannel
		if result.Err != nil {
			return nil, benchmarkInputs, fmt.Errorf("could not process value from LLM for batch %d: %s", i, result.Err)
		}

		processedValue := client.PreprocessJSON(result.Value)

		fmt.Println(processedValue)

		item, err := llmResponseToItems(processedValue)
		if err != nil {
			return nil, benchmarkInputs, fmt.Errorf("could not convert llm output to json. %s: %w", processedValue, err)
		}

		fmt.Printf("Processed item %d\n", i)
		items = append(items, item)
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
