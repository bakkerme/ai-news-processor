package rss

import (
	"context"
	"fmt"
)

// FeedProvider defines the interface for fetching and processing RSS feed data.
type FeedProvider interface {
	// FetchFeed retrieves and processes a feed from the given URL
	FetchFeed(ctx context.Context, url string) (*Feed, error)

	// FetchComments retrieves and processes comments for a specific entry
	FetchComments(ctx context.Context, entry Entry) (*CommentFeed, error)

	// GetMockFeed returns a mock feed for testing purposes
	GetMockFeed(ctx context.Context, personaName string) (*Feed, error)

	// GetMockComments returns mock comments for testing purposes
	GetMockComments(ctx context.Context, personaName string, entryID string) (*CommentFeed, error)
}

// DefaultFeedProvider implements the FeedProvider interface using the standard RSS functions
type DefaultFeedProvider struct {
	// Can add configuration options here if needed
}

// NewFeedProvider creates a new instance of the default feed provider
func NewFeedProvider() *DefaultFeedProvider {
	return &DefaultFeedProvider{}
}

// FetchFeed implements FeedProvider.FetchFeed
func (p *DefaultFeedProvider) FetchFeed(ctx context.Context, url string) (*Feed, error) {
	rssString, err := FetchRSS(url)
	if err != nil {
		return nil, fmt.Errorf("could not fetch RSS from %s: %w", url, err)
	}

	feed := &Feed{}
	err = ProcessRSSFeed(rssString, feed)
	if err != nil {
		return nil, fmt.Errorf("could not process RSS feed from %s: %w", url, err)
	}

	return feed, nil
}

// FetchComments implements FeedProvider.FetchComments
func (p *DefaultFeedProvider) FetchComments(ctx context.Context, entry Entry) (*CommentFeed, error) {
	commentURL := entry.GetCommentRSSURL()
	commentFeedString, err := FetchRSS(commentURL)
	if err != nil {
		return nil, fmt.Errorf("failed to load comment feed for entry %s: %w", entry.ID, err)
	}

	commentFeed := &CommentFeed{}
	err = ProcessCommentsRSSFeed(commentFeedString, commentFeed)
	if err != nil {
		return nil, fmt.Errorf("could not process comment feed: %w", err)
	}

	return commentFeed, nil
}

// GetMockFeed implements FeedProvider.GetMockFeed
func (p *DefaultFeedProvider) GetMockFeed(ctx context.Context, personaName string) (*Feed, error) {
	feed := ReturnFakeRSS(personaName)

	return feed, nil
}

// GetMockComments implements FeedProvider.GetMockComments
func (p *DefaultFeedProvider) GetMockComments(ctx context.Context, personaName string, entryID string) (*CommentFeed, error) {
	commentFeed := ReturnFakeCommentRSS(personaName, entryID)

	return commentFeed, nil
}
