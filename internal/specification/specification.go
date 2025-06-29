package specification

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Specification struct {
	LlmUrl    string
	LlmApiKey string
	LlmModel  string

	LlmImageEnabled      bool
	LlmImageModel        string
	LlmUrlSummaryEnabled bool

	EmailTo       string
	EmailFrom     string
	EmailHost     string
	EmailPort     string
	EmailUsername string
	EmailPassword string

	DebugMockFeeds       bool
	DebugMockLLM         bool
	DebugSkipEmail       bool
	DebugOutputBenchmark bool
	DebugMaxEntries      int
	DebugRedditDump      bool

	QualityFilterThreshold int

	PersonasPath string

	AuditServiceUrl string

	SendBenchmarkToAuditService bool

	// Reddit API configuration
	RedditClientID string
	RedditSecret   string
	RedditUsername string
	RedditPassword string
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

	// Reddit API configuration validation (required unless using mock feeds)
	if !s.DebugMockFeeds {
		if s.RedditClientID == "" {
			return fmt.Errorf("Reddit client ID is required")
		}
		if s.RedditSecret == "" {
			return fmt.Errorf("Reddit client secret is required")
		}
		if s.RedditUsername == "" {
			return fmt.Errorf("Reddit username is required")
		}
		if s.RedditPassword == "" {
			return fmt.Errorf("Reddit password is required")
		}
	}

	return nil
}

func GetConfig() (*Specification, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found or error loading it: %v", err)
	}

	s := &Specification{
		LlmUrl:    os.Getenv("ANP_LLM_URL"),
		LlmApiKey: os.Getenv("ANP_LLM_API_KEY"),
		LlmModel:  os.Getenv("ANP_LLM_MODEL"),

		LlmImageEnabled:      getBoolEnv("ANP_LLM_IMAGE_ENABLED", false),
		LlmImageModel:        os.Getenv("ANP_LLM_IMAGE_MODEL"),
		LlmUrlSummaryEnabled: getBoolEnv("ANP_LLM_URL_SUMMARY_ENABLED", true),

		EmailTo:       os.Getenv("ANP_EMAIL_TO"),
		EmailFrom:     os.Getenv("ANP_EMAIL_FROM"),
		EmailHost:     os.Getenv("ANP_EMAIL_HOST"),
		EmailPort:     os.Getenv("ANP_EMAIL_PORT"),
		EmailUsername: os.Getenv("ANP_EMAIL_USERNAME"),
		EmailPassword: os.Getenv("ANP_EMAIL_PASSWORD"),

		DebugMockFeeds:       getBoolEnv("ANP_DEBUG_MOCK_FEEDS", false),
		DebugMockLLM:         getBoolEnv("ANP_DEBUG_MOCK_LLM", false),
		DebugSkipEmail:       getBoolEnv("ANP_DEBUG_SKIP_EMAIL", false),
		DebugOutputBenchmark: getBoolEnv("ANP_DEBUG_OUTPUT_BENCHMARK", false),
		DebugMaxEntries:      getIntEnv("ANP_DEBUG_MAX_ENTRIES", 0),
		DebugRedditDump:      getBoolEnv("ANP_DEBUG_REDDIT_DUMP", false),

		QualityFilterThreshold: getIntEnv("ANP_QUALITY_FILTER_THRESHOLD", 10),

		PersonasPath: os.Getenv("ANP_PERSONAS_PATH"),

		AuditServiceUrl: os.Getenv("ANP_AUDIT_SERVICE_URL"),

		SendBenchmarkToAuditService: getBoolEnv("ANP_SEND_BENCHMARK_TO_AUDIT_SERVICE", false),

		// Reddit API configuration
		RedditClientID: os.Getenv("ANP_REDDIT_CLIENT_ID"),
		RedditSecret:   os.Getenv("ANP_REDDIT_CLIENT_SECRET"),
		RedditUsername: os.Getenv("ANP_REDDIT_USERNAME"),
		RedditPassword: os.Getenv("ANP_REDDIT_PASSWORD"),
	}

	// Validate the configuration
	if err := s.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return s, nil
}

// getBoolEnv gets a boolean environment variable with a default value
func getBoolEnv(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return boolValue
}

// getIntEnv gets an integer environment variable with a default value
func getIntEnv(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}
