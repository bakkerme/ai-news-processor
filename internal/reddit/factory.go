package reddit

import (
	"fmt"

	"github.com/bakkerme/ai-news-processor/internal/rss"
	"github.com/bakkerme/ai-news-processor/internal/specification"
)

// CreateFeedProvider creates appropriate feed provider based on configuration
func CreateFeedProvider(spec *specification.Specification, selectedPersonas []interface{}) (rss.FeedProvider, error) {
	// Handle mock RSS case first
	if spec.DebugMockRss {
		// For mock RSS, we need a persona name, but we don't have access to personas here
		// Return error to maintain the existing pattern in run.go
		return nil, fmt.Errorf("mock RSS provider should be created in run.go")
	}

	// Use Reddit API if enabled
	if spec.UseRedditAPI {
		provider, err := NewRedditAPIProvider(
			spec.RedditClientID,
			spec.RedditSecret,
			spec.RedditUsername,
			spec.RedditPassword,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create Reddit API provider: %w", err)
		}
		return provider, nil
	}

	// Default to RSS provider
	return rss.NewFeedProvider(), nil
}