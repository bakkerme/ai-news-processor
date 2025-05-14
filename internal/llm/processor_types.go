package llm

import (
	"time"

	"github.com/bakkerme/ai-news-processor/internal/contentextractor"
	"github.com/bakkerme/ai-news-processor/internal/fetcher"
	"github.com/bakkerme/ai-news-processor/internal/http"
	"github.com/bakkerme/ai-news-processor/internal/openai"
	"github.com/bakkerme/ai-news-processor/internal/urlextraction"
)

// EntryProcessConfig holds configuration for entry processing
type EntryProcessConfig struct {
	InitialBackoff       time.Duration
	BackoffFactor        float64
	MaxRetries           int
	MaxBackoff           time.Duration
	ImageEnabled         bool // Whether image processing is enabled
	DebugOutputBenchmark bool // Whether to output benchmark inputs
	URLSummaryEnabled    bool // Whether URL summarization is enabled
}

// DefaultEntryProcessConfig provides default configuration for entry processing
var DefaultEntryProcessConfig = EntryProcessConfig{
	InitialBackoff:       1 * time.Second,
	BackoffFactor:        2.0,
	MaxRetries:           3,
	MaxBackoff:           10 * time.Second,
	ImageEnabled:         false,
	DebugOutputBenchmark: false,
	URLSummaryEnabled:    true,
}

// Processor handles the processing of RSS entries with LLM integration
type Processor struct {
	client               openai.OpenAIClient               // Main LLM client
	imageClient          openai.OpenAIClient               // Client for image processing
	urlFetcher           fetcher.Fetcher                   // HTTP client for fetching URLs
	config               EntryProcessConfig                // Configuration for retries and backoff
	urlSummaryEnabled    bool                              // Whether URL summarization is enabled
	urlExtractor         urlextraction.Extractor           // URL extraction implementation
	imageEnabled         bool                              // Whether image processing is enabled
	debugOutputBenchmark bool                              // Whether to output benchmark inputs
	imageFetcher         http.ImageFetcher                 // Fetcher for images
	articleExtractor     contentextractor.ArticleExtractor // Article content extractor
}
