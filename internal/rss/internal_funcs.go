package rss

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bakkerme/ai-news-processor/internal/http/retry"
	strip "github.com/grokify/html-strip-tags-go"
)

// DefaultRSSRetryConfig provides default retry settings for RSS fetching
var DefaultRSSRetryConfig = retry.RetryConfig{
	MaxRetries:      3,
	InitialBackoff:  1 * time.Second,
	MaxBackoff:      30 * time.Second,
	BackoffFactor:   2.0,
	MaxTotalTimeout: 1 * time.Minute,
}

// fetchRSS retrieves RSS content from a URL
func fetchRSS(url string) (string, error) {
	resp, err := fetchWithRetry(url, DefaultRSSRetryConfig)
	if err != nil {
		return "", fmt.Errorf("could not fetch RSS: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("could not read response body: %w", err)
	}

	return string(body), nil
}

func processRSSFeed(input string, feed *Feed) error {
	feed.RawRSS = input // Store the raw RSS data
	if err := xml.Unmarshal([]byte(input), feed); err != nil {
		return err
	}

	return nil
}

func processCommentsRSSFeed(input string, commentFeed *CommentFeed) error {
	commentFeed.RawRSS = input // Store the raw RSS data
	if err := xml.Unmarshal([]byte(input), commentFeed); err != nil {
		return err
	}

	return nil
}

func cleanContent(s string, maxLen int, disableTruncation bool) string {
	stripped := strip.StripTags(s)
	stripped = strings.ReplaceAll(stripped, "&#39;", "'")
	stripped = strings.ReplaceAll(stripped, "&#32;", " ")
	stripped = strings.ReplaceAll(stripped, "&quot;", "\"")

	if disableTruncation {
		return stripped
	}

	lenToUse := maxLen
	strLen := len(stripped)

	if strLen < lenToUse {
		lenToUse = strLen
	}

	truncated := stripped[0:lenToUse]

	// Tack a ... on the end to signify it's truncated to the llm
	if lenToUse != strLen {
		truncated += "..."
	}

	return truncated
}

// fetchWithRetry attempts to fetch a URL with exponential backoff retry
func fetchWithRetry(url string, config retry.RetryConfig) (*http.Response, error) {
	ctx := context.Background()

	// Define the retryable function that performs the HTTP request
	fetchFn := func(ctx context.Context) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to execute request: %w", err)
		}

		// Check for rate limiting
		if retry.IsRateLimitError(resp) {
			retryAfter := retry.GetRetryAfterDuration(resp)
			resp.Body.Close() // Close the body before returning error
			return nil, fmt.Errorf("rate limited, retry after %v", retryAfter)
		}

		if resp.StatusCode >= 400 {
			resp.Body.Close() // Close the body before returning error
			return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
		}

		return resp, nil
	}

	// Define retry condition
	shouldRetry := func(err error) bool {
		if err == nil {
			return false
		}
		// Retry on network errors and rate limits
		return strings.Contains(err.Error(), "rate limited") ||
			strings.Contains(err.Error(), "connection refused") ||
			strings.Contains(err.Error(), "no such host") ||
			strings.Contains(err.Error(), "timeout")
	}

	// Execute with retry
	resp, err := retry.RetryWithBackoff(ctx, config, fetchFn, shouldRetry)
	if err != nil {
		return nil, fmt.Errorf("failed after retries: %w", err)
	}

	return resp, nil
}

// dumpFeed saves the raw RSS content to disk for debugging purposes
func dumpFeed(feedURL string, content Feedlike, personaName, itemName string) error {
	fmt.Printf("Dumping RSS for %s\n", feedURL)

	feedString := content.FeedString()

	// Create a safe filename from the itemName
	filename := itemName + ".rss"

	// Create the directory path
	dir := filepath.Join("..", "feed_mocks", "rss", personaName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write the content to file
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(feedString), 0644); err != nil {
		return fmt.Errorf("failed to write RSS content: %w", err)
	}

	return nil
}

// ensureValidImageURL ensures a URL has a scheme (http:// or https://)
func ensureValidImageURL(imgURL string) string {
	if !strings.HasPrefix(imgURL, "http://") && !strings.HasPrefix(imgURL, "https://") {
		return "https://" + imgURL
	}
	return imgURL
}
