package persona

import (
	"fmt"

	"github.com/bakkerme/ai-news-processor/internal/common"
)

// LoadAndSelect loads personas from a given path and selects based on criteria
func LoadAndSelect(path string, personaName string) ([]common.Persona, error) {
	personas, err := common.LoadPersonas(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load personas: %w", err)
	}
	if len(personas) == 0 {
		return nil, fmt.Errorf("no personas found in directory")
	}

	// Select specific persona or all
	if personaName == "all" || personaName == "" {
		return personas, nil
	}

	// Filter for specific persona
	selectedPersonas := []common.Persona{}
	for _, p := range personas {
		if p.Name == personaName {
			selectedPersonas = append(selectedPersonas, p)
		}
	}
	if len(selectedPersonas) == 0 {
		return nil, fmt.Errorf("persona '%s' not found", personaName)
	}

	return selectedPersonas, nil
}
