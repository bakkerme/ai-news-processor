package reddit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/bakkerme/ai-news-processor/internal/rss"
)

// RedditMockProvider implements the rss.FeedProvider interface using JSON mock data
type RedditMockProvider struct {
	PersonaName string
}

// NewRedditMockProvider creates a new Reddit mock provider for the specified persona
func NewRedditMockProvider(personaName string) *RedditMockProvider {
	return &RedditMockProvider{
		PersonaName: personaName,
	}
}

// FetchFeed implements rss.FeedProvider.FetchFeed for Reddit mocks
func (m *RedditMockProvider) FetchFeed(ctx context.Context, url string) (*rss.Feed, error) {
	return m.GetMockFeed(ctx, m.PersonaName)
}

// FetchComments implements rss.FeedProvider.FetchComments for Reddit mocks
func (m *RedditMockProvider) FetchComments(ctx context.Context, entry rss.Entry) (*rss.CommentFeed, error) {
	return m.GetMockComments(ctx, m.PersonaName, entry.ID)
}

// GetMockFeed reads Reddit JSON mock data and converts to RSS Feed format
func (m *RedditMockProvider) GetMockFeed(ctx context.Context, personaName string) (*rss.Feed, error) {
	// Read JSON mock data
	path := filepath.Join("feed_mocks", "reddit", personaName, fmt.Sprintf("%s.json", personaName))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read Reddit mock feed: %w", err)
	}

	// Parse JSON data
	var feedData RedditFeedData
	if err := json.Unmarshal(data, &feedData); err != nil {
		return nil, fmt.Errorf("failed to parse Reddit mock feed: %w", err)
	}

	// Convert JSON data to RSS entries
	entries := make([]rss.Entry, len(feedData.Posts))
	for i, post := range feedData.Posts {
		entries[i] = mockPostToEntry(post)
	}

	feed := &rss.Feed{
		Entries: entries,
		RawRSS:  fmt.Sprintf("Reddit API mock feed for r/%s", feedData.Subreddit),
	}

	return feed, nil
}

// GetMockComments reads Reddit JSON comment mock data and converts to RSS CommentFeed format
func (m *RedditMockProvider) GetMockComments(ctx context.Context, personaName string, entryID string) (*rss.CommentFeed, error) {
	// Read JSON mock data
	path := filepath.Join("feed_mocks", "reddit", personaName, fmt.Sprintf("%s.json", entryID))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read Reddit mock comments: %w", err)
	}

	// Parse JSON data
	var commentData RedditCommentData
	if err := json.Unmarshal(data, &commentData); err != nil {
		return nil, fmt.Errorf("failed to parse Reddit mock comments: %w", err)
	}

	// Convert JSON data to RSS comment entries
	// Filter for top-level comments only (matching Reddit API provider behavior)
	var commentEntries []rss.EntryComments
	for _, comment := range commentData.Comments {
		// Only include top-level comments to match RSS behavior
		if comment.ParentID == "t3_"+entryID {
			commentEntries = append(commentEntries, rss.EntryComments{
				Content: comment.Body,
			})
		}
	}

	commentFeed := &rss.CommentFeed{
		Entries: commentEntries,
		RawRSS:  fmt.Sprintf("Reddit API mock comments for post %s", entryID),
	}

	return commentFeed, nil
}

// mockPostToEntry converts a mock Reddit post to an RSS Entry
func mockPostToEntry(post RedditPostData) rss.Entry {
	entry := rss.Entry{
		Title:     post.Title,
		ID:        post.ID,
		Published: post.Created,
		Content:   post.Body,
	}

	// Set the link - use full Reddit permalink
	entry.Link = rss.Link{
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
		entry.MediaThumbnail = rss.MediaThumbnail{
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