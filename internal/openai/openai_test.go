package openai

import "testing"

func TestPreprocessJSON(t *testing.T) {
	client := &Client{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic json with code blocks",
			input:    "```json\n{\"key\": \"value\"}\n```",
			expected: "{\"key\": \"value\"}",
		},
		{
			name:     "json with think tags",
			input:    "<think>Let me process this</think>```json\n{\"key\": \"value\"}\n```",
			expected: "{\"key\": \"value\"}",
		},
		{
			name:     "multiple think tags",
			input:    "<think>First thought</think>```json\n{\"data\": 123}\n```<think>Another thought</think>",
			expected: "{\"data\": 123}",
		},
		{
			name:     "think tags without json markers",
			input:    "<think>Some thought</think>{\"raw\": \"json\"}",
			expected: "{\"raw\": \"json\"}",
		},
		{
			name:     "nested json content",
			input:    "```json\n{\"outer\": {\"inner\": \"value\"}}\n```",
			expected: "{\"outer\": {\"inner\": \"value\"}}",
		},
		{
			name:     "think tags with whitespace",
			input:    "<think>\n  Processing...\n</think>```json\n{\"key\": \"value\"}\n```",
			expected: "{\"key\": \"value\"}",
		},
		{
			name:     "no json markers",
			input:    "{\"plain\": \"json\"}",
			expected: "{\"plain\": \"json\"}",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "only think tags",
			input:    "<think>Just thinking</think>",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.PreprocessJSON(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
