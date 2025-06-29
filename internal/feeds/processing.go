package feeds

import (
	"context"
	"fmt"
	"log"

	"github.com/bakkerme/ai-news-processor/internal/urlextraction"
)

// FeedProvider defines the interface for fetching and processing feed data
type FeedProvider interface {
	// FetchFeed retrieves and processes a feed from the given subreddit
	FetchFeed(ctx context.Context, subreddit string) (*Feed, error)

	// FetchComments retrieves and processes comments for a specific entry
	FetchComments(ctx context.Context, entry Entry) (*CommentFeed, error)
}

// FetchAndProcessFeed fetches a feed from the given subreddit and processes it
func FetchAndProcessFeed(provider FeedProvider, urlExtractor urlextraction.Extractor, subreddit string, debugDump bool, personaName string) ([]Entry, error) {
	log.Printf("Loading feed for subreddit: %s\n", subreddit)

	feed, err := provider.FetchFeed(context.Background(), subreddit)
	if err != nil {
		return nil, fmt.Errorf("failed to load feed data: %w", err)
	}

	entries := feed.Entries
	if len(entries) == 0 {
		return nil, fmt.Errorf("no entries found in feed")
	}

	for i, entry := range entries {
		commentFeed, err := provider.FetchComments(context.Background(), entry)
		if err != nil {
			return nil, fmt.Errorf("failed to load comment data for entry %s: %w", entry.ID, err)
		}

		// Filter out the original post from comments (Reddit includes the original post as first comment entry)
		var filteredComments []EntryComments
		for _, comment := range commentFeed.Entries {
			// Skip comment entries that have the same ID as the main post (this prevents duplication)
			if comment.Content != "" && len(comment.Content) > 0 {
				// Check if this comment entry is actually the original post by comparing a portion of content
				// or simply filter based on position (first entry is typically the original post)
				filteredComments = append(filteredComments, comment)
			}
		}
		
		// Remove the first comment entry if it exists, as Reddit comment feeds include the original post as the first entry
		if len(filteredComments) > 0 {
			filteredComments = filteredComments[1:]
		}
		
		entries[i].Comments = filteredComments

		// extract image urls
		imageURLs, err := urlExtractor.ExtractImageURLsFromEntry(entry)
		if err != nil {
			return nil, fmt.Errorf("failed to extract image URLs: %w", err)
		}

		entries[i].ImageURLs = imageURLs

		// extract external urls
		externalURLs, err := urlExtractor.ExtractExternalURLsFromEntry(entry)
		if err != nil {
			return nil, fmt.Errorf("failed to extract external URLs: %w", err)
		}

		entries[i].ExternalURLs = externalURLs
	}

	return entries, nil
}

// FindEntryByID finds a feed entry with the given ID
func FindEntryByID(id string, entries []Entry) *Entry {
	for _, entry := range entries {
		if entry.ID == id {
			return &entry
		}
	}
	return nil
}