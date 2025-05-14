package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/bakkerme/ai-news-processor/internal/bench"
	"github.com/bakkerme/ai-news-processor/internal/email"
	"github.com/bakkerme/ai-news-processor/internal/llm"
	"github.com/bakkerme/ai-news-processor/internal/models"
	"github.com/bakkerme/ai-news-processor/internal/openai"
	"github.com/bakkerme/ai-news-processor/internal/persona"
	"github.com/bakkerme/ai-news-processor/internal/prompts"
	"github.com/bakkerme/ai-news-processor/internal/qualityfilter"
	"github.com/bakkerme/ai-news-processor/internal/rss"
	"github.com/bakkerme/ai-news-processor/internal/specification"
)

func main() {
	s, err := specification.GetConfig()
	if err != nil {
		panic(err)
	}

	// Print the duration it took to run the job
	startTime := time.Now()
	defer func() {
		fmt.Printf("Job took %v\n", time.Since(startTime))
	}()

	// Initialize the OpenAI client
	openaiClient := openai.New(s.LlmUrl, s.LlmApiKey, s.LlmModel)

	// Initialize the image client if image processing is enabled
	var imageClient openai.OpenAIClient
	if s.LlmImageEnabled {
		imageClient = openai.New(s.LlmUrl, s.LlmApiKey, s.LlmImageModel)
		fmt.Println("Image processing enabled with model:", s.LlmImageModel)
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
		fmt.Println("Using mock feed provider")
		// Use the persona name from the first selected persona for mock data
		// Each persona will still use its own mock data in processing
		feedProvider = rss.NewMockFeedProvider(selectedPersonas[0].Name)
	} else {
		feedProvider = rss.NewFeedProvider()
	}

	// Process each persona
	for _, persona := range selectedPersonas {
		fmt.Printf("Processing persona: %s\n", persona.Name)

		// 1. Fetch and process RSS feed using FeedProvider
		entries, err := rss.FetchAndProcessFeed(feedProvider, persona.FeedURL, s.DebugRssDump, persona.Name)
		if err != nil {
			fmt.Printf("Failed to process RSS feed for persona %s: %v\n", persona.Name, err)
			continue
		}

		// Limit entries if DebugMaxEntries is set
		if s.DebugMaxEntries > 0 && len(entries) > s.DebugMaxEntries {
			entries = entries[:s.DebugMaxEntries]
		}

		// 2. Enrich entries with comments
		entries, err = rss.FetchAndEnrichWithComments(feedProvider, entries, s.DebugRssDump, persona.Name)
		if err != nil {
			fmt.Printf("Failed to enrich entries with comments for persona %s: %v\n", persona.Name, err)
			continue
		}

		// 3. Filter entries with quality filter
		entries = qualityfilter.Filter(entries, s.QualityFilterThreshold)

		// Store all raw inputs for benchmarking
		var benchmarkLLMInputs []string
		var items []models.Item

		// 4. Process entries with LLM
		if !s.DebugMockLLM {
			fmt.Println("Sending to LLM")
			systemPrompt, err := prompts.ComposePrompt(persona, "")
			if err != nil {
				fmt.Printf("Could not compose prompt for persona %s: %v\n", persona.Name, err)
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
			processor := llm.NewProcessor(openaiClient, imageClient, processorConfig)

			// Process the entries using the processor
			items, benchmarkLLMInputs, err = processor.ProcessEntries(systemPrompt, entries, persona)
			if err != nil {
				fmt.Printf("Could not process entries with LLM for persona %s: %v\n", persona.Name, err)
				continue
			}
		} else {
			fmt.Println("Loading fake LLM response")
			items = GetMockLLMResponse()
		}

		// 5. Enrich items with links from RSS entries
		items = llm.EnrichItems(items, entries)

		// Output benchmark data if requested
		if s.DebugOutputBenchmark {
			benchData := &bench.BenchmarkData{
				RawInput: benchmarkLLMInputs,
				Results:  items,
				Persona:  persona.Name,
			}
			outputBenchmarkData(benchData)
		}

		// 6. Filter for relevant items
		relevantItems := llm.FilterRelevantItems(items)
		if len(relevantItems) == 0 {
			fmt.Println("no items to render as an email")
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
			summaryResponse = GetMockSummaryResponse()
		}

		// 10. Render and send email
		if !s.DebugSkipEmail {
			err = emailService.RenderAndSend(relevantItems, summaryResponse, persona.Name)
			if err != nil {
				fmt.Printf("Could not send email for persona %s: %v\n", persona.Name, err)
				continue
			}
		} else {
			fmt.Println("Skipping email")
		}
	}
}

// outputBenchmarkData writes benchmark data to a file
func outputBenchmarkData(data *bench.BenchmarkData) {
	filename := "../bench/results/benchmark.json"

	// Ensure the directory exists
	err := os.MkdirAll("bench/results", 0755)
	if err != nil {
		fmt.Printf("Error creating benchmark directory: %v\n", err)
		return
	}

	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Error creating benchmark file: %v\n", err)
		return
	}
	defer file.Close()

	jsonData, err := bench.SerializeBenchmarkData(data)
	if err != nil {
		fmt.Printf("Error serializing benchmark data: %v\n", err)
		return
	}

	_, err = file.Write(jsonData)
	if err != nil {
		fmt.Printf("Error writing benchmark data: %v\n", err)
		return
	}

	fmt.Printf("Benchmark data written to %s\n", filename)
}
