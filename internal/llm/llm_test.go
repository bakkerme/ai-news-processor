package llm

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/bakkerme/ai-news-processor/internal/contentextractor"
	"github.com/bakkerme/ai-news-processor/internal/customerrors"
	"github.com/bakkerme/ai-news-processor/internal/http/retry"
	"github.com/bakkerme/ai-news-processor/internal/openai"
	"github.com/bakkerme/ai-news-processor/internal/urlextraction"
	"github.com/bakkerme/ai-news-processor/models"
	"github.com/stretchr/testify/assert"
)

// Mock implementations for dependencies
type mockOpenAIClient struct {
	ChatCompletionFunc func(systemPrompt string, userPrompts []string, imageURLs []string, schemaParams *openai.SchemaParameters, temperature float64, maxTokens int, results chan customerrors.ErrorString)
}

func (m *mockOpenAIClient) ChatCompletion(systemPrompt string, userPrompts []string, imageURLs []string, schemaParams *openai.SchemaParameters, temperature float64, maxTokens int, results chan customerrors.ErrorString) {
	if m.ChatCompletionFunc != nil {
		m.ChatCompletionFunc(systemPrompt, userPrompts, imageURLs, schemaParams, temperature, maxTokens, results)
		return
	}
	// Default mock behavior if ChatCompletionFunc is not set
	close(results) // Or send a default response
}
func (m *mockOpenAIClient) PreprocessJSON(s string) string          { return s }
func (m *mockOpenAIClient) SetRetryConfig(config retry.RetryConfig) {}
func (m *mockOpenAIClient) PreprocessYAML(response string) string   { return response }
func (m *mockOpenAIClient) GetModelName() string                    { return "mock-model" }

type mockArticleExtractor struct{}

func (m *mockArticleExtractor) Extract(body io.Reader, url *url.URL) (*contentextractor.ArticleData, error) {
	return &contentextractor.ArticleData{}, nil
}

type mockFetcher struct{}

func (m *mockFetcher) Fetch(ctx context.Context, url *url.URL) (*http.Response, error) {
	return nil, nil
}

type mockURLExtractor struct{}

func (m *mockURLExtractor) ExtractExternalURLsFromEntries(entries []urlextraction.ContentProvider) (map[string][]url.URL, error) {
	return nil, nil
}

func (m *mockURLExtractor) ExtractImageURLsFromEntries(entries []urlextraction.ContentProvider) (map[string][]url.URL, error) {
	return nil, nil
}

func (m *mockURLExtractor) ExtractExternalURLsFromEntry(entry urlextraction.ContentProvider) ([]url.URL, error) {
	return nil, nil
}

func (m *mockURLExtractor) ExtractImageURLsFromEntry(entry urlextraction.ContentProvider) ([]url.URL, error) {
	return nil, nil
}

type mockImageFetcher struct{}

func (m *mockImageFetcher) FetchAsBase64(url string) (string, error) {
	return "", nil
}

func TestNewProcessor(t *testing.T) {
	mockClient := &mockOpenAIClient{}
	mockImageClient := &mockOpenAIClient{}
	mockArtclExtractor := &mockArticleExtractor{}
	mockURLFetcher := &mockFetcher{}
	mockURLExtrctor := &mockURLExtractor{}
	mockImgFetcher := &mockImageFetcher{}

	config := EntryProcessConfig{
		URLSummaryEnabled:    true,
		ImageEnabled:         true,
		DebugOutputBenchmark: true,
		InitialBackoff:       1 * time.Second,
		BackoffFactor:        2.0,
		MaxRetries:           3,
		MaxBackoff:           30 * time.Second,
	}

	processor := NewProcessor(mockClient, mockImageClient, config, mockArtclExtractor, mockURLFetcher, mockURLExtrctor, mockImgFetcher)

	assert.Equal(t, mockClient, processor.client.(*mockOpenAIClient), "client should match")
	assert.Equal(t, mockImageClient, processor.imageClient.(*mockOpenAIClient), "imageClient should match")
	assert.Equal(t, mockURLFetcher, processor.urlFetcher.(*mockFetcher), "urlFetcher should match")
	assert.Equal(t, config, processor.config, "config should match")
	assert.Equal(t, config.URLSummaryEnabled, processor.urlSummaryEnabled, "urlSummaryEnabled should match")
	assert.Equal(t, mockURLExtrctor, processor.urlExtractor.(*mockURLExtractor), "urlExtractor should match")
	assert.Equal(t, config.ImageEnabled, processor.imageEnabled, "imageEnabled should match")
	assert.Equal(t, config.DebugOutputBenchmark, processor.debugOutputBenchmark, "debugOutputBenchmark should match")
	assert.Equal(t, mockImgFetcher, processor.imageFetcher.(*mockImageFetcher), "imageFetcher should match")
	assert.Equal(t, mockArtclExtractor, processor.articleExtractor.(*mockArticleExtractor), "articleExtractor should match")
}


