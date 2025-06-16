package bench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bakkerme/ai-news-processor/models"
)

// EntrySummary, ImageSummary, WebContentSummary, and BenchmarkData structs are defined in internal/models

// SerializeRunData converts RunData to JSON byte array
func SerializeRunData(data *models.RunData) ([]byte, error) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal run data: %w", err)
	}
	return jsonData, nil
}

var benchmarkDir = "benchmarkresults" // This can remain, as it's about file storage

// WriteRunDataToDisk writes run data to a file and creates a backup if needed
func WriteRunDataToDisk(data *models.RunData) error {
	personaName := "unknown"
	if data.Persona.Name != "" {
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
		if errReadFile == nil {
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

	log.Printf("Run data written to %s and %s\n", benchFilePath, defaultPath)
	return nil
}

// SubmitRunDataToAuditService sends the run data to the ai-news-auditability-service.
func SubmitRunDataToAuditService(data *models.RunData, auditServiceURL string) error {
	if !strings.HasSuffix(auditServiceURL, "/runs") {
		if strings.HasSuffix(auditServiceURL, "/") {
			auditServiceURL += "runs"
		} else {
			auditServiceURL += "/runs"
		}
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal audit service payload: %w", err)
	}

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

	log.Printf("Run data successfully submitted to audit service at %s\n", auditServiceURL)
	return nil
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
			log.Printf("Warning: failed to read run data file %s: %v\n", filePath, err)
			continue
		}

		var runData models.RunData // Changed type
		err = json.Unmarshal(dataBytes, &runData)
		if err != nil {
			log.Printf("Warning: failed to unmarshal run data from file %s: %v\n", filePath, err)
			continue
		}

		runDataList = append(runDataList, runData)
	}

	return runDataList, nil
}
