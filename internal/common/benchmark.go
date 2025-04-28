package common

import (
	"encoding/json"
	"fmt"
)

// BenchmarkData represents the data collected during benchmarking
type BenchmarkData struct {
	RawInput []string `json:"raw_input"` // The raw input strings sent to the LLM
	Results  []Item   `json:"results"`   // The processed results from the LLM
	Persona  string   `json:"persona"`   // The persona used for this benchmark
}

// SerializeBenchmarkData converts benchmark data to JSON byte array
func SerializeBenchmarkData(data *BenchmarkData) ([]byte, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal benchmark data: %w", err)
	}
	return jsonData, nil
}
