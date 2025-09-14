package providers

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bakkerme/ai-news-processor/internal/feeds"
	"github.com/bakkerme/ai-news-processor/internal/persona"
)

// MockProvider implements the feeds.FeedProvider interface using JSON mock data
type MockProvider struct {
	PersonaName string
}

// NewMockProvider creates a new mock provider for the specified persona
func NewMockProvider(personaName string) *MockProvider {
	processedName := processPersonaName(personaName)
	return &MockProvider{
		PersonaName: processedName,
	}
}

// FetchFeed implements feeds.FeedProvider.FetchFeed for mocks
func (m *MockProvider) FetchFeed(ctx context.Context, p persona.Persona) (*feeds.Feed, error) {
	return m.GetMockFeed(ctx, p)
}

// FetchComments implements feeds.FeedProvider.FetchComments for mocks
func (m *MockProvider) FetchComments(ctx context.Context, entry feeds.Entry) (*feeds.CommentFeed, error) {
	return m.GetMockComments(ctx, m.PersonaName, entry.ID)
}

// GetMockFeed reads mock data (JSON for Reddit, XML for RSS) and converts to feeds.Feed format
func (m *MockProvider) GetMockFeed(ctx context.Context, p persona.Persona) (*feeds.Feed, error) {
	processedName := processPersonaName(p.Name)

	// Determine provider type and load appropriate mock data
	providerType := p.GetProvider()
	switch providerType {
	case "reddit":
		return m.getMockRedditFeed(processedName)
	case "rss":
		return m.getMockRSSFeed(processedName, p.FeedURL)
	default:
		return nil, fmt.Errorf("unsupported provider type for mock: %s", providerType)
	}
}

// getMockRedditFeed reads Reddit JSON mock data and converts to feeds.Feed format
func (m *MockProvider) getMockRedditFeed(processedName string) (*feeds.Feed, error) {
	// Read JSON mock data
	path := filepath.Join("feed_mocks", "reddit", processedName, fmt.Sprintf("%s.json", processedName))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read Reddit mock feed: %w", err)
	}

	// Parse JSON data
	var feedData RedditFeedData
	if err := json.Unmarshal(data, &feedData); err != nil {
		return nil, fmt.Errorf("failed to parse Reddit mock feed: %w", err)
	}

	// Convert JSON data to feed entries
	entries := make([]feeds.Entry, len(feedData.Posts))
	for i, post := range feedData.Posts {
		entries[i] = mockPostToEntry(post)
	}

	feed := &feeds.Feed{
		Entries: entries,
		RawData: fmt.Sprintf("Mock feed for r/%s", feedData.Subreddit),
	}

	return feed, nil
}

// getMockRSSFeed reads RSS XML mock data and converts to feeds.Feed format
func (m *MockProvider) getMockRSSFeed(processedName string, feedURL string) (*feeds.Feed, error) {
	// Read XML mock data
	path := filepath.Join("feed_mocks", "rss", processedName, fmt.Sprintf("%s.xml", processedName))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read RSS mock feed: %w", err)
	}

	// Parse XML data using the same structures as RSS provider
	var rss RSSFeed
	if err := xml.Unmarshal(data, &rss); err != nil {
		return nil, fmt.Errorf("failed to parse RSS mock feed: %w", err)
	}

	// Convert RSS items to feed entries
	entries := make([]feeds.Entry, len(rss.Channel.Items))
	for i, item := range rss.Channel.Items {
		entries[i] = mockRSSItemToEntry(item)
	}

	feed := &feeds.Feed{
		Entries: entries,
		RawData: string(data), // Store raw XML for debugging
	}

	return feed, nil
}

// GetMockComments reads Reddit JSON comment mock data and converts to feeds.CommentFeed format
func (m *MockProvider) GetMockComments(ctx context.Context, personaName string, entryID string) (*feeds.CommentFeed, error) {
	processedName := processPersonaName(personaName)

	// Read JSON mock data
	path := filepath.Join("feed_mocks", "reddit", processedName, fmt.Sprintf("%s.json", entryID))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read mock comments: %w", err)
	}

	// Parse JSON data
	var commentData RedditCommentData
	if err := json.Unmarshal(data, &commentData); err != nil {
		return nil, fmt.Errorf("failed to parse mock comments: %w", err)
	}

	// Convert JSON data to feed comment entries
	// Filter for top-level comments only (matching Reddit API provider behavior)
	var commentEntries []feeds.EntryComments
	for _, comment := range commentData.Comments {
		// Only include top-level comments to match RSS behavior
		if comment.ParentID == "t3_"+entryID {
			commentEntries = append(commentEntries, feeds.EntryComments{
				Content: comment.Body,
			})
		}
	}

	commentFeed := &feeds.CommentFeed{
		Entries: commentEntries,
		RawData: fmt.Sprintf("Mock comments for post %s", entryID),
	}

	return commentFeed, nil
}

