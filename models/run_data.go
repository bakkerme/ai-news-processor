package models

import (
	"net/url"
	"time"

	"github.com/bakkerme/ai-news-processor/internal/persona" // Import for persona.Persona
)

// EntrySummary represents the raw input and results for the entire processing pipeline
type EntrySummary struct {
	RawInput       string `json:"rawInput"`         // The raw input strings sent to the LLM
	Results        Item   `json:"results"`          // The processed results from the LLM, uses models.Item
	ProcessingTime int64  `json:"processingTimeMs"` // Time taken to process the entry in milliseconds
}

// ImageSummary represents the benchmark data for image processing
type ImageSummary struct {
	ImageURL         string `json:"imageURL"`          // URL of the image processed
	ImageDescription string `json:"imageDescription"`  // The description generated for the image
	Title            string `json:"title,omitempty"`   // Title associated with the image
	EntryID          string `json:"entryID,omitempty"` // ID of the entry the image belongs to
	ProcessingTime   int64  `json:"processingTimeMs"`  // Time taken to process the image in milliseconds
}

// WebContentSummary represents the benchmark data for web content processing
type WebContentSummary struct {
	URL             url.URL `json:"url"`               // URL of the web content
	OriginalContent string  `json:"originalContent"`   // Original content from the URL
	Summary         string  `json:"summary"`           // Summary generated for the web content
	Title           string  `json:"title,omitempty"`   // Title of the web content
	EntryID         string  `json:"entryID,omitempty"` // ID of the entry the web content belongs to
	ProcessingTime  int64   `json:"processingTimeMs"`  // Time taken to process the web content in milliseconds
}

// RunData represents the data collected during a run, intended for auditing and benchmarking.
// This was formerly BenchmarkData in bench.go
type RunData struct {
	EntrySummaries                []EntrySummary      `json:"entrySummaries"`
	ImageSummaries                []ImageSummary      `json:"imageSummaries"`
	WebContentSummaries           []WebContentSummary `json:"webContentSummaries"`
	OverallSummary                *SummaryResponse    `json:"overallSummary"`
	Persona                       persona.Persona     `json:"persona"`
	RunDate                       time.Time           `json:"runDate"`
	OverallModelUsed              string              `json:"overallModelUsed,omitempty"`
	ImageModelUsed                string              `json:"imageModelUsed,omitempty"`
	WebContentModelUsed           string              `json:"webContentModelUsed,omitempty"`
	TotalProcessingTime           int64               `json:"totalProcessingTime,omitempty"`
	EntryTotalProcessingTime      int64               `json:"entryTotalProcessingTime,omitempty"`
	ImageTotalProcessingTime      int64               `json:"imageTotalProcessingTime,omitempty"`
	WebContentTotalProcessingTime int64               `json:"webContentTotalProcessingTime,omitempty"`
	SuccessRate                   float64             `json:"successRate,omitempty"`
}
