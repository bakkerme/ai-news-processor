package reddit

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/bakkerme/ai-news-processor/internal/rss"
	"github.com/vartanbeno/go-reddit/v2/reddit"
)

// mapPostToEntry converts a Reddit API post to an RSS Entry
func mapPostToEntry(post *reddit.Post) rss.Entry {
	entry := rss.Entry{
		Title:     post.Title,
		ID:        post.ID,
		Published: post.Created.Time,
		Content:   post.Body, // Selftext for text posts
	}

	// Set the link - use full Reddit permalink
	entry.Link = rss.Link{
		Href: fmt.Sprintf("https://www.reddit.com%s", post.Permalink),
	}

	// Handle different post types
	if post.IsSelfPost {
		// Text post - content is in Body (selftext)
		entry.Content = post.Body
	} else {
		// Link post - URL points to external content
		entry.Content = fmt.Sprintf("Link: %s", post.URL)
		
		// Extract external URLs
		if post.URL != "" {
			if parsedURL, err := url.Parse(post.URL); err == nil {
				entry.ExternalURLs = []url.URL{*parsedURL}
			}
		}
	}

	// Extract image URLs if this is an image post
	entry.ImageURLs = extractImageURLsFromPost(post)

	// Set media thumbnail if available
	entry.MediaThumbnail = extractThumbnailFromPost(post)

	// Initialize empty maps/slices for compatibility
	if entry.ExternalURLs == nil {
		entry.ExternalURLs = []url.URL{}
	}
	if entry.ImageURLs == nil {
		entry.ImageURLs = []url.URL{}
	}
	if entry.WebContentSummaries == nil {
		entry.WebContentSummaries = make(map[string]string)
	}

	return entry
}

// mapCommentToEntryComment converts a Reddit API comment to an RSS EntryComments
func mapCommentToEntryComment(comment *reddit.Comment) rss.EntryComments {
	return rss.EntryComments{
		Content: comment.Body,
	}
}

// extractImageURLsFromPost extracts image URLs from a Reddit post
func extractImageURLsFromPost(post *reddit.Post) []url.URL {
	var imageURLs []url.URL

	// Check if the post URL is a direct image
	if post.URL != "" && isImageURL(post.URL) {
		if parsedURL, err := url.Parse(post.URL); err == nil {
			imageURLs = append(imageURLs, *parsedURL)
		}
	}

	// TODO: Could extract from Reddit's preview data if available
	// This would require accessing raw API response for preview.images

	return imageURLs
}

// extractThumbnailFromPost extracts thumbnail information from a Reddit post
func extractThumbnailFromPost(post *reddit.Post) rss.MediaThumbnail {
	// For image posts, use the post URL as thumbnail
	// TODO: Could access actual thumbnail URL from raw API response
	if post.URL != "" && isImageURL(post.URL) {
		return rss.MediaThumbnail{
			URL: post.URL,
		}
	}

	return rss.MediaThumbnail{}
}

// isImageURL checks if a URL points to an image
func isImageURL(urlStr string) bool {
	if urlStr == "" {
		return false
	}

	// Check for common image extensions
	lowerURL := strings.ToLower(urlStr)
	imageExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg"}
	
	for _, ext := range imageExtensions {
		if strings.Contains(lowerURL, ext) {
			return true
		}
	}

	// Check for common image hosting domains
	imageHosts := []string{
		"i.imgur.com",
		"i.redd.it",
		"preview.redd.it",
		"i.reddit.com",
		"imgur.com/",
	}

	for _, host := range imageHosts {
		if strings.Contains(lowerURL, host) {
			return true
		}
	}

	return false
}

// timestampToTime converts Reddit timestamp to time.Time
func timestampToTime(timestamp *reddit.Timestamp) time.Time {
	if timestamp == nil {
		return time.Time{}
	}
	return timestamp.Time
}