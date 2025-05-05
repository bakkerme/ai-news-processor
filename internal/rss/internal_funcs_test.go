package rss

import (
	"context"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestProcessRSSFeed(t *testing.T) {
	// Test valid RSS processing
	validRSS := `<feed><entry><title>Test Title</title><link href="http://example.com/1"/><id>1</id><published>2023-01-01T00:00:00Z</published><content>Test content</content></entry></feed>`
	feed := &Feed{}
	err := processRSSFeed(validRSS, feed)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(feed.Entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(feed.Entries))
	}
	if feed.Entries[0].Title != "Test Title" {
		t.Errorf("Expected title 'Test Title', got '%s'", feed.Entries[0].Title)
	}

	// Test invalid XML
	invalidRSS := `<feed><entry><broken>`
	feed = &Feed{}
	err = processRSSFeed(invalidRSS, feed)

	if err == nil {
		t.Fatal("Expected error for invalid XML, got none")
	}
}

func TestProcessCommentsRSSFeed(t *testing.T) {
	// Test valid comments RSS processing
	validComments := `<feed><entry><content>Comment 1</content></entry><entry><content>Comment 2</content></entry></feed>`
	commentFeed := &CommentFeed{}
	err := processCommentsRSSFeed(validComments, commentFeed)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(commentFeed.Entries) != 2 {
		t.Fatalf("Expected 2 comments, got %d", len(commentFeed.Entries))
	}
	if commentFeed.Entries[0].Content != "Comment 1" {
		t.Errorf("Expected content 'Comment 1', got '%s'", commentFeed.Entries[0].Content)
	}
}

