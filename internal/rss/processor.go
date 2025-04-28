package rss

import (
	"fmt"
	"io"
	"net/http"
)

// FetchAndProcessFeed fetches an RSS feed from the given URL and processes it
func FetchAndProcessFeed(feedURL string, mockRSS bool) ([]Entry, error) {
	var rssString string
	var err error

	if !mockRSS {
		fmt.Printf("Loading RSS feed: %s\n", feedURL)
		rssString, err = FetchRSS(feedURL)
		if err != nil {
			return nil, fmt.Errorf("failed to load rss data: %w", err)
		}
	} else {
		fmt.Println("Loading Mock RSS feed")
		rssString = ReturnFakeRSS()
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

// EnrichWithComments adds comments to each RSS entry
func EnrichWithComments(entries []Entry, mockRSS bool) ([]Entry, error) {
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
		} else {
			commentFeedString = ReturnFakeCommentRSS(entry.ID)
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
