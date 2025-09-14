package rss

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/bakkerme/ai-news-processor/internal/feeds"
	"github.com/bakkerme/ai-news-processor/internal/persona"
)

// RSSProvider implements the feeds.FeedProvider interface for generic RSS feeds
// This provider can work with any standards-compliant RSS feed, making the
// ai-news-processor a generic system for news processing beyond Reddit
type RSSProvider struct {
	httpClient *http.Client
	enableDump bool
}

// NewRSSProvider creates a new generic RSS provider
func NewRSSProvider(enableDump bool) *RSSProvider {
	return &RSSProvider{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		enableDump: enableDump,
	}
}

// FetchFeed implements feeds.FeedProvider.FetchFeed for RSS feeds
func (r *RSSProvider) FetchFeed(ctx context.Context, p persona.Persona) (*feeds.Feed, error) {
	// TODO: Extract RSS URL from persona - for now use a placeholder
	// This will be updated when persona struct is extended with RSS fields
	rssURL := "" // Will be extracted from persona.FeedURL when available
	if rssURL == "" {
		return nil, fmt.Errorf("RSS URL not configured for persona %s - persona system needs RSS URL field", p.Name)
	}

	log.Printf("Fetching generic RSS feed from %s for persona %s", rssURL, p.Name)

	// Fetch RSS content
	rssContent, err := r.fetchRSSContent(ctx, rssURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch RSS content: %w", err)
	}

	// Parse RSS content
	feed, err := r.parseRSSFeed(rssContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSS feed: %w", err)
	}

	// Set raw data for debugging
	feed.RawData = fmt.Sprintf("RSS feed from %s", rssURL)

	return feed, nil
}

// FetchComments implements feeds.FeedProvider.FetchComments for RSS feeds
// Note: Most generic RSS feeds do not support comments, so this returns an empty comment feed
func (r *RSSProvider) FetchComments(ctx context.Context, entry feeds.Entry) (*feeds.CommentFeed, error) {
	// Generic RSS feeds typically don't have comment feeds
	// Return empty comment feed to satisfy interface requirements
	log.Printf("Generic RSS feeds do not support comments for entry %s", entry.ID)
	return &feeds.CommentFeed{
		Entries: []feeds.EntryComments{},
		RawData: fmt.Sprintf("Comments not supported for generic RSS entry %s", entry.ID),
	}, nil
}

// fetchRSSContent retrieves RSS content from a URL
func (r *RSSProvider) fetchRSSContent(ctx context.Context, rssURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", rssURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent to identify as a generic RSS reader
	req.Header.Set("User-Agent", "ai-news-processor/1.0 (Generic RSS Reader)")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch RSS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

// parseRSSFeed parses RSS XML into a feeds.Feed
func (r *RSSProvider) parseRSSFeed(rssContent string) (*feeds.Feed, error) {
	var rss RSSFeed
	if err := xml.Unmarshal([]byte(rssContent), &rss); err != nil {
		return nil, fmt.Errorf("failed to unmarshal RSS: %w", err)
	}

	entries := make([]feeds.Entry, len(rss.Channel.Items))
	for i, item := range rss.Channel.Items {
		entries[i] = r.rssItemToEntry(item)
	}

	return &feeds.Feed{
		Entries: entries,
	}, nil
}

// rssItemToEntry converts an RSS item to a feeds.Entry
func (r *RSSProvider) rssItemToEntry(item RSSItem) feeds.Entry {
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

// extractIDFromGUID extracts an ID from RSS GUID for generic RSS feeds
func extractIDFromGUID(guid string) string {
	// Try to extract a meaningful ID from the GUID
	// First try: if it's a URL, use the last path segment
	if strings.HasPrefix(guid, "http") {
		parts := strings.Split(strings.TrimRight(guid, "/"), "/")
		if len(parts) > 0 {
			lastPart := parts[len(parts)-1]
			// Remove common file extensions and query parameters
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

	// Fallback: use the full GUID, but clean it up
	guid = strings.TrimSpace(guid)
	// If it's still a URL, try to make a shorter ID
	if len(guid) > 50 {
		// Create a simple hash-like ID from the GUID
		return fmt.Sprintf("id_%d", len(guid)+int(guid[0]))
	}

	return guid
}

// cleanHTMLContent removes HTML tags and entities from content
func cleanHTMLContent(content string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]+>`)
	cleaned := re.ReplaceAllString(content, "")

	// Clean HTML entities
	cleaned = strings.ReplaceAll(cleaned, "&#39;", "'")
	cleaned = strings.ReplaceAll(cleaned, "&#32;", " ")
	cleaned = strings.ReplaceAll(cleaned, "&quot;", "\"")
	cleaned = strings.ReplaceAll(cleaned, "&amp;", "&")
	cleaned = strings.ReplaceAll(cleaned, "&lt;", "<")
	cleaned = strings.ReplaceAll(cleaned, "&gt;", ">")

	return strings.TrimSpace(cleaned)
}

// extractURLsFromContent extracts URLs from HTML content for generic RSS feeds
func extractURLsFromContent(content string) []url.URL {
	var urls []url.URL

	// Extract URLs from href attributes
	hrefRegex := regexp.MustCompile(`href="([^"]+)"`)
	matches := hrefRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 1 {
			if parsedURL, err := url.Parse(match[1]); err == nil {
				// Include all external URLs (no filtering)
				if parsedURL.Host != "" && (parsedURL.Scheme == "http" || parsedURL.Scheme == "https") {
					urls = append(urls, *parsedURL)
				}
			}
		}
	}

	return urls
}

// extractImageURLsFromContent extracts image URLs from HTML content
func extractImageURLsFromContent(content string) []url.URL {
	var urls []url.URL

	// Extract URLs from img src attributes
	imgRegex := regexp.MustCompile(`<img[^>]+src="([^"]+)"`)
	matches := imgRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 1 {
			if parsedURL, err := url.Parse(match[1]); err == nil {
				if isImageURL(parsedURL.String()) {
					urls = append(urls, *parsedURL)
				}
			}
		}
	}

	return urls
}

// isImageURL checks if a URL points to an image (generic implementation)
func isImageURL(urlStr string) bool {
	lowerURL := strings.ToLower(urlStr)
	imageExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}

	// Check for image file extensions
	for _, ext := range imageExtensions {
		if strings.HasSuffix(lowerURL, ext) || strings.Contains(lowerURL, ext+"?") {
			return true
		}
	}

	return false
}

// RSS XML structures for parsing
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

// RSSTimestamp handles various RSS date formats
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
	log.Printf("Warning: Failed to parse date '%s', using current time", content)
	t.Time = time.Now()
	return nil
}
