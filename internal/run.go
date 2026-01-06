package internal

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/bakkerme/ai-news-processor/internal/bench"
	"github.com/bakkerme/ai-news-processor/internal/contentextractor"
	"github.com/bakkerme/ai-news-processor/internal/email"
	"github.com/bakkerme/ai-news-processor/internal/feeds"
	"github.com/bakkerme/ai-news-processor/internal/fetcher"
	httputil "github.com/bakkerme/ai-news-processor/internal/http"
	"github.com/bakkerme/ai-news-processor/internal/http/retry"
	"github.com/bakkerme/ai-news-processor/internal/llm"
	"github.com/bakkerme/ai-news-processor/internal/openai"
	"github.com/bakkerme/ai-news-processor/internal/persona"
	"github.com/bakkerme/ai-news-processor/internal/prompts"
	"github.com/bakkerme/ai-news-processor/internal/providers"
	"github.com/bakkerme/ai-news-processor/internal/providers/rss"
	"github.com/bakkerme/ai-news-processor/internal/qualityfilter"
	"github.com/bakkerme/ai-news-processor/internal/sentlog"
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

	// Initialize the OpenAI client with safe timeouts to prevent infinite generation
	openaiClient := openai.NewWithSafeTimeouts(s.LlmUrl, s.LlmApiKey, s.LlmModel)

	// Initialize the image client if image processing is enabled
	var imageClient openai.OpenAIClient
	if s.LlmImageEnabled {
		imageClient = openai.NewWithSafeTimeouts(s.LlmUrl, s.LlmApiKey, s.LlmImageModel)
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

	// Create provider factory function
	createProvider := func(providerType string, personaName string) (feeds.FeedProvider, error) {
		if s.DebugMockFeeds {
			log.Printf("Using mock feed provider for persona %s", personaName)
			return providers.NewMockProvider(personaName), nil
		}

		switch providerType {
		case "reddit":
			log.Printf("Using Reddit API provider for persona %s", personaName)
			return providers.NewRedditProvider(
				s.RedditClientID,
				s.RedditSecret,
				s.RedditUsername,
				s.RedditPassword,
				s.DebugRedditDump,
			)
		case "rss":
			log.Printf("Using RSS provider for persona %s", personaName)
			return rss.NewRSSProvider(s.DebugRedditDump), nil // Reuse debug flag for RSS dumps
		default:
			return nil, fmt.Errorf("unsupported provider type: %s", providerType)
		}
	}

	// Process each persona
	sentLogBase := s.SentLogBasePath
	if sentLogBase == "" {
		sentLogBase = "."
	}
	sentLogPath := filepath.Join(sentLogBase, "sent_post_ids.json")
	sentIDs, err := sentlog.LoadSentIDs(sentLogPath)
	if err != nil {
		log.Printf("Warning: could not load sent log: %v", err)
		sentIDs = make(map[string]struct{})
	}

	for _, persona := range selectedPersonas {
		log.Printf("Processing persona: %s (provider: %s)\n", persona.Name, persona.GetProvider())

		// Create provider specific to this persona
		feedProvider, err := createProvider(persona.GetProvider(), persona.Name)
		if err != nil {
			log.Printf("Failed to create provider for persona %s: %v\n", persona.Name, err)
			continue
		}

		// Create appropriate URL extractor based on provider type
		var urlExtractor urlextraction.Extractor
		switch persona.GetProvider() {
		case "reddit":
			urlExtractor = urlextraction.NewRedditExtractor()
		case "rss":
			// For now, use the Reddit extractor as it handles generic URLs well
			// TODO: Consider creating a generic URL extractor in the future
			urlExtractor = urlextraction.NewRedditExtractor()
		default:
			urlExtractor = urlextraction.NewRedditExtractor()
		}

		// 1. Fetch and process feed using FeedProvider
		entries, err := feeds.FetchAndProcessFeed(feedProvider, urlExtractor, persona, s.DebugRedditDump)
		if err != nil {
			log.Printf("Failed to process feed for persona %s: %v\n", persona.Name, err)
			continue
		}

		// Limit entries if DebugMaxEntries is set
		if s.DebugMaxEntries > 0 && len(entries) > s.DebugMaxEntries {
			entries = entries[:s.DebugMaxEntries]
		}

		// 2. Filter entries with quality filter (use persona-specific threshold)
		threshold := persona.GetCommentThreshold(s.QualityFilterThreshold)
		entries = qualityfilter.Filter(entries, threshold)

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

		// 6. Filter for relevant items
		relevantItems := llm.FilterRelevantItems(items)
		relevantItems = filterUnsentItems(relevantItems, sentIDs)
		if len(relevantItems) == 0 {
			log.Println("no items to render as an email")
			continue
		}

		// 9. Generate summary for relevant items
		var summaryResponse *models.SummaryResponse
		if !s.DebugMockLLM {
			summaryResponse, err = llm.GenerateSummary(openaiClient, relevantItems, persona)
			if err != nil {
				log.Printf("Could not generate summary for persona %s: %v\n", persona.Name, err)
				continue
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
			// Persist newly emailed items so future runs skip them.
			for _, item := range relevantItems {
				if item.ID == "" {
					continue
				}
				sentIDs[item.ID] = struct{}{}
			}
			if err := sentlog.SaveSentIDs(sentLogPath, sentIDs); err != nil {
				log.Printf("Warning: could not persist sent log: %v", err)
			}
		} else {
			log.Println("Skipping email")
		}
	}
}

func filterUnsentItems(items []models.Item, sentIDs map[string]struct{}) []models.Item {
	sentCount := 0
	unsentItems := make([]models.Item, 0, len(items))
	for _, item := range items {
		if item.ID == "" {
			continue
		}
		if _, exists := sentIDs[item.ID]; exists {
			sentCount++
			continue
		}
		unsentItems = append(unsentItems, item)
	}
	if sentCount > 0 {
		log.Printf("Skipping %d items already emailed", sentCount)
	}
	return unsentItems
}
