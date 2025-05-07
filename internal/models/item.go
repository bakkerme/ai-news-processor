package models

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// Item represents the structure of the JSON/YAML object
type Item struct {
	Title          string `yaml:"title"`
	ID             string `yaml:"id"`
	Summary        string `yaml:"overview"`
	CommentSummary string `yaml:"comment_overview"`
	Link           string `yaml:"link"`
	Relevance      string `yaml:"relevance"`
	IsRelevant     bool   `yaml:"is_relevant"`
}

// KeyDevelopment represents a key development and its referenced item
type KeyDevelopment struct {
	Text   string `yaml:"text"`
	ItemID string `yaml:"item_id"`
}

// SummaryResponse represents an overall summary of multiple relevant AI news items
type SummaryResponse struct {
	OverallSummary     string           `yaml:"overall_summary"`
	KeyDevelopments    []KeyDevelopment `yaml:"key_developments"`
	EmergingTrends     []string         `yaml:"emerging_trends"`
	TechnicalHighlight string           `yaml:"technical_highlight"`
}

// UnmarshalYAML implements custom unmarshaling for SummaryResponse to clean up fields
func (s *SummaryResponse) UnmarshalYAML(value *yaml.Node) error {
	type Alias SummaryResponse
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(s),
	}
	if err := value.Decode(aux); err != nil {
		return err
	}
	// Clean up each trend by removing trailing backslashes and whitespace
	for i, trend := range s.EmergingTrends {
		s.EmergingTrends[i] = strings.TrimRight(trend, "\\\n\r\t ")
	}
	return nil
}

// Helper function to unmarshal YAML into SummaryResponse
func UnmarshalSummaryResponseYAML(data []byte) (*SummaryResponse, error) {
	var summary SummaryResponse
	if err := yaml.Unmarshal(data, &summary); err != nil {
		return nil, err
	}
	return &summary, nil
}
