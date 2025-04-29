package common

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Persona struct {
	Name    string `yaml:"name"`     // Unique name for the persona (e.g., "LocalLLaMA")
	FeedURL string `yaml:"feed_url"` // URL of the RSS feed (e.g., "https://reddit.com/r/localllama.rss")
	Topic   string `yaml:"topic"`    // Main subject area (e.g., "AI Technology", "Gardening")

	// Persona identity (separated from specific task instructions)
	PersonaIdentity string `yaml:"persona_identity"` // Core identity and expertise of the persona

	// Task-specific instructions
	BasePromptTask    string `yaml:"base_prompt_task"`    // Task description for individual item analysis
	SummaryPromptTask string `yaml:"summary_prompt_task"` // Task description for summary generation

	// Content focus and criteria
	FocusAreas        []string `yaml:"focus_areas"`        // List of topics/keywords to prioritize
	RelevanceCriteria []string `yaml:"relevance_criteria"` // List of criteria for relevance analysis
	SummaryAnalysis   []string `yaml:"summary_analysis"`   // Focus areas for summary analysis
	ExclusionCriteria []string `yaml:"exclusion_criteria"` // List of criteria to explicitly exclude items
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
