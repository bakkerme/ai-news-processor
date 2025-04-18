package main

import "fmt"

func formatPrompt(systemPrompt string, userPrompt string) string {
	return fmt.Sprintf(`System: %s
User: %s`, systemPrompt, userPrompt)
}

func queryLlamaCPP(systemPrompt, userPrompt string) {
	prompt := formatPrompt(systemPrompt, userPrompt)

	// Example usage of the CompletionRequest struct
	req := &CompletionRequest{
		Prompt:      prompt,
		Temperature: 1.0,
		CachePrompt: true,
	}

	// URL of the API endpoint
	url := "http://192.168.1.115:8080/completion"

	// Send the request
	response, err := sendCompletionRequest(req, url)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}

	fmt.Println(response.Content)
}
