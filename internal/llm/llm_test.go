package llm

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/bakkerme/ai-news-processor/internal/contentextractor"
	"github.com/bakkerme/ai-news-processor/internal/customerrors"
	"github.com/bakkerme/ai-news-processor/internal/http/retry"
	"github.com/bakkerme/ai-news-processor/internal/models"
	"github.com/bakkerme/ai-news-processor/internal/openai"
	"github.com/bakkerme/ai-news-processor/internal/rss"
)

// Mock implementations for dependencies
type mockOpenAIClient struct {
	ChatCompletionFunc func(systemPrompt string, userPrompts []string, imageURLs []string, schemaParams *openai.SchemaParameters, results chan customerrors.ErrorString)
}

func (m *mockOpenAIClient) ChatCompletion(systemPrompt string, userPrompts []string, imageURLs []string, schemaParams *openai.SchemaParameters, results chan customerrors.ErrorString) {
	if m.ChatCompletionFunc != nil {
		m.ChatCompletionFunc(systemPrompt, userPrompts, imageURLs, schemaParams, results)
		return
	}
	// Default mock behavior if ChatCompletionFunc is not set
	close(results) // Or send a default response
}
func (m *mockOpenAIClient) PreprocessJSON(s string) string          { return s }
func (m *mockOpenAIClient) SetRetryConfig(config retry.RetryConfig) {}
func (m *mockOpenAIClient) PreprocessYAML(response string) string   { return response }

type mockArticleExtractor struct{}

func (m *mockArticleExtractor) Extract(body io.Reader, url *url.URL) (*contentextractor.ArticleData, error) {
	return &contentextractor.ArticleData{}, nil
}

type mockFetcher struct{}

func (m *mockFetcher) Fetch(ctx context.Context, url string) (*http.Response, error) {
	return nil, nil
}

type mockURLExtractor struct{}

func (m *mockURLExtractor) ExtractURLsFromEntry(entry rss.Entry) ([]string, error) {
	return nil, nil
}

func (m *mockURLExtractor) ExtractURLsFromEntries(entries []rss.Entry) (map[string][]string, error) {
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

	if processor.client.(*mockOpenAIClient) != mockClient {
		t.Errorf("Expected client to be %v, got %v", mockClient, processor.client)
	}
	if processor.imageClient.(*mockOpenAIClient) != mockImageClient {
		t.Errorf("Expected imageClient to be %v, got %v", mockImageClient, processor.imageClient)
	}
	if processor.urlFetcher.(*mockFetcher) != mockURLFetcher {
		t.Errorf("Expected urlFetcher to be %v, got %v", mockURLFetcher, processor.urlFetcher)
	}
	if !reflect.DeepEqual(processor.config, config) {
		t.Errorf("Expected config to be %v, got %v", config, processor.config)
	}
	if processor.urlSummaryEnabled != config.URLSummaryEnabled {
		t.Errorf("Expected urlSummaryEnabled to be %v, got %v", config.URLSummaryEnabled, processor.urlSummaryEnabled)
	}
	if processor.urlExtractor.(*mockURLExtractor) != mockURLExtrctor {
		t.Errorf("Expected urlExtractor to be %v, got %v", mockURLExtrctor, processor.urlExtractor)
	}
	if processor.imageEnabled != config.ImageEnabled {
		t.Errorf("Expected imageEnabled to be %v, got %v", config.ImageEnabled, processor.imageEnabled)
	}
	if processor.debugOutputBenchmark != config.DebugOutputBenchmark {
		t.Errorf("Expected debugOutputBenchmark to be %v, got %v", config.DebugOutputBenchmark, processor.debugOutputBenchmark)
	}
	if processor.imageFetcher.(*mockImageFetcher) != mockImgFetcher {
		t.Errorf("Expected imageFetcher to be %v, got %v", mockImgFetcher, processor.imageFetcher)
	}
	if processor.articleExtractor.(*mockArticleExtractor) != mockArtclExtractor {
		t.Errorf("Expected articleExtractor to be %v, got %v", mockArtclExtractor, processor.articleExtractor)
	}
}

func TestEnrichItems(t *testing.T) {
	items := []models.Item{
		{ID: "1", Title: "Item 1"},
		{ID: "2", Title: "Item 2"},
		{ID: "3", Title: "Item 3"}, // No corresponding entry
	}
	entries := []rss.Entry{
		{ID: "1", Link: rss.Link{Href: "http://example.com/1"}},
		{ID: "2", Link: rss.Link{Href: "http://example.com/2"}},
		{ID: "nonexistent", Link: rss.Link{Href: "http://example.com/nonexistent"}},
	}

	expectedItems := []models.Item{
		{ID: "1", Title: "Item 1", Link: "http://example.com/1"},
		{ID: "2", Title: "Item 2", Link: "http://example.com/2"},
		{ID: "3", Title: "Item 3"},
	}

	enrichedItems := EnrichItems(items, entries)

	if !reflect.DeepEqual(enrichedItems, expectedItems) {
		t.Errorf("Expected enriched items %v, got %v", expectedItems, enrichedItems)
	}

	// Test with an item that has no ID
	itemsWithNoID := []models.Item{
		{Title: "Item No ID"},
		{ID: "1", Title: "Item 1"},
	}
	expectedWithNoID := []models.Item{
		{Title: "Item No ID"},
		{ID: "1", Title: "Item 1", Link: "http://example.com/1"},
	}
	enrichedNoID := EnrichItems(itemsWithNoID, entries)
	if !reflect.DeepEqual(enrichedNoID, expectedWithNoID) {
		t.Errorf("Expected enriched items with no ID %v, got %v", expectedWithNoID, enrichedNoID)
	}

	// Test with empty items
	emptyItems := []models.Item{}
	expectedEmpty := []models.Item{}
	enrichedEmpty := EnrichItems(emptyItems, entries)
	if !reflect.DeepEqual(enrichedEmpty, expectedEmpty) {
		t.Errorf("Expected empty enriched items %v, got %v", expectedEmpty, enrichedEmpty)
	}

	// Test with empty entries
	emptyEntries := []rss.Entry{}
	expectedEmptyEntries := []models.Item{
		{ID: "1", Title: "Item 1"},
		{ID: "2", Title: "Item 2"},
		{ID: "3", Title: "Item 3"},
	}
	enrichedEmptyEntries := EnrichItems(items, emptyEntries)
	if !reflect.DeepEqual(enrichedEmptyEntries, expectedEmptyEntries) {
		t.Errorf("Expected enriched items with empty entries %v, got %v", expectedEmptyEntries, enrichedEmptyEntries)
	}
}

