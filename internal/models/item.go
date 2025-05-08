package models

import (
	"encoding/json"
	"strings"
)

// Item represents the structure of the JSON/YAML object
type Item struct {
	Title          string `json:"title"`
	ID             string `json:"id"`
	Summary        string `json:"overview"`
	CommentSummary string `json:"comment_overview"`
	Link           string `json:"link"`
	Relevance      string `json:"relevance"`
	IsRelevant     bool   `json:"is_relevant"`
}

// KeyDevelopment represents a key development and its referenced item
type KeyDevelopment struct {
	Text   string `json:"text"`
	ItemID string `json:"item_id"`
}

// SummaryResponse represents an overall summary of multiple relevant AI news items
type SummaryResponse struct {
	OverallSummary     string           `json:"overall_summary"`
	KeyDevelopments    []KeyDevelopment `json:"key_developments"`
	EmergingTrends     []string         `json:"emerging_trends"`
	TechnicalHighlight string           `json:"technical_highlight"`
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
	// Clean up each trend by removing trailing backslashes and whitespace
	for i, trend := range s.EmergingTrends {
		s.EmergingTrends[i] = strings.TrimRight(trend, "\\\n\r\t ")
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
