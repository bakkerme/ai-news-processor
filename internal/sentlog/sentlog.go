package sentlog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// LoadSentIDs loads a set of sent item IDs from disk.
// If the file does not exist, an empty set is returned.
func LoadSentIDs(path string) (map[string]struct{}, error) {
	ids := make(map[string]struct{})
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ids, nil
		}
		return nil, fmt.Errorf("could not read sent log: %w", err)
	}

	var list []string
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("could not parse sent log: %w", err)
	}
	for _, id := range list {
		if id == "" {
			continue
		}
		ids[id] = struct{}{}
	}
	return ids, nil
}

// SaveSentIDs persists the sent item IDs to disk as a JSON array.
func SaveSentIDs(path string, ids map[string]struct{}) error {
	list := make([]string, 0, len(ids))
	for id := range ids {
		list = append(list, id)
	}
	sort.Strings(list)

	payload, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("could not encode sent log: %w", err)
	}

	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("could not create sent log directory: %w", err)
		}
	}

	if err := os.WriteFile(path, payload, 0644); err != nil {
		return fmt.Errorf("could not write sent log: %w", err)
	}

	return nil
}
