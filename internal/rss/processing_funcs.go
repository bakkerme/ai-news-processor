package rss

import (
	"context"
	"fmt"
	"log"

	"github.com/bakkerme/ai-news-processor/internal/urlextraction"
)

// FetchAndProcessFeed fetches an RSS feed from the given URL and processes it
func FetchAndProcessFeed(provider FeedProvider, urlExtractor urlextraction.Extractor, feedURL string, debugRssDump bool, personaName string) ([]Entry, error) {
	log.Printf("Loading RSS feed: %s\n", feedURL)

	rssFeed, err := provider.FetchFeed(context.Background(), feedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to load rss data: %w", err)
	}

	// Dump RSS content if debug flag is enabled
	if debugRssDump {
		if err := dumpFeed(feedURL, rssFeed, personaName, personaName); err != nil {
			log.Printf("Warning: Failed to dump RSS feed: %v\n", err)
		}
	}

	entries := rssFeed.Entries
	if len(entries) == 0 {
		return nil, fmt.Errorf("no entries found in feed")
	}

	for i, entry := range entries {
		commentFeed, err := provider.FetchComments(context.Background(), entry)
		if err != nil {
			return nil, fmt.Errorf("failed to load rss comment data for entry %s: %w", entry.ID, err)
		}

		if debugRssDump {
			if err := dumpFeed(entry.GetCommentRSSURL(), commentFeed, personaName, entry.ID); err != nil {
				log.Printf("Warning: Failed to dump RSS comment feed: %v\n", err)
			}
		}

		entries[i].Comments = commentFeed.Entries

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

// FindEntryByID finds an RSS entry with the given ID
func FindEntryByID(id string, entries []Entry) *Entry {
	for _, entry := range entries {
		if entry.ID == id {
			return &entry
		}
	}
	return nil
}
