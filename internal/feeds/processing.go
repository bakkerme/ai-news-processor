package feeds

import (
	"context"
	"fmt"
	"log"

	"github.com/bakkerme/ai-news-processor/internal/persona"
	"github.com/bakkerme/ai-news-processor/internal/urlextraction"
)

// FeedProvider defines the interface for fetching and processing feed data
type FeedProvider interface {
	// FetchFeed retrieves and processes a feed for the given persona
	FetchFeed(ctx context.Context, persona persona.Persona) (*Feed, error)

	// FetchComments retrieves and processes comments for a specific entry
	FetchComments(ctx context.Context, entry Entry) (*CommentFeed, error)
}

// FetchAndProcessFeed fetches a feed for the given persona and processes it
// TODO: most of this logic should be in the reddit provider itself
func FetchAndProcessFeed(provider FeedProvider, urlExtractor urlextraction.Extractor, persona persona.Persona, debugDump bool) ([]Entry, error) {
	log.Printf("Loading feed for persona: %s\n", persona.Name)

	feed, err := provider.FetchFeed(context.Background(), persona)
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

		if len(entries[i].ImageURLs) == 0 {
			// extract image urls
			imageURLs, err := urlExtractor.ExtractImageURLsFromEntry(entry)
			if err != nil {
				return nil, fmt.Errorf("failed to extract image URLs: %w", err)
			}

			entries[i].ImageURLs = imageURLs
		}

		if len(entries[i].ExternalURLs) == 0 {
			// extract external urls
			externalURLs, err := urlExtractor.ExtractExternalURLsFromEntry(entry)
			if err != nil {
				return nil, fmt.Errorf("failed to extract external URLs: %w", err)
			}

			entries[i].ExternalURLs = externalURLs
		}
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
