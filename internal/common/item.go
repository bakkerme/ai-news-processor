package common

import (
	"encoding/json"
	"strings"
)

// Item represents the structure of the JSON object
type Item struct {
	Title          string `json:"Title" jsonschema_description:"Title of the post" jsonschema:"required"`
	ID             string `json:"ID" jsonschema_description:"Post ID" jsonschema:"required"`
	Summary        string `json:"Summary" jsonschema_description:"Provide a summary of the post content" jsonschema:"required"`
	CommentSummary string `json:"CommentSummary" jsonschema_description:"Provide a summary and semtiment of the comments" jsonschema:"required"`
	Link           string `json:"Link" jsonschema_description:"A link to the post" jsonschema:"required"`
	Relevance      string `json:"Relevance" jsonschema_description:"Why is this relevant?" jsonschema:"required"`
	IsRelevant     bool   `json:"IsRelevant" jsonschema_description:"Should this be included?" jsonschema:"required"`
}

// KeyDevelopment represents a key development and its referenced item
type KeyDevelopment struct {
	Text   string `json:"Text" jsonschema_description:"Description of the key development" jsonschema:"required"`
	ItemID string `json:"ItemID" jsonschema_description:"ID of the referenced post" jsonschema:"required"`
}

// SummaryResponse represents an overall summary of multiple relevant AI news items
type SummaryResponse struct {
	OverallSummary     string           `json:"OverallSummary" jsonschema_description:"A high-level summary of the major AI developments and trends" jsonschema:"required"`
	KeyDevelopments    []KeyDevelopment `json:"KeyDevelopments" jsonschema_description:"List of the most significant developments, with references to items" jsonschema:"required"`
	EmergingTrends     []string         `json:"EmergingTrends" jsonschema_description:"List of 3-5 emerging trends visible across the articles" jsonschema:"required"`
	TechnicalHighlight string           `json:"TechnicalHighlight" jsonschema_description:"Most technically significant development" jsonschema:"required"`
}

// UnmarshalJSON implements custom unmarshaling for SummaryResponse to clean up fields
func (s *SummaryResponse) UnmarshalJSON(data []byte) error {
	type Alias SummaryResponse
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(s),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Clean up each trend by removing trailing backslashes and whitespace
	for i, trend := range s.EmergingTrends {
		s.EmergingTrends[i] = strings.TrimRight(trend, "\\\n\r\t ")
	}
	return nil
}
