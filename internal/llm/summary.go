package llm

import (
	"log"

	"github.com/bakkerme/ai-news-processor/internal/contentextractor"
	"github.com/bakkerme/ai-news-processor/internal/fetcher"
	"github.com/bakkerme/ai-news-processor/internal/http"
	"github.com/bakkerme/ai-news-processor/internal/http/retry"
	"github.com/bakkerme/ai-news-processor/internal/openai"
	"github.com/bakkerme/ai-news-processor/internal/persona"
	"github.com/bakkerme/ai-news-processor/internal/rss"
	"github.com/bakkerme/ai-news-processor/internal/urlextraction"
	"github.com/bakkerme/ai-news-processor/models"
)

// GenerateSummary creates a summary for a set of relevant RSS entries with retry support
func GenerateSummary(client openai.OpenAIClient, entries []rss.Entry, p persona.Persona) (*models.SummaryResponse, error) {
	log.Println("Generating summary of relevant items")

	// Create processor config for retry logic
	processorConfig := EntryProcessConfig{
		InitialBackoff: DefaultEntryProcessConfig.InitialBackoff,
		BackoffFactor:  DefaultEntryProcessConfig.BackoffFactor,
		MaxRetries:     DefaultEntryProcessConfig.MaxRetries,
		MaxBackoff:     DefaultEntryProcessConfig.MaxBackoff,
	}

	// Create retry config from entry process config
	retryConfig := retry.RetryConfig{
		InitialBackoff: processorConfig.InitialBackoff,
		BackoffFactor:  processorConfig.BackoffFactor,
		MaxRetries:     processorConfig.MaxRetries,
		MaxBackoff:     processorConfig.MaxBackoff,
	}

	// Initialize minimal dependencies for the processor (only needed for retry logic)
	urlFetcher := fetcher.NewHTTPFetcher(nil, retryConfig, fetcher.DefaultUserAgent)
	imageFetcher := &http.DefaultImageFetcher{}
	articleExtractor := &contentextractor.DefaultArticleExtractor{}
	urlExtractor := urlextraction.NewRedditExtractor()

	// Create processor instance to use retry logic
	processor := NewProcessor(
		client,
		client, // Use same client for both regular and image operations
		processorConfig,
		articleExtractor,
		urlFetcher,
		urlExtractor,
		imageFetcher,
	)

	// Use the retry-enabled summary generation
	return processor.generateSummaryWithRetry(entries, p)
}
