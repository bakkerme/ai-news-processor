package feeds

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Feedlike is an interface that can be used to represent any type that has a FeedString method
type Feedlike interface {
	FeedString() string
}

// Feed represents a collection of entries from a feed source
type Feed struct {
	Entries []Entry `json:"entries"`
	RawData string  `json:"rawData,omitempty"` // Raw data from the source (JSON, XML, etc.)
}

func (f *Feed) FeedString() string {
	return f.RawData
}

// CommentFeed represents a collection of comments for a specific entry
type CommentFeed struct {
	Entries []EntryComments `json:"entries"`
	RawData string          `json:"rawData,omitempty"` // Raw data from the source
}

func (cf *CommentFeed) FeedString() string {
	return cf.RawData
}

// Entry represents a single content item (post, article, etc.)
type Entry struct {
	Title               string            `json:"title"`
	Link                Link              `json:"link"`
	ID                  string            `json:"id"`
	Published           time.Time         `json:"published"`
	Content             string            `json:"content"`
	Comments            []EntryComments   `json:"comments"`
	ExternalURLs        []url.URL         `json:"externalURLs"`        // External URLs found in content
	ImageURLs           []url.URL         `json:"imageURLs"`           // Extracted image URLs
	MediaThumbnail      MediaThumbnail    `json:"mediaThumbnail"`      // Thumbnail information
	ImageDescription    string            `json:"imageDescription"`    // Generated image descriptions
	WebContentSummaries map[string]string `json:"webContentSummaries"` // Summaries of external URLs
}

// EntryComments represents a comment on an entry
type EntryComments struct {
	Content string `json:"content"`
}

// Link represents a link with an href
type Link struct {
	Href string `json:"href"`
}

// MediaThumbnail represents thumbnail information
type MediaThumbnail struct {
	URL string `json:"url"`
}

// String generates a string representation of the Entry for processing
func (e *Entry) String(disableTruncation bool) string {
	var s strings.Builder
	s.WriteString(fmt.Sprintf("Title: %s\nID: %s\nContent: %s\nImageDescription: %s\n",
		strings.Trim(e.Title, " "),
		e.ID,
		cleanContent(e.Content, 1200, disableTruncation),
		e.ImageDescription,
	))

	if len(e.ExternalURLs) > 0 {
		s.WriteString("\nExternal URLs:\n")
		for _, url := range e.ExternalURLs {
			s.WriteString(fmt.Sprintf("- %s\n", url.String()))
		}
	}

	if len(e.WebContentSummaries) > 0 {
		s.WriteString("\nExternal URL Summaries:\n")
		for url, summary := range e.WebContentSummaries {
			s.WriteString(fmt.Sprintf("- %s: %s\n", url, summary))
		}
	}

	s.WriteString("Comments:\n")
	for _, comment := range e.Comments {
		s.WriteString(fmt.Sprintf("- %s\n", cleanContent(comment.Content, 600, disableTruncation)))
	}

	return s.String()
}

// GetCommentURL returns a URL for fetching comments (Reddit-specific implementation)
func (e *Entry) GetCommentURL() string {
	return fmt.Sprintf("%s.rss?depth=1", e.Link.Href)
}

// GetID returns the Entry's ID, implementing the ContentProvider interface
func (e Entry) GetID() string {
	return e.ID
}

// GetContent returns the Entry's Content, implementing the ContentProvider interface
func (e Entry) GetContent() string {
	return e.Content
}

// cleanContent cleans and optionally truncates content
func cleanContent(s string, maxLen int, disableTruncation bool) string {
	// Basic HTML entity cleanup
	cleaned := strings.ReplaceAll(s, "&#39;", "'")
	cleaned = strings.ReplaceAll(cleaned, "&#32;", " ")
	cleaned = strings.ReplaceAll(cleaned, "&quot;", "\"")
	cleaned = strings.ReplaceAll(cleaned, "&amp;", "&")
	cleaned = strings.ReplaceAll(cleaned, "&lt;", "<")
	cleaned = strings.ReplaceAll(cleaned, "&gt;", ">")

	if disableTruncation {
		return cleaned
	}

	lenToUse := maxLen
	strLen := len(cleaned)

	if strLen < lenToUse {
		lenToUse = strLen
	}

	truncated := cleaned[0:lenToUse]

	// Add ellipsis if truncated
	if lenToUse != strLen {
		truncated += "..."
	}

	return truncated
}