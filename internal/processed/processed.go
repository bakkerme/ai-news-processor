package processed

import (
	"encoding/json"
	"os"
)

const processedIDsFile = "/tmp/processed_ids.json"

// IDs stores the IDs of items that have already been processed
type IDs struct {
	IDs map[string]bool `json:"ids"`
}

// New creates a new IDs tracker
func New() *IDs {
	return &IDs{IDs: make(map[string]bool)}
}

// Load loads the processed IDs from a file
func Load() (*IDs, error) {
	data, err := os.ReadFile(processedIDsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return New(), nil
		}
		return nil, err
	}

	var processedIDs IDs
	if err := json.Unmarshal(data, &processedIDs); err != nil {
		return nil, err
	}
	return &processedIDs, nil
}

// Save saves the processed IDs to a file
func (p *IDs) Save() error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return os.WriteFile(processedIDsFile, data, 0644)
}

// Add adds an ID to the processed list
func (p *IDs) Add(id string) {
	p.IDs[id] = true
}

// Has checks if an ID has been processed
func (p *IDs) Has(id string) bool {
	return p.IDs[id]
}
