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
	"github.com/bakkerme/ai-news-processor/internal/rss"
	"github.com/invopop/jsonschema"

	"github.com/bakkerme/ai-news-processor/internal/models"
)

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
	client.ChatCompletion(
		systemPrompt,
		userPrompts,
		imageURLs,
		ItemResponseSchema,
		"post_item",
		"an object representing a post",
		results,
	)
}

// ChatCompletionForFeedSummary sends a ChatCompletion to get a summary for an entire feed
func ChatCompletionForFeedSummary(client openai.OpenAIClient, systemPrompt string, userPrompts []string, results chan customerrors.ErrorString) {
	// Feed summaries don't include images directly
	client.ChatCompletion(
		systemPrompt,
		userPrompts,
		[]string{}, // No images for feed summaries
		SummaryResponseSchema,
		"summary",
		"a summary of multiple AI news items",
		results,
	)
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

// ensureValidImageURL ensures a URL has a scheme (http:// or https://)
func ensureValidImageURL(imgURL string) string {
	if !strings.HasPrefix(imgURL, "http://") && !strings.HasPrefix(imgURL, "https://") {
		return "https://" + imgURL
	}
	return imgURL
}

// ProcessEntries takes RSS entries, processes them through an LLM in batches, and returns processed items
func ProcessEntries(client openai.OpenAIClient, systemPrompt string, entries []rss.Entry, batchSize int, multiMode bool, debugOutputBenchmark bool) ([]models.Item, []string, error) {
	var items []models.Item
	var benchmarkInputs []string

	completionChannel := make(chan customerrors.ErrorString, len(entries))
	batchCounter := 0

	// Process entries in batches
	for i := 0; i < len(entries); i += batchSize {
		batch := entries[i:min(i+batchSize, len(entries))]
		fmt.Printf("Sending batch %d with %d items\n", i/batchSize, len(batch))

		batchStrings := make([]string, len(batch))
		var batchImageURLs []string // Collect image URLs from the batch

		if multiMode {
			for j, entry := range batch {
				batchStrings[j] = entry.String(false)

				// Extract image URLs from entries
				if len(entry.ImageURLs) > 0 {
					// Add up to 3 images per entry to avoid overwhelming the model
					maxImages := min(3, len(entry.ImageURLs))
					for k := 0; k < maxImages; k++ {
						// Get the URL, ensure it has a scheme
						imgURL := ensureValidImageURL(entry.ImageURLs[k].String())

						fmt.Println("Adding image", imgURL)

						// Fetch and convert to base64
						dataURI := fetchImageAsBase64(imgURL)
						if dataURI != "" {
							batchImageURLs = append(batchImageURLs, dataURI)
						}
					}
				}

				// Also check for MediaThumbnail if no other images were found
				if len(entry.ImageURLs) == 0 && entry.MediaThumbnail.URL != "" {
					imgURL := ensureValidImageURL(entry.MediaThumbnail.URL)
					dataURI := fetchImageAsBase64(imgURL)
					if dataURI != "" {
						batchImageURLs = append(batchImageURLs, dataURI)
					}
				}
			}
		}

		// Store inputs for benchmarking
		if debugOutputBenchmark {
			benchmarkInputs = append(benchmarkInputs, batchStrings...)
		}

		go ChatCompletionForEntrySummary(client, systemPrompt, batchStrings, batchImageURLs, completionChannel)
		batchCounter++
	}

	// Process results from all batches
	for i := 0; i < batchCounter; i++ {
		fmt.Printf("Waiting for batch %d\n", i)
		result := <-completionChannel
		if result.Err != nil {
			return nil, benchmarkInputs, fmt.Errorf("could not process value from LLM for batch %d: %s", i, result.Err)
		}

		fmt.Println(result.Value)

		processedValue := client.PreprocessJSON(result.Value)

		batchItems, err := llmResponseToItems(processedValue)
		if err != nil {
			return nil, benchmarkInputs, fmt.Errorf("could not convert llm output to json. %s: %w", result.Value, err)
		}

		fmt.Printf("Processed batch %d, found %d items\n", i, len(batchItems))
		items = append(items, batchItems...)
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
func llmResponseToItems(jsonStr string) ([]models.Item, error) {
	var items []models.Item
	err := json.Unmarshal([]byte(jsonStr), &items)
	if err != nil {
		return nil, err
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
