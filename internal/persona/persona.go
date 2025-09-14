package persona

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Persona struct {
	Name      string `yaml:"name" json:"name"`           // Unique name for the persona (e.g., "LocalLLaMA")
	Provider  string `yaml:"provider" json:"provider"`   // Data source provider: "reddit" or "rss" (defaults to "reddit" if not specified)
	Subreddit string `yaml:"subreddit" json:"subreddit"` // Subreddit name (e.g., "localllama") - used for reddit provider
	FeedURL   string `yaml:"feed_url" json:"feedURL"`    // RSS feed URL - used for rss provider
	Topic     string `yaml:"topic" json:"topic"`         // Main subject area (e.g., "AI Technology", "Gardening")

	// Persona identity (separated from specific task instructions)
	PersonaIdentity string `yaml:"persona_identity" json:"personaIdentity"` // Core identity and expertise of the persona

	// Task-specific instructions
	BasePromptTask    string `yaml:"base_prompt_task" json:"basePromptTask"`       // Task description for individual item analysis
	SummaryPromptTask string `yaml:"summary_prompt_task" json:"summaryPromptTask"` // Task description for summary generation

	// Content focus and criteria
	FocusAreas        []string `yaml:"focus_areas" json:"focusAreas"`               // List of topics/keywords to prioritize
	RelevanceCriteria []string `yaml:"relevance_criteria" json:"relevanceCriteria"` // List of criteria for relevance analysis
	SummaryAnalysis   []string `yaml:"summary_analysis" json:"summaryAnalysis"`     // Focus areas for summary analysis
	ExclusionCriteria []string `yaml:"exclusion_criteria" json:"exclusionCriteria"` // List of criteria to explicitly exclude items

	// Quality filtering
	CommentThreshold *int `yaml:"comment_threshold,omitempty" json:"commentThreshold,omitempty"` // Minimum number of comments for posts (optional, uses global default if not specified)
}

// GetProvider returns the effective provider for this persona.
// If the persona has a provider set, it uses that. Otherwise, it defaults to "reddit" for backward compatibility.
func (p *Persona) GetProvider() string {
	if p.Provider != "" {
		return p.Provider
	}
	return "reddit" // Default to reddit for backward compatibility
}

// GetCommentThreshold returns the effective comment threshold for this persona.
// If the persona has a specific threshold set, it uses that. Otherwise, it falls back to the provided default.
func (p *Persona) GetCommentThreshold(defaultThreshold int) int {
	if p.CommentThreshold != nil {
		return *p.CommentThreshold
	}
	return defaultThreshold
}

// Validate checks if the persona configuration is valid for its provider type
func (p *Persona) Validate() error {
	provider := p.GetProvider()
	
	switch provider {
	case "reddit":
		if p.Subreddit == "" {
			return fmt.Errorf("persona %s: subreddit is required for reddit provider", p.Name)
		}
	case "rss":
		if p.FeedURL == "" {
			return fmt.Errorf("persona %s: feed_url is required for rss provider", p.Name)
		}
		// Basic URL validation
		if !strings.HasPrefix(p.FeedURL, "http://") && !strings.HasPrefix(p.FeedURL, "https://") {
			return fmt.Errorf("persona %s: feed_url must be a valid HTTP/HTTPS URL", p.Name)
		}
	default:
		return fmt.Errorf("persona %s: unsupported provider '%s', must be 'reddit' or 'rss'", p.Name, provider)
	}
	
	return nil
}

// LoadPersonas loads all persona YAML files from the given directory
func LoadPersonas(dir string) ([]Persona, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var personas []Persona
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".yaml" {
			continue
		}
		path := filepath.Join(dir, file.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var persona Persona
		if err := yaml.Unmarshal(data, &persona); err != nil {
			return nil, err
		}
		
		// Validate persona configuration
		if err := persona.Validate(); err != nil {
			return nil, fmt.Errorf("invalid persona in file %s: %w", file.Name(), err)
		}
		
		personas = append(personas, persona)
	}
	return personas, nil
}
