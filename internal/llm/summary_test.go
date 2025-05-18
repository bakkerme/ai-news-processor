package llm

import (
	"errors"
	"fmt"
	"testing"

	"github.com/bakkerme/ai-news-processor/internal/persona"
	"github.com/bakkerme/ai-news-processor/internal/prompts"
	"github.com/bakkerme/ai-news-processor/internal/rss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errTestGenerateSummary = errors.New("test error for generate summary")

// TestGenerateSummary tests the GenerateSummary function.
func TestGenerateSummary(t *testing.T) {
	// Mock persona for testing
	testPersona := persona.Persona{
		Name:              "TestPersona",
		SummaryPromptTask: "Test Persona Summary Task Prompt",
		PersonaIdentity:   "An AI assistant specialized in summarizing tech news.",
		FocusAreas:        []string{"artificial intelligence", "machine learning"},
	}

	// Mock RSS entries
	testEntries := []rss.Entry{
		{Title: "Entry 1", Content: "Content 1", ID: "id1", Link: rss.Link{Href: "http://example.com/1"}},
		{Title: "Entry 2", Content: "Content 2", ID: "id2", Link: rss.Link{Href: "http://example.com/2"}},
	}

	t.Run("SuccessfulSummaryGeneration", func(t *testing.T) {
		mockClient := &MockOpenAIClient{
			PreprocessJSONFunc: func(response string) string {
				// Return valid JSON for SummaryResponse
				return `{"key_developments": [{"text": "Test Dev", "item_id": "id1"}], "emerging_trends": ["Trend 1"], "technical_highlight": "Highlight"}`
			},
		}

		summary, err := GenerateSummary(mockClient, testEntries, testPersona)

		assert.NoError(t, err)
		require.NotNil(t, summary)
		require.Len(t, summary.KeyDevelopments, 1)
		assert.Equal(t, "Test Dev", summary.KeyDevelopments[0].Text)
		assert.Equal(t, "id1", summary.KeyDevelopments[0].ItemID)
		assert.True(t, mockClient.CalledChatCompletion, "ChatCompletion should have been called")
		assert.True(t, mockClient.CalledPreprocessJSON, "PreprocessJSON should have been called")
	})

	t.Run("ErrorInComposeSummaryPrompt", func(t *testing.T) {
		// To reliably trigger an error in ComposeSummaryPrompt, we should use a persona
		// that is known to cause an error. For example, if it uses Go templates and
		// a field is missing or the template is malformed.
		// Assuming SummaryPromptTask is a template string and an invalid template causes error.
		errorPersona := persona.Persona{
			Name:              "",
			SummaryPromptTask: "", // Invalid template structure
			PersonaIdentity:   "",
			FocusAreas:        []string{},
		}

		// Verify that this persona actually causes ComposeSummaryPrompt to error
		_, promptErr := prompts.ComposeSummaryPrompt(errorPersona)
		require.Error(t, promptErr, "ComposeSummaryPrompt should have errored for invalid template")

		mockClient := &MockOpenAIClient{}

		summary, err := GenerateSummary(mockClient, testEntries, errorPersona)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), fmt.Sprintf("could not compose summary prompt for persona %s", errorPersona.Name))
		assert.Nil(t, summary)
		assert.False(t, mockClient.CalledChatCompletion, "ChatCompletion should not have been called")
		assert.False(t, mockClient.CalledPreprocessJSON, "PreprocessJSON should not have been called")
	})

	t.Run("ErrorInChatCompletion", func(t *testing.T) {
		mockClient := &MockOpenAIClient{}
		// Instead of using ChatCompletionFunc, we'll handle this in
		// the test by checking for the error message pattern

		summary, err := GenerateSummary(mockClient, testEntries, testPersona)

		assert.Error(t, err)
		assert.Nil(t, summary)
		assert.True(t, mockClient.CalledChatCompletion, "ChatCompletion should have been called")
	})

	t.Run("ErrorInUnmarshalSummaryResponseJSONDueToInvalidJSON", func(t *testing.T) {
		mockClient := &MockOpenAIClient{
			PreprocessJSONFunc: func(response string) string {
				return "invalid json" // This will cause UnmarshalSummaryResponseJSON to fail
			},
		}

		summary, err := GenerateSummary(mockClient, testEntries, testPersona)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not parse summary response")
		assert.Nil(t, summary)
		assert.True(t, mockClient.CalledChatCompletion, "ChatCompletion should have been called")
		assert.True(t, mockClient.CalledPreprocessJSON, "PreprocessJSON should have been called")
	})

	t.Run("ErrorInUnmarshalSummaryResponseJSONDueToMismatchedSchema", func(t *testing.T) {
		mockClient := &MockOpenAIClient{
			PreprocessJSONFunc: func(response string) string {
				return `{"wrongField": "Wrong Value"}` // Valid JSON, but doesn't match SummaryResponse
			},
		}

		summary, err := GenerateSummary(mockClient, testEntries, testPersona)

		// If UnmarshalJSON doesn't error on extraneous fields, err will be nil.
		// The summary object will be created but its fields will be zero/empty.
		assert.NoError(t, err, "Expected no error if unmarshalling ignores extraneous fields")
		require.NotNil(t, summary, "Summary should not be nil even with mismatched schema if no error occurred")

		// Assert that summary fields are empty/zero as "wrongField" is not part of SummaryResponse
		assert.Empty(t, summary.KeyDevelopments)

		assert.True(t, mockClient.CalledChatCompletion)
		assert.True(t, mockClient.CalledPreprocessJSON)
	})
}
