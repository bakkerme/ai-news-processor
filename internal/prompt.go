package main

import "github.com/bakkerme/ai-news-processor/internal/prompts"

func getSystemPrompt() string {
	return prompts.GetSystemPrompt()
}

func getSummarySystemPrompt() string {
	return prompts.GetSummarySystemPrompt()
}
