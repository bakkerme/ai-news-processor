package reddit

import (
	"testing"
)

func TestExtractSubredditFromURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expected    string
		expectError bool
	}{
		{
			name:        "valid RSS URL",
			url:         "https://www.reddit.com/r/LocalLLaMA/.rss",
			expected:    "LocalLLaMA",
			expectError: false,
		},
		{
			name:        "valid RSS URL without .rss",
			url:         "https://www.reddit.com/r/cursor/",
			expected:    "cursor",
			expectError: false,
		},
		{
			name:        "invalid URL format",
			url:         "https://example.com/invalid",
			expected:    "",
			expectError: true,
		},
		{
			name:        "empty URL",
			url:         "",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractSubredditFromURL(tt.url)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestExtractSubredditFromPermalink(t *testing.T) {
	tests := []struct {
		name        string
		permalink   string
		expected    string
		expectError bool
	}{
		{
			name:        "valid permalink",
			permalink:   "https://www.reddit.com/r/LocalLLaMA/comments/abc123/title/",
			expected:    "LocalLLaMA",
			expectError: false,
		},
		{
			name:        "permalink without trailing slash",
			permalink:   "https://www.reddit.com/r/cursor/comments/def456/another-title",
			expected:    "cursor",
			expectError: false,
		},
		{
			name:        "invalid permalink format",
			permalink:   "https://example.com/invalid/path",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractSubredditFromPermalink(tt.permalink)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestIsImageURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "direct image URL",
			url:      "https://i.imgur.com/abc123.jpg",
			expected: true,
		},
		{
			name:     "reddit image URL",
			url:      "https://i.redd.it/xyz789.png",
			expected: true,
		},
		{
			name:     "imgur gallery URL",
			url:      "https://imgur.com/abc123",
			expected: true,
		},
		{
			name:     "non-image URL",
			url:      "https://example.com/article",
			expected: false,
		},
		{
			name:     "empty URL",
			url:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isImageURL(tt.url)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}