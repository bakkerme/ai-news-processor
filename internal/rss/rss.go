package rss

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bakkerme/ai-news-processor/internal/common"
	strip "github.com/grokify/html-strip-tags-go"
)

// DefaultRSSRetryConfig provides default retry settings for RSS fetching
var DefaultRSSRetryConfig = common.RetryConfig{
	MaxRetries:      3,
	InitialBackoff:  1 * time.Second,
	MaxBackoff:      30 * time.Second,
	BackoffFactor:   2.0,
	MaxTotalTimeout: 1 * time.Minute,
}

// fetchWithRetry attempts to fetch a URL with exponential backoff retry
func fetchWithRetry(url string, config common.RetryConfig) (*http.Response, error) {
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
		if common.IsRateLimitError(resp) {
			retryAfter := common.GetRetryAfterDuration(resp)
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
	resp, err := common.RetryWithBackoff(ctx, config, fetchFn, shouldRetry)
	if err != nil {
		return nil, fmt.Errorf("failed after retries: %w", err)
	}

	return resp, nil
}

type Feed struct {
	Entries []Entry `xml:"entry"`
}

type Entry struct {
	Title     string    `xml:"title"`
	Link      Link      `xml:"link"`
	ID        string    `xml:"id"`
	Published time.Time `xml:"published"`
	Content   string    `xml:"content"`
	Comments  []EntryComments
}

type CommentFeed struct {
	Entries []EntryComments `xml:"entry"`
}

type EntryComments struct {
	Content string `xml:"content"`
}

type Link struct {
	Href string `xml:"href,attr"`
}

func (e *Entry) String(disableTruncation bool) string {
	var s strings.Builder
	s.WriteString(fmt.Sprintf("Title: %s\nID: %s\nSummary: %s\n",
		strings.Trim(e.Title, " "),
		e.ID,
		cleanContent(e.Content, 1200, disableTruncation),
	))

	for _, comment := range e.Comments {
		s.WriteString(fmt.Sprintf("Comment: %s\n", cleanContent(comment.Content, 600, disableTruncation)))
	}

	return s.String()
}

func (e *Entry) GetCommentRSSURL() string {
	return fmt.Sprintf("%s.rss?depth=1", e.Link.Href)
}

func ProcessRSSFeed(input string) (*Feed, error) {
	var feed Feed
	if err := xml.Unmarshal([]byte(input), &feed); err != nil {
		return nil, err
	}

	return &feed, nil
}

func ProcessCommentsRSSFeed(input string) (*CommentFeed, error) {
	var commentFeed CommentFeed
	if err := xml.Unmarshal([]byte(input), &commentFeed); err != nil {
		return nil, err
	}

	return &commentFeed, nil
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

// UnmarshalXML implements xml.Unmarshaler for custom time parsing
func (e *Entry) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type Alias Entry
	aux := &struct {
		Published string `xml:"published"`
		*Alias
	}{
		Alias: (*Alias)(e),
	}
	if err := d.DecodeElement(aux, &start); err != nil {
		return err
	}

	// Parse the time string
	if aux.Published != "" {
		t, err := time.Parse(time.RFC3339, aux.Published)
		if err != nil {
			return fmt.Errorf("failed to parse published time: %w", err)
		}
		e.Published = t
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

		feed, err := ProcessRSSFeed(rssString)
		if err != nil {
			return nil, fmt.Errorf("could not process RSS feed from %s: %w", url, err)
		}

		feeds = append(feeds, feed)
	}
	return feeds, nil
}

// GetMockFeeds returns mock RSS feeds for testing
func GetMockFeeds(personaName string) []*Feed {
	feedString := ReturnFakeRSS(personaName)
	feed, err := ProcessRSSFeed(feedString)
	if err != nil {
		panic(fmt.Sprintf("could not process mock feed: %v", err))
	}
	return []*Feed{feed}
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
