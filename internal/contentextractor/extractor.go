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

// ArticleExtractor defines the interface for extracting article data from an HTML source.
type ArticleExtractor interface {
	Extract(body io.Reader, sourceURL *url.URL) (*ArticleData, error)
}

// DefaultArticleExtractor is the default implementation of ArticleExtractor
// that uses the go-readability library.
type DefaultArticleExtractor struct{}

// Extract calls the package-level ExtractArticle function.
func (d *DefaultArticleExtractor) Extract(body io.Reader, sourceURL *url.URL) (*ArticleData, error) {
	// Check for nil inputs
	if body == nil {
		return nil, fmt.Errorf("contentextractor: body cannot be nil")
	}
	if sourceURL == nil {
		return nil, fmt.Errorf("contentextractor: sourceURL cannot be nil")
	}

	article, err := readability.FromReader(body, sourceURL)
	if err != nil {
		return nil, fmt.Errorf("contentextractor: failed to extract article using go-readability from %s: %w", sourceURL.String(), err)
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
