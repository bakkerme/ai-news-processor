package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bakkerme/ai-news-processor/internal/contentextractor"
	"github.com/bakkerme/ai-news-processor/internal/customerrors"
	"github.com/bakkerme/ai-news-processor/internal/fetcher"
	httputil "github.com/bakkerme/ai-news-processor/internal/http"
	"github.com/bakkerme/ai-news-processor/internal/http/retry"
	"github.com/bakkerme/ai-news-processor/internal/models"
	"github.com/bakkerme/ai-news-processor/internal/openai"
	"github.com/bakkerme/ai-news-processor/internal/persona"
	"github.com/bakkerme/ai-news-processor/internal/prompts"
	"github.com/bakkerme/ai-news-processor/internal/rss"
	"github.com/bakkerme/ai-news-processor/internal/urlextraction"
)

// NewProcessor creates a new LLM processor with the given clients and configuration
func NewProcessor(client openai.OpenAIClient, imageClient openai.OpenAIClient, config EntryProcessConfig) *Processor {
	// Create retry config for fetcher from entry process config
	retryConfig := retry.RetryConfig{
		InitialBackoff: config.InitialBackoff,
		BackoffFactor:  config.BackoffFactor,
		MaxRetries:     config.MaxRetries,
		MaxBackoff:     config.MaxBackoff,
	}

	// Initialize fetcher with default client and config
	urlFetcher := fetcher.NewHTTPFetcher(nil, retryConfig, fetcher.DefaultUserAgent)

	// Initialize URL extractor
	urlExtractor := urlextraction.NewRedditExtractor()

	return &Processor{
		client:               client,
		imageClient:          imageClient,
		urlFetcher:           urlFetcher,
		config:               config,
		urlSummaryEnabled:    config.URLSummaryEnabled,
		urlExtractor:         urlExtractor,
		imageEnabled:         config.ImageEnabled,
		debugOutputBenchmark: config.DebugOutputBenchmark,
	}
}

// ProcessEntries takes RSS entries, processes them through an LLM, and returns processed items
func (p *Processor) ProcessEntries(systemPrompt string, entries []rss.Entry, persona persona.Persona) ([]models.Item, []string, error) {
	var items []models.Item
	var benchmarkInputs []string
	var processingErrors []error

	// PHASE 1: Process all images first if image processing is enabled. This needs to be done first because the image processing uses a seperate model that takes time to load.
	if p.imageEnabled {
		fmt.Println("Phase 1: Processing all images")
		for i := range entries {
			if len(entries[i].ImageURLs) > 0 {
				// Create the image prompt
				imagePrompt, err := prompts.ComposeImagePrompt(persona, entries[i].Title)
				if err != nil {
					fmt.Printf("Error creating image prompt for entry %d: %v\n", i, err)
					continue
				}

				fmt.Printf("Processing image for entry %d: %s\n", i, entries[i].ImageURLs[0].String())
				imageDescription, err := p.processImageWithRetry(entries[i], imagePrompt)
				if err != nil {
					fmt.Printf("Error processing image for entry %d: %v\n", i, err)
				} else {
					entries[i].ImageDescription = imageDescription
					fmt.Printf("Image processing successful for entry %d\n", i)
				}
			}
		}
	}

	// PHASE 2: Process all external URLs
	if p.urlSummaryEnabled {
		fmt.Println("Phase 2: Processing all external URLs")
		for i := range entries {
			fmt.Printf("Processing external URLs for entry %d\n", i)
			summaries, err := p.processExternalURLs(&entries[i], persona)
			if err != nil {
				fmt.Printf("Error processing external URLs for entry %d: %v\n", i, err)
				processingErrors = append(processingErrors, fmt.Errorf("entry %d: %w", i, err))
				continue
			}

			// Add the summaries to the entry
			entries[i].ExternalURLSummaries = summaries
		}
	}

	// Store benchmark inputs if needed
	if p.debugOutputBenchmark {
		for _, entry := range entries {
			benchmarkInputs = append(benchmarkInputs, entry.String(true))
		}
	}

	// PHASE 3: Process the main entry text summarization for all entries
	fmt.Println("Phase 3: Processing all text summarizations")
	for i, entry := range entries {
		fmt.Printf("Processing entry text %d\n", i)

		// Process the main entry text (including external URL summaries if available)
		item, err := p.processEntryWithRetry(systemPrompt, entry)
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
		fmt.Printf("warning: %d entries failed processing\n", len(processingErrors))
	}

	return items, benchmarkInputs, nil
}