func TestFilterRelevantItems(t *testing.T) {
	t.Run("mixed relevant and irrelevant items", func(t *testing.T) {
		items := []models.Item{
			{ID: "1", IsRelevant: true, Title: "Relevant Item 1"},
			{ID: "", IsRelevant: true, Title: "Relevant Item No ID"},
			{ID: "3", IsRelevant: false, Title: "Irrelevant Item"},
			{ID: "4", IsRelevant: true, Title: "Relevant Item 2"},
		}

		expectedItems := []models.Item{
			{ID: "1", IsRelevant: true, Title: "Relevant Item 1"},
			{ID: "4", IsRelevant: true, Title: "Relevant Item 2"},
		}

		filteredItems := FilterRelevantItems(items)
		assert.Equal(t, expectedItems, filteredItems, "should filter out irrelevant items and items without ID")
	})

	t.Run("no relevant items", func(t *testing.T) {
		noRelevantItems := []models.Item{
			{ID: "1", IsRelevant: false, Title: "Irrelevant 1"},
			{ID: "2", IsRelevant: false, Title: "Irrelevant 2"},
		}

		filteredNoRelevant := FilterRelevantItems(noRelevantItems)
		assert.Empty(t, filteredNoRelevant, "should return empty slice when no items are relevant")
	})

	t.Run("all relevant items", func(t *testing.T) {
		allRelevantItems := []models.Item{
			{ID: "1", IsRelevant: true, Title: "Relevant 1"},
			{ID: "2", IsRelevant: true, Title: "Relevant 2"},
		}
		expectedAllRelevant := []models.Item{
			{ID: "1", IsRelevant: true, Title: "Relevant 1"},
			{ID: "2", IsRelevant: true, Title: "Relevant 2"},
		}

		filteredAllRelevant := FilterRelevantItems(allRelevantItems)
		assert.Equal(t, expectedAllRelevant, filteredAllRelevant, "should return all items when all are relevant")
	})

	t.Run("empty input", func(t *testing.T) {
		emptyItems := []models.Item{}

		filteredEmpty := FilterRelevantItems(emptyItems)
		assert.Empty(t, filteredEmpty, "should return empty slice for empty input")
	})
}

func TestLlmResponseToItems(t *testing.T) {
	t.Run("valid json", func(t *testing.T) {
		jsonStr := `{"id":"123","title":"Test Title","summary":"Test Summary","isRelevant":true}`
		expectedItem := models.Item{
			ID:         "123",
			Title:      "Test Title",
			Summary:    "Test Summary",
			IsRelevant: true,
		}

		item, err := llmResponseToItems(jsonStr)
		assert.NoError(t, err, "should not return error for valid JSON")
		assert.Equal(t, expectedItem, item, "parsed item should match expected")
	})

	t.Run("invalid json", func(t *testing.T) {
		jsonStr := `{"id":"123","title":"Test Title",}` // Invalid JSON, trailing comma
		_, err := llmResponseToItems(jsonStr)
		assert.Error(t, err, "should return error for invalid JSON")
	})

	t.Run("empty json string", func(t *testing.T) {
		jsonStr := ""
		_, err := llmResponseToItems(jsonStr)
		assert.Error(t, err, "should return error for empty JSON string")
	})

	t.Run("json with missing fields", func(t *testing.T) {
		jsonStr := `{"id":"123","title":"Test Title"}`
		expectedItem := models.Item{
			ID:    "123",
			Title: "Test Title",
		}

		item, err := llmResponseToItems(jsonStr)
		assert.NoError(t, err, "should not return error for JSON with missing optional fields")
		assert.Equal(t, expectedItem, item, "parsed item should match expected with missing fields")
	})

	t.Run("json with extra fields", func(t *testing.T) {
		jsonStr := `{"id":"123","title":"Test Title","extra_field":"should be ignored","summary":"Test Overview"}`
		expectedItem := models.Item{
			ID:      "123",
			Title:   "Test Title",
			Summary: "Test Overview",
		}

		item, err := llmResponseToItems(jsonStr)
		assert.NoError(t, err, "should not return error for JSON with extra fields")
		assert.Equal(t, expectedItem, item, "parsed item should match expected, ignoring extra fields")
	})
}
