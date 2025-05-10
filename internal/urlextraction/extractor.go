package urlextraction

import (
	"fmt"
	"html"
	"net/url"
	"strings"

	xhtml "golang.org/x/net/html"

	"github.com/bakkerme/ai-news-processor/internal/rss"
)

// Extractor defines the interface for URL extraction from RSS entries
type Extractor interface {
	// ExtractURLsFromEntry processes a single RSS entry and returns a slice of external URLs
	ExtractURLsFromEntry(entry rss.Entry) ([]string, error)
	// ExtractURLsFromEntries processes multiple RSS entries and returns a map of entry IDs to their external URLs
	ExtractURLsFromEntries(entries []rss.Entry) (map[string][]string, error)
}

// RedditExtractor implements the Extractor interface for Reddit-specific URL extraction
type RedditExtractor struct{}

// NewRedditExtractor creates a new RedditExtractor instance
func NewRedditExtractor() *RedditExtractor {
	return &RedditExtractor{}
}

// ExtractURLsFromEntries processes a slice of RSS entries and extracts external URLs
// from the Content field of each entry, filtering out reddit.com and redd.it URLs.
// It returns a map where the key is the Entry ID and the value is a slice of unique external URLs.
// This function is kept for potential batch processing needs but the primary task focuses on single entry processing.
func (re *RedditExtractor) ExtractURLsFromEntries(entries []rss.Entry) (map[string][]string, error) {
	results := make(map[string][]string)

	for _, entry := range entries {
		if entry.ID == "" {
			// Potentially log a warning or handle entries with no ID if necessary
			continue
		}

		extractedUrls, err := re.ExtractURLsFromEntry(entry)
		if err != nil {
			// For now, return error immediately. Consider collecting errors or partial results later.
			return nil, fmt.Errorf("error extracting external URLs for entry ID %s: %w", entry.ID, err)
		}

		results[entry.ID] = extractedUrls
	}

	return results, nil
}

// extractURLsFromHTML extracts all href attributes from anchor tags in an HTML string.
// It parses the HTML and traverses the node tree to find all <a> elements and their href attributes.
func (re *RedditExtractor) extractURLsFromHTML(htmlContent string) ([]string, error) {
	if strings.TrimSpace(htmlContent) == "" {
		return []string{}, nil
	}

	// First unescape any HTML entities in the content
	unescaped := html.UnescapeString(htmlContent)

	doc, err := xhtml.Parse(strings.NewReader(unescaped))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML content: %w", err)
	}

	var urls []string
	var f func(*xhtml.Node)
	f = func(n *xhtml.Node) {
		if n.Type == xhtml.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					if a.Val != "" { // Ensure URL is not empty
						urls = append(urls, a.Val)
					}
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return urls, nil
}

// isRedditDomain checks if the given URL belongs to any Reddit domain.
func (re *RedditExtractor) isRedditDomain(urlStr string) (bool, error) {
	if urlStr == "" {
		return false, nil
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		// This case handles completely unparseable strings.
		return false, fmt.Errorf("failed to parse URL: %w", err)
	}

	// url.Parse can successfully parse strings that are not valid absolute URLs
	// (e.g., "not-a-url" becomes u.Path = "not-a-url", u.Host = "").
	// We consider a URL valid for domain checking only if it has a scheme and a host.
	if u.Scheme == "" || u.Host == "" {
		return false, fmt.Errorf("invalid URL for domain check: %s", urlStr)
	}

	host := strings.ToLower(u.Hostname())
	return strings.Contains(host, "reddit") || strings.Contains(host, "redd.it"), nil
}

// ExtractURLsFromEntry processes a single RSS entry and extracts external URLs
// from its Content field. It filters out URLs belonging to reddit.com or redd.it.
func (re *RedditExtractor) ExtractURLsFromEntry(entry rss.Entry) ([]string, error) {
	allURLs, err := re.extractURLsFromHTML(entry.Content)
	if err != nil {
		return nil, fmt.Errorf("error extracting all URLs from entry ID %s: %w", entry.ID, err)
	}

	var externalURLs []string
	for _, u := range allURLs {
		isReddit, err := re.isRedditDomain(u)
		if err != nil {
			// Log or handle URL parsing errors for individual URLs if needed
			// For now, we can skip malformed URLs that cannot be parsed for domain checking
			// Or return the error: return nil, fmt.Errorf("error checking domain for URL %s: %w", u, err)
			fmt.Printf("Warning: Skipping URL '%s' due to parsing error: %v\n", u, err) // Example logging
			continue
		}
		if !isReddit {
			externalURLs = append(externalURLs, u)
		}
	}

	return externalURLs, nil
}