// processExternalURLs extracts and processes external URLs from an entry
func (p *Processor) processExternalURLs(entry *rss.Entry, persona persona.Persona) (map[string]string, error) {
	// 1. Extract external URLs
	extractedURLs, err := p.urlExtractor.ExtractURLsFromEntry(*entry)
	if err != nil {
		return nil, fmt.Errorf("failed to extract external URLs: %w", err)
	}

	// Initialize the map for summaries if needed
	if entry.ExternalURLSummaries == nil {
		entry.ExternalURLSummaries = make(map[string]string)
	}

	if len(extractedURLs) == 0 {
		return nil, nil
	}

	// Only process the first URL for now
	extractedURLs = []string{extractedURLs[0]}
	summaries := make(map[string]string)

	// 2. Process each extracted URL
	for _, extractedURLStr := range extractedURLs {
		fmt.Printf("processing external URL: %s\n", extractedURLStr)

		// Parse URL string to *url.URL
		parsedURL, err := url.Parse(extractedURLStr)
		if err != nil {
			fmt.Printf("warning: Failed to parse URL %s: %v\n", extractedURLStr, err)
			continue // Skip to the next URL if parsing fails
		}

		// 2a. Fetch the content
		resp, err := p.urlFetcher.Fetch(context.Background(), extractedURLStr)
		if err != nil {
			fmt.Printf("warning: Failed to fetch content for %s: %v\n", extractedURLStr, err)
			continue // Skip to the next URL if fetching fails
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("warning: Received non-OK status code for %s: %d\n", extractedURLStr, resp.StatusCode)
			continue // Skip to the next URL for non-OK status codes
		}

		// 2b. Extract the article text
		articleData, err := contentextractor.ExtractArticle(resp.Body, parsedURL)
		if err != nil {
			fmt.Printf("warning: Failed to extract article content for %s: %v\n", extractedURLStr, err)
			continue // Skip to the next URL if extraction fails
		}

		// fmt.Printf("extracted article content: %s\n", articleData.CleanedText)

		// 2c. Summarize the extracted content with LLM
		summary, err := p.summarizeWebSite(articleData.Title, extractedURLStr, articleData.CleanedText, persona)
		if err != nil {
			fmt.Printf("warning: Failed to summarize content for %s: %v\n", extractedURLStr, err)
			continue // Skip to the next URL if summarization fails
		}

		// 2d. Store the summary
		summaries[extractedURLStr] = summary
	}

	return summaries, nil
}

// summarizeTextWithLLM summarizes given content using an LLM
func (p *Processor) summarizeWebSite(pageTitle, url, content string, persona persona.Persona) (string, error) {
	// Create a system prompt for summarization
	systemPrompt := fmt.Sprintf("You are a concise summarizer for %s. Provide brief, informative summaries of web content.", persona.Name)

	// Use simple prompt for initial implementation
	userPrompt := fmt.Sprintf("Please provide a concise summary of the following article content:\n\n%s\n\nTitle: %s\n\nURL: %s", content, pageTitle, url)

	// disable qwen thinking
	userPrompt += "\n/no_thinking"

	// Function to execute the LLM call
	processFn := func() (string, error) {
		result, err := p.chatCompletionForWebSummary(systemPrompt, userPrompt)

		if err != nil {
			return "", fmt.Errorf("could not process value from LLM: %w", err)
		}

		// strip the result of any think tags. They should be empty with the /no_thinking flag
		result = strings.ReplaceAll(result, "<think>", "")
		result = strings.ReplaceAll(result, "</think>", "")

		return result, nil
	}

	// Retry the LLM call if it fails
	return p.retryStringFunc(processFn)
}

