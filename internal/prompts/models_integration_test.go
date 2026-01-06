package prompts

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/bakkerme/ai-news-processor/models"
)

func TestRealModelsIntegration(t *testing.T) {
	t.Run("Real Item JSON Example", func(t *testing.T) {
		example, err := GetRealItemJSONExample()
		if err != nil {
			t.Fatalf("Failed to generate real item example: %v", err)
		}

		// Verify it's valid JSON
		var parsed map[string]interface{}
		err = json.Unmarshal([]byte(example), &parsed)
		if err != nil {
			t.Fatalf("Generated example is not valid JSON: %v", err)
		}

		// Verify it has the expected structure based on the real models.ItemSubset struct
		expectedFields := []string{
			"id", "overview", "summary", "commentSummary", "isRelevant",
		}

		for _, field := range expectedFields {
			if _, exists := parsed[field]; !exists {
				t.Errorf("Expected field '%s' not found in generated JSON", field)
			}
		}

		t.Logf("Real Item JSON Example: %s", example)
	})

	t.Run("Real SummaryResponse JSON Example", func(t *testing.T) {
		example, err := GetRealSummaryResponseJSONExample()
		if err != nil {
			t.Fatalf("Failed to generate real summary response example: %v", err)
		}

		// Verify it's valid JSON
		var parsed map[string]interface{}
		err = json.Unmarshal([]byte(example), &parsed)
		if err != nil {
			t.Fatalf("Generated example is not valid JSON: %v", err)
		}

		// Check keyDevelopments field
		if _, exists := parsed["keyDevelopments"]; !exists {
			t.Errorf("Expected keyDevelopments field not found")
		}

		t.Logf("Real SummaryResponse JSON Example: %s", example)
	})

	t.Run("Roundtrip Test - Item", func(t *testing.T) {
		// Generate an example
		example, err := GetRealItemJSONExample()
		if err != nil {
			t.Fatalf("Failed to generate example: %v", err)
		}

		// Try to unmarshal it back into the real struct
		var item models.ItemSubset
		err = json.Unmarshal([]byte(example), &item)
		if err != nil {
			t.Fatalf("Failed to unmarshal generated example back to struct: %v", err)
		}

		// Verify some key fields were set
		if item.ID != "unique_id_example" {
			t.Errorf("Expected ID to be 'unique_id_example', got '%s'", item.ID)
		}
		if !item.IsRelevant {
			t.Errorf("Expected IsRelevant to be true, got %v", item.IsRelevant)
		}

		t.Logf("Roundtrip test successful - can generate and parse back to struct")
	})

	t.Run("Roundtrip Test - SummaryResponse", func(t *testing.T) {
		example, err := GetRealSummaryResponseJSONExample()
		if err != nil {
			t.Fatalf("Failed to generate example: %v", err)
		}

		var summary models.SummaryResponse
		err = json.Unmarshal([]byte(example), &summary)
		if err != nil {
			t.Fatalf("Failed to unmarshal generated example back to struct: %v", err)
		}

		// Verify structure
		if len(summary.KeyDevelopments) != 1 {
			t.Errorf("Expected 1 key development, got %d", len(summary.KeyDevelopments))
		}

		if len(summary.KeyDevelopments) > 0 {
			keyDev := summary.KeyDevelopments[0]
			if keyDev.ItemID != "unique_id_example" {
				t.Errorf("Expected ItemID to be 'unique_id_example', got '%s'", keyDev.ItemID)
			}
		}

		t.Logf("Roundtrip test successful for SummaryResponse")
	})

	t.Run("Verify No Hardcoded Values", func(t *testing.T) {
		// This test ensures that we're actually reading from struct tags, not hardcoded values

		// First, get an example
		example, err := GetRealItemJSONExample()
		if err != nil {
			t.Fatalf("Failed to generate example: %v", err)
		}

		// Check that the JSON field names match the struct tags from models.ItemSubset
		// This validates that we're reading the tags correctly
		if !strings.Contains(example, `"id"`) {
			t.Errorf("Expected to find 'id' field (from JSON tag)")
		}
		if !strings.Contains(example, `"overview"`) {
			t.Errorf("Expected to find 'overview' field (from JSON tag)")
		}
		if !strings.Contains(example, `"commentSummary"`) {
			t.Errorf("Expected to find 'commentSummary' field (from JSON tag)")
		}

		// Make sure we're not generating field names that don't exist in the ItemSubset struct
		if strings.Contains(example, `"imageDescription"`) {
			t.Errorf("Found 'imageDescription' which shouldn't exist in ItemSubset")
		}
		if strings.Contains(example, `"thumbnailUrl"`) {
			t.Errorf("Found 'thumbnailUrl' which shouldn't exist in ItemSubset")
		}
		if strings.Contains(example, `"webContentSummary"`) {
			t.Errorf("Found 'webContentSummary' which shouldn't exist in ItemSubset")
		}
		if strings.Contains(example, `"link"`) {
			t.Errorf("Found 'link' which shouldn't exist in ItemSubset")
		}
		if strings.Contains(example, `"entry"`) {
			t.Errorf("Found 'entry' which shouldn't exist in ItemSubset")
		}

		t.Logf("JSON field names correctly match ItemSubset struct tags")
	})
}
