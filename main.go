package main

import (
	"fmt"
	"io"
	"net/http"
)

func main() {
	openaiClient := NewOpenAIClient("http://192.168.1.115:8080/v1", "")

	// rss, err := getRSS()
	// if err != nil {
	// panic(fmt.Errorf("failed to load rss data %w", err))
	// }

	rssString := returnFakeRSS()
	rss, err := processRSSFeed(rssString)
	if err != nil {
		panic(err)
	}

	completionChannel := make(chan string, len(rss.Entries))
	systemPrompt := getSystemPrompt()

	for _, rssEntry := range rss.Entries {
		userPrompt := entryToString(rssEntry)
		go openaiClient.Query(systemPrompt, userPrompt, completionChannel)
	}

	items := make([]Item, len(rss.Entries))
	for i := range rss.Entries {
		result := <-completionChannel
		// fmt.Println(result)
		item := llmResponseToItem(result)

		fmt.Printf("Processed %d\n", i)
		items[i] = item
	}

	for _, item := range items {
		if item.ShouldThisBeIncluded {
			fmt.Println(item.FormatItem())
		}
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
