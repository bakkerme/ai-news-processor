package llm

import (
	"testing"

	"github.com/bakkerme/ai-news-processor/internal/customerrors"
	"github.com/bakkerme/ai-news-processor/internal/http/retry"
	"github.com/bakkerme/ai-news-processor/internal/openai"
	"github.com/stretchr/testify/assert"
)

// MockOpenAIClient is a mock implementation of the openai.OpenAIClient interface.
type MockOpenAIClient struct {
	SetRetryConfigFunc func(config retry.RetryConfig)
	PreprocessYAMLFunc func(response string) string
	PreprocessJSONFunc func(response string) string
	GetModelNameFunc   func() string

	// Store calls to verify
	CalledChatCompletion bool
	CalledSetRetryConfig bool
	CalledPreprocessYAML bool
	CalledPreprocessJSON bool
	CalledGetModelName   bool

	LastSystemPrompt string
	LastUserPrompts  []string
	LastImageURLs    []string
	LastSchemaParams *openai.SchemaParameters
	LastTemperature  float64
	LastMaxTokens    int
	LastRetryConfig  retry.RetryConfig
	LastYAMLResponse string
	LastJSONResponse string
}

// ChatCompletion implements the openai.OpenAIClient interface.
func (m *MockOpenAIClient) ChatCompletion(systemPrompt string, userPrompts []string, imageURLs []string, schemaParams *openai.SchemaParameters, temperature float64, maxTokens int, results chan customerrors.ErrorString) {
	m.CalledChatCompletion = true
	m.LastSystemPrompt = systemPrompt
	m.LastUserPrompts = userPrompts
	m.LastImageURLs = imageURLs
	m.LastSchemaParams = schemaParams
	m.LastTemperature = temperature
	m.LastMaxTokens = maxTokens

	// Default behavior: send an empty successful result
	go func() {
		results <- customerrors.ErrorString{Value: "mocked response", Err: nil}
	}()
}

// SetRetryConfig implements the openai.OpenAIClient interface.
func (m *MockOpenAIClient) SetRetryConfig(config retry.RetryConfig) {
	m.CalledSetRetryConfig = true
	m.LastRetryConfig = config
	if m.SetRetryConfigFunc != nil {
		m.SetRetryConfigFunc(config)
	}
}

// PreprocessYAML implements the openai.OpenAIClient interface.
func (m *MockOpenAIClient) PreprocessYAML(response string) string {
	m.CalledPreprocessYAML = true
	m.LastYAMLResponse = response
	if m.PreprocessYAMLFunc != nil {
		return m.PreprocessYAMLFunc(response)
	}
	return response // Default behavior
}

// PreprocessJSON implements the openai.OpenAIClient interface.
func (m *MockOpenAIClient) PreprocessJSON(response string) string {
	m.CalledPreprocessJSON = true
	m.LastJSONResponse = response
	if m.PreprocessJSONFunc != nil {
		return m.PreprocessJSONFunc(response)
	}
	return response // Default behavior
}

// GetModelName implements the openai.OpenAIClient interface.
func (m *MockOpenAIClient) GetModelName() string {
	m.CalledGetModelName = true
	if m.GetModelNameFunc != nil {
		return m.GetModelNameFunc()
	}
	return "mock-model" // Default behavior
}

// TestChatCompletionForEntrySummary tests chatCompletionForEntrySummary.
func TestChatCompletionForEntrySummary(t *testing.T) {
	mockClient := &MockOpenAIClient{}
	systemPrompt := "test system prompt for entry summary"
	userPrompts := []string{"user prompt 1", "user prompt 2"}
	imageURLs := []string{"http://example.com/image1.jpg"}
	results := make(chan customerrors.ErrorString, 1)

	chatCompletionForEntrySummary(mockClient, systemPrompt, userPrompts, imageURLs, results)

	// Wait for the goroutine in ChatCompletion to send a result
	<-results

	assert.True(t, mockClient.CalledChatCompletion, "ChatCompletion should have been called")
	assert.Equal(t, systemPrompt, mockClient.LastSystemPrompt)
	assert.Equal(t, userPrompts, mockClient.LastUserPrompts)
	assert.Equal(t, imageURLs, mockClient.LastImageURLs)
	// SchemaParams are currently nil in the tested function
	assert.Nil(t, mockClient.LastSchemaParams)
	assert.Equal(t, 0.5, mockClient.LastTemperature)
	assert.Equal(t, MaxTokensEntrySummary, mockClient.LastMaxTokens)
}

