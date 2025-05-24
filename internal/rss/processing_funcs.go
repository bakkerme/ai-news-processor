package rss

import (
	"context"
	"fmt"
)

// FetchAndProcessFeed fetches an RSS feed from the given URL and processes it
func FetchAndProcessFeed(provider FeedProvider, feedURL string, debugRssDump bool, personaName string) ([]Entry, error) {
	fmt.Printf("Loading RSS feed: %s\n", feedURL)

	rssFeed, err := provider.FetchFeed(context.Background(), feedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to load rss data: %w", err)
	}

	// Dump RSS content if debug flag is enabled
	if debugRssDump {
		if err := dumpFeed(feedURL, rssFeed, personaName, personaName); err != nil {
			fmt.Printf("Warning: Failed to dump RSS feed: %v\n", err)
		}
	}

	entries := rssFeed.Entries
	if len(entries) == 0 {
		return nil, fmt.Errorf("no entries found in feed")
	}

	return entries, nil
}

// FetchAndEnrichWithComments adds comments to each RSS entry
func FetchAndEnrichWithComments(provider FeedProvider, entries []Entry, debugRssDump bool, personaName string) ([]Entry, error) {
	enrichedEntries := make([]Entry, len(entries))
	copy(enrichedEntries, entries)

	for i, entry := range enrichedEntries {
		commentFeed, err := provider.FetchComments(context.Background(), entry)
		if err != nil {
			return nil, fmt.Errorf("failed to load rss comment data for entry %s: %w", entry.ID, err)
		}

		if debugRssDump {
			if err := dumpFeed(entry.GetCommentRSSURL(), commentFeed, personaName, entry.ID); err != nil {
				fmt.Printf("Warning: Failed to dump RSS comment feed: %v\n", err)
			}
		}

		enrichedEntries[i].Comments = commentFeed.Entries
	}

	return enrichedEntries, nil
}

// FindEntryByID finds an RSS entry with the given ID
func FindEntryByID(id string, entries []Entry) *Entry {
	for _, entry := range entries {
		if entry.ID == id {
			return &entry
		}
	}
	return nil
}
