package rss

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
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

// FetchRSS retrieves RSS content from a URL
func FetchRSS(url string) (string, error) {
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

func ProcessRSSFeed(input string, feed *Feed) error {
	feed.RawRSS = input // Store the raw RSS data
	if err := xml.Unmarshal([]byte(input), feed); err != nil {
		return err
	}

	return nil
}

func ProcessCommentsRSSFeed(input string, commentFeed *CommentFeed) error {
	commentFeed.RawRSS = input // Store the raw RSS data
	if err := xml.Unmarshal([]byte(input), commentFeed); err != nil {
		return err
	}

	return nil
}

// GetFeeds retrieves RSS feeds from the provided URLs
func GetFeeds(urls []string) ([]*Feed, error) {
	var feeds []*Feed
	for _, url := range urls {
		rssString, err := FetchRSS(url)
		if err != nil {
			return nil, fmt.Errorf("could not fetch RSS from %s: %w", url, err)
		}

		feed := &Feed{}
		err = ProcessRSSFeed(rssString, feed)
		if err != nil {
			return nil, fmt.Errorf("could not process RSS feed from %s: %w", url, err)
		}

		feeds = append(feeds, feed)
	}
	return feeds, nil
}

// GetMockFeeds returns mock RSS feeds for testing
func GetMockFeeds(personaName string) []*Feed {
	feed := ReturnFakeRSS(personaName)
	return []*Feed{feed}
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
