package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bakkerme/ai-news-processor/internal/contentextractor"
	"github.com/bakkerme/ai-news-processor/internal/customerrors"
	"github.com/bakkerme/ai-news-processor/internal/feeds"
	"github.com/bakkerme/ai-news-processor/internal/fetcher"
	httputil "github.com/bakkerme/ai-news-processor/internal/http"
	"github.com/bakkerme/ai-news-processor/internal/http/retry"
	"github.com/bakkerme/ai-news-processor/internal/openai"
	"github.com/bakkerme/ai-news-processor/internal/persona"
	"github.com/bakkerme/ai-news-processor/internal/prompts"
	"github.com/bakkerme/ai-news-processor/internal/urlextraction"
	"github.com/bakkerme/ai-news-processor/models"
)

// Note: Processor and EntryProcessConfig are defined in processor_types.go

// NewProcessor creates a new LLM processor with the given clients and configuration
func NewProcessor(client openai.OpenAIClient, imageClient openai.OpenAIClient, config EntryProcessConfig, articleExtractor contentextractor.ArticleExtractor, urlFetcher fetcher.Fetcher, urlExtractor urlextraction.Extractor, imageFetcher httputil.ImageFetcher) *Processor {
	return &Processor{
		client:               client,
		imageClient:          imageClient,
		urlFetcher:           urlFetcher,
		config:               config,
		urlSummaryEnabled:    config.URLSummaryEnabled,
		urlExtractor:         urlExtractor,
		imageEnabled:         config.ImageEnabled,
		debugOutputBenchmark: config.DebugOutputBenchmark,
		imageFetcher:         imageFetcher,
		articleExtractor:     articleExtractor,
	}
}

// ProcessEntries takes RSS entries, processes them through an LLM, and returns processed items
func (p *Processor) ProcessEntries(systemPrompt string, entries []feeds.Entry, persona persona.Persona) ([]models.Item, models.RunData, error) {
	var items []models.Item
	var processingErrors []error

	benchmarkData := models.RunData{
		EntrySummaries:                []models.EntrySummary{},
		ImageSummaries:                []models.ImageSummary{},
		WebContentSummaries:           []models.WebContentSummary{}, // This feature is unused for now, since web summaries do not use llm
		RunDate:                       time.Now(),
		Persona:                       persona,
		OverallModelUsed:              p.client.GetModelName(),
		ImageModelUsed:                p.imageClient.GetModelName(),
		WebContentModelUsed:           p.client.GetModelName(),
		TotalProcessingTime:           0,
		EntryTotalProcessingTime:      0,
		ImageTotalProcessingTime:      0,
		WebContentTotalProcessingTime: 0,
		SuccessRate:                   0,
	}

	// Track total processing time if benchmarking is enabled
	startTime := time.Now()

	// PHASE 1: Process all images first if image processing is enabled. This needs to be done first because the image processing uses a seperate model that takes time to load.
	if p.imageEnabled {
		log.Println("Phase 1: Processing all images")

		imageStartTime := time.Now()
		for i := range entries {
			if len(entries[i].ImageURLs) > 0 {
				// Create the image prompt
				imagePrompt, err := prompts.ComposeImagePrompt(persona, entries[i].Title)
				if err != nil {
					log.Printf("Error creating image prompt for entry %d: %v\n", i, err)
					continue
				}

				log.Printf("Processing image for entry %d: %s\n", i, entries[i].ImageURLs[0].String())

				// Track image processing time if benchmarking is enabled
				imgStartTime := time.Now()

				imageDescription, err := p.processImageWithRetry(entries[i], imagePrompt)

				// Calculate processing time for benchmarking
				imgProcessingTime := time.Since(imgStartTime).Milliseconds()

				if err != nil {
					log.Printf("Error processing image for entry %d: %v\n", i, err)
				} else {
					entries[i].ImageDescription = imageDescription
					log.Printf("Image processing successful for entry %d\n", i)

					// Add to benchmark data
					imgSummary := models.ImageSummary{
						ImageURL:         entries[i].ImageURLs[0].String(),
						ImageDescription: imageDescription,
						Title:            entries[i].Title,
						EntryID:          entries[i].ID,
						ProcessingTime:   imgProcessingTime,
					}
					benchmarkData.ImageSummaries = append(benchmarkData.ImageSummaries, imgSummary)
				}
			}
		}

		benchmarkData.ImageTotalProcessingTime = time.Since(imageStartTime).Milliseconds()
	}

	// PHASE 2: Process all external URLs
	if p.urlSummaryEnabled {
		log.Println("Phase 2: Processing all external URLs")

		webStartTime := time.Now()
		for i := range entries {
			log.Printf("Processing external URLs for entry %d\n", i)
			summaries, err := p.processExternalURLs(&entries[i], persona, &benchmarkData)
			if err != nil {
				log.Printf("Error processing external URLs for entry %d: %v\n", i, err)
				processingErrors = append(processingErrors, fmt.Errorf("entry %d: %w", i, err))
				continue
			}

			// Add the summaries to the entry
			entries[i].WebContentSummaries = summaries
		}

		benchmarkData.WebContentTotalProcessingTime = time.Since(webStartTime).Milliseconds()
	}

	// PHASE 3: Process the main entry text summarization for all entries
	log.Println("Phase 3: Processing all text summarizations")
	overallStartTime := time.Now()
	for i, entry := range entries {
		log.Printf("Processing entry text %d\n", i)

		entryStartTime := time.Now()

		// Process the main entry text (including external URL summaries if available)
		item, err := p.processEntryWithRetry(systemPrompt, entry)

		if err != nil {
			log.Printf("Error processing entry %d: %v\n", i, err)
			processingErrors = append(processingErrors, fmt.Errorf("entry %d: %w", i, err))
			continue
		}

		entryProcessingTime := time.Since(entryStartTime).Milliseconds()

		log.Printf("Processed item %d successfully\n", i)
		items = append(items, item)

		// Add to benchmark data
		entrySummary := models.EntrySummary{
			RawInput:       entry.String(true),
			Results:        item,
			ProcessingTime: entryProcessingTime,
		}
		benchmarkData.EntrySummaries = append(benchmarkData.EntrySummaries, entrySummary)
	}
	benchmarkData.EntryTotalProcessingTime = time.Since(overallStartTime).Milliseconds()

	// If all entries failed, return an error
	if len(items) == 0 && len(processingErrors) > 0 {
		return nil, benchmarkData, fmt.Errorf("all entries failed processing: %v", processingErrors[0])
	}

	// If some entries failed but we have some successes, just log the errors
	if len(processingErrors) > 0 {
		log.Printf("warning: %d entries failed processing\n", len(processingErrors))
	}

	// Finalize benchmark data
	benchmarkData.TotalProcessingTime = time.Since(startTime).Milliseconds()

	if len(entries) > 0 {
		successCount := len(items)
		benchmarkData.SuccessRate = float64(successCount) / float64(len(entries))
	}

	return items, benchmarkData, nil
}

