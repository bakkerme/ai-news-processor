package internal

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/bakkerme/ai-news-processor/internal/bench"
	"github.com/bakkerme/ai-news-processor/internal/contentextractor"
	"github.com/bakkerme/ai-news-processor/internal/email"
	"github.com/bakkerme/ai-news-processor/internal/fetcher"
	httputil "github.com/bakkerme/ai-news-processor/internal/http"
	"github.com/bakkerme/ai-news-processor/internal/http/retry"
	"github.com/bakkerme/ai-news-processor/internal/llm"
	"github.com/bakkerme/ai-news-processor/internal/openai"
	"github.com/bakkerme/ai-news-processor/internal/persona"
	"github.com/bakkerme/ai-news-processor/internal/prompts"
	"github.com/bakkerme/ai-news-processor/internal/qualityfilter"
	"github.com/bakkerme/ai-news-processor/internal/rss"
	"github.com/bakkerme/ai-news-processor/internal/specification"
	"github.com/bakkerme/ai-news-processor/internal/urlextraction"
	"github.com/bakkerme/ai-news-processor/models"
)

func Run() {
	s, err := specification.GetConfig()
	if err != nil {
		panic(err)
	}

	// Print the duration it took to run the job
	startTime := time.Now()
	defer func() {
		log.Printf("Job took %v\n", time.Since(startTime))
	}()

	// Initialize the OpenAI client
	openaiClient := openai.New(s.LlmUrl, s.LlmApiKey, s.LlmModel)

	// Initialize the image client if image processing is enabled
	var imageClient openai.OpenAIClient
	if s.LlmImageEnabled {
		imageClient = openai.New(s.LlmUrl, s.LlmApiKey, s.LlmImageModel)
		log.Println("Image processing enabled with model:", s.LlmImageModel)
	} else {
		// Use the main client as a fallback
		imageClient = openaiClient
	}

	// Initialize email service
	emailService, err := email.NewService(s)
	if err != nil {
		panic(fmt.Errorf("could not initialize email service: %w", err))
	}

	// Set up persona handling
	personaPath := s.PersonasPath
	if personaPath == "" {
		personaPath = "/app/personas/" // default to Docker path
	}

	personaFlag := flag.String("persona", "", "Persona to use (name or 'all')")
	flag.Parse()

	// Load and select personas
	selectedPersonas, err := persona.LoadAndSelect(personaPath, *personaFlag)
	if err != nil {
		panic(err)
	}

	// Create appropriate feed provider based on debug settings
	var feedProvider rss.FeedProvider
	if s.DebugMockRss {
		log.Println("Using mock feed provider")
		// Use the persona name from the first selected persona for mock data
		// Each persona will still use its own mock data in processing
		feedProvider = rss.NewMockFeedProvider(selectedPersonas[0].Name)
	} else {
		feedProvider = rss.NewFeedProvider()
	}

	// Process each persona
	for _, persona := range selectedPersonas {
		log.Printf("Processing persona: %s\n", persona.Name)
		urlExtractor := urlextraction.NewRedditExtractor()

		// 1. Fetch and process RSS feed using FeedProvider
		entries, err := rss.FetchAndProcessFeed(feedProvider, urlExtractor, persona.FeedURL, s.DebugRssDump, persona.Name)
		if err != nil {
			log.Printf("Failed to process RSS feed for persona %s: %v\n", persona.Name, err)
			continue
		}

		// Limit entries if DebugMaxEntries is set
		if s.DebugMaxEntries > 0 && len(entries) > s.DebugMaxEntries {
			entries = entries[:s.DebugMaxEntries]
		}

		// 2. Filter entries with quality filter
		entries = qualityfilter.Filter(entries, s.QualityFilterThreshold)

		// Store all raw inputs for benchmarking
		var benchmarkData models.RunData
		var items []models.Item

		// 3. Process entries with LLM
		if !s.DebugMockLLM {
			log.Println("Sending to LLM")
			systemPrompt, err := prompts.ComposePrompt(persona, "")
			if err != nil {
				log.Printf("Could not compose prompt for persona %s: %v\n", persona.Name, err)
				continue
			}

			// Create the LLM processor with the configured clients
			processorConfig := llm.EntryProcessConfig{
				InitialBackoff:       llm.DefaultEntryProcessConfig.InitialBackoff,
				BackoffFactor:        llm.DefaultEntryProcessConfig.BackoffFactor,
				MaxRetries:           llm.DefaultEntryProcessConfig.MaxRetries,
				MaxBackoff:           llm.DefaultEntryProcessConfig.MaxBackoff,
				ImageEnabled:         s.LlmImageEnabled,
				URLSummaryEnabled:    s.LlmUrlSummaryEnabled,
				DebugOutputBenchmark: s.DebugOutputBenchmark,
			}

			// Create retry config from entry process config
			retryConfig := retry.RetryConfig{
				InitialBackoff: processorConfig.InitialBackoff,
				BackoffFactor:  processorConfig.BackoffFactor,
				MaxRetries:     processorConfig.MaxRetries,
				MaxBackoff:     processorConfig.MaxBackoff,
			}

			// Initialize dependencies for the processor
			urlFetcher := fetcher.NewHTTPFetcher(nil, retryConfig, fetcher.DefaultUserAgent)
			imageFetcher := &httputil.DefaultImageFetcher{}
			articleExtractor := &contentextractor.DefaultArticleExtractor{}

			// Initialize the processor with the dependencies
			processor := llm.NewProcessor(
				openaiClient,
				imageClient,
				processorConfig,
				articleExtractor,
				urlFetcher,
				urlExtractor,
				imageFetcher,
			)

			// Process the entries using the processor
			items, benchmarkData, err = processor.ProcessEntries(systemPrompt, entries, persona)
			if err != nil {
				log.Printf("Could not process entries with LLM for persona %s: %v\n", persona.Name, err)
				continue
			}
		} else {
			log.Println("Loading fake LLM response")
			items = GetMockLLMResponse()
			// Generate mock benchmark data using the mock items, the current persona, and the original entries
			benchmarkData = GetMockBenchmarkData(items, persona, entries)
			// Since this is a mock, there is no error from processing
			err = nil
		}

		// 5. Enrich items with links from RSS entries
		items = llm.EnrichItems(items, entries)

		// 6. Filter for relevant items
		relevantItems := llm.FilterRelevantItems(items)
		if len(relevantItems) == 0 {
			log.Println("no items to render as an email")
			continue
		}

		// 7. Get relevant entries for summary
		relevantEntries := make([]rss.Entry, 0, len(relevantItems))
		for _, item := range relevantItems {
			entry := rss.FindEntryByID(item.ID, entries)
			if entry != nil {
				relevantEntries = append(relevantEntries, *entry)
			}
		}

		// 9. Generate summary for relevant items
		var summaryResponse *models.SummaryResponse
		if !s.DebugMockLLM {
			summaryResponse, err = llm.GenerateSummary(openaiClient, relevantEntries, persona)
			if err != nil {
				panic(fmt.Errorf("could not generate summary: %w", err))
			}
		} else {
			// Mock summary for debug mode
			summaryResponse = GetMockSummaryResponse(relevantItems)
		}

		// Store the overall summary in the benchmark data
		benchmarkData.OverallSummary = summaryResponse

		// Output benchmark data if requested
		if s.DebugOutputBenchmark {
			err := bench.WriteRunDataToDisk(&benchmarkData)
			if err != nil {
				log.Printf("Error writing benchmark data to disk for persona %s: %v\n", persona.Name, err)
			}
		}

		if s.SendBenchmarkToAuditService {
			err = bench.SubmitRunDataToAuditService(&benchmarkData, s.AuditServiceUrl)
			if err != nil {
				log.Printf("Warning: Failed to submit run data to audit service for persona %s: %v\n", persona.Name, err)
			}
		}

		// 10. Render and send email
		if !s.DebugSkipEmail {
			err = emailService.RenderAndSend(relevantItems, summaryResponse, persona.Name)
			if err != nil {
				log.Printf("Could not send email for persona %s: %v\n", persona.Name, err)
				continue
			}
		} else {
			log.Println("Skipping email")
		}
	}
}
