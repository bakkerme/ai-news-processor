package urlextraction

import (
	"fmt"
	"html"
	"net/url"
	"strings"

	xhtml "golang.org/x/net/html"
)

// ContentProvider defines the interface for objects that can provide content for URL extraction
type ContentProvider interface {
	GetID() string
	GetContent() string
}

// Extractor defines the interface for URL extraction from content providers
type Extractor interface {
	// ExtractExternalURLsFromEntries processes multiple content providers and returns a map of IDs to their external URLs
	ExtractExternalURLsFromEntries(entries []ContentProvider) (map[string][]url.URL, error)
	ExtractImageURLsFromEntries(entries []ContentProvider) (map[string][]url.URL, error)

	ExtractExternalURLsFromEntry(entry ContentProvider) ([]url.URL, error)
	ExtractImageURLsFromEntry(entry ContentProvider) ([]url.URL, error)
}

// RedditExtractor implements the Extractor interface for Reddit-specific URL extraction
type RedditExtractor struct{}

// NewRedditExtractor creates a new RedditExtractor instance
func NewRedditExtractor() *RedditExtractor {
	return &RedditExtractor{}
}

// ExtractExternalURLsFromEntries processes a slice of content providers and extracts external URLs
// from the Content field of each entry, filtering out reddit.com and redd.it URLs.
// It returns a map where the key is the Entry ID and the value is a slice of unique external URLs.
// This function is kept for potential batch processing needs but the primary task focuses on single entry processing.
func (re *RedditExtractor) ExtractExternalURLsFromEntries(entries []ContentProvider) (map[string][]url.URL, error) {
	results := make(map[string][]url.URL)

	for _, entry := range entries {
		if entry.GetID() == "" {
			// Potentially log a warning or handle entries with no ID if necessary
			continue
		}

		extractedUrls, err := re.ExtractExternalURLsFromEntry(entry)
		if err != nil {
			// For now, return error immediately. Consider collecting errors or partial results later.
			return nil, fmt.Errorf("error extracting external URLs for entry ID %s: %w", entry.GetID(), err)
		}

		results[entry.GetID()] = extractedUrls
	}

	return results, nil
}

// ExtractImageURLsFromEntries processes a slice of content providers and extracts image URLs
// from the Content field of each entry, filtering out URLs that are not likely images or contain excluded terms.
func (re *RedditExtractor) ExtractImageURLsFromEntries(entries []ContentProvider) (map[string][]url.URL, error) {
	results := make(map[string][]url.URL)

	for _, entry := range entries {
		extractedUrls, err := re.ExtractImageURLsFromEntry(entry)
		if err != nil {
			return nil, fmt.Errorf("error extracting image URLs for entry ID %s: %w", entry.GetID(), err)
		}

		results[entry.GetID()] = extractedUrls
	}

	return results, nil
}

// ExtractExternalURLsFromEntry processes a single content provider and extracts external URLs
// from its Content field. It filters out URLs belonging to reddit.com or redd.it.
func (re *RedditExtractor) ExtractExternalURLsFromEntry(entry ContentProvider) ([]url.URL, error) {
	allURLs, err := re.extractURLsFromEntry(entry)
	if err != nil {
		return nil, fmt.Errorf("error extracting external URLs from entry ID %s: %w", entry.GetID(), err)
	}

	var externalURLs []url.URL
	for _, u := range allURLs {
		isReddit, err := re.isRedditDomain(u.String())
		if err != nil {
			return nil, fmt.Errorf("error checking if URL is Reddit domain: %w", err)
		}
		if !isReddit {
			externalURLs = append(externalURLs, u)
		}
	}

	return externalURLs, nil
}

