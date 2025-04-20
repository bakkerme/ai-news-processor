package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/bakkerme/ai-news-processor/internal/common"
	"github.com/bakkerme/ai-news-processor/internal/email"
	"github.com/bakkerme/ai-news-processor/internal/openai"
	"github.com/bakkerme/ai-news-processor/internal/rss"
)

func main() {
	s, err := GetConfig()
	if err != nil {
		panic(s)
	}

	emailer, err := email.New(s.EmailHost, s.EmailPort, s.EmailUsername, s.EmailPassword, s.EmailFrom)
	if err != nil {
		panic(fmt.Errorf("could not set up emailer: %w", err))
	}

	openaiClient := openai.NewOpenAIClient(s.LlmUrl, s.LlmApiKey, s.LlmModel)

	rssString := ""
	if !s.DebugMockRss {
		fmt.Println("Loading RSS feed")
		rssString, err = getRSS()
		if err != nil {
			panic(fmt.Errorf("failed to load rss data %w", err))
		}
	} else {
		fmt.Println("Loading Mock RSS feed")
		rssString = rss.ReturnFakeRSS()
	}

	rss, err := rss.ProcessRSSFeed(rssString)
	if err != nil {
		panic(fmt.Errorf("could not process rss feed: %w", err))
	}

	// fmt.Println(rss)
	// for _, r := range rss.Entries {
	// fmt.Printf("%+v", r)
	// fmt.Println(r.String())
	// }

	items := make([]common.Item, len(rss.Entries))
	if !s.DebugMockLLM {
		fmt.Println("Sending to LLM")

		completionChannel := make(chan common.ErrorString, len(rss.Entries))
		systemPrompt := getSystemPrompt()

		batchCounter := 0
		batchSize := 5
		for i := 0; i < len(rss.Entries); i += batchSize {
			batch := rss.Entries[i:min(i+batchSize, len(rss.Entries))]

			fmt.Printf("Sending batch %d with %d items\n", i/batchSize, len(batch))

			batchStrings := make([]string, len(batch))
			for j, entry := range batch {
				batchStrings[j] = entry.String()
			}

			// openaiClient.Query(systemPrompt, batchStrings, completionChannel)
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

	toInclude := []common.Item{}
	// for _, item := range items {
	// if item.ShouldThisBeIncluded {
	// toInclude = append(toInclude, item)
	// }
	// }

	email, err := email.RenderEmail(toInclude)
	if err != nil {
		panic(fmt.Errorf("could not render email: %w", err))
	}

	if !s.DebugMockSkipEmail {
		fmt.Printf("Sending email to %s", s.EmailTo)
		emailer.Send(s.EmailTo, "AI News", email)
	} else {
		// fmt.Println(email)
	}
}

func getRSS() (string, error) {
	resp, err := http.Get("https://reddit.com/r/localllama.rss")
	if err != nil {
		return "", fmt.Errorf("could not get from reddit rss: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("could not load response body: %w", err)
	}

	return string(body), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
