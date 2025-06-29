package reddit

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/vartanbeno/go-reddit/v2/reddit"
)

// RedditFeedData represents a Reddit feed dump in JSON format
type RedditFeedData struct {
	Subreddit string               `json:"subreddit"`
	FetchedAt time.Time            `json:"fetched_at"`
	Posts     []RedditPostData     `json:"posts"`
	RawAPIURL string               `json:"raw_api_url,omitempty"`
}

// RedditCommentData represents Reddit comments dump in JSON format
type RedditCommentData struct {
	PostID     string                `json:"post_id"`
	FetchedAt  time.Time             `json:"fetched_at"`
	Comments   []RedditCommentEntry  `json:"comments"`
	RawAPIURL  string                `json:"raw_api_url,omitempty"`
}

// RedditPostData represents a Reddit post in JSON dump format
type RedditPostData struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	URL         string    `json:"url"`
	Permalink   string    `json:"permalink"`
	Created     time.Time `json:"created"`
	Score       int       `json:"score"`
	NumComments int       `json:"num_comments"`
	Author      string    `json:"author"`
	IsSelf      bool      `json:"is_self"`
	NSFW        bool      `json:"nsfw,omitempty"`
	Spoiler     bool      `json:"spoiler,omitempty"`
}

// RedditCommentEntry represents a Reddit comment in JSON dump format
type RedditCommentEntry struct {
	ID               string    `json:"id"`
	Body             string    `json:"body"`
	ParentID         string    `json:"parent_id"`
	Author           string    `json:"author"`
	Score            int       `json:"score"`
	Created          time.Time `json:"created"`
	Controversiality int       `json:"controversiality,omitempty"`
}

// dumpRedditFeed saves Reddit API feed data as JSON for debugging/mocking
func dumpRedditFeed(subreddit string, posts []*reddit.Post, personaName string) error {
	log.Printf("Dumping Reddit API feed for r/%s", subreddit)

	// Convert Reddit posts to dump format
	postData := make([]RedditPostData, len(posts))
	for i, post := range posts {
		postData[i] = RedditPostData{
			ID:          post.ID,
			Title:       post.Title,
			Body:        post.Body,
			URL:         post.URL,
			Permalink:   post.Permalink,
			Created:     post.Created.Time,
			Score:       post.Score,
			NumComments: post.NumberOfComments,
			Author:      post.Author,
			IsSelf:      post.IsSelfPost,
			NSFW:        post.NSFW,
			Spoiler:     post.Spoiler,
		}
	}

	feedData := RedditFeedData{
		Subreddit: subreddit,
		FetchedAt: time.Now(),
		Posts:     postData,
		RawAPIURL: fmt.Sprintf("/r/%s/hot", subreddit),
	}

	// Create directory structure
	dir := filepath.Join("feed_mocks", "reddit", personaName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write JSON to file
	filename := fmt.Sprintf("%s.json", personaName)
	path := filepath.Join(dir, filename)
	
	jsonData, err := json.MarshalIndent(feedData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal feed data: %w", err)
	}

	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write feed data: %w", err)
	}

	log.Printf("Reddit feed dumped to: %s", path)
	return nil
}

// dumpRedditComments saves Reddit API comment data as JSON for debugging/mocking
func dumpRedditComments(postID string, comments []*reddit.Comment, personaName string) error {
	log.Printf("Dumping Reddit API comments for post %s", postID)

	// Convert Reddit comments to dump format
	commentData := make([]RedditCommentEntry, len(comments))
	for i, comment := range comments {
		commentData[i] = RedditCommentEntry{
			ID:               comment.ID,
			Body:             comment.Body,
			ParentID:         comment.ParentID,
			Author:           comment.Author,
			Score:            comment.Score,
			Created:          comment.Created.Time,
			Controversiality: comment.Controversiality,
		}
	}

	commentsData := RedditCommentData{
		PostID:    postID,
		FetchedAt: time.Now(),
		Comments:  commentData,
		RawAPIURL: fmt.Sprintf("/comments/%s", postID),
	}

	// Create directory structure
	dir := filepath.Join("feed_mocks", "reddit", personaName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write JSON to file
	filename := fmt.Sprintf("%s.json", postID)
	path := filepath.Join(dir, filename)
	
	jsonData, err := json.MarshalIndent(commentsData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal comments data: %w", err)
	}

	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write comments data: %w", err)
	}

	log.Printf("Reddit comments dumped to: %s", path)
	return nil
}