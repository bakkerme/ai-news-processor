package rss

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"regexp"
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
	Title          string    `xml:"title"`
	Link           Link      `xml:"link"`
	ID             string    `xml:"id"`
	Published      time.Time `xml:"published"`
	Content        string    `xml:"content"`
	Comments       []EntryComments
	ImageURLs      []url.URL      // New field to store extracted image URLs
	MediaThumbnail MediaThumbnail `xml:"http://search.yahoo.com/mrss/ thumbnail"` // Field to store thumbnail information from media namespace
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

	// Check for image links in the "link" span elements that appear at the end of Reddit posts
	// These often look like: <span><a href="https://i.redd.it/image.jpeg">[link]</a></span>
	linkURLs := extractLinkURLsFromContent(e.Content)
	for _, linkURL := range linkURLs {
		if isLikelyImageURL(linkURL) && !containsExcludedTerms(linkURL) {
			u, parseErr := url.Parse(linkURL)
			if parseErr == nil {
				e.ImageURLs = append(e.ImageURLs, *u)
			}
		}
	}

	// Parse the content HTML for regular <img> tags
	doc, err := html.Parse(strings.NewReader(e.Content))
	if err != nil {
		return fmt.Errorf("failed to parse HTML content: %w", err)
	}

	var extract func(*html.Node)
	extract = func(n *html.Node) {
		// Check for <img> tags
		if n.Type == html.ElementNode && n.Data == "img" {
			for _, a := range n.Attr {
				if a.Key == "src" {
					imgURL := a.Val
					// Use more relaxed validation for img tags since these are definitely images
					if !containsExcludedTerms(imgURL) {
						u, parseErr := url.Parse(imgURL)
						if parseErr == nil {
							e.ImageURLs = append(e.ImageURLs, *u)
						}
					}
					break
				}
			}
		}

		// Check for <a> tags with image links
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					linkURL := a.Val
					// Check if the link directly points to an image
					if isLikelyImageURL(linkURL) && !containsExcludedTerms(linkURL) {
						u, parseErr := url.Parse(linkURL)
						if parseErr == nil {
							e.ImageURLs = append(e.ImageURLs, *u)
						}
					}
					break
				}
			}
		}

		// Continue traversing the DOM
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}

	extract(doc)

	// Deduplicate image URLs
	e.ImageURLs = deduplicateURLs(e.ImageURLs)

	return nil
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

// extractLinkURLsFromContent looks for URLs in [link] elements common in Reddit feeds
func extractLinkURLsFromContent(content string) []string {
	// Simple regex to find links in patterns like <span><a href="URL">[link]</a></span>
	re := regexp.MustCompile(`<span>\s*<a href="([^"]+)">\s*\[link\]\s*</a>\s*</span>`)
	matches := re.FindAllStringSubmatch(content, -1)

	var urls []string
	for _, match := range matches {
		if len(match) >= 2 {
			urls = append(urls, match[1])
		}
	}
	return urls
}

// deduplicateURLs removes duplicate URLs from a slice
func deduplicateURLs(urls []url.URL) []url.URL {
	urlMap := make(map[string]url.URL)
	for _, u := range urls {
		urlMap[u.String()] = u
	}

	deduplicated := make([]url.URL, 0, len(urlMap))
	for _, u := range urlMap {
		deduplicated = append(deduplicated, u)
	}
	return deduplicated
}
