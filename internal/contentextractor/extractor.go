package contentextractor

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/go-shiori/go-readability"
)

// ArticleData holds the extracted information from a web page.
type ArticleData struct {
	Title       string
	CleanedText string
	// Future fields: Excerpt, SiteName, Favicon, Language, etc.
}

// ExtractArticle uses go-readability to extract the main content and metadata from an HTML page.
// It takes an io.Reader for the HTML body and the original page URL.
func ExtractArticle(htmlContent io.Reader, pageURL *url.URL) (*ArticleData, error) {
	// Check for nil inputs
	if htmlContent == nil {
		return nil, fmt.Errorf("contentextractor: htmlContent cannot be nil")
	}
	if pageURL == nil {
		return nil, fmt.Errorf("contentextractor: pageURL cannot be nil")
	}

	article, err := readability.FromReader(htmlContent, pageURL)
	if err != nil {
		return nil, fmt.Errorf("contentextractor: failed to extract article using go-readability from %s: %w", pageURL.String(), err)
	}

	// TODO: Add logic to check if content is substantial enough
	// For example, if article.TextContent is too short, return an error or a specific status.

	// strip out excessive whitespace
	cleanedText := strings.TrimSpace(article.TextContent)
	cleanedText = strings.ReplaceAll(cleanedText, "\n", " ")
	cleanedText = strings.ReplaceAll(cleanedText, "\r", " ")
	cleanedText = strings.ReplaceAll(cleanedText, "\t", " ")

	return &ArticleData{
		Title:       article.Title,
		CleanedText: cleanedText,
	}, nil
}
