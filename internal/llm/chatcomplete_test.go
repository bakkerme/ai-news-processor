package llm

import (
	"errors"
	"testing"

	"github.com/bakkerme/ai-news-processor/internal/customerrors"
	"github.com/bakkerme/ai-news-processor/internal/http/retry"
	"github.com/bakkerme/ai-news-processor/internal/openai"
	"github.com/stretchr/testify/assert"
)

// MockOpenAIClient is a mock implementation of the openai.OpenAIClient interface.
type MockOpenAIClient struct {
	ChatCompletionFunc func(systemPrompt string, userPrompts []string, imageURLs []string, schemaParams *openai.SchemaParameters, results chan customerrors.ErrorString)
	SetRetryConfigFunc func(config retry.RetryConfig)
	PreprocessYAMLFunc func(response string) string
	PreprocessJSONFunc func(response string) string

	// Store calls to verify
	CalledChatCompletion bool
	CalledSetRetryConfig bool
	CalledPreprocessYAML bool
	CalledPreprocessJSON bool

	LastSystemPrompt string
	LastUserPrompts  []string
	LastImageURLs    []string
	LastSchemaParams *openai.SchemaParameters
	LastRetryConfig  retry.RetryConfig
	LastYAMLResponse string
	LastJSONResponse string
}

// ChatCompletion implements the openai.OpenAIClient interface.
func (m *MockOpenAIClient) ChatCompletion(systemPrompt string, userPrompts []string, imageURLs []string, schemaParams *openai.SchemaParameters, results chan customerrors.ErrorString) {
	m.CalledChatCompletion = true
	m.LastSystemPrompt = systemPrompt
	m.LastUserPrompts = userPrompts
	m.LastImageURLs = imageURLs
	m.LastSchemaParams = schemaParams
	if m.ChatCompletionFunc != nil {
		m.ChatCompletionFunc(systemPrompt, userPrompts, imageURLs, schemaParams, results)
	} else {
		// Default behavior: send an empty successful result
		go func() {
			results <- customerrors.ErrorString{Value: "mocked response", Err: nil}
		}()
	}
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

var errTest = errors.New("test error")

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
}

func TestChatCompletionImageSummary(t *testing.T) {
	t.Run("SuccessfulResponse", func(t *testing.T) {
		mockClient := &MockOpenAIClient{}
		systemPrompt := "test system prompt for image summary"
		imageURLs := []string{"http://example.com/image2.png"}
		expectedResponse := "mocked image description"

		// Override default mock behavior to control the response
		mockClient.ChatCompletionFunc = func(sp string, up []string, iu []string, sparam *openai.SchemaParameters, res chan customerrors.ErrorString) {
			res <- customerrors.ErrorString{Value: expectedResponse, Err: nil}
		}

		description, err := chatCompletionImageSummary(mockClient, systemPrompt, imageURLs)

		assert.NoError(t, err)
		assert.Equal(t, expectedResponse, description)
		assert.True(t, mockClient.CalledChatCompletion, "ChatCompletion should have been called")
		assert.Equal(t, systemPrompt, mockClient.LastSystemPrompt)
		assert.Equal(t, []string{}, mockClient.LastUserPrompts, "UserPrompts should be empty for image summary")
		assert.Equal(t, imageURLs, mockClient.LastImageURLs)
		assert.Nil(t, mockClient.LastSchemaParams, "SchemaParams should be nil for image summary")
	})

	t.Run("ErrorResponse", func(t *testing.T) {
		mockClient := &MockOpenAIClient{}
		systemPrompt := "test system prompt for image summary error"
		imageURLs := []string{"http://example.com/image_error.png"}

		// Override default mock behavior to send an error
		mockClient.ChatCompletionFunc = func(sp string, up []string, iu []string, sparam *openai.SchemaParameters, res chan customerrors.ErrorString) {
			res <- customerrors.ErrorString{Value: "", Err: errTest}
		}

		description, err := chatCompletionImageSummary(mockClient, systemPrompt, imageURLs)

		assert.Error(t, err)
		assert.Equal(t, errTest, err)
		assert.Equal(t, "", description)
		assert.True(t, mockClient.CalledChatCompletion, "ChatCompletion should have been called")
		assert.Equal(t, systemPrompt, mockClient.LastSystemPrompt)
		assert.Equal(t, []string{}, mockClient.LastUserPrompts)
		assert.Equal(t, imageURLs, mockClient.LastImageURLs)
		assert.Nil(t, mockClient.LastSchemaParams)
	})
}

func TestChatCompletionForWebSummary(t *testing.T) {
	t.Run("SuccessfulResponse", func(t *testing.T) {
		mockClient := &MockOpenAIClient{}
		processor := &Processor{client: mockClient} // Processor uses the client field
		systemPrompt := "test system prompt for web summary"
		userPrompt := "user prompt for web"
		expectedResponse := "mocked web summary"

		// Override default mock behavior to control the response
		mockClient.ChatCompletionFunc = func(sp string, up []string, iu []string, sparam *openai.SchemaParameters, res chan customerrors.ErrorString) {
			res <- customerrors.ErrorString{Value: expectedResponse, Err: nil}
		}

		summary, err := processor.chatCompletionForWebSummary(systemPrompt, userPrompt)

		assert.NoError(t, err)
		assert.Equal(t, expectedResponse, summary)
		assert.True(t, mockClient.CalledChatCompletion, "ChatCompletion should have been called")
		assert.Equal(t, systemPrompt, mockClient.LastSystemPrompt)
		assert.Equal(t, []string{userPrompt}, mockClient.LastUserPrompts)
		assert.Equal(t, []string{}, mockClient.LastImageURLs, "ImageURLs should be empty for web summary")
		assert.Nil(t, mockClient.LastSchemaParams, "SchemaParams should be nil for web summary")
	})

	t.Run("ErrorResponse", func(t *testing.T) {
		mockClient := &MockOpenAIClient{}
		processor := &Processor{client: mockClient}
		systemPrompt := "test system prompt for web summary error"
		userPrompt := "user prompt for web error"

		// Override default mock behavior to send an error
		mockClient.ChatCompletionFunc = func(sp string, up []string, iu []string, sparam *openai.SchemaParameters, res chan customerrors.ErrorString) {
			res <- customerrors.ErrorString{Value: "", Err: errTest}
		}

		summary, err := processor.chatCompletionForWebSummary(systemPrompt, userPrompt)

		assert.Error(t, err)
		assert.Equal(t, errTest, err)
		assert.Equal(t, "", summary)
		assert.True(t, mockClient.CalledChatCompletion, "ChatCompletion should have been called")
		assert.Equal(t, systemPrompt, mockClient.LastSystemPrompt)
		assert.Equal(t, []string{userPrompt}, mockClient.LastUserPrompts)
		assert.Equal(t, []string{}, mockClient.LastImageURLs)
		assert.Nil(t, mockClient.LastSchemaParams)
	})
}

// TODO: Add tests for error cases, e.g., when the client.ChatCompletion sends an error on the results channel.
// TODO: Add tests to ensure the Schema an Name and Description are passed correctly if/when they are re-enabled in chatcomplete.go.
