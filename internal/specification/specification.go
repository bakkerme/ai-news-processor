package specification

import (
	"fmt"
	"strconv"

	"github.com/kelseyhightower/envconfig"
)

type Specification struct {
	LlmUrl    string `split_words:"true"`
	LlmApiKey string `split_words:"true"`
	LlmModel  string `split_words:"true"`

	LlmImageEnabled      bool   `split_words:"true" default:"false"`
	LlmImageModel        string `split_words:"true"`
	LlmUrlSummaryEnabled bool   `split_words:"true" default:"true"`

	EmailTo       string `split_words:"true"`
	EmailFrom     string `split_words:"true"`
	EmailHost     string `split_words:"true"`
	EmailPort     string `split_words:"true"`
	EmailUsername string `split_words:"true"`
	EmailPassword string `split_words:"true"`

	DebugMockRss         bool `split_words:"true"`
	DebugMockLLM         bool `split_words:"true"`
	DebugSkipEmail       bool `split_words:"true"`
	DebugOutputBenchmark bool `split_words:"true"`
	DebugMaxEntries      int  `split_words:"true"`
	DebugRssDump         bool `split_words:"true"`

	QualityFilterThreshold int `split_words:"true" default:"10"`

	PersonasPath string `split_words:"true"`

	AuditServiceUrl string `split_words:"true"`

	SendBenchmarkToAuditService bool `split_words:"true" default:"false"`
}

// Validate checks if the specification is valid
func (s *Specification) Validate() error {
	// Email configuration validation
	if s.EmailHost == "" {
		return fmt.Errorf("email host is required")
	}
	if s.EmailPort == "" {
		return fmt.Errorf("email port is required")
	}
	if _, err := strconv.Atoi(s.EmailPort); err != nil {
		return fmt.Errorf("invalid email port: %w", err)
	}
	if s.EmailUsername == "" {
		return fmt.Errorf("email username is required")
	}
	if s.EmailPassword == "" {
		return fmt.Errorf("email password is required")
	}
	if s.EmailFrom == "" {
		return fmt.Errorf("email from address is required")
	}
	if s.EmailTo == "" {
		return fmt.Errorf("email to address is required")
	}

	// LLM configuration validation
	if !s.DebugMockLLM {
		if s.LlmUrl == "" {
			return fmt.Errorf("LLM URL is required when not in mock mode")
		}
		// if s.LlmApiKey == "" {
		// 	return fmt.Errorf("LLM API key is required when not in mock mode")
		// }
		if s.LlmModel == "" {
			return fmt.Errorf("LLM model is required when not in mock mode")
		}

		// Multi-modal validation
		if s.LlmImageEnabled && s.LlmImageModel == "" {
			return fmt.Errorf("LLM image model is required when image processing is enabled")
		}
	}

	// Debug configuration validation
	if s.DebugMaxEntries < 0 {
		return fmt.Errorf("debug max entries cannot be negative")
	}

	if s.DebugOutputBenchmark && s.AuditServiceUrl == "" {
		return fmt.Errorf("audit service URL is required when benchmark output is enabled")
	}

	return nil
}

func GetConfig() (*Specification, error) {
	var s Specification
	err := envconfig.Process("anp", &s)
	if err != nil {
		return nil, fmt.Errorf("could not load specification: %w", err)
	}

	// Validate the configuration
	if err := s.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &s, nil
}
