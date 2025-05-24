package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/bakkerme/ai-news-processor/internal/persona"
	"github.com/bakkerme/ai-news-processor/internal/rss"
	"github.com/bakkerme/ai-news-processor/models"
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

func GetMockBenchmarkData(items []models.Item, personaObj persona.Persona, entries []rss.Entry) models.RunData {
	// First, enrich the items with Entry field (like in real processing)
	enrichedItems := make([]models.Item, len(items))
	copy(enrichedItems, items)

	// Create entry map for efficient lookup
	entryMap := make(map[string]rss.Entry)
	for _, entry := range entries {
		entryMap[entry.ID] = entry
	}

	// Enrich items with Entry field and other data
	for i, item := range enrichedItems {
		if item.ID != "" {
			if entry, ok := entryMap[item.ID]; ok {
				enrichedItems[i].Entry = entry
				enrichedItems[i].Link = entry.Link.Href
				if len(entry.ImageURLs) > 0 {
					enrichedItems[i].ThumbnailURL = entry.ImageURLs[0].String()
				} else if entry.MediaThumbnail.URL != "" {
					enrichedItems[i].ThumbnailURL = entry.MediaThumbnail.URL
				}
			}
		}
	}

	// Initialize slices
	entrySummaries := make([]models.EntrySummary, len(enrichedItems))
	imageSummaries := make([]models.ImageSummary, 0)
	webContentSummaries := make([]models.WebContentSummary, 0)

	// Timing variables
	var entryTotalProcessingTime int64 = 0
	var imageTotalProcessingTime int64 = 0
	var webContentTotalProcessingTime int64 = 0

	// Process each item to create mock benchmark data
	for i, item := range enrichedItems {
		// Mock entry processing time (10-50ms range)
		entryProcessingTime := int64(10 + (i%5)*10)
		rawInput := fmt.Sprintf("Mock raw input for item ID: %s, Title: %s", item.ID, item.Title)

		// If we found a corresponding entry, use its String() method for a more realistic RawInput
		if item.Entry.ID != "" {
			rawInput = item.Entry.String(true)
		}

		entrySummaries[i] = models.EntrySummary{
			RawInput:       rawInput,
			Results:        item,
			ProcessingTime: entryProcessingTime,
		}
		entryTotalProcessingTime += entryProcessingTime

		// Mock image processing if item has image data
		if item.ImageSummary != "" {
			imageProcessingTime := int64(1000) // 1 minute
			imageURL := item.Entry.ImageURLs[0].String()

			imageSummary := models.ImageSummary{
				ImageURL:         imageURL,
				ImageDescription: item.ImageSummary,
				Title:            item.Title,
				EntryID:          item.ID,
				ProcessingTime:   imageProcessingTime,
			}
			imageSummaries = append(imageSummaries, imageSummary)
			imageTotalProcessingTime += imageProcessingTime
		}

		// Mock web content processing if item has web content data
		if item.WebContentSummary != "" {
			webProcessingTime := int64(200 + (i%4)*75) // 200-425ms range
			webURL := item.Entry.Link.Href
			mockOriginalContent := fmt.Sprintf("Mock original web content for %s. This would be the extracted article text that was summarized. Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.", item.Title)

			// Use actual link if available
			if item.Link != "" {
				webURL = item.Link
			}

			parsedURL, err := url.Parse(webURL)
			if err != nil {
				log.Fatalf("Error parsing URL: %v", err)
			}

			webSummary := models.WebContentSummary{
				URL:             *parsedURL,
				OriginalContent: mockOriginalContent,
				Summary:         item.WebContentSummary,
				Title:           fmt.Sprintf("Web Content for %s", item.Title),
				EntryID:         item.ID,
				ProcessingTime:  webProcessingTime,
			}
			webContentSummaries = append(webContentSummaries, webSummary)
			webContentTotalProcessingTime += webProcessingTime
		}
	}

	// Calculate total processing time
	totalProcessingTime := entryTotalProcessingTime + imageTotalProcessingTime + webContentTotalProcessingTime

	// Generate overall summary for relevant items
	relevantItems := make([]models.Item, 0)
	for _, item := range enrichedItems {
		if item.IsRelevant && item.ID != "" {
			relevantItems = append(relevantItems, item)
		}
	}
	overallSummary := GetMockSummaryResponse(relevantItems)

	// Calculate success rate (assume all items processed successfully for mock)
	successRate := 1.0
	if len(items) > 0 {
		successRate = float64(len(enrichedItems)) / float64(len(items))
	}

	return models.RunData{
		EntrySummaries:                entrySummaries,
		ImageSummaries:                imageSummaries,
		WebContentSummaries:           webContentSummaries,
		OverallSummary:                overallSummary,
		Persona:                       personaObj,
		RunDate:                       time.Now(),
		OverallModelUsed:              "mock-llm-model",
		ImageModelUsed:                "mock-image-model",
		WebContentModelUsed:           "mock-webcontent-model",
		TotalProcessingTime:           totalProcessingTime,
		EntryTotalProcessingTime:      entryTotalProcessingTime,
		ImageTotalProcessingTime:      imageTotalProcessingTime,
		WebContentTotalProcessingTime: webContentTotalProcessingTime,
		SuccessRate:                   successRate,
	}
}