func TestCleanContent(t *testing.T) {
	tests := []struct {
		name              string
		input             string
		maxLen            int
		disableTruncation bool
		expected          string
	}{
		{
			name:              "Basic HTML stripping",
			input:             "<p>Test content</p>",
			maxLen:            100,
			disableTruncation: false,
			expected:          "Test content",
		},
		{
			name:              "HTML entities conversion",
			input:             "Quote: &quot;Hello&#39;s&quot; text",
			maxLen:            100,
			disableTruncation: false,
			expected:          `Quote: "Hello's" text`,
		},
		{
			name:              "Truncation works",
			input:             "This is a long text that should be truncated",
			maxLen:            10,
			disableTruncation: false,
			expected:          "This is a ...",
		},
		{
			name:              "Truncation disabled",
			input:             "This is a long text that should not be truncated",
			maxLen:            10,
			disableTruncation: true,
			expected:          "This is a long text that should not be truncated",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := cleanContent(tc.input, tc.maxLen, tc.disableTruncation)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestFetchRSS(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/success" {
			w.Write([]byte("<feed><entry><title>Test</title></entry></feed>"))
		} else if r.URL.Path == "/rate-limit" {
			w.Header().Set("Retry-After", "2")
			w.WriteHeader(http.StatusTooManyRequests)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	// Test successful fetch
	rss, err := fetchRSS(server.URL + "/success")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if rss != "<feed><entry><title>Test</title></entry></feed>" {
		t.Errorf("Unexpected RSS content: %s", rss)
	}

	// Test server error
	_, err = fetchRSS(server.URL + "/error")
	if err == nil {
		t.Fatal("Expected error for server error, got none")
	}
}

func TestDefaultFeedProvider(t *testing.T) {
	var serverURL string

	// Create a test HTTP server with a mock RSS feed
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/feed") {
			w.Write([]byte(`<feed><entry><title>Test Entry</title><link href="` + serverURL + `/entry/1"/><id>entry1</id><published>2023-01-01T00:00:00Z</published><content>Test content</content></entry></feed>`))
		} else if strings.Contains(r.URL.Path, "/entry/") && strings.HasSuffix(r.URL.Path, ".rss") {
			w.Write([]byte(`<feed><entry><content>Test comment</content></entry></feed>`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	// Create provider
	provider := NewFeedProvider()

	// Test FetchFeed
	ctx := context.Background()
	feed, err := provider.FetchFeed(ctx, serverURL+"/feed")

	if err != nil {
		t.Fatalf("Expected no error from FetchFeed, got %v", err)
	}
	if len(feed.Entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(feed.Entries))
	}
	if feed.Entries[0].Title != "Test Entry" {
		t.Errorf("Expected title 'Test Entry', got '%s'", feed.Entries[0].Title)
	}

	// Test FetchComments
	commentFeed, err := provider.FetchComments(ctx, feed.Entries[0])

	if err != nil {
		t.Fatalf("Expected no error from FetchComments, got %v", err)
	}
	if len(commentFeed.Entries) != 1 {
		t.Fatalf("Expected 1 comment, got %d", len(commentFeed.Entries))
	}
	if commentFeed.Entries[0].Content != "Test comment" {
		t.Errorf("Expected content 'Test comment', got '%s'", commentFeed.Entries[0].Content)
	}
}

func TestFindEntryByID(t *testing.T) {
	entries := []Entry{
		{ID: "entry1", Title: "Entry 1"},
		{ID: "entry2", Title: "Entry 2"},
		{ID: "entry3", Title: "Entry 3"},
	}

	// Test finding an existing entry
	result := FindEntryByID("entry2", entries)
	if result == nil {
		t.Fatal("Expected to find entry2, got nil")
	}
	if result.Title != "Entry 2" {
		t.Errorf("Expected title 'Entry 2', got '%s'", result.Title)
	}

	// Test finding a non-existent entry
	result = FindEntryByID("entry4", entries)
	if result != nil {
		t.Errorf("Expected nil for non-existent entry, got %+v", result)
	}
}

func TestEntryString(t *testing.T) {
	entry := Entry{
		Title:   "Test Title",
		Link:    Link{Href: "http://example.com/1"},
		ID:      "entry1",
		Content: "<p>This is some <b>HTML</b> content</p>",
		Comments: []EntryComments{
			{Content: "<p>Comment 1</p>"},
			{Content: "<p>Comment 2</p>"},
		},
	}

	// Test with truncation enabled
	result := entry.String(false)
	if !strings.Contains(result, "Title: Test Title") {
		t.Errorf("Expected title in output, got: %s", result)
	}
	if !strings.Contains(result, "This is some HTML content") {
		t.Errorf("Expected cleaned content, got: %s", result)
	}
	if !strings.Contains(result, "Comment: Comment 1") {
		t.Errorf("Expected comment in output, got: %s", result)
	}

	// Test with truncation disabled
	result = entry.String(true)
	if !strings.Contains(result, "Title: Test Title") {
		t.Errorf("Expected title in output, got: %s", result)
	}
}

func TestGetCommentRSSURL(t *testing.T) {
	entry := Entry{
		Link: Link{Href: "http://example.com/entry/123"},
	}

	url := entry.GetCommentRSSURL()
	expected := "http://example.com/entry/123.rss?depth=1"

	if url != expected {
		t.Errorf("Expected URL '%s', got '%s'", expected, url)
	}
}

func TestEntryUnmarshalXML(t *testing.T) {
	// Test valid published date
	validXML := `<entry><title>Test</title><link href="http://example.com"/><id>1</id><published>2023-01-01T12:34:56Z</published><content>Test</content></entry>`
	var entry Entry
	err := xml.Unmarshal([]byte(validXML), &entry)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expectedTime := time.Date(2023, 1, 1, 12, 34, 56, 0, time.UTC)
	if !entry.Published.Equal(expectedTime) {
		t.Errorf("Expected time %v, got %v", expectedTime, entry.Published)
	}

	// Test invalid published date
	invalidXML := `<entry><title>Test</title><link href="http://example.com"/><id>1</id><published>invalid-date</published><content>Test</content></entry>`
	entry = Entry{}
	err = xml.Unmarshal([]byte(invalidXML), &entry)

	if err == nil {
		t.Fatal("Expected error for invalid date, got none")
	}
}