// mockPostToEntry converts a mock Reddit post to a feeds.Entry
func mockPostToEntry(post RedditPostData) feeds.Entry {
	entry := feeds.Entry{
		Title:     post.Title,
		ID:        post.ID,
		Published: post.Created,
		Content:   post.Body,
	}

	// Set the link - use full Reddit permalink
	entry.Link = feeds.Link{
		Href: fmt.Sprintf("https://www.reddit.com%s", post.Permalink),
	}

	// Handle different post types
	if post.IsSelf {
		// Text post - content is in Body (selftext)
		entry.Content = post.Body
	} else {
		// Link post - URL points to external content
		entry.Content = fmt.Sprintf("Link: %s", post.URL)

		// Extract external URLs
		if post.URL != "" {
			if parsedURL, err := url.Parse(post.URL); err == nil {
				entry.ExternalURLs = []url.URL{*parsedURL}
			}
		}
	}

	// Extract image URLs if this is an image post
	if !post.IsSelf && isImageURL(post.URL) {
		if parsedURL, err := url.Parse(post.URL); err == nil {
			entry.ImageURLs = []url.URL{*parsedURL}
		}
	}

	// Set media thumbnail if available
	if !post.IsSelf && isImageURL(post.URL) {
		entry.MediaThumbnail = feeds.MediaThumbnail{
			URL: post.URL,
		}
	}

	// Initialize empty maps/slices for compatibility
	if entry.ExternalURLs == nil {
		entry.ExternalURLs = []url.URL{}
	}
	if entry.ImageURLs == nil {
		entry.ImageURLs = []url.URL{}
	}
	if entry.WebContentSummaries == nil {
		entry.WebContentSummaries = make(map[string]string)
	}

	return entry
}

// mockRSSItemToEntry converts a mock RSS item to a feeds.Entry (same logic as RSS provider)
func mockRSSItemToEntry(item RSSItem) feeds.Entry {
	entry := feeds.Entry{
		Title:     item.Title,
		ID:        extractIDFromGUID(item.GUID),
		Content:   cleanHTMLContent(item.Description),
		Published: item.PubDate.Time,
	}

	// Set the link
	if item.Link != "" {
		entry.Link = feeds.Link{Href: item.Link}
	}

	// Extract external URLs from content
	entry.ExternalURLs = extractURLsFromContent(item.Description)

	// Extract image URLs from content
	entry.ImageURLs = extractImageURLsFromContent(item.Description)

	// Initialize empty maps/slices
	if entry.ExternalURLs == nil {
		entry.ExternalURLs = []url.URL{}
	}
	if entry.ImageURLs == nil {
		entry.ImageURLs = []url.URL{}
	}
	if entry.WebContentSummaries == nil {
		entry.WebContentSummaries = make(map[string]string)
	}

	return entry
}

// RSS XML structures for parsing (same as RSS provider)
type RSSFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Channel RSSChannel `xml:"channel"`
}

type RSSChannel struct {
	Title       string    `xml:"title"`
	Description string    `xml:"description"`
	Items       []RSSItem `xml:"item"`
}

type RSSItem struct {
	Title       string       `xml:"title"`
	Link        string       `xml:"link"`
	Description string       `xml:"description"`
	GUID        string       `xml:"guid"`
	PubDate     RSSTimestamp `xml:"pubDate"`
}

// RSSTimestamp handles various RSS date formats (same as RSS provider)
type RSSTimestamp struct {
	time.Time
}

// UnmarshalXML implements custom time parsing for RSS pubDate
func (t *RSSTimestamp) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var content string
	if err := d.DecodeElement(&content, &start); err != nil {
		return err
	}

	// Try various RSS date formats
	formats := []string{
		time.RFC1123Z,               // Mon, 02 Jan 2006 15:04:05 -0700
		time.RFC1123,                // Mon, 02 Jan 2006 15:04:05 MST
		time.RFC822Z,                // 02 Jan 06 15:04 -0700
		time.RFC822,                 // 02 Jan 06 15:04 MST
		"2006-01-02T15:04:05Z",      // ISO format
		"2006-01-02T15:04:05-07:00", // ISO with timezone
		"2006-01-02 15:04:05",       // Simple format
	}

	for _, format := range formats {
		if parsed, err := time.Parse(format, content); err == nil {
			t.Time = parsed
			return nil
		}
	}

	// If all parsing fails, use current time and log warning
	// log.Printf("Warning: Failed to parse date '%s', using current time", content)
	t.Time = time.Now()
	return nil
}

// Helper functions from RSS provider
func extractIDFromGUID(guid string) string {
	if strings.HasPrefix(guid, "http") {
		parts := strings.Split(strings.TrimRight(guid, "/"), "/")
		if len(parts) > 0 {
			lastPart := parts[len(parts)-1]
			if idx := strings.Index(lastPart, "?"); idx != -1 {
				lastPart = lastPart[:idx]
			}
			if idx := strings.Index(lastPart, "#"); idx != -1 {
				lastPart = lastPart[:idx]
			}
			if lastPart != "" {
				return lastPart
			}
		}
	}

	guid = strings.TrimSpace(guid)
	if len(guid) > 50 {
		return fmt.Sprintf("id_%d", len(guid)+int(guid[0]))
	}

	return guid
}

func cleanHTMLContent(content string) string {
	// Simple HTML tag removal for mock data
	content = strings.ReplaceAll(content, "<", "&lt;")
	content = strings.ReplaceAll(content, ">", "&gt;")
	content = strings.ReplaceAll(content, "&#39;", "'")
	content = strings.ReplaceAll(content, "&quot;", "\"")
	content = strings.ReplaceAll(content, "&amp;", "&")
	return strings.TrimSpace(content)
}

func extractURLsFromContent(content string) []url.URL {
	// Simple URL extraction for mock data
	return []url.URL{}
}

func extractImageURLsFromContent(content string) []url.URL {
	// Simple image URL extraction for mock data
	return []url.URL{}
}
