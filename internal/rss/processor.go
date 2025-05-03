package rss

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// FetchAndProcessFeed fetches an RSS feed from the given URL and processes it
func FetchAndProcessFeed(provider FeedProvider, feedURL string, mockRSS bool, personaName string, debugRssDump bool) ([]Entry, error) {
	var rssFeed *Feed
	var err error

	if !mockRSS {
		fmt.Printf("Loading RSS feed: %s\n", feedURL)
		rssFeed, err = provider.FetchFeed(context.Background(), feedURL)
		if err != nil {
			return nil, fmt.Errorf("failed to load rss data: %w", err)
		}

		// Dump RSS content if debug flag is enabled
		if debugRssDump {
			if err := DumpFeed(feedURL, rssFeed, personaName, personaName); err != nil {
				fmt.Printf("Warning: Failed to dump RSS feed: %v\n", err)
			}
		}
	} else {
		fmt.Println("Loading Mock RSS feed")
		rssFeed = ReturnFakeRSS(personaName)
	}

	entries := rssFeed.Entries
	if len(entries) == 0 {
		return nil, fmt.Errorf("no entries found in feed")
	}

	return entries, nil
}

// FetchAndEnrichWithComments adds comments to each RSS entry
func FetchAndEnrichWithComments(provider FeedProvider, entries []Entry, mockRSS bool, debugRssDump bool, personaName string) ([]Entry, error) {
	enrichedEntries := make([]Entry, len(entries))
	copy(enrichedEntries, entries)

	for i, entry := range enrichedEntries {
		var commentFeed *CommentFeed
		var err error

		if !mockRSS {
			commentFeed, err = provider.FetchComments(context.Background(), entry)
			if err != nil {
				return nil, fmt.Errorf("failed to load rss comment data for entry %s: %w", entry.ID, err)
			}

			if debugRssDump {
				if err := DumpFeed(entry.GetCommentRSSURL(), commentFeed, personaName, entry.ID); err != nil {
					fmt.Printf("Warning: Failed to dump RSS comment feed: %v\n", err)
				}
			}
		} else {
			commentFeed = ReturnFakeCommentRSS(personaName, entry.ID)
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

// DumpFeed saves the raw RSS content to disk for debugging purposes
func DumpFeed(feedURL string, content Feedlike, personaName, itemName string) error {
	fmt.Printf("Dumping RSS for %s\n", feedURL)

	feedString := content.FeedString()

	// Create a safe filename from the itemName
	filename := itemName + ".rss"

	// Create the directory path
	dir := filepath.Join("..", "feed_mocks", "rss", personaName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write the content to file
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(feedString), 0644); err != nil {
		return fmt.Errorf("failed to write RSS content: %w", err)
	}

	return nil
}
