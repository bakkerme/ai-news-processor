package persona

import (
	"os"
	"path/filepath"
	"strings"
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

	// Create test persona YAML with comment threshold (RSS provider)
	personaWithThreshold := `name: "TestPersona"
provider: "rss"
feed_url: "https://example.com/test.rss"
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

	// Create test persona YAML without comment threshold (Reddit provider)
	personaWithoutThreshold := `name: "TestPersona2"
provider: "reddit"
subreddit: "test2"
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

func TestPersona_GetProvider(t *testing.T) {
	tests := []struct {
		name     string
		persona  Persona
		expected string
	}{
		{
			name: "uses explicit reddit provider",
			persona: Persona{
				Name:     "Test",
				Provider: "reddit",
			},
			expected: "reddit",
		},
		{
			name: "uses explicit rss provider",
			persona: Persona{
				Name:     "Test",
				Provider: "rss",
			},
			expected: "rss",
		},
		{
			name: "defaults to reddit when provider is empty",
			persona: Persona{
				Name:     "Test",
				Provider: "",
			},
			expected: "reddit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.persona.GetProvider()
			if result != tt.expected {
				t.Errorf("GetProvider() = %s, expected %s", result, tt.expected)
			}
		})
	}
}

func TestPersona_Validate(t *testing.T) {
	tests := []struct {
		name        string
		persona     Persona
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid reddit persona",
			persona: Persona{
				Name:      "Test",
				Provider:  "reddit",
				Subreddit: "test",
			},
			expectError: false,
		},
		{
			name: "valid rss persona",
			persona: Persona{
				Name:     "Test",
				Provider: "rss",
				FeedURL:  "https://example.com/feed.rss",
			},
			expectError: false,
		},
		{
			name: "reddit persona missing subreddit",
			persona: Persona{
				Name:     "Test",
				Provider: "reddit",
			},
			expectError: true,
			errorMsg:    "subreddit is required for reddit provider",
		},
		{
			name: "rss persona missing feed_url",
			persona: Persona{
				Name:     "Test",
				Provider: "rss",
			},
			expectError: true,
			errorMsg:    "feed_url is required for rss provider",
		},
		{
			name: "rss persona with invalid URL",
			persona: Persona{
				Name:     "Test",
				Provider: "rss",
				FeedURL:  "invalid-url",
			},
			expectError: true,
			errorMsg:    "feed_url must be a valid HTTP/HTTPS URL",
		},
		{
			name: "unsupported provider",
			persona: Persona{
				Name:     "Test",
				Provider: "unsupported",
			},
			expectError: true,
			errorMsg:    "unsupported provider 'unsupported'",
		},
		{
			name: "default reddit provider missing subreddit",
			persona: Persona{
				Name: "Test",
				// Provider defaults to "reddit"
			},
			expectError: true,
			errorMsg:    "subreddit is required for reddit provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.persona.Validate()
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain '%s', but got: %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %s", err.Error())
				}
			}
		})
	}
}

// Helper function to create an int pointer
func intPtr(i int) *int {
	return &i
}