// processExternalURLs extracts and processes external URLs from an entry
func (p *Processor) processExternalURLs(entry *feeds.Entry, persona persona.Persona, benchmarkData *models.RunData) (map[string]string, error) {
	// 1. Extract external URLs
	extractedURLs, err := p.urlExtractor.ExtractExternalURLsFromEntry(*entry)
	if err != nil {
		return nil, fmt.Errorf("failed to extract external URLs: %w", err)
	}

	// Store all extracted URLs in the ExternalURLs field
	entry.ExternalURLs = extractedURLs

	// Initialize the map for summaries if needed
	if entry.WebContentSummaries == nil {
		entry.WebContentSummaries = make(map[string]string)
	}

	if len(extractedURLs) == 0 {
		return nil, nil
	}

	// Only process the first URL for now
	extractedURLs = []url.URL{extractedURLs[0]}
	summaries := make(map[string]string)

	// 2. Process each extracted URL
	for _, extractedURLStr := range extractedURLs {
		log.Printf("processing external URL: %s\n", extractedURLStr.String())

		// Start timing for benchmarking
		webStartTime := time.Now()

		// 2a. Fetch the content
		resp, err := p.urlFetcher.Fetch(context.Background(), &extractedURLStr)
		if err != nil {
			log.Printf("warning: Failed to fetch content for %s: %v\n", extractedURLStr.String(), err)
			continue // Skip to the next URL if fetching fails
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("warning: Received non-OK status code for %s: %d\n", extractedURLStr.String(), resp.StatusCode)
			continue // Skip to the next URL for non-OK status codes
		}

		// 2b. Extract the article text
		articleData, err := p.articleExtractor.Extract(resp.Body, &extractedURLStr)
		if err != nil {
			log.Printf("warning: Failed to extract article content for %s: %v\n", extractedURLStr.String(), err)
			continue // Skip to the next URL if extraction fails
		}

		// 2c. Summarize the extracted content with LLM
		summary, err := p.summarizeWebSite(articleData.Title, &extractedURLStr, articleData.CleanedText, persona)
		if err != nil {
			log.Printf("warning: Failed to summarize content for %s: %v\n", extractedURLStr.String(), err)
			continue // Skip to the next URL if summarization fails
		}

		// Calculate processing time for benchmarking
		webProcessingTime := time.Since(webStartTime).Milliseconds()

		// 2d. Store the summary
		summaries[extractedURLStr.String()] = summary

		// Add to benchmark data if benchmarking is enabled
		if benchmarkData != nil {
			webSummary := models.WebContentSummary{
				URL:             extractedURLStr.String(),
				OriginalContent: articleData.CleanedText,
				Summary:         summary,
				Title:           articleData.Title,
				EntryID:         entry.ID,
				ProcessingTime:  webProcessingTime,
			}
			benchmarkData.WebContentSummaries = append(benchmarkData.WebContentSummaries, webSummary)
		}
	}

	return summaries, nil
}

