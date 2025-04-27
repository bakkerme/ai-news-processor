package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/bakkerme/ai-news-processor/internal/common"
	"github.com/bakkerme/ai-news-processor/internal/rss"
)

func getMainRSS() (string, error) {
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

func getCommentRSS(entry rss.Entry) (string, error) {
	resp, err := http.Get(entry.GetCommentRSSURL())
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

func outputBenchmark(data *common.BenchmarkData) error {
	// Marshal the benchmark data
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal benchmark data: %w", err)
	}

	// Write to benchmark.json
	err = os.WriteFile("../bench/benchmark.json", jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write benchmark data: %w", err)
	}

	return nil
}

func writeEmailToDisk(email string) error {
	return os.WriteFile("./email.html", []byte(email), 0644)
}