func TestChatCompletionForFeedSummary(t *testing.T) {
	mockClient := &MockOpenAIClient{}
	systemPrompt := "test system prompt for feed summary"
	userPrompts := []string{"feed user prompt 1", "feed user prompt 2"}
	results := make(chan customerrors.ErrorString, 1)

	chatCompletionForFeedSummary(mockClient, systemPrompt, userPrompts, results)

	// Wait for the goroutine in ChatCompletion to send a result
	<-results

	assert.True(t, mockClient.CalledChatCompletion, "ChatCompletion should have been called")
	assert.Equal(t, systemPrompt, mockClient.LastSystemPrompt)
	assert.Equal(t, userPrompts, mockClient.LastUserPrompts)
	assert.Equal(t, []string{}, mockClient.LastImageURLs, "ImageURLs should be empty for feed summary")
	// SchemaParams are currently nil in the tested function
	assert.Nil(t, mockClient.LastSchemaParams)
	assert.Equal(t, 0.5, mockClient.LastTemperature)
	assert.Equal(t, MaxTokensFeedSummary, mockClient.LastMaxTokens)
}

// TestChatCompletionImageSummary tests chatCompletionImageSummary.
func TestChatCompletionImageSummary(t *testing.T) {
	mockClient := &MockOpenAIClient{}
	systemPrompt := "test system prompt for image summary"
	imageURLs := []string{"http://example.com/image2.png"}

	// Use the default mock behavior
	description, err := chatCompletionImageSummary(mockClient, systemPrompt, imageURLs)

	assert.NoError(t, err)
	assert.Equal(t, "mocked response", description)
	assert.True(t, mockClient.CalledChatCompletion, "ChatCompletion should have been called")
	assert.Equal(t, systemPrompt, mockClient.LastSystemPrompt)
	assert.Equal(t, []string{}, mockClient.LastUserPrompts, "UserPrompts should be empty for image summary")
	assert.Equal(t, imageURLs, mockClient.LastImageURLs)
	assert.Nil(t, mockClient.LastSchemaParams, "SchemaParams should be nil for image summary")
	assert.Equal(t, 0.1, mockClient.LastTemperature)
	assert.Equal(t, MaxTokensImageSummary, mockClient.LastMaxTokens)
}

// TestMaxTokenLimitsPreventInfiniteGeneration verifies that max token limits are properly set
func TestMaxTokenLimitsPreventInfiniteGeneration(t *testing.T) {
	t.Run("EntrySummary", func(t *testing.T) {
		mockClient := &MockOpenAIClient{}
		results := make(chan customerrors.ErrorString, 1)

		chatCompletionForEntrySummary(mockClient, "test", []string{"test"}, nil, results)
		<-results

		assert.Equal(t, MaxTokensEntrySummary, mockClient.LastMaxTokens, "Entry summary should use MaxTokensEntrySummary")
		assert.Greater(t, MaxTokensEntrySummary, 0, "MaxTokensEntrySummary should be greater than 0")
	})

	t.Run("FeedSummary", func(t *testing.T) {
		mockClient := &MockOpenAIClient{}
		results := make(chan customerrors.ErrorString, 1)

		chatCompletionForFeedSummary(mockClient, "test", []string{"test"}, results)
		<-results

		assert.Equal(t, MaxTokensFeedSummary, mockClient.LastMaxTokens, "Feed summary should use MaxTokensFeedSummary")
		assert.Greater(t, MaxTokensFeedSummary, 0, "MaxTokensFeedSummary should be greater than 0")
	})

	t.Run("ImageSummary", func(t *testing.T) {
		mockClient := &MockOpenAIClient{}

		_, err := chatCompletionImageSummary(mockClient, "test", []string{"test"})
		assert.NoError(t, err)

		assert.Equal(t, MaxTokensImageSummary, mockClient.LastMaxTokens, "Image summary should use MaxTokensImageSummary")
		assert.Greater(t, MaxTokensImageSummary, 0, "MaxTokensImageSummary should be greater than 0")
	})

	t.Run("WebSummary", func(t *testing.T) {
		mockClient := &MockOpenAIClient{}
		processor := &Processor{client: mockClient}

		_, err := processor.chatCompletionForWebSummary("test", "test")
		assert.NoError(t, err)

		assert.Equal(t, MaxTokensWebSummary, mockClient.LastMaxTokens, "Web summary should use MaxTokensWebSummary")
		assert.Greater(t, MaxTokensWebSummary, 0, "MaxTokensWebSummary should be greater than 0")
	})
}

// TODO: Add tests for error cases, e.g., when the client.ChatCompletion sends an error on the results channel.