// processEntryWithRetry processes a single entry with retry support
func (p *Processor) processEntryWithRetry(systemPrompt string, entry rss.Entry) (models.Item, error) {
	entryString := entry.String(true)

	processFn := func() (models.Item, error) {
		// Process the entry
		results := make(chan customerrors.ErrorString, 1)
		chatCompletionForEntrySummary(p.client, systemPrompt, []string{entryString}, nil, results)
		result := <-results
		close(results)

		if result.Err != nil {
			return models.Item{}, fmt.Errorf("could not process value from LLM: %w", result.Err)
		}

		processedValue := p.client.PreprocessJSON(result.Value)

		item, err := llmResponseToItems(processedValue)
		if err != nil {
			return models.Item{}, fmt.Errorf("could not convert llm output to json. %s: %w", processedValue, err)
		}

		return item, nil
	}

	return p.retryItemFunc(processFn, "entry")
}

// processImageWithRetry processes an image with retry support
func (p *Processor) processImageWithRetry(entry rss.Entry, imagePrompt string) (string, error) {
	if len(entry.ImageURLs) == 0 {
		return "", nil // No image to process
	}

	imgURL := entry.ImageURLs[0].String()
	dataURI := httputil.FetchImageAsBase64(imgURL)
	if dataURI == "" {
		return "", fmt.Errorf("could not fetch image from URL: %s", imgURL)
	}

	processFn := func() (string, error) {
		// Process the image
		return chatCompletionImageSummary(p.imageClient, imagePrompt, []string{dataURI})
	}

	return p.retryStringFunc(processFn)
}

// retryStringFunc is a helper to retry a function that returns a string and error
func (p *Processor) retryStringFunc(processFn func() (string, error)) (string, error) {
	// Create retry config from processor's config
	retryConfig := retry.RetryConfig{
		InitialBackoff: p.config.InitialBackoff,
		BackoffFactor:  p.config.BackoffFactor,
		MaxRetries:     p.config.MaxRetries,
		MaxBackoff:     p.config.MaxBackoff,
	}

	// Create a basic shouldRetry function that handles common errors
	shouldRetry := func(err error) bool {
		if err == nil {
			return false // No error, no need to retry
		}
		// Add more sophisticated retry logic as needed
		return true // For now, retry on any error
	}

	return retry.RetryWithBackoff(context.Background(), retryConfig, func(ctx context.Context) (string, error) {
		// The provided processFn might not take a context, but RetryWithBackoff requires one.
		return processFn()
	}, shouldRetry)
}

// retryItemFunc is a helper to retry a function that returns a models.Item and error
func (p *Processor) retryItemFunc(processFn func() (models.Item, error), processType string) (models.Item, error) {
	// Create retry config from processor's config
	retryConfig := retry.RetryConfig{
		InitialBackoff: p.config.InitialBackoff,
		BackoffFactor:  p.config.BackoffFactor,
		MaxRetries:     p.config.MaxRetries,
		MaxBackoff:     p.config.MaxBackoff,
	}

	// Create a basic shouldRetry function that handles common errors
	shouldRetry := func(err error) bool {
		if err == nil {
			return false // No error, no need to retry
		}
		// Add more sophisticated retry logic as needed
		return true // For now, retry on any error
	}

	emptyItem := models.Item{}
	var result models.Item
	var lastErr error

	// Manually implement retry logic since we can't use type parameters on methods
	// and retry.RetryWithBackoff expects T to match for both the function and return value
	backoff := retryConfig.InitialBackoff
	for attempt := 0; attempt <= retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			fmt.Printf("retrying %s processing (attempt %d/%d) after error: %v\n",
				processType, attempt, retryConfig.MaxRetries, lastErr)
			time.Sleep(backoff)
			backoff = time.Duration(float64(backoff) * retryConfig.BackoffFactor)
			if backoff > retryConfig.MaxBackoff {
				backoff = retryConfig.MaxBackoff
			}
		}

		var err error
		result, err = processFn()
		if err == nil {
			return result, nil // Success
		}

		lastErr = err
		if !shouldRetry(err) {
			break // Don't retry non-retryable errors
		}
	}

	if lastErr != nil {
		return emptyItem, fmt.Errorf("max retries exceeded for %s: %w", processType, lastErr)
	}
	return result, nil
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
