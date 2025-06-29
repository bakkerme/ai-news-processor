package qualityfilter

import "github.com/bakkerme/ai-news-processor/internal/feeds"

// Filter returns a list of entries that have more comments than the specified threshold
func Filter(entries []feeds.Entry, threshold int) []feeds.Entry {
	filtered := make([]feeds.Entry, 0)
	for _, entry := range entries {
		if len(entry.Comments) >= threshold {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}
