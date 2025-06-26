package persona

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Persona struct {
	Name    string `yaml:"name" json:"name"`        // Unique name for the persona (e.g., "LocalLLaMA")
	FeedURL string `yaml:"feed_url" json:"feedUrl"` // URL of the RSS feed (e.g., "https://reddit.com/r/localllama.rss")
	Topic   string `yaml:"topic" json:"topic"`      // Main subject area (e.g., "AI Technology", "Gardening")

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

// GetCommentThreshold returns the effective comment threshold for this persona.
// If the persona has a specific threshold set, it uses that. Otherwise, it falls back to the provided default.
func (p *Persona) GetCommentThreshold(defaultThreshold int) int {
	if p.CommentThreshold != nil {
		return *p.CommentThreshold
	}
	return defaultThreshold
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
		personas = append(personas, persona)
	}
	return personas, nil
}
