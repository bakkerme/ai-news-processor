package contentextractor

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractArticle(t *testing.T) {
	tests := []struct {
		name              string
		htmlFile          string
		urlStr            string
		expectTitle       string
		expectTextSnippet string
		expectError       bool
	}{
		{
			name:              "Clean article",
			htmlFile:          "clean_article.html",
			urlStr:            "https://example.com/article",
			expectTitle:       "Sample Clean Article: AI Advancements in 2025",
			expectTextSnippet: "Researchers at leading tech companies have announced breakthrough advancements",
			expectError:       false,
		},
		{
			name:              "Complex page with boilerplate",
			htmlFile:          "complex_page.html",
			urlStr:            "https://example.com/tech/quantum-computing",
			expectTitle:       "Tech News Central - Quantum Computing Breakthrough",
			expectTextSnippet: "Scientists at Quantum Labs have achieved a significant breakthrough",
			expectError:       false,
		},
		{
			name:              "Invalid HTML",
			htmlFile:          "invalid.html",
			urlStr:            "https://example.com/invalid",
			expectTitle:       "",
			expectTextSnippet: "This is not valid HTML content",
			expectError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Read the HTML file
			htmlPath := filepath.Join("testdata", tt.htmlFile)
			htmlContent, err := os.ReadFile(htmlPath)
			if err != nil {
				t.Fatalf("Failed to read test file %s: %v", htmlPath, err)
			}

			// Parse the URL
			testURL, err := url.Parse(tt.urlStr)
			if err != nil {
				t.Fatalf("Failed to parse URL %s: %v", tt.urlStr, err)
			}

			// Call the function to test
			result, err := ExtractArticle(strings.NewReader(string(htmlContent)), testURL)

			// Check error expectation
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// If we expect no error, verify the result
			if !tt.expectError && result != nil {
				// Check title
				if result.Title != tt.expectTitle {
					t.Errorf("Title mismatch\nExpected: %s\nGot: %s", tt.expectTitle, result.Title)
				}

				// Check for text content snippet
				if !strings.Contains(result.CleanedText, tt.expectTextSnippet) {
					t.Errorf("Expected text to contain: %s\nGot: %s", tt.expectTextSnippet, result.CleanedText)
				}
			}
		})
	}
}

func TestExtractArticleErrors(t *testing.T) {
	// Test with nil reader
	t.Run("Nil reader", func(t *testing.T) {
		testURL, _ := url.Parse("https://example.com")
		_, err := ExtractArticle(nil, testURL)
		if err == nil {
			t.Error("Expected error with nil reader, got none")
		}
	})

	// Test with nil URL
	t.Run("Nil URL", func(t *testing.T) {
		_, err := ExtractArticle(strings.NewReader("<html><body>Test</body></html>"), nil)
		if err == nil {
			t.Error("Expected error with nil URL, got none")
		}
	})
}
