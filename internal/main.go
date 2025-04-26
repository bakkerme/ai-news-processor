package main

import (
	"fmt"
	"time"

	"github.com/bakkerme/ai-news-processor/internal/common"
	"github.com/bakkerme/ai-news-processor/internal/email"
	"github.com/bakkerme/ai-news-processor/internal/openai"
	"github.com/bakkerme/ai-news-processor/internal/rss"
)

func main() {
	s, err := GetConfig()
	if err != nil {
		panic(err)
	}

	emailer, err := email.New(s.EmailHost, s.EmailPort, s.EmailUsername, s.EmailPassword, s.EmailFrom)
	if err != nil {
		panic(fmt.Errorf("could not set up emailer: %w", err))
	}

	openaiClient := openai.New(s.LlmUrl, s.LlmApiKey, s.LlmModel)

	// Print the duration it took to run the job
	startTime := time.Now()
	defer func() {
		fmt.Printf("Job took %v\n", time.Since(startTime))
	}()

	rssString := ""
	if !s.DebugMockRss {
		fmt.Println("Loading RSS feed")
		rssString, err = getMainRSS()
		if err != nil {
			panic(fmt.Errorf("failed to load rss data %w", err))
		}
	} else {
		fmt.Println("Loading Mock RSS feed")
		rssString = rss.ReturnFakeRSS()
	}

	rssFeed, err := rss.ProcessRSSFeed(rssString)
	if err != nil {
		panic(fmt.Errorf("could not process rss feed: %w", err))
	}

	entries := rssFeed.Entries

	if len(entries) == 0 {
		panic("no entries found")
	}

	// Limit entries if DebugMaxEntries is set
	if s.DebugMaxEntries > 0 && len(entries) > s.DebugMaxEntries {
		entries = entries[:s.DebugMaxEntries]
	}

	for i, entry := range entries {
		commentFeedString := ""
		if !s.DebugMockRss {
			commentFeedString, err = getCommentRSS(entry)
			if err != nil {
				panic(fmt.Errorf("failed to load rss comment data %w", err))
			}
		} else {
			commentFeedString = rss.ReturnFakeCommentRSS(entry.ID)
		}

		commentFeed, err := rss.ProcessCommentsRSSFeed(commentFeedString)
		if err != nil {
			panic(fmt.Errorf("could not process rss coment feed: %w", err))
		}

		entry.Comments = commentFeed.Entries
		entries[i] = entry
	}

	items := make([]common.Item, len(entries))
	if !s.DebugMockLLM {
		fmt.Println("Sending to LLM")

		completionChannel := make(chan common.ErrorString, len(entries))
		systemPrompt := getSystemPrompt()

		batchCounter := 0
		batchSize := 5
		for i := 0; i < len(entries); i += batchSize {
			batch := entries[i:min(i+batchSize, len(entries))]

			fmt.Printf("Sending batch %d with %d items\n", i/batchSize, len(batch))

			batchStrings := make([]string, len(batch))
			for j, entry := range batch {
				batchStrings[j] = entry.String()
			}

			go openaiClient.Query(systemPrompt, batchStrings, completionChannel)
			batchCounter++
		}

		for i := 0; i < batchCounter; i++ {
			fmt.Printf("Waiting for batch %d\n", i)
			result := <-completionChannel
			if result.Err != nil {
				panic(fmt.Errorf("could not process value from LLM for entry %d: %s", i, result.Err))
			}

			fmt.Println(result.Value)

			processedValue := openaiClient.PreprocessJSON(result.Value)

			item, err := llmResponseToItem(processedValue)
			if err != nil {
				panic(fmt.Errorf("could not convert llm output to json. %s: %w", result.Value, err))
			}

			fmt.Printf("Processed batch %d, found %d items\n", i, len(item))
			items = append(items, item...)
		}
	} else {
		fmt.Println("Loading fake LLM response")
		items = returnFakeLLMResponse()
	}

	// add the link from the RSS Entries to the Items
	for i, item := range items {
		id := item.ID
		if id == "" {
			continue
		}

		entry := getRSSEntryWithID(id, entries)
		if entry == nil {
			fmt.Printf("could not find item with ID %s in RSS entry\n", id)
			continue
		}

		items[i].Link = entry.Link.Href
	}

	if s.DebugOutputBenchmark {
		itemsToInclude := []common.Item{}
		for _, item := range items {
			if item.IsRelevant && item.ID != "" {
				if item.ID != "" {
					itemsToInclude = append(itemsToInclude, item)
				}
			}
		}

		outputBenchmark(itemsToInclude)
	}

	itemsToInclude := []common.Item{}
	for _, item := range items {
		if item.IsRelevant && item.ID != "" {
			itemsToInclude = append(itemsToInclude, item)
		}
	}

	if len(itemsToInclude) == 0 {
		panic("no items render as an email")
	}

	// Generate summary for relevant items
	fmt.Println("Generating summary of relevant items")
	relevantEntries := make([]rss.Entry, 0, len(itemsToInclude))
	for _, item := range itemsToInclude {
		if entry := getRSSEntryWithID(item.ID, entries); entry != nil {
			relevantEntries = append(relevantEntries, *entry)
		}
	}

	// Create input for summary
	summaryInputs := make([]string, len(relevantEntries))
	for i, entry := range relevantEntries {
		summaryInputs[i] = entry.String()
	}

	summaryChannel := make(chan common.ErrorString, 1)
	go openaiClient.QueryForSummary(getSummarySystemPrompt(), summaryInputs, summaryChannel)

	summaryResult := <-summaryChannel
	if summaryResult.Err != nil {
		panic(fmt.Errorf("could not generate summary: %w", summaryResult.Err))
	}

	processedSummary := openaiClient.PreprocessJSON(summaryResult.Value)
	summary, err := openaiClient.ParseSummaryResponse(processedSummary)
	if err != nil {
		panic(fmt.Errorf("could not parse summary response: %w", err))
	}

	email, err := email.RenderEmail(itemsToInclude, summary)
	if err != nil {
		panic(fmt.Errorf("could not render email: %w", err))
	}

	if !s.DebugSkipEmail {
		fmt.Printf("Sending email to %s\n", s.EmailTo)
		emailer.Send(s.EmailTo, "AI News", email)
	} else {
		writeEmailToDisk(email)
	}
}
