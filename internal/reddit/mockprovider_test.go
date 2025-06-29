package reddit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRedditMockProvider(t *testing.T) {
	// Create a temporary mock data file for testing
	testDir := filepath.Join("testdata", "reddit", "TestPersona")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll("testdata")

	// Create mock feed data
	feedData := RedditFeedData{
		Subreddit: "TestPersona",
		FetchedAt: time.Now(),
		Posts: []RedditPostData{
			{
				ID:          "abc123",
				Title:       "Test Post",
				Body:        "This is a test post body",
				URL:         "",
				Permalink:   "/r/TestPersona/comments/abc123/test_post/",
				Created:     time.Now(),
				Score:       100,
				NumComments: 5,
				Author:      "testuser",
				IsSelf:      true,
			},
		},
	}

	// Write mock feed data
	feedPath := filepath.Join(testDir, "TestPersona.json")
	feedJSON, _ := json.Marshal(feedData)
	if err := os.WriteFile(feedPath, feedJSON, 0644); err != nil {
		t.Fatalf("Failed to write mock feed data: %v", err)
	}

	// Create mock comment data
	commentData := RedditCommentData{
		PostID:    "abc123",
		FetchedAt: time.Now(),
		Comments: []RedditCommentEntry{
			{
				ID:       "def456",
				Body:     "This is a test comment",
				ParentID: "t3_abc123",
				Author:   "commenter",
				Score:    10,
				Created:  time.Now(),
			},
		},
	}

	// Write mock comment data
	commentPath := filepath.Join(testDir, "abc123.json")
	commentJSON, _ := json.Marshal(commentData)
	if err := os.WriteFile(commentPath, commentJSON, 0644); err != nil {
		t.Fatalf("Failed to write mock comment data: %v", err)
	}

	// Test the mock provider
	provider := NewRedditMockProvider("TestPersona")

	// Change working directory to point to our test data
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	// Create the expected path structure
	feedMocksDir := "feed_mocks"
	if err := os.MkdirAll(filepath.Join(feedMocksDir, "reddit", "TestPersona"), 0755); err != nil {
		t.Fatalf("Failed to create feed_mocks directory: %v", err)
	}
	defer os.RemoveAll(feedMocksDir)

	// Copy test files to the expected location
	if err := copyFile(feedPath, filepath.Join(feedMocksDir, "reddit", "TestPersona", "TestPersona.json")); err != nil {
		t.Fatalf("Failed to copy feed file: %v", err)
	}
	if err := copyFile(commentPath, filepath.Join(feedMocksDir, "reddit", "TestPersona", "abc123.json")); err != nil {
		t.Fatalf("Failed to copy comment file: %v", err)
	}

	// Test FetchFeed
	feed, err := provider.FetchFeed(context.Background(), "dummy_url")
	if err != nil {
		t.Fatalf("FetchFeed failed: %v", err)
	}

	if len(feed.Entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(feed.Entries))
	}

	entry := feed.Entries[0]
	if entry.ID != "abc123" {
		t.Errorf("Expected ID 'abc123', got '%s'", entry.ID)
	}
	if entry.Title != "Test Post" {
		t.Errorf("Expected title 'Test Post', got '%s'", entry.Title)
	}

	// Test FetchComments
	commentFeed, err := provider.FetchComments(context.Background(), entry)
	if err != nil {
		t.Fatalf("FetchComments failed: %v", err)
	}

	if len(commentFeed.Entries) != 1 {
		t.Errorf("Expected 1 comment, got %d", len(commentFeed.Entries))
	}

	comment := commentFeed.Entries[0]
	if comment.Content != "This is a test comment" {
		t.Errorf("Expected comment 'This is a test comment', got '%s'", comment.Content)
	}
}

// Helper function to copy files
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}