// summarizeTextWithLLM summarizes given content using an LLM
func (p *Processor) summarizeWebSite(pageTitle string, url *url.URL, content string, persona persona.Persona) (string, error) {
	// Create a system prompt for summarization
	systemPrompt := fmt.Sprintf("You are a concise summarizer for %s. Provide brief, informative summaries of web content. Keep summaries to 300-500 words and focus on key technical insights.", persona.Name)

	// Use simple prompt for initial implementation
	userPrompt := fmt.Sprintf("Please provide a concise summary of the following article content (aim for 300-500 words):\n\n%s\n\nTitle: %s\n\nURL: %s", content, pageTitle, url)

	// disable qwen thinking
	// userPrompt += "\n/no_thinking"

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
func (p *Processor) processEntryWithRetry(systemPrompt string, entry feeds.Entry) (models.Item, error) {
	entryString := entry.String(true)

	// noThink := "/no_thinking"
	noThink := ""

	processFn := func() (models.Item, error) {
		// Process the entry
		results := make(chan customerrors.ErrorString, 1)
		chatCompletionForEntrySummary(p.client, systemPrompt, []string{entryString, noThink}, nil, results)
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

		item.Entry = entry // Associate the processed item with the original entry
		return item, nil
	}

	return p.retryItemFunc(processFn, "entry")
}

// processImageWithRetry processes an image with retry support
func (p *Processor) processImageWithRetry(entry feeds.Entry, imagePrompt string) (string, error) {
	if len(entry.ImageURLs) == 0 {
		return "", nil // No image to process
	}

	imgURL := entry.ImageURLs[0].String()
	dataURI, err := p.imageFetcher.FetchAsBase64(imgURL)
	if err != nil {
		return "", fmt.Errorf("could not fetch image using imageFetcher from URL %s: %w", imgURL, err)
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
			log.Printf("retrying %s processing (attempt %d/%d) after error: %v\n",
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

// retrySummaryFunc is a helper to retry a function that returns a models.SummaryResponse and error
func (p *Processor) retrySummaryFunc(processFn func() (*models.SummaryResponse, error), processType string) (*models.SummaryResponse, error) {
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

	var result *models.SummaryResponse
	var lastErr error

	// Manually implement retry logic since we can't use type parameters on methods
	// and retry.RetryWithBackoff expects T to match for both the function and return value
	backoff := retryConfig.InitialBackoff
	for attempt := 0; attempt <= retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("retrying %s processing (attempt %d/%d) after error: %v\n",
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
		return nil, fmt.Errorf("max retries exceeded for %s: %w", processType, lastErr)
	}
	return result, nil
}

// EnrichItems adds links from RSS entries to items based on item ID
func EnrichItems(items []models.Item, entries []feeds.Entry) []models.Item {
	enrichedItems := make([]models.Item, len(items))
	copy(enrichedItems, items)

	for i, item := range enrichedItems {
		id := item.ID
		if id == "" {
			continue
		}

		entry := feeds.FindEntryByID(id, entries)
		if entry == nil {
			log.Printf("could not find item with ID %s in RSS entry\n", id)
			continue
		}

		enrichedItems[i].Title = entry.Title

		// Populate the Entry field with the associated RSS entry
		enrichedItems[i].Entry = *entry
		enrichedItems[i].Link = entry.Link.Href

		if len(entry.ImageURLs) > 0 {
			enrichedItems[i].ThumbnailURL = entry.ImageURLs[0].String()
		} else if entry.MediaThumbnail.URL != "" {
			enrichedItems[i].ThumbnailURL = entry.MediaThumbnail.URL
		}
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

// llmResponseToItems converts a JSON LLM response to a single models.Item
func llmResponseToItems(jsonStr string) (models.Item, error) {
	var items models.Item
	err := json.Unmarshal([]byte(jsonStr), &items)
	if err != nil {
		return models.Item{}, fmt.Errorf("could not unmarshal llm response to items: %w", err)
	}
	return items, nil
}

// generateSummaryWithRetry generates a summary with retry support
func (p *Processor) generateSummaryWithRetry(items []models.Item, persona persona.Persona) (*models.SummaryResponse, error) {
	processFn := func() (*models.SummaryResponse, error) {
		// Create input for summary
		summaryInputs := make([]string, len(items))
		for i, item := range items {
			summaryInputs[i] = item.ToSummaryString()
		}

		summaryChannel := make(chan customerrors.ErrorString, 1)
		summaryPrompt, err := prompts.ComposeSummaryPrompt(persona)
		if err != nil {
			return nil, fmt.Errorf("could not compose summary prompt for persona %s: %w", persona.Name, err)
		}

		go chatCompletionForFeedSummary(p.client, summaryPrompt, summaryInputs, summaryChannel)

		summaryResult := <-summaryChannel
		if summaryResult.Err != nil {
			return nil, fmt.Errorf("could not generate summary: %w", summaryResult.Err)
		}

		processedSummary := p.client.PreprocessJSON(summaryResult.Value)
		summary, err := models.UnmarshalSummaryResponseJSON([]byte(processedSummary))
		if err != nil {
			return nil, fmt.Errorf("could not parse summary response: %w", err)
		}

		return summary, nil
	}

	return p.retrySummaryFunc(processFn, "summary")
}
