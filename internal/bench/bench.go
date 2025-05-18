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

	"github.com/bakkerme/ai-news-processor/models"
	// persona import might not be directly needed here anymore if RunData contains the full persona
	// and OutputRunData's signature changes.
)

// EntrySummary, ImageSummary, WebContentSummary, and BenchmarkData structs are now defined in internal/models
// and will be imported as models.EntrySummary, models.ImageSummary, models.WebContentSummary, and models.RunData respectively.

// SerializeRunData converts RunData to JSON byte array
func SerializeRunData(data *models.RunData) ([]byte, error) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal run data: %w", err)
	}
	return jsonData, nil
}

var benchmarkDir = "benchmarkresults" // This can remain, as it's about file storage

// OutputRunData writes run data to a file and submits to audit service
// The p *persona.Persona parameter is removed as data.Persona is now the full persona.Persona.
func OutputRunData(data *models.RunData, auditServiceURL string) error {
	// Create filename with persona name and timestamp
	personaName := "unknown"
	if data.Persona.Name != "" { // data.Persona is now persona.Persona
		personaName = data.Persona.Name
	}

	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("benchmark_%s_%s.json", personaName, timestamp)
	benchFilePath := filepath.Join(benchmarkDir, filename)

	err := os.MkdirAll(benchmarkDir, 0755)
	if err != nil {
		return fmt.Errorf("error creating benchmark directory: %w", err)
	}

	defaultPath := filepath.Join(benchmarkDir, "benchmark.json")
	if _, err := os.Stat(defaultPath); err == nil {
		backupDir := filepath.Join(benchmarkDir, "backup")
		err := os.MkdirAll(backupDir, 0755)
		if err != nil {
			return fmt.Errorf("error creating backup directory: %w", err)
		}

		backupPath := filepath.Join(backupDir, "benchmark.json")
		backupData, errReadFile := os.ReadFile(defaultPath)
		if errReadFile == nil { // Check error for ReadFile
			errWrite := os.WriteFile(backupPath, backupData, 0644)
			if errWrite != nil {
				return fmt.Errorf("error creating backup: %w", errWrite)
			}
		}
	}

	jsonData, err := SerializeRunData(data)
	if err != nil {
		return fmt.Errorf("error serializing run data: %w", err)
	}

	err = os.WriteFile(benchFilePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("error writing to timestamped benchmark file: %w", err)
	}

	err = os.WriteFile(defaultPath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("error writing to default benchmark file: %w", err)
	}

	fmt.Printf("Run data written to %s and %s\n", benchFilePath, defaultPath)

	err = submitToAuditService(data, auditServiceURL)
	if err != nil {
		fmt.Printf("Warning: Failed to submit run data to audit service: %v\n", err)
	}

	return nil
}

// submitToAuditService sends the run data to the ai-news-auditability-service.
func submitToAuditService(data *models.RunData, auditServiceURL string) error {
	if !strings.HasSuffix(auditServiceURL, "/runs") {
		if strings.HasSuffix(auditServiceURL, "/") {
			auditServiceURL += "runs"
		} else {
			auditServiceURL += "/runs"
		}
	}

	jsonData, err := json.Marshal(data) // data is already *models.RunData
	if err != nil {
		return fmt.Errorf("failed to marshal audit service payload: %w", err)
	}

	// fmt.Printf("jsonData: %s\n", string(jsonData)) // Keep for debugging if necessary

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
		if resp.Body != nil {
			bodyBytes, readErr = io.ReadAll(resp.Body)
		}

		if readErr != nil {
			return fmt.Errorf("audit service returned status %s; failed to read response body: %v", resp.Status, readErr)
		}
		return fmt.Errorf("audit service returned status %s: %s", resp.Status, string(bodyBytes))
	}

	fmt.Printf("Run data successfully submitted to audit service at %s\n", auditServiceURL)
	return nil
}

// AddImageSummaryToRunData adds an image summary to the run data
func AddImageSummaryToRunData(rd *models.RunData, summary models.ImageSummary) {
	rd.ImageSummaries = append(rd.ImageSummaries, summary)
}

// AddWebContentSummaryToRunData adds a web content summary to the run data
func AddWebContentSummaryToRunData(rd *models.RunData, summary models.WebContentSummary) {
	rd.WebContentSummaries = append(rd.WebContentSummaries, summary)
}

// AddEntrySummaryToRunData adds an entry summary to the run data
func AddEntrySummaryToRunData(rd *models.RunData, summary models.EntrySummary) {
	rd.EntrySummaries = append(rd.EntrySummaries, summary)
}

// LoadRunData loads the most recent run data for each persona from a file
func LoadRunData() ([]models.RunData, error) {
	// read all benchmark files
	files, err := os.ReadDir(filepath.Join(benchmarkDir)) // Assuming benchmarkDir is relative to where this runs or an absolute path
	if err != nil {
		return nil, fmt.Errorf("failed to read benchmark files from %s: %w", benchmarkDir, err)
	}

	mostRecentRuns := make(map[string]string)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		// format is fmt.Sprintf("benchmark_%s_%s.json", personaName, timestamp)
		// robust parsing:
		nameParts := strings.SplitN(file.Name(), ".", 2)
		if len(nameParts) < 2 || nameParts[1] != "json" {
			continue
		}
		baseNameParts := strings.SplitN(nameParts[0], "_", 3)
		if len(baseNameParts) < 3 || baseNameParts[0] != "benchmark" {
			continue
		}
		personaName := baseNameParts[1]
		timestamp := baseNameParts[2]

		if _, exists := mostRecentRuns[personaName]; !exists || timestamp > mostRecentRuns[personaName] {
			mostRecentRuns[personaName] = timestamp
		}
	}

	runDataList := []models.RunData{} // Changed type

	for personaName, timestamp := range mostRecentRuns {
		filename := fmt.Sprintf("benchmark_%s_%s.json", personaName, timestamp)
		filePath := filepath.Join(benchmarkDir, filename) // Use benchmarkDir
		dataBytes, err := os.ReadFile(filePath)
		if err != nil {
			// It's possible a file was deleted between listing and reading, log and continue or handle
			fmt.Printf("Warning: failed to read run data file %s: %v\n", filePath, err)
			continue
		}

		var runData models.RunData // Changed type
		err = json.Unmarshal(dataBytes, &runData)
		if err != nil {
			fmt.Printf("Warning: failed to unmarshal run data from file %s: %v\n", filePath, err)
			continue
		}

		runDataList = append(runDataList, runData)
	}

	return runDataList, nil
}
