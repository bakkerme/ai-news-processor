package providers

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"

	"github.com/bakkerme/ai-news-processor/internal/feeds"
	"github.com/bakkerme/ai-news-processor/internal/persona"
	"github.com/vartanbeno/go-reddit/v2/reddit"
)

// RedditProvider implements the feeds.FeedProvider interface using Reddit API
type RedditProvider struct {
	client     *reddit.Client
	enableDump bool
}

// NewRedditProvider creates a new Reddit API provider
func NewRedditProvider(clientID, clientSecret, username, password string, enableDump bool) (*RedditProvider, error) {
	credentials := reddit.Credentials{
		ID:       clientID,
		Secret:   clientSecret,
		Username: username,
		Password: password,
	}

	client, err := reddit.NewClient(credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to create Reddit client: %w", err)
	}

	return &RedditProvider{
		client:     client,
		enableDump: enableDump,
	}, nil
}

// FetchFeed implements feeds.FeedProvider.FetchFeed
func (r *RedditProvider) FetchFeed(ctx context.Context, p persona.Persona) (*feeds.Feed, error) {
	log.Printf("Fetching posts from r/%s via Reddit API", p.Subreddit)

	// Fetch posts from Reddit API
	posts, _, err := r.client.Subreddit.HotPosts(ctx, p.Subreddit, &reddit.ListOptions{
		Limit: 25, // Match RSS default limit
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch posts from r/%s: %w", p.Subreddit, err)
	}

	// Dump Reddit API data if enabled
	if r.enableDump {
		if err := r.dumpRedditFeed(p.Subreddit, posts, p.Subreddit); err != nil {
			log.Printf("Warning: Failed to dump Reddit feed: %v", err)
		}
	}

	// Convert Reddit posts to feed entries
	entries := make([]feeds.Entry, len(posts))
	for i, post := range posts {
		entries[i] = r.mapPostToEntry(post)
	}

	feed := &feeds.Feed{
		Entries: entries,
		RawData: fmt.Sprintf("Reddit API feed for r/%s", p.Subreddit),
	}

	return feed, nil
}

// FetchComments implements feeds.FeedProvider.FetchComments
func (r *RedditProvider) FetchComments(ctx context.Context, entry feeds.Entry) (*feeds.CommentFeed, error) {
	log.Printf("Fetching comments for post %s via Reddit API", entry.ID)

	// Fetch comments from Reddit API
	postAndComments, _, err := r.client.Post.Get(ctx, entry.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch comments for post %s: %w", entry.ID, err)
	}

	// Dump Reddit API comment data if enabled
	if r.enableDump && postAndComments != nil {
		// Extract subreddit for dump organization
		subreddit, err := extractSubredditFromPermalink(entry.Link.Href)
		if err != nil {
			log.Printf("Warning: Could not extract subreddit for dump: %v", err)
		} else {
			if err := r.dumpRedditComments(entry.ID, postAndComments.Comments, subreddit); err != nil {
				log.Printf("Warning: Failed to dump Reddit comments: %v", err)
			}
		}
	}

	// Convert Reddit comments to feed comment entries (top-level only to match RSS depth=1)
	var commentEntries []feeds.EntryComments
	if postAndComments != nil {
		// Access comments from the PostAndComments struct
		for _, comment := range postAndComments.Comments {
			// Only include top-level comments to match RSS behavior
			if comment.ParentID == "t3_"+entry.ID {
				commentEntries = append(commentEntries, feeds.EntryComments{
					Content: comment.Body,
				})
			}
		}
	}

	commentFeed := &feeds.CommentFeed{
		Entries: commentEntries,
		RawData: fmt.Sprintf("Reddit API comments for post %s", entry.ID),
	}

	return commentFeed, nil
}

// mapPostToEntry converts a Reddit API post to a feeds.Entry
func (r *RedditProvider) mapPostToEntry(post *reddit.Post) feeds.Entry {
	entry := feeds.Entry{
		Title:     post.Title,
		ID:        post.ID,
		Published: post.Created.Time,
		Content:   post.Body, // Selftext for text posts
	}

	// Set the link - use full Reddit permalink
	entry.Link = feeds.Link{
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
	entry.ImageURLs = r.extractImageURLsFromPost(post)

	// Set media thumbnail if available
	entry.MediaThumbnail = r.extractThumbnailFromPost(post)

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

// extractImageURLsFromPost extracts image URLs from a Reddit post
func (r *RedditProvider) extractImageURLsFromPost(post *reddit.Post) []url.URL {
	var imageURLs []url.URL

	// Check if the post URL is a direct image
	if post.URL != "" && isImageURL(post.URL) {
		if parsedURL, err := url.Parse(post.URL); err == nil {
			imageURLs = append(imageURLs, *parsedURL)
		}
	}

	return imageURLs
}

// extractThumbnailFromPost extracts thumbnail information from a Reddit post
func (r *RedditProvider) extractThumbnailFromPost(post *reddit.Post) feeds.MediaThumbnail {
	// For image posts, use the post URL as thumbnail
	if post.URL != "" && isImageURL(post.URL) {
		return feeds.MediaThumbnail{
			URL: post.URL,
		}
	}

	return feeds.MediaThumbnail{}
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


// extractSubredditFromPermalink extracts subreddit from Reddit permalink
// Example: "https://www.reddit.com/r/LocalLLaMA/comments/abc123/title/" -> "LocalLLaMA"
func extractSubredditFromPermalink(permalink string) (string, error) {
	// Use regex to extract subreddit from permalink
	re := regexp.MustCompile(`/r/([^/]+)/`)
	matches := re.FindStringSubmatch(permalink)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not extract subreddit from permalink: %s", permalink)
	}
	return matches[1], nil
}