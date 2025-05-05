package qualityfilter

import "github.com/bakkerme/ai-news-processor/internal/rss"

// Filter returns a list of entries that have over 10 comments
func Filter(entries []rss.Entry) []rss.Entry {
	filtered := make([]rss.Entry, 0)
	for _, entry := range entries {
		if len(entry.Comments) > 10 {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}
