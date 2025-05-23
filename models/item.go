package models

import (
	"encoding/json"
)

// Item represents the structure of the JSON/YAML object
type Item struct {
	Title              string `json:"title"`
	ID                 string `json:"id"`
	Summary            string `json:"summary"`
	CommentSummary     string `json:"commentSummary,omitempty"`
	ImageSummary       string `json:"imageDescription,omitempty"`
	ExternalURLSummary string `json:"externalURLSummary,omitempty"`
	Link               string `json:"link,omitempty"`
	IsRelevant         bool   `json:"isRelevant"`
	ThumbnailURL       string `json:"thumbnailUrl,omitempty"`
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
