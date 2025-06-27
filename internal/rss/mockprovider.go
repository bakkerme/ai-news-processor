package rss

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// MockFeedProvider implements the FeedProvider interface for testing
type MockFeedProvider struct {
	PersonaName string
}

// NewMockFeedProvider creates a new mock feed provider for the specified persona
func NewMockFeedProvider(personaName string) *MockFeedProvider {
	return &MockFeedProvider{
		PersonaName: personaName,
	}
}

// FetchFeed implements FeedProvider.FetchFeed for mocks
func (m *MockFeedProvider) FetchFeed(ctx context.Context, url string) (*Feed, error) {
	return m.GetMockFeed(ctx, m.PersonaName)
}

// FetchComments implements FeedProvider.FetchComments for mocks
func (m *MockFeedProvider) FetchComments(ctx context.Context, entry Entry) (*CommentFeed, error) {
	return m.GetMockComments(ctx, m.PersonaName, entry.ID)
}

// GetMockFeed implements FeedProvider.GetMockFeed
func (m *MockFeedProvider) GetMockFeed(ctx context.Context, personaName string) (*Feed, error) {
	// print current working dir
	dir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current working directory: %w", err)
	}

	fmt.Println("Current working directory:", dir)

	path := filepath.Join("feed_mocks", "rss", personaName, fmt.Sprintf("%s.rss", personaName))
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read mock feed: %w", err)
	}

	rssFeed := &Feed{}
	err = processRSSFeed(string(b), rssFeed)
	if err != nil {
		return nil, fmt.Errorf("failed to process mock feed: %w", err)
	}

	return rssFeed, nil
}

// GetMockComments implements FeedProvider.GetMockComments
func (m *MockFeedProvider) GetMockComments(ctx context.Context, personaName string, entryID string) (*CommentFeed, error) {
	path := filepath.Join("feed_mocks", "rss", personaName, fmt.Sprintf("%s.rss", entryID))
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read mock comments: %w", err)
	}

	commentFeed := &CommentFeed{}
	err = processCommentsRSSFeed(string(b), commentFeed)
	if err != nil {
		return nil, fmt.Errorf("failed to process mock comments: %w", err)
	}

	return commentFeed, nil
}
