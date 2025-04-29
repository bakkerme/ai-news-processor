package llm

import (
	"encoding/json"
	"fmt"

	"github.com/bakkerme/ai-news-processor/internal/common"
	"github.com/bakkerme/ai-news-processor/internal/openai"
	"github.com/bakkerme/ai-news-processor/internal/rss"
)

// ProcessEntries takes RSS entries, processes them through an LLM in batches, and returns processed items
func ProcessEntries(client *openai.Client, systemPrompt string, entries []rss.Entry, batchSize int, debugOutputBenchmark bool) ([]common.Item, []string, error) {
	var items []common.Item
	var benchmarkInputs []string

	completionChannel := make(chan common.ErrorString, len(entries))
	batchCounter := 0

	// Process entries in batches
	for i := 0; i < len(entries); i += batchSize {
		batch := entries[i:min(i+batchSize, len(entries))]
		fmt.Printf("Sending batch %d with %d items\n", i/batchSize, len(batch))

		batchStrings := make([]string, len(batch))
		for j, entry := range batch {
			batchStrings[j] = entry.String(false)
		}

		// Store inputs for benchmarking
		if debugOutputBenchmark {
			benchmarkInputs = append(benchmarkInputs, batchStrings...)
		}

		go client.QueryForEntrySummary(systemPrompt, batchStrings, completionChannel)
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
func EnrichItems(items []common.Item, entries []rss.Entry) []common.Item {
	enrichedItems := make([]common.Item, len(items))
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
func FilterRelevantItems(items []common.Item) []common.Item {
	var relevantItems []common.Item
	for _, item := range items {
		if item.IsRelevant && item.ID != "" {
			relevantItems = append(relevantItems, item)
		}
	}
	return relevantItems
}

// llmResponseToItems converts a JSON LLM response to a slice of Items
func llmResponseToItems(jsonStr string) ([]common.Item, error) {
	var items []common.Item
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
