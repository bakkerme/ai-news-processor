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

	// Prompt customization settings (optional, uses defaults if not specified)
	PromptConfig *PromptConfig `yaml:"prompt_config,omitempty" json:"promptConfig,omitempty"`
}

// PromptConfig contains settings to customize the prompt template for this persona
type PromptConfig struct {
	// Content sections
	IncludeCommentSummary *bool `yaml:"include_comment_summary,omitempty" json:"includeCommentSummary,omitempty"` // Whether to include CommentSummary section (defaults to true for reddit, false for rss)
	IncludeImageAnalysis  *bool `yaml:"include_image_analysis,omitempty" json:"includeImageAnalysis,omitempty"`   // Whether to include image analysis (defaults to true)

	// Content length specifications
	SummaryParagraphs    *int `yaml:"summary_paragraphs,omitempty" json:"summaryParagraphs,omitempty"`       // Number of paragraphs for Summary section (default: 1-2)
	SummaryWordCount     *int `yaml:"summary_word_count,omitempty" json:"summaryWordCount,omitempty"`         // Target word count for Summary (default: 500-800)
	CommentParagraphs    *int `yaml:"comment_paragraphs,omitempty" json:"commentParagraphs,omitempty"`       // Number of paragraphs for CommentSummary (default: 1-2)
	CommentWordCount     *int `yaml:"comment_word_count,omitempty" json:"commentWordCount,omitempty"`         // Target word count for CommentSummary (default: 300-600)
	OverviewBulletPoints *int `yaml:"overview_bullet_points,omitempty" json:"overviewBulletPoints,omitempty"` // Number of overview bullet points (default: 2-3)

	// Technical depth and style
	TechnicalDepth   *string `yaml:"technical_depth,omitempty" json:"technicalDepth,omitempty"`     // "basic", "moderate", "advanced" (default: "moderate")
	WritingStyle     *string `yaml:"writing_style,omitempty" json:"writingStyle,omitempty"`         // "conversational", "formal", "technical" (default: "conversational")
	IncludeTechGeek  *bool   `yaml:"include_tech_geek,omitempty" json:"includeTechGeek,omitempty"`   // Whether to include "geek out" technical details (default: true)
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

// GetIncludeCommentSummary returns whether to include comment summary in prompts.
// Defaults based on provider type: true for reddit, false for rss.
func (p *Persona) GetIncludeCommentSummary() bool {
	if p.PromptConfig != nil && p.PromptConfig.IncludeCommentSummary != nil {
		return *p.PromptConfig.IncludeCommentSummary
	}
	// Default based on provider: reddit has comments, rss typically doesn't
	return p.GetProvider() == "reddit"
}

// GetIncludeImageAnalysis returns whether to include image analysis in prompts.
func (p *Persona) GetIncludeImageAnalysis() bool {
	if p.PromptConfig != nil && p.PromptConfig.IncludeImageAnalysis != nil {
		return *p.PromptConfig.IncludeImageAnalysis
	}
	return true // Default to including image analysis
}

// GetSummaryParagraphs returns the number of paragraphs for Summary section.
func (p *Persona) GetSummaryParagraphs() string {
	if p.PromptConfig != nil && p.PromptConfig.SummaryParagraphs != nil {
		count := *p.PromptConfig.SummaryParagraphs
		if count == 1 {
			return "1 paragraph"
		}
		return fmt.Sprintf("%d paragraphs", count)
	}
	return "1 - 2 paragraphs" // Default
}

// GetSummaryWordCount returns the target word count for Summary section.
func (p *Persona) GetSummaryWordCount() string {
	if p.PromptConfig != nil && p.PromptConfig.SummaryWordCount != nil {
		return fmt.Sprintf("%d words total", *p.PromptConfig.SummaryWordCount)
	}
	return "500-800 words total" // Default
}

// GetCommentParagraphs returns the number of paragraphs for CommentSummary section.
func (p *Persona) GetCommentParagraphs() string {
	if p.PromptConfig != nil && p.PromptConfig.CommentParagraphs != nil {
		count := *p.PromptConfig.CommentParagraphs
		if count == 1 {
			return "1 paragraph"
		}
		return fmt.Sprintf("%d paragraphs", count)
	}
	return "1 - 2 paragraphs" // Default
}

// GetCommentWordCount returns the target word count for CommentSummary section.
func (p *Persona) GetCommentWordCount() string {
	if p.PromptConfig != nil && p.PromptConfig.CommentWordCount != nil {
		return fmt.Sprintf("%d words total", *p.PromptConfig.CommentWordCount)
	}
	return "300-600 words total" // Default
}

// GetOverviewBulletPoints returns the number of overview bullet points.
func (p *Persona) GetOverviewBulletPoints() string {
	if p.PromptConfig != nil && p.PromptConfig.OverviewBulletPoints != nil {
		count := *p.PromptConfig.OverviewBulletPoints
		return fmt.Sprintf("%d", count)
	}
	return "2-3" // Default
}

// GetTechnicalDepth returns the technical depth setting.
func (p *Persona) GetTechnicalDepth() string {
	if p.PromptConfig != nil && p.PromptConfig.TechnicalDepth != nil {
		return *p.PromptConfig.TechnicalDepth
	}
	return "moderate" // Default
}

// GetWritingStyle returns the writing style setting.
func (p *Persona) GetWritingStyle() string {
	if p.PromptConfig != nil && p.PromptConfig.WritingStyle != nil {
		return *p.PromptConfig.WritingStyle
	}
	return "conversational" // Default
}

// GetIncludeTechGeek returns whether to include technical "geek out" details.
func (p *Persona) GetIncludeTechGeek() bool {
	if p.PromptConfig != nil && p.PromptConfig.IncludeTechGeek != nil {
		return *p.PromptConfig.IncludeTechGeek
	}
	return true // Default
}

// GetTechnicalDepthInstruction returns the instruction text based on technical depth setting.
func (p *Persona) GetTechnicalDepthInstruction() string {
	switch p.GetTechnicalDepth() {
	case "basic":
		return "Keep technical details accessible to a general audience. Avoid jargon and explain technical concepts in simple terms."
	case "advanced":
		return "Provide in-depth technical analysis with detailed explanations of implementation details, algorithms, and technical implications."
	default: // "moderate"
		return "Provide technical analysis appropriate for a knowledgeable audience, balancing accessibility with technical depth."
	}
}

// GetWritingStyleInstruction returns the instruction text based on writing style setting.
func (p *Persona) GetWritingStyleInstruction() string {
	switch p.GetWritingStyle() {
	case "formal":
		return "Write in a formal, professional tone suitable for academic or business contexts."
	case "technical":
		return "Write in a precise, technical style focusing on accuracy and detailed specifications."
	default: // "conversational"
		return "Write in a conversational, engaging style while maintaining technical accuracy."
	}
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
