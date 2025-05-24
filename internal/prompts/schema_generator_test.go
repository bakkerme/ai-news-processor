package prompts

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestJSONExampleGenerator(t *testing.T) {
	generator := &JSONExampleGenerator{}

	t.Run("Generate Item Example", func(t *testing.T) {
		example, err := generator.GenerateJSONExample(createItemExample())
		if err != nil {
			t.Fatalf("Failed to generate item example: %v", err)
		}

		// Verify it's valid JSON
		var parsed map[string]interface{}
		err = json.Unmarshal([]byte(example), &parsed)
		if err != nil {
			t.Fatalf("Generated example is not valid JSON: %v", err)
		}

		// Check that key fields are present and have expected values
		if parsed["id"].(string) != "t3_1keo3te" {
			t.Errorf("Expected id to be 't3_1keo3te', got %v", parsed["id"])
		}
		if parsed["isRelevant"].(bool) != true {
			t.Errorf("Expected isRelevant to be true, got %v", parsed["isRelevant"])
		}
		if parsed["title"].(string) != "Example Article Title" {
			t.Errorf("Expected title to be 'Example Article Title', got %v", parsed["title"])
		}

		t.Logf("Generated Item JSON:\n%s", example)
	})

	t.Run("Generate Summary Response Example", func(t *testing.T) {
		example, err := generator.GenerateJSONExample(createSummaryResponseExample())
		if err != nil {
			t.Fatalf("Failed to generate summary response example: %v", err)
		}

		// Verify it's valid JSON
		var parsed map[string]interface{}
		err = json.Unmarshal([]byte(example), &parsed)
		if err != nil {
			t.Fatalf("Generated example is not valid JSON: %v", err)
		}

		// Check that keyDevelopments array is present
		keyDevs, ok := parsed["keyDevelopments"].([]interface{})
		if !ok {
			t.Fatalf("Expected keyDevelopments to be an array, got %T", parsed["keyDevelopments"])
		}

		if len(keyDevs) != 1 {
			t.Errorf("Expected 1 key development example, got %d", len(keyDevs))
		}

		// Check the structure of the key development
		if len(keyDevs) > 0 {
			keyDev := keyDevs[0].(map[string]interface{})
			if keyDev["text"].(string) != "Key development description..." {
				t.Errorf("Expected text to be 'Key development description...', got %v", keyDev["text"])
			}
			if keyDev["itemID"].(string) != "t3_1keo3te" {
				t.Errorf("Expected itemID to be 't3_1keo3te', got %v", keyDev["itemID"])
			}
		}

		t.Logf("Generated Summary Response JSON:\n%s", example)
	})

	t.Run("Generate Compact Examples", func(t *testing.T) {
		itemExample, err := GetItemJSONExample()
		if err != nil {
			t.Fatalf("Failed to get item JSON example: %v", err)
		}

		summaryExample, err := GetSummaryResponseJSONExample()
		if err != nil {
			t.Fatalf("Failed to get summary response JSON example: %v", err)
		}

		// Verify they're compact (single line)
		if strings.Contains(itemExample, "\n") {
			t.Errorf("Item example should be compact (single line), but contains newlines")
		}
		if strings.Contains(summaryExample, "\n") {
			t.Errorf("Summary example should be compact (single line), but contains newlines")
		}

		t.Logf("Compact Item JSON: %s", itemExample)
		t.Logf("Compact Summary JSON: %s", summaryExample)
	})
}

func TestPromptGeneration(t *testing.T) {

	t.Run("Test Base Prompt with JSON Generation", func(t *testing.T) {
		// This is a conceptual test - in the real implementation, we'd need to adapt
		// the ComposePrompt function to work with our test persona structure
		itemExample, err := GetItemJSONExample()
		if err != nil {
			t.Fatalf("Failed to get item JSON example: %v", err)
		}

		// Verify the JSON example contains the expected structure
		if !strings.Contains(itemExample, `"id":"t3_1keo3te"`) {
			t.Errorf("Item example should contain expected ID format")
		}
		if !strings.Contains(itemExample, `"isRelevant":true`) {
			t.Errorf("Item example should contain isRelevant field")
		}

		t.Logf("Generated item example for prompt: %s", itemExample)
	})

	t.Run("Test Summary Prompt with JSON Generation", func(t *testing.T) {
		summaryExample, err := GetSummaryResponseJSONExample()
		if err != nil {
			t.Fatalf("Failed to get summary JSON example: %v", err)
		}

		// Verify the JSON example contains the expected structure
		if !strings.Contains(summaryExample, `"keyDevelopments"`) {
			t.Errorf("Summary example should contain keyDevelopments field")
		}
		if !strings.Contains(summaryExample, `"text"`) {
			t.Errorf("Summary example should contain text field in key developments")
		}
		if !strings.Contains(summaryExample, `"itemID"`) {
			t.Errorf("Summary example should contain itemID field in key developments")
		}

		t.Logf("Generated summary example for prompt: %s", summaryExample)
	})
}

// TestStructSynchronization ensures that changes to structs are reflected in generated examples
func TestStructSynchronization(t *testing.T) {
	generator := &JSONExampleGenerator{}

	// Test with the current item structure
	t.Run("Item struct fields synchronization", func(t *testing.T) {
		example, err := generator.GenerateJSONExample(createItemExample())
		if err != nil {
			t.Fatalf("Failed to generate example: %v", err)
		}

		// Parse the generated JSON to check field presence
		var parsed map[string]interface{}
		err = json.Unmarshal([]byte(example), &parsed)
		if err != nil {
			t.Fatalf("Generated JSON is invalid: %v", err)
		}

		// Expected fields based on itemExample struct
		expectedFields := []string{
			"title", "id", "summary", "commentSummary",
			"imageDescription", "webContentSummary",
			"link", "isRelevant", "thumbnailUrl",
		}

		for _, field := range expectedFields {
			if _, exists := parsed[field]; !exists {
				t.Errorf("Expected field '%s' not found in generated JSON", field)
			}
		}

		t.Logf("All expected fields present in generated JSON")
	})

	t.Run("Summary response struct fields synchronization", func(t *testing.T) {
		example, err := generator.GenerateJSONExample(createSummaryResponseExample())
		if err != nil {
			t.Fatalf("Failed to generate example: %v", err)
		}

		var parsed map[string]interface{}
		err = json.Unmarshal([]byte(example), &parsed)
		if err != nil {
			t.Fatalf("Generated JSON is invalid: %v", err)
		}

		// Check keyDevelopments structure
		keyDevs, exists := parsed["keyDevelopments"]
		if !exists {
			t.Fatalf("keyDevelopments field not found")
		}

		keyDevsArray, ok := keyDevs.([]interface{})
		if !ok {
			t.Fatalf("keyDevelopments should be an array")
		}

		if len(keyDevsArray) > 0 {
			keyDev := keyDevsArray[0].(map[string]interface{})
			expectedSubFields := []string{"text", "itemID"}
			for _, field := range expectedSubFields {
				if _, exists := keyDev[field]; !exists {
					t.Errorf("Expected field '%s' not found in keyDevelopments item", field)
				}
			}
		}

		t.Logf("Summary response structure validated")
	})
}
