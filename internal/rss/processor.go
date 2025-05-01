package rss

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// FetchAndProcessFeed fetches an RSS feed from the given URL and processes it
func FetchAndProcessFeed(feedURL string, mockRSS bool, personaName string, debugRssDump bool) ([]Entry, error) {
	var rssString string
	var err error

	if !mockRSS {
		fmt.Printf("Loading RSS feed: %s\n", feedURL)
		rssString, err = FetchRSS(feedURL)
		if err != nil {
			return nil, fmt.Errorf("failed to load rss data: %w", err)
		}

		// Dump RSS content if debug flag is enabled
		if debugRssDump {
			if err := DumpRSS(feedURL, rssString, personaName, personaName); err != nil {
				fmt.Printf("Warning: Failed to dump RSS feed: %v\n", err)
			}
		}
	} else {
		fmt.Println("Loading Mock RSS feed")
		rssString = ReturnFakeRSS(personaName)
	}

	rssFeed, err := ProcessRSSFeed(rssString)
	if err != nil {
		return nil, fmt.Errorf("could not process rss feed: %w", err)
	}

	entries := rssFeed.Entries
	if len(entries) == 0 {
		return nil, fmt.Errorf("no entries found in feed")
	}

	return entries, nil
}

// FetchAndEnrichWithComments adds comments to each RSS entry
func FetchAndEnrichWithComments(entries []Entry, mockRSS bool, debugRssDump bool, personaName string) ([]Entry, error) {
	enrichedEntries := make([]Entry, len(entries))
	copy(enrichedEntries, entries)

	for i, entry := range enrichedEntries {
		commentFeedString := ""
		var err error

		if !mockRSS {
			commentFeedString, err = getCommentRSS(entry)
			if err != nil {
				return nil, fmt.Errorf("failed to load rss comment data for entry %s: %w", entry.ID, err)
			}

			if debugRssDump {
				if err := DumpRSS(entry.GetCommentRSSURL(), commentFeedString, personaName, entry.ID); err != nil {
					fmt.Printf("Warning: Failed to dump RSS comment feed: %v\n", err)
				}
			}
		} else {
			commentFeedString = ReturnFakeCommentRSS(personaName, entry.ID)
		}

		commentFeed, err := ProcessCommentsRSSFeed(commentFeedString)
		if err != nil {
			return nil, fmt.Errorf("could not process rss comment feed: %w", err)
		}

		enrichedEntries[i].Comments = commentFeed.Entries
	}

	return enrichedEntries, nil
}

// getCommentRSS fetches the RSS feed for comments on a specific entry
func getCommentRSS(entry Entry) (string, error) {
	resp, err := http.Get(entry.GetCommentRSSURL())
	if err != nil {
		return "", fmt.Errorf("could not get from reddit rss: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("could not load response body: %w", err)
	}

	return string(body), nil
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

// DumpRSS saves the raw RSS content to disk for debugging purposes
func DumpRSS(feedURL, content, personaName, itemName string) error {
	fmt.Printf("Dumping RSS for %s\n", feedURL)

	// Create a safe filename from the itemName
	filename := itemName + ".rss"

	// Create the directory path
	dir := filepath.Join("..", "feed_mocks", "rss", personaName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write the content to file
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write RSS content: %w", err)
	}

	return nil
}
