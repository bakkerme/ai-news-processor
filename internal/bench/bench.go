package bench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bakkerme/ai-news-processor/internal/models"
	"github.com/bakkerme/ai-news-processor/internal/persona"
)

// EntrySummary represents the raw input and results for the entire processing pipeline
type EntrySummary struct {
	RawInput       string      `json:"raw_input"`          // The raw input strings sent to the LLM
	Results        models.Item `json:"results"`            // The processed results from the LLM
	ProcessingTime int64       `json:"processing_time_ms"` // Time taken to process the entry in milliseconds
}

// ImageSummary represents the benchmark data for image processing
type ImageSummary struct {
	ImageURL         string `json:"image_url"`          // URL of the image processed
	ImageDescription string `json:"image_description"`  // The description generated for the image
	Title            string `json:"title"`              // Title associated with the image
	EntryID          string `json:"entry_id,omitempty"` // ID of the entry the image belongs to
	ProcessingTime   int64  `json:"processing_time_ms"` // Time taken to process the image in milliseconds
}

// WebContentSummary represents the benchmark data for web content processing
type WebContentSummary struct {
	URL             string `json:"url"`                // URL of the web content
	OriginalContent string `json:"original_content"`   // Original content from the URL
	Summary         string `json:"summary"`            // Summary generated for the web content
	Title           string `json:"title,omitempty"`    // Title of the web content
	EntryID         string `json:"entry_id,omitempty"` // ID of the entry the web content belongs to
	ProcessingTime  int64  `json:"processing_time_ms"` // Time taken to process the web content in milliseconds
}

// BenchmarkData represents the data collected during benchmarking
type BenchmarkData struct {
	EntrySummaries      []EntrySummary      `json:"entrySummaries"`                // Overall input-output pairs for entire pipeline
	ImageSummaries      []ImageSummary      `json:"imageSummaries,omitempty"`      // Image URL to description benchmarks
	WebContentSummaries []WebContentSummary `json:"webContentSummaries,omitempty"` // URL to summary benchmarks
	Persona             persona.Persona     `json:"persona"`                       // The full persona used for this benchmark
	RunDate             time.Time           `json:"runDate"`                       // The date the benchmark was run, ISO 8601 format (time.RFC3339)
	OverallModelUsed    string              `json:"overallModelUsed,omitempty"`    // The LLM model used for the benchmark
	ImageModelUsed      string              `json:"imageModelUsed,omitempty"`      // The LLM model used for the image processing
	WebContentModelUsed string              `json:"webContentModelUsed,omitempty"` // The LLM model used for the web content processing

	TotalProcessingTime int64 `json:"totalProcessingTime,omitempty"` // Total time taken for processing in milliseconds

	// Performance metrics
	EntryTotalProcessingTime      int64 `json:"entryTotalProcessingTime,omitempty"`      // Total time taken for entry processing in milliseconds
	ImageTotalProcessingTime      int64 `json:"imageTotalProcessingTime,omitempty"`      // Total time taken for image processing in milliseconds
	WebContentTotalProcessingTime int64 `json:"webContentTotalProcessingTime,omitempty"` // Total time taken for web content processing in milliseconds

	SuccessRate float64 `json:"successRate,omitempty"` // Percentage of successful processing attempts
}

// SerializeBenchmarkData converts benchmark data to JSON byte array
func SerializeBenchmarkData(data *BenchmarkData) ([]byte, error) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal benchmark data: %w", err)
	}
	return jsonData, nil
}

var benchmarkDir = "benchmarkresults"

