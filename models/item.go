package models

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bakkerme/ai-news-processor/internal/rss"
)

// Item represents the structure of the JSON/YAML object
type Item struct {
	Title             string    `json:"title"`
	ID                string    `json:"id"`
	Overview          string    `json:"overview"`
	Summary           string    `json:"summary"`
	CommentSummary    string    `json:"commentSummary,omitempty"`
	ImageSummary      string    `json:"imageDescription,omitempty"`
	WebContentSummary string    `json:"webContentSummary,omitempty"`
	Link              string    `json:"link,omitempty"`
	IsRelevant        bool      `json:"isRelevant"`
	ThumbnailURL      string    `json:"thumbnailUrl,omitempty"`
	Entry             rss.Entry `json:"entry,omitempty"`
}

// ToSummaryString creates a concise string representation of the Item for summary generation
// This includes ID, Title, Summary, and CommentSummary (if present)
func (item *Item) ToSummaryString() string {
	var itemStr strings.Builder
	fmt.Fprintf(&itemStr, "ID: %s\n", item.ID)
	fmt.Fprintf(&itemStr, "Title: %s\n", item.Title)
	fmt.Fprintf(&itemStr, "Summary: %s\n", item.Summary)
	if item.CommentSummary != "" {
		fmt.Fprintf(&itemStr, "Comment Summary: %s\n", item.CommentSummary)
	}
	return itemStr.String()
}

type ItemSubset struct {
	Title          string `json:"title"`
	ID             string `json:"id"`
	Overview       string `json:"overview"`
	Summary        string `json:"summary"`
	CommentSummary string `json:"commentSummary,omitempty"`
	IsRelevant     bool   `json:"isRelevant"`
}

// KeyDevelopment represents a key development and its referenced item
type KeyDevelopment struct {
	Text   string `json:"text"`
	ItemID string `json:"itemID"`
}

// SummaryResponse represents an overall summary of multiple relevant AI news items
type SummaryResponse struct {
	KeyDevelopments []KeyDevelopment `json:"keyDevelopments"`
}

// UnmarshalJSON implements custom unmarshaling for SummaryResponse to clean up fields
func (s *SummaryResponse) UnmarshalJSON(data []byte) error {
	type Alias SummaryResponse
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(s),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	return nil
}

// Helper function to unmarshal JSON into SummaryResponse
func UnmarshalSummaryResponseJSON(data []byte) (*SummaryResponse, error) {
	var summary SummaryResponse
	if err := json.Unmarshal(data, &summary); err != nil {
		return nil, err
	}
	return &summary, nil
}
