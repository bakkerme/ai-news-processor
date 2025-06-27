package specification

import (
	"fmt"
	"os"
	"strconv"

	parentSpec "github.com/bakkerme/ai-news-processor/internal/specification"
	"github.com/joho/godotenv"
)

type BenchmarkSpecification struct {
	*parentSpec.Specification

	BenchmarkIterations  int
	BenchmarkConcurrency int
	BenchmarkOutputPath  string
}

func (s *BenchmarkSpecification) Validate() error {
	if s.BenchmarkIterations <= 0 {
		return fmt.Errorf("benchmark iterations must be greater than 0")
	}
	if s.BenchmarkConcurrency <= 0 {
		return fmt.Errorf("benchmark concurrency must be greater than 0")
	}
	if s.BenchmarkOutputPath == "" {
		return fmt.Errorf("benchmark output path is required")
	}

	if !s.DebugMockLLM {
		if s.LlmUrl == "" {
			return fmt.Errorf("LLM URL is required when not in mock mode")
		}
		if s.LlmModel == "" {
			return fmt.Errorf("LLM model is required when not in mock mode")
		}
		if s.LlmImageEnabled && s.LlmImageModel == "" {
			return fmt.Errorf("LLM image model is required when image processing is enabled")
		}
	}

	if s.DebugMaxEntries < 0 {
		return fmt.Errorf("debug max entries cannot be negative")
	}

	return nil
}

func GetConfig() (*BenchmarkSpecification, error) {
	// First log working directory to help with debugging
	dir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current working directory: %w", err)
	}
	fmt.Println("Current working directory:", dir)

	err = godotenv.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load .env file: %w", err)
	}

	parentConfig, err := parentSpec.GetConfig()
	if err != nil {
		// Create a minimal parent config for benchmark use
		parentConfig = &parentSpec.Specification{
			LlmUrl:                      os.Getenv("ANP_LLM_URL"),
			LlmApiKey:                   os.Getenv("ANP_LLM_API_KEY"),
			LlmModel:                    os.Getenv("ANP_LLM_MODEL"),
			LlmImageEnabled:             getBoolEnv("ANP_LLM_IMAGE_ENABLED", false),
			LlmImageModel:               os.Getenv("ANP_LLM_IMAGE_MODEL"),
			LlmUrlSummaryEnabled:        getBoolEnv("ANP_LLM_URL_SUMMARY_ENABLED", true),
			DebugMockRss:                getBoolEnv("ANP_DEBUG_MOCK_RSS", false),
			DebugMockLLM:                getBoolEnv("ANP_DEBUG_MOCK_LLM", false),
			DebugSkipEmail:              getBoolEnv("ANP_DEBUG_SKIP_EMAIL", true),
			DebugOutputBenchmark:        getBoolEnv("ANP_DEBUG_OUTPUT_BENCHMARK", true),
			DebugMaxEntries:             getIntEnv("ANP_DEBUG_MAX_ENTRIES", 0),
			DebugRssDump:                getBoolEnv("ANP_DEBUG_RSS_DUMP", false),
			QualityFilterThreshold:      getIntEnv("ANP_QUALITY_FILTER_THRESHOLD", 10),
			PersonasPath:                os.Getenv("ANP_PERSONAS_PATH"),
			AuditServiceUrl:             os.Getenv("ANP_AUDIT_SERVICE_URL"),
			SendBenchmarkToAuditService: getBoolEnv("ANP_SEND_BENCHMARK_TO_AUDIT_SERVICE", false),
		}
	}

	s := &BenchmarkSpecification{
		Specification:        parentConfig,
		BenchmarkIterations:  getIntEnv("ANP_BENCHMARK_ITERATIONS", 5),
		BenchmarkConcurrency: getIntEnv("ANP_BENCHMARK_CONCURRENCY", 1),
		BenchmarkOutputPath:  getEnvOrDefault("ANP_BENCHMARK_OUTPUT_PATH", "./benchmark_results.json"),
	}

	if err := s.Validate(); err != nil {
		return nil, fmt.Errorf("invalid benchmark configuration: %w", err)
	}

	return s, nil
}

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

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