// OutputBenchmarkData writes benchmark data to a file and submits to audit service
func OutputBenchmarkData(data *BenchmarkData, auditServiceURL string) error {
	// Create filename with persona name and timestamp
	personaName := "unknown"
	if data.Persona.Name != "" {
		personaName = data.Persona.Name
	}

	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("benchmark_%s_%s.json", personaName, timestamp)
	benchFilePath := filepath.Join(benchmarkDir, filename)

	// Ensure the directory exists
	err := os.MkdirAll(benchmarkDir, 0755)
	if err != nil {
		return fmt.Errorf("error creating benchmark directory: %w", err)
	}

	// Create backup of previous benchmark.json if it exists
	defaultPath := filepath.Join(benchmarkDir, "benchmark.json")
	if _, err := os.Stat(defaultPath); err == nil {
		backupDir := filepath.Join(benchmarkDir, "backup")
		err := os.MkdirAll(backupDir, 0755)
		if err != nil {
			return fmt.Errorf("error creating backup directory: %w", err)
		}

		backupPath := filepath.Join(backupDir, "benchmark.json")
		backupData, err := os.ReadFile(defaultPath)
		if err == nil {
			err = os.WriteFile(backupPath, backupData, 0644)
			if err != nil {
				return fmt.Errorf("error creating backup: %w", err)
			}
		}
	}

	// Serialize data
	jsonData, err := SerializeBenchmarkData(data)
	if err != nil {
		return fmt.Errorf("error serializing benchmark data: %w", err)
	}

	// Write to timestamped file
	err = os.WriteFile(benchFilePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("error writing to timestamped benchmark file: %w", err)
	}

	// Also write to default benchmark.json for backward compatibility
	err = os.WriteFile(defaultPath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("error writing to default benchmark file: %w", err)
	}

	fmt.Printf("Benchmark data written to %s and %s\n", benchFilePath, defaultPath)

	// Attempt to submit data to the auditability service
	err = submitToAuditService(data, auditServiceURL)
	if err != nil {
		// Log the error but don't fail the whole process if submission fails
		fmt.Printf("Warning: Failed to submit benchmark data to audit service: %v\n", err)
	}

	return nil
}

// submitToAuditService sends the benchmark data to the ai-news-auditability-service.
func submitToAuditService(data *BenchmarkData, auditServiceURL string) error {
	if !strings.HasSuffix(auditServiceURL, "/runs") {
		if strings.HasSuffix(auditServiceURL, "/") {
			auditServiceURL += "runs"
		} else {
			auditServiceURL += "/runs"
		}
	}

	// BenchmarkData is now directly usable
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal audit service payload: %w", err)
	}

	fmt.Printf("jsonData: %s\n", string(jsonData))

	req, err := http.NewRequest("POST", auditServiceURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create audit service request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to audit service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var bodyBytes []byte
		var readErr error
		// A more robust way to read the body
		if resp.Body != nil {
			bodyBytes, readErr = io.ReadAll(resp.Body)
		}

		if readErr != nil {
			return fmt.Errorf("audit service returned status %s; failed to read response body: %v", resp.Status, readErr)
		}
		return fmt.Errorf("audit service returned status %s: %s", resp.Status, string(bodyBytes))
	}

	fmt.Printf("Benchmark data successfully submitted to audit service at %s\n", auditServiceURL)
	return nil
}

// AddImageSummary adds an image summary to the benchmark data
func (bd *BenchmarkData) AddImageSummary(summary ImageSummary) {
	bd.ImageSummaries = append(bd.ImageSummaries, summary)
}

// AddWebContentSummary adds a web content summary to the benchmark data
func (bd *BenchmarkData) AddWebContentSummary(summary WebContentSummary) {
	bd.WebContentSummaries = append(bd.WebContentSummaries, summary)
}

// AddEntrySummary adds an entry summary to the benchmark data
func (bd *BenchmarkData) AddEntrySummary(summary EntrySummary) {
	bd.EntrySummaries = append(bd.EntrySummaries, summary)
}

// LoadBenchmarkData loads the most recent benchmark data for each persona from a file
func LoadBenchmarkData() ([]BenchmarkData, error) {
	// read all benchmark files
	files, err := os.ReadDir("bench/results")
	if err != nil {
		return nil, fmt.Errorf("failed to read benchmark files: %w", err)
	}

	// find the most recent run file for each persona
	// format is fmt.Sprintf("benchmark_%s_%s.json", personaName, timestamp)
	mostRecentRuns := make(map[string]string)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		parts := strings.Split(file.Name(), "_")
		if len(parts) < 2 {
			continue
		}
		personaName := parts[1]
		timestamp := parts[2]

		if _, exists := mostRecentRuns[personaName]; !exists || timestamp > mostRecentRuns[personaName] {
			mostRecentRuns[personaName] = timestamp
		}
	}

	benchmarkDataList := []BenchmarkData{}

	// load the most recent run for each persona
	for personaName, timestamp := range mostRecentRuns {
		filename := fmt.Sprintf("benchmark_%s_%s.json", personaName, timestamp)
		data, err := os.ReadFile(filepath.Join("bench/results", filename))
		if err != nil {
			return nil, fmt.Errorf("failed to read benchmark data: %w", err)
		}

		var benchmarkData BenchmarkData
		err = json.Unmarshal(data, &benchmarkData)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal benchmark data: %w", err)
		}

		benchmarkDataList = append(benchmarkDataList, benchmarkData)
	}

	return benchmarkDataList, nil
}
