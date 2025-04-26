package common

import (
	"os"
)

// SaveLLMResponse saves the LLM response to a file
func SaveLLMResponse(response string) error {
	return os.WriteFile("llmresponse.json", []byte(response), 0644)
}
