package main

import (
	"encoding/json"
	"fmt"
	"github.com/bakkerme/ai-news-processor/internal/common"
	"github.com/bakkerme/ai-news-processor/internal/rss"
	"io"
	"net/http"
	"os"
)

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

func getRSSEntryWithID(id string, entries []rss.Entry) *rss.Entry {
	for _, entry := range entries {
		if entry.ID == id {
			return &entry
		}
	}

	return nil
}

func outputBenchmark(items []common.Item) error {
	data, err := json.Marshal(items)
	if err != nil {
		return err
	}
	return os.WriteFile("./bench/benchmark.json", data, 0644)
}
