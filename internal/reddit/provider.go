package reddit

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"

	"github.com/bakkerme/ai-news-processor/internal/rss"
	"github.com/vartanbeno/go-reddit/v2/reddit"
)

// RedditAPIProvider implements the rss.FeedProvider interface using Reddit API
type RedditAPIProvider struct {
	client       *reddit.Client
	enableDump   bool
}

// NewRedditAPIProvider creates a new Reddit API provider
func NewRedditAPIProvider(clientID, clientSecret, username, password string, enableDump bool) (*RedditAPIProvider, error) {
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

	return &RedditAPIProvider{
		client:     client,
		enableDump: enableDump,
	}, nil
}

// FetchFeed implements rss.FeedProvider.FetchFeed
func (r *RedditAPIProvider) FetchFeed(ctx context.Context, url string) (*rss.Feed, error) {
	// Extract subreddit name from RSS URL
	subreddit, err := extractSubredditFromURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to extract subreddit from URL %s: %w", url, err)
	}

	log.Printf("Fetching posts from r/%s via Reddit API", subreddit)

	// Fetch posts from Reddit API
	posts, _, err := r.client.Subreddit.HotPosts(ctx, subreddit, &reddit.ListOptions{
		Limit: 25, // Match RSS default limit
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch posts from r/%s: %w", subreddit, err)
	}

	// Dump Reddit API data if enabled
	if r.enableDump {
		if err := dumpRedditFeed(subreddit, posts, subreddit); err != nil {
			log.Printf("Warning: Failed to dump Reddit feed: %v", err)
		}
	}

	// Convert Reddit posts to RSS entries
	entries := make([]rss.Entry, len(posts))
	for i, post := range posts {
		entries[i] = mapPostToEntry(post)
	}

	feed := &rss.Feed{
		Entries: entries,
		RawRSS:  fmt.Sprintf("Reddit API feed for r/%s", subreddit),
	}

	return feed, nil
}

// FetchComments implements rss.FeedProvider.FetchComments
func (r *RedditAPIProvider) FetchComments(ctx context.Context, entry rss.Entry) (*rss.CommentFeed, error) {
	log.Printf("Fetching comments for post %s via Reddit API", entry.ID)

	// Fetch comments from Reddit API - correct method signature
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
			if err := dumpRedditComments(entry.ID, postAndComments.Comments, subreddit); err != nil {
				log.Printf("Warning: Failed to dump Reddit comments: %v", err)
			}
		}
	}

	// Convert Reddit comments to RSS comment entries (top-level only to match RSS depth=1)
	var commentEntries []rss.EntryComments
	if postAndComments != nil {
		// Access comments from the PostAndComments struct
		for _, comment := range postAndComments.Comments {
			// Only include top-level comments to match RSS behavior
			if comment.ParentID == "t3_"+entry.ID {
				commentEntries = append(commentEntries, mapCommentToEntryComment(comment))
			}
		}
	}

	commentFeed := &rss.CommentFeed{
		Entries: commentEntries,
		RawRSS:  fmt.Sprintf("Reddit API comments for post %s", entry.ID),
	}

	return commentFeed, nil
}

// extractSubredditFromURL extracts subreddit name from RSS URL
// Example: "https://www.reddit.com/r/LocalLLaMA/.rss" -> "LocalLLaMA"
func extractSubredditFromURL(rssURL string) (string, error) {
	// Parse URL to extract subreddit name
	parsedURL, err := url.Parse(rssURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	// Extract subreddit from path like "/r/LocalLLaMA/.rss"
	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(pathParts) < 2 || pathParts[0] != "r" {
		return "", fmt.Errorf("invalid subreddit URL format: %s", rssURL)
	}

	subreddit := pathParts[1]
	// Remove .rss suffix if present
	subreddit = strings.TrimSuffix(subreddit, ".rss")

	return subreddit, nil
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
