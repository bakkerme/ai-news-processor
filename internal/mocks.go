package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/bakkerme/ai-news-processor/internal/bench"
	"github.com/bakkerme/ai-news-processor/internal/models"
	"github.com/bakkerme/ai-news-processor/internal/persona"
	"github.com/bakkerme/ai-news-processor/internal/rss"
)

func GetMockLLMResponse() []models.Item {
	// Look for llmresponse.json in the root directory
	jsonData, err := os.ReadFile("llmresponse.json")
	if err != nil {
		panic(err)
	}

	// Unmarshal the JSON string into an Item struct
	var items []models.Item
	if err := json.Unmarshal([]byte(jsonData), &items); err != nil {
		log.Fatalf("Error unmarshalling JSON: %v", err)
	}

	return items
}

func GetMockSummaryResponse(relevantItems []models.Item) *models.SummaryResponse {
	if len(relevantItems) == 0 {
		return &models.SummaryResponse{
			KeyDevelopments: []models.KeyDevelopment{},
		}
	}

	keyDevs := make([]models.KeyDevelopment, 0, len(relevantItems))
	for _, item := range relevantItems {
		// Create a simple key development text from the item's title.
		text := fmt.Sprintf("Mock summary for: %s", item.Title)
		// Add a snippet of the overview if available, keeping it concise for a mock.
		if len(item.Summary) > 75 {
			text = fmt.Sprintf("Mock summary for: %s - %s...", item.Title, item.Summary[:75])
		} else if len(item.Summary) > 0 {
			text = fmt.Sprintf("Mock summary for: %s - %s", item.Title, item.Summary)
		}

		keyDevs = append(keyDevs, models.KeyDevelopment{
			Text:   text,
			ItemID: item.ID,
		})
	}

	return &models.SummaryResponse{
		KeyDevelopments: keyDevs,
	}
}

func GetMockBenchmarkData(items []models.Item, personaObj persona.Persona, entries []rss.Entry) bench.BenchmarkData {
	entrySummaries := make([]bench.EntrySummary, len(items))
	var totalProcessingTime int64 = 0
	var entryTotalProcessingTime int64 = 0

	// Find the corresponding rss.Entry for each models.Item to populate RawInput
	entryMap := make(map[string]rss.Entry)
	for _, entry := range entries {
		entryMap[entry.ID] = entry
	}

	for i, item := range items {
		mockProcessingTime := int64(10 + i%5) // Mock processing time between 10-14ms
		rawInput := fmt.Sprintf("Mock raw input for item ID: %s, Title: %s", item.ID, item.Title)

		// If we found a corresponding entry, use its String() method for a more realistic RawInput
		if entry, ok := entryMap[item.ID]; ok {
			rawInput = entry.String(true) // Assuming String(true) gives a good representation
		}

		entrySummaries[i] = bench.EntrySummary{
			RawInput:       rawInput,
			Results:        item,
			ProcessingTime: mockProcessingTime,
		}
		entryTotalProcessingTime += mockProcessingTime
	}
	totalProcessingTime = entryTotalProcessingTime // Assuming only entry processing for mock

	return bench.BenchmarkData{
		EntrySummaries:                entrySummaries,
		ImageSummaries:                []bench.ImageSummary{},      // Empty for mock
		WebContentSummaries:           []bench.WebContentSummary{}, // Empty for mock
		Persona:                       personaObj,
		RunDate:                       time.Now(),
		OverallModelUsed:              "mock-llm-model",
		ImageModelUsed:                "mock-image-model",      // Or empty if not applicable
		WebContentModelUsed:           "mock-webcontent-model", // Or empty if not applicable
		TotalProcessingTime:           totalProcessingTime,
		EntryTotalProcessingTime:      entryTotalProcessingTime,
		ImageTotalProcessingTime:      0,   // Mocked as 0
		WebContentTotalProcessingTime: 0,   // Mocked as 0
		SuccessRate:                   1.0, // Assuming all mock items are "successful"
	}
}
