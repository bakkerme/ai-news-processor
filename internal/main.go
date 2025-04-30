package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/bakkerme/ai-news-processor/internal/common"
	"github.com/bakkerme/ai-news-processor/internal/email"
	"github.com/bakkerme/ai-news-processor/internal/llm"
	"github.com/bakkerme/ai-news-processor/internal/openai"
	"github.com/bakkerme/ai-news-processor/internal/persona"
	"github.com/bakkerme/ai-news-processor/internal/prompts"
	"github.com/bakkerme/ai-news-processor/internal/rss"
	"github.com/bakkerme/ai-news-processor/internal/specification"
	"github.com/bakkerme/ai-news-processor/internal/summary"
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

	// Process each persona
	for _, persona := range selectedPersonas {
		fmt.Printf("Processing persona: %s\n", persona.Name)

		// 1. Fetch and process RSS feed
		entries, err := rss.FetchAndProcessFeed(persona.FeedURL, s.DebugMockRss)
		if err != nil {
			panic(fmt.Errorf("failed to process RSS feed for persona %s: %w", persona.Name, err))
		}

		// Limit entries if DebugMaxEntries is set
		if s.DebugMaxEntries > 0 && len(entries) > s.DebugMaxEntries {
			entries = entries[:s.DebugMaxEntries]
		}

		// 2. Enrich entries with comments
		entries, err = rss.EnrichWithComments(entries, s.DebugMockRss)
		if err != nil {
			panic(fmt.Errorf("failed to enrich entries with comments for persona %s: %w", persona.Name, err))
		}

		// Store all raw inputs for benchmarking
		var benchmarkInputs []string
		var items []common.Item

		// 3. Process entries with LLM
		if !s.DebugMockLLM {
			fmt.Println("Sending to LLM")
			systemPrompt, err := prompts.ComposePrompt(persona)
			if err != nil {
				panic(fmt.Errorf("could not compose prompt for persona %s: %w", persona.Name, err))
			}

			batchSize := 1

			items, benchmarkInputs, err = llm.ProcessEntries(openaiClient, systemPrompt, entries, batchSize, s.DebugOutputBenchmark)
			if err != nil {
				panic(fmt.Errorf("could not process entries with LLM: %w", err))
			}
		} else {
			fmt.Println("Loading fake LLM response")
			items = GetMockLLMResponse()
		}

		// 4. Enrich items with links from RSS entries
		items = llm.EnrichItems(items, entries)

		// Output benchmark data if requested
		if s.DebugOutputBenchmark {
			benchData := &common.BenchmarkData{
				RawInput: benchmarkInputs,
				Results:  items,
				Persona:  persona.Name,
			}
			outputBenchmarkData(benchData)
		}

		// 5. Filter for relevant items
		relevantItems := llm.FilterRelevantItems(items)
		if len(relevantItems) == 0 {
			panic("no items to render as an email")
		}

		// 6. Get relevant entries for summary
		relevantEntries := make([]rss.Entry, 0, len(relevantItems))
		for _, item := range relevantItems {
			entry := rss.FindEntryByID(item.ID, entries)
			if entry != nil {
				relevantEntries = append(relevantEntries, *entry)
			}
		}

		// 7. Generate summary for relevant items
		var summaryResponse *common.SummaryResponse
		if !s.DebugMockLLM {
			summaryResponse, err = summary.Generate(openaiClient, relevantEntries, persona)
			if err != nil {
				panic(fmt.Errorf("could not generate summary: %w", err))
			}
		} else {
			// Mock summary for debug mode
			summaryResponse = GetMockSummaryResponse()
		}

		// 8. Render and send email
		if !s.DebugSkipEmail {
			err = emailService.RenderAndSend(relevantItems, summaryResponse)
			if err != nil {
				panic(fmt.Errorf("could not send email: %w", err))
			}
		} else {
			fmt.Println("Skipping email")
		}
	}
}

// outputBenchmarkData writes benchmark data to a file
func outputBenchmarkData(data *common.BenchmarkData) {
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

	jsonData, err := common.SerializeBenchmarkData(data)
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
