package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/bakkerme/ai-news-processor/internal/feeds"
)

// MockProvider implements the feeds.FeedProvider interface using JSON mock data
type MockProvider struct {
	PersonaName string
}

// NewMockProvider creates a new mock provider for the specified persona
func NewMockProvider(personaName string) *MockProvider {
	processedName := processPersonaName(personaName)
	return &MockProvider{
		PersonaName: processedName,
	}
}

// FetchFeed implements feeds.FeedProvider.FetchFeed for mocks
func (m *MockProvider) FetchFeed(ctx context.Context, subreddit string) (*feeds.Feed, error) {
	return m.GetMockFeed(ctx, m.PersonaName)
}

// FetchComments implements feeds.FeedProvider.FetchComments for mocks
func (m *MockProvider) FetchComments(ctx context.Context, entry feeds.Entry) (*feeds.CommentFeed, error) {
	return m.GetMockComments(ctx, m.PersonaName, entry.ID)
}

// GetMockFeed reads Reddit JSON mock data and converts to feeds.Feed format
func (m *MockProvider) GetMockFeed(ctx context.Context, personaName string) (*feeds.Feed, error) {
	processedName := processPersonaName(personaName)

	// Read JSON mock data
	path := filepath.Join("feed_mocks", "reddit", processedName, fmt.Sprintf("%s.json", processedName))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read mock feed: %w", err)
	}

	// Parse JSON data
	var feedData RedditFeedData
	if err := json.Unmarshal(data, &feedData); err != nil {
		return nil, fmt.Errorf("failed to parse mock feed: %w", err)
	}

	// Convert JSON data to feed entries
	entries := make([]feeds.Entry, len(feedData.Posts))
	for i, post := range feedData.Posts {
		entries[i] = mockPostToEntry(post)
	}

	feed := &feeds.Feed{
		Entries: entries,
		RawData: fmt.Sprintf("Mock feed for r/%s", feedData.Subreddit),
	}

	return feed, nil
}

// GetMockComments reads Reddit JSON comment mock data and converts to feeds.CommentFeed format
func (m *MockProvider) GetMockComments(ctx context.Context, personaName string, entryID string) (*feeds.CommentFeed, error) {
	processedName := processPersonaName(personaName)

	// Read JSON mock data
	path := filepath.Join("feed_mocks", "reddit", processedName, fmt.Sprintf("%s.json", entryID))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read mock comments: %w", err)
	}

	// Parse JSON data
	var commentData RedditCommentData
	if err := json.Unmarshal(data, &commentData); err != nil {
		return nil, fmt.Errorf("failed to parse mock comments: %w", err)
	}

	// Convert JSON data to feed comment entries
	// Filter for top-level comments only (matching Reddit API provider behavior)
	var commentEntries []feeds.EntryComments
	for _, comment := range commentData.Comments {
		// Only include top-level comments to match RSS behavior
		if comment.ParentID == "t3_"+entryID {
			commentEntries = append(commentEntries, feeds.EntryComments{
				Content: comment.Body,
			})
		}
	}

	commentFeed := &feeds.CommentFeed{
		Entries: commentEntries,
		RawData: fmt.Sprintf("Mock comments for post %s", entryID),
	}

	return commentFeed, nil
}

// mockPostToEntry converts a mock Reddit post to a feeds.Entry
func mockPostToEntry(post RedditPostData) feeds.Entry {
	entry := feeds.Entry{
		Title:     post.Title,
		ID:        post.ID,
		Published: post.Created,
		Content:   post.Body,
	}

	// Set the link - use full Reddit permalink
	entry.Link = feeds.Link{
		Href: fmt.Sprintf("https://www.reddit.com%s", post.Permalink),
	}

	// Handle different post types
	if post.IsSelf {
		// Text post - content is in Body (selftext)
		entry.Content = post.Body
	} else {
		// Link post - URL points to external content
		entry.Content = fmt.Sprintf("Link: %s", post.URL)

		// Extract external URLs
		if post.URL != "" {
			if parsedURL, err := url.Parse(post.URL); err == nil {
				entry.ExternalURLs = []url.URL{*parsedURL}
			}
		}
	}

	// Extract image URLs if this is an image post
	if !post.IsSelf && isImageURL(post.URL) {
		if parsedURL, err := url.Parse(post.URL); err == nil {
			entry.ImageURLs = []url.URL{*parsedURL}
		}
	}

	// Set media thumbnail if available
	if !post.IsSelf && isImageURL(post.URL) {
		entry.MediaThumbnail = feeds.MediaThumbnail{
			URL: post.URL,
		}
	}

	// Initialize empty maps/slices for compatibility
	if entry.ExternalURLs == nil {
		entry.ExternalURLs = []url.URL{}
	}
	if entry.ImageURLs == nil {
		entry.ImageURLs = []url.URL{}
	}
	if entry.WebContentSummaries == nil {
		entry.WebContentSummaries = make(map[string]string)
	}

	return entry
}
