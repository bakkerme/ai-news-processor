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
	rssString, err := fetchRSS(url)
	if err != nil {
		return nil, fmt.Errorf("could not fetch RSS from %s: %w", url, err)
	}

	feed := &Feed{}
	err = processRSSFeed(rssString, feed)
	if err != nil {
		return nil, fmt.Errorf("could not process RSS feed from %s: %w", url, err)
	}

	return feed, nil
}

// FetchComments implements FeedProvider.FetchComments
func (p *DefaultFeedProvider) FetchComments(ctx context.Context, entry Entry) (*CommentFeed, error) {
	commentURL := entry.GetCommentRSSURL()
	commentFeedString, err := fetchRSS(commentURL)
	if err != nil {
		return nil, fmt.Errorf("failed to load comment feed for entry %s: %w", entry.ID, err)
	}

	commentFeed := &CommentFeed{}
	err = processCommentsRSSFeed(commentFeedString, commentFeed)
	if err != nil {
		return nil, fmt.Errorf("could not process comment feed: %w", err)
	}

	return commentFeed, nil
}