func TestFilterRelevantItems(t *testing.T) {
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

	if !reflect.DeepEqual(filteredItems, expectedItems) {
		t.Errorf("Expected filtered items %v, got %v", expectedItems, filteredItems)
	}

	// Test with no relevant items
	noRelevantItems := []models.Item{
		{ID: "1", IsRelevant: false, Title: "Irrelevant 1"},
		{ID: "2", IsRelevant: false, Title: "Irrelevant 2"},
	}
	expectedNoRelevant := []models.Item{}
	filteredNoRelevant := FilterRelevantItems(noRelevantItems)
	if len(expectedNoRelevant) == 0 && len(filteredNoRelevant) == 0 {
		// Both are empty (one might be nil, other empty non-nil), consider it a pass for this case
	} else if !reflect.DeepEqual(filteredNoRelevant, expectedNoRelevant) {
		t.Errorf("Expected no relevant items %v, got %v", expectedNoRelevant, filteredNoRelevant)
	}

	// Test with all relevant items
	allRelevantItems := []models.Item{
		{ID: "1", IsRelevant: true, Title: "Relevant 1"},
		{ID: "2", IsRelevant: true, Title: "Relevant 2"},
	}
	expectedAllRelevant := []models.Item{
		{ID: "1", IsRelevant: true, Title: "Relevant 1"},
		{ID: "2", IsRelevant: true, Title: "Relevant 2"},
	}
	filteredAllRelevant := FilterRelevantItems(allRelevantItems)
	if !reflect.DeepEqual(filteredAllRelevant, expectedAllRelevant) {
		t.Errorf("Expected all relevant items %v, got %v", expectedAllRelevant, filteredAllRelevant)
	}

	// Test with empty input
	emptyItems := []models.Item{}
	expectedEmpty := []models.Item{}
	filteredEmpty := FilterRelevantItems(emptyItems)
	if len(expectedEmpty) == 0 && len(filteredEmpty) == 0 {
		// Both are empty, consider it a pass
	} else if !reflect.DeepEqual(filteredEmpty, expectedEmpty) {
		t.Errorf("Expected empty filtered items %v, got %v", expectedEmpty, filteredEmpty)
	}
}

func TestLlmResponseToItems(t *testing.T) {
	t.Run("valid json", func(t *testing.T) {
		jsonStr := `{"id":"123","title":"Test Title","overview":"Test Summary","is_relevant":true}`
		expectedItem := models.Item{
			ID:         "123",
			Title:      "Test Title",
			Summary:    "Test Summary",
			IsRelevant: true,
		}

		item, err := llmResponseToItems(jsonStr)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if !reflect.DeepEqual(item, expectedItem) {
			t.Errorf("Expected item %v, got %v", expectedItem, item)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		jsonStr := `{"id":"123","title":"Test Title",}` // Invalid JSON, trailing comma
		_, err := llmResponseToItems(jsonStr)
		if err == nil {
			t.Fatal("Expected an error for invalid JSON, got nil")
		}
	})

	t.Run("empty json string", func(t *testing.T) {
		jsonStr := ""
		_, err := llmResponseToItems(jsonStr)
		if err == nil {
			t.Fatal("Expected an error for empty JSON string, got nil")
		}
	})

	t.Run("json with missing fields", func(t *testing.T) {
		jsonStr := `{"id":"123","title":"Test Title"}`
		expectedItem := models.Item{
			ID:    "123",
			Title: "Test Title",
		}
		item, err := llmResponseToItems(jsonStr)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if !reflect.DeepEqual(item, expectedItem) {
			t.Errorf("Expected item %v, got %v", expectedItem, item)
		}
	})

	t.Run("json with extra fields", func(t *testing.T) {
		jsonStr := `{"id":"123","title":"Test Title","extra_field":"should be ignored","overview":"Test Overview"}`
		expectedItem := models.Item{
			ID:      "123",
			Title:   "Test Title",
			Summary: "Test Overview",
		}
		item, err := llmResponseToItems(jsonStr)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if !reflect.DeepEqual(item, expectedItem) {
			t.Errorf("Expected item %v, got %v", expectedItem, item)
		}
	})
}
