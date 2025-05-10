package qualityfilter

import (
	"github.com/bakkerme/ai-news-processor/internal/rss"
)

// Filter returns a list of entries that have more comments than the specified threshold
func Filter(entries []rss.Entry, threshold int) []rss.Entry {
	filtered := make([]rss.Entry, 0)
	for _, entry := range entries {
		if len(entry.Comments) >= threshold {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}
