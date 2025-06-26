package persona

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPersona_GetCommentThreshold(t *testing.T) {
	tests := []struct {
		name             string
		persona          Persona
		defaultThreshold int
		expected         int
	}{
		{
			name: "uses persona threshold when set",
			persona: Persona{
				Name:             "Test",
				CommentThreshold: intPtr(5),
			},
			defaultThreshold: 10,
			expected:         5,
		},
		{
			name: "uses default threshold when persona threshold is nil",
			persona: Persona{
				Name:             "Test",
				CommentThreshold: nil,
			},
			defaultThreshold: 10,
			expected:         10,
		},
		{
			name: "uses persona threshold of zero when explicitly set",
			persona: Persona{
				Name:             "Test",
				CommentThreshold: intPtr(0),
			},
			defaultThreshold: 10,
			expected:         0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.persona.GetCommentThreshold(tt.defaultThreshold)
			if result != tt.expected {
				t.Errorf("GetCommentThreshold() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

func TestLoadPersonas_WithCommentThreshold(t *testing.T) {
	// Create a temporary directory for test personas
	tmpDir, err := os.MkdirTemp("", "persona_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test persona YAML with comment threshold
	personaWithThreshold := `name: "TestPersona"
feed_url: "https://reddit.com/r/test.rss"
topic: "Test Topic"
persona_identity: "test persona"
base_prompt_task: "test task"
summary_prompt_task: "summary task"
focus_areas:
  - "test area"
relevance_criteria:
  - "test criteria"
exclusion_criteria:
  - "test exclusion"
summary_analysis:
  - "test analysis"
comment_threshold: 15`

	// Create test persona YAML without comment threshold
	personaWithoutThreshold := `name: "TestPersona2"
feed_url: "https://reddit.com/r/test2.rss"
topic: "Test Topic 2"
persona_identity: "test persona 2"
base_prompt_task: "test task 2"
summary_prompt_task: "summary task 2"
focus_areas:
  - "test area 2"
relevance_criteria:
  - "test criteria 2"
exclusion_criteria:
  - "test exclusion 2"
summary_analysis:
  - "test analysis 2"`

	// Write test files
	err = os.WriteFile(filepath.Join(tmpDir, "persona1.yaml"), []byte(personaWithThreshold), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	err = os.WriteFile(filepath.Join(tmpDir, "persona2.yaml"), []byte(personaWithoutThreshold), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Load personas
	personas, err := LoadPersonas(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load personas: %v", err)
	}

	if len(personas) != 2 {
		t.Fatalf("Expected 2 personas, got %d", len(personas))
	}

	// Find the personas by name
	var personaWith, personaWithout *Persona
	for _, p := range personas {
		if p.Name == "TestPersona" {
			personaWith = &p
		} else if p.Name == "TestPersona2" {
			personaWithout = &p
		}
	}

	// Verify the persona with threshold
	if personaWith == nil {
		t.Fatal("TestPersona not found")
	}
	if personaWith.CommentThreshold == nil {
		t.Error("Expected comment threshold to be set")
	} else if *personaWith.CommentThreshold != 15 {
		t.Errorf("Expected comment threshold 15, got %d", *personaWith.CommentThreshold)
	}

	// Verify the persona without threshold
	if personaWithout == nil {
		t.Fatal("TestPersona2 not found")
	}
	if personaWithout.CommentThreshold != nil {
		t.Error("Expected comment threshold to be nil")
	}

	// Test GetCommentThreshold with default
	if personaWith.GetCommentThreshold(10) != 15 {
		t.Error("Expected persona-specific threshold to be used")
	}
	if personaWithout.GetCommentThreshold(10) != 10 {
		t.Error("Expected default threshold to be used")
	}
}

func TestLoadActualPersonas(t *testing.T) {
	// Test loading the actual persona files to ensure they are valid
	personas, err := LoadPersonas("../../personas/")
	if err != nil {
		t.Fatalf("Failed to load actual personas: %v", err)
	}

	if len(personas) == 0 {
		t.Fatal("No personas loaded")
	}

	// Verify all personas have the comment threshold set correctly
	for _, p := range personas {
		if p.CommentThreshold == nil {
			t.Errorf("Persona %s does not have comment threshold set", p.Name)
			continue
		}

		// Define expected thresholds for specific personas
		expectedThreshold := 10 // Default for most personas
		if p.Name == "LLMDevs" {
			expectedThreshold = 0 // LLMDevs has low traffic, so threshold is 0
		}

		if *p.CommentThreshold != expectedThreshold {
			t.Errorf("Persona %s has comment threshold %d, expected %d", p.Name, *p.CommentThreshold, expectedThreshold)
		}

		// Verify GetCommentThreshold returns the expected value
		threshold := p.GetCommentThreshold(5) // Use different default to ensure persona value is used
		if threshold != expectedThreshold {
			t.Errorf("Persona %s GetCommentThreshold returned %d, expected %d", p.Name, threshold, expectedThreshold)
		}
	}
}

// Helper function to create an int pointer
func intPtr(i int) *int {
	return &i
}
