package rss

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// Feedlike is an interface that can be used to represent any type that has a FeedString method, i.e. Feed and CommentFeed
type Feedlike interface {
	FeedString() string
}

// Feed and Comment Feed are used as intermediate types for RSS feeds
type Feed struct {
	Entries []Entry `xml:"entry"`
	RawRSS  string  // Added field to store raw RSS data
}

func (f *Feed) FeedString() string {
	return f.RawRSS // Method to return the raw RSS data
}

type CommentFeed struct {
	Entries []EntryComments `xml:"entry"`
	RawRSS  string          // Added field to store raw RSS data
}

func (cf *CommentFeed) FeedString() string {
	return cf.RawRSS // Method to return the raw RSS data
}

// Entry and EntryComments are used throughout the codebase for RSS feeds
type Entry struct {
	Title            string    `xml:"title"`
	Link             Link      `xml:"link"`
	ID               string    `xml:"id"`
	Published        time.Time `xml:"published"`
	Content          string    `xml:"content"`
	Comments         []EntryComments
	ImageURLs        []url.URL      // New field to store extracted image URLs
	MediaThumbnail   MediaThumbnail `xml:"http://search.yahoo.com/mrss/ thumbnail"` // Field to store thumbnail information from media namespace
	ImageDescription string         // Field to store image descriptions from dedicated image processing
}

type EntryComments struct {
	Content string `xml:"content"`
}

type Link struct {
	Href string `xml:"href,attr"`
}

// MediaThumbnail represents the media:thumbnail element in RSS feeds
type MediaThumbnail struct {
	URL string `xml:"url,attr"`
}

func (e *Entry) String(disableTruncation bool) string {
	var s strings.Builder
	s.WriteString(fmt.Sprintf("Title: %s\nID: %s\nSummary: %s\nImageDescription: %s\n",
		strings.Trim(e.Title, " "),
		e.ID,
		cleanContent(e.Content, 0, true),
		e.ImageDescription,
	))

	for _, comment := range e.Comments {
		s.WriteString(fmt.Sprintf("Comment: %s\n", cleanContent(comment.Content, 600, disableTruncation)))
	}

	return s.String()
}

func (e *Entry) GetCommentRSSURL() string {
	return fmt.Sprintf("%s.rss?depth=1", e.Link.Href)
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

// ExtractImageURLs finds image URLs in the entry content and stores them in the ImageURLs field.
func (e *Entry) ExtractImageURLs() error {
	// Reset the ImageURLs slice
	e.ImageURLs = nil
	urlMap := make(map[string]url.URL) // Use a map to automatically deduplicate

	// Parse the content HTML for regular <img> tags and <a> tags with image links
	doc, err := html.Parse(strings.NewReader(e.Content))
	if err != nil {
		return fmt.Errorf("failed to parse HTML content: %w", err)
	}

	// Traverse the DOM to find image URLs
	extractURLsFromNode(doc, urlMap)

	// Convert map to slice
	e.ImageURLs = make([]url.URL, 0, len(urlMap))
	for _, u := range urlMap {
		e.ImageURLs = append(e.ImageURLs, u)
	}

	return nil
}

// extractURLsFromNode recursively traverses HTML nodes to extract image URLs
func extractURLsFromNode(n *html.Node, urlMap map[string]url.URL) {
	if n.Type == html.ElementNode {
		// Check for <img> tags
		if n.Data == "img" {
			for _, a := range n.Attr {
				if a.Key == "src" {
					addImageURLIfValid(a.Val, urlMap)
					break
				}
			}
		}

		// Check for <a> tags with image links
		if n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					addImageURLIfValid(a.Val, urlMap)
					break
				}
			}
		}
	}

	// Continue traversing the DOM
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractURLsFromNode(c, urlMap)
	}
}

// addImageURLIfValid adds a URL to the map if it's a valid image URL
func addImageURLIfValid(urlStr string, urlMap map[string]url.URL) {
	if (isLikelyImageURL(urlStr) || hasImageExtension(urlStr)) && !containsExcludedTerms(urlStr) {
		validURL := ensureValidImageURL(urlStr)
		u, err := url.Parse(validURL)
		if err == nil {
			urlMap[u.String()] = *u
		}
	}
}

// isLikelyImageURL checks if a URL is likely an image based on extension or known image hosting patterns
func isLikelyImageURL(urlStr string) bool {
	// If it ends with a common image extension, it's definitely an image
	if hasImageExtension(urlStr) {
		return true
	}

	// Check for common image hosting patterns
	lowerURL := strings.ToLower(urlStr)

	// i.redd.it, i.imgur.com are dedicated image hosts
	if strings.Contains(lowerURL, "i.redd.it") ||
		strings.Contains(lowerURL, "i.imgur.com") {
		return true
	}

	return false
}

// hasImageExtension checks if a URL ends with a common image file extension
func hasImageExtension(urlStr string) bool {
	lowerURL := strings.ToLower(urlStr)
	return strings.HasSuffix(lowerURL, ".jpg") ||
		strings.HasSuffix(lowerURL, ".jpeg") ||
		strings.HasSuffix(lowerURL, ".png") ||
		strings.HasSuffix(lowerURL, ".gif") ||
		strings.HasSuffix(lowerURL, ".bmp") ||
		strings.HasSuffix(lowerURL, ".webp")
}

// containsExcludedTerms checks if a URL contains terms that indicate it's a low-quality image
func containsExcludedTerms(urlStr string) bool {
	lowerURL := strings.ToLower(urlStr)
	return strings.Contains(lowerURL, "thumb") || strings.Contains(lowerURL, "preview")
}