// ExtractImageURLsFromEntry processes a single content provider and extracts image URLs
// from its Content field. It filters out URLs that are not likely images or contain excluded terms.
func (re *RedditExtractor) ExtractImageURLsFromEntry(entry ContentProvider) ([]url.URL, error) {
	imageURLs, err := re.extractURLsFromEntry(entry)
	if err != nil {
		return nil, fmt.Errorf("error extracting image URLs from entry ID %s: %w", entry.GetID(), err)
	}

	var validImageURLs []url.URL
	for _, u := range imageURLs {
		validURL := ensureValidImageURL(u.String())
		if isLikelyImageURL(validURL) && !containsExcludedTerms(validURL) {
			u, err := url.Parse(validURL)
			if err == nil {
				validImageURLs = append(validImageURLs, *u)
			}
		}
	}
	return validImageURLs, nil
}

// extractURLsFromHTML extracts all href attributes from anchor tags and src attributes from img tags in an HTML string.
// It parses the HTML and traverses the node tree to find all <a> and <img> elements and their href/src attributes.
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
		if n.Type == xhtml.ElementNode {
			if n.Data == "a" {
				for _, a := range n.Attr {
					if a.Key == "href" {
						if a.Val != "" { // Ensure URL is not empty
							urls = append(urls, a.Val)
						}
						break
					}
				}
			} else if n.Data == "img" {
				for _, a := range n.Attr {
					if a.Key == "src" {
						if a.Val != "" { // Ensure URL is not empty
							urls = append(urls, a.Val)
						}
						break
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	// Filter out invalid or relative URLs
	var validURLs []string
	for _, u := range urls {
		parsed, err := url.Parse(u)
		if err == nil && parsed.IsAbs() {
			validURLs = append(validURLs, u)
		}
	}

	return validURLs, nil
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

	// Handle mailto schemes explicitly: they are not Reddit domains and don't have a host.
	if u.Scheme == "mailto" {
		return false, nil
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

// filterNonHTTPProtocols filters a slice of URL strings, returning only those with http or https schemes.
// Malformed URLs or those that cannot be parsed are also filtered out.
func filterNonHTTPProtocols(urls []string) []string {
	var httpURLs []string
	for _, urlStr := range urls {
		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			// Skip unparseable URLs
			continue
		}
		if parsedURL.Scheme == "http" || parsedURL.Scheme == "https" {
			httpURLs = append(httpURLs, urlStr)
		}
	}
	return httpURLs
}

// ExtractURLsFromEntry processes a single content provider and extracts external URLs
// from its Content field.
func (re *RedditExtractor) extractURLsFromEntry(entry ContentProvider) ([]url.URL, error) {
	allURLs, err := re.extractURLsFromHTML(entry.GetContent())
	if err != nil {
		return nil, fmt.Errorf("error extracting all URLs from entry ID %s: %w", entry.GetID(), err)
	}

	// Filter out non-HTTP/HTTPS URLs first
	httpURLs := filterNonHTTPProtocols(allURLs)

	var externalURLs []url.URL
	for _, u := range httpURLs {
		url, err := url.Parse(u)
		if err != nil {
			return nil, fmt.Errorf("error parsing URL: %w", err)
		}
		externalURLs = append(externalURLs, *url)
	}

	return externalURLs, nil
}

// isLikelyImageURL checks if a URL is likely an image based on extension or known image hosting patterns
func isLikelyImageURL(urlStr string) bool {
	// Check for common image hosting patterns
	lowerURL := strings.ToLower(urlStr)

	// i.redd.it, i.imgur.com are dedicated image hosts
	if strings.Contains(lowerURL, "i.redd.it") ||
		strings.Contains(lowerURL, "preview.redd.it") ||
		strings.Contains(lowerURL, "i.imgur.com") {
		return true
	}

	// Check for common image extensions
	return hasImageExtension(urlStr)
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
	return strings.Contains(lowerURL, "thumb") ||
		strings.Contains(lowerURL, "external-preview")
}

// ensureValidImageURL ensures a URL has a scheme (http:// or https://)
func ensureValidImageURL(imgURL string) string {
	if !strings.HasPrefix(imgURL, "http://") && !strings.HasPrefix(imgURL, "https://") {
		return "https://" + imgURL
	}
	return imgURL
}
