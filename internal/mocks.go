package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/bakkerme/ai-news-processor/internal/models"
)

func GetMockLLMResponse() []models.Item {
	// Assuming localllama.rss is located in the same directory as this file
	jsonData, err := os.ReadFile("./llmresponse.json")
	if err != nil {
		panic(err)
	}

	// Unmarshal the JSON string into an Item struct
	var items []models.Item
	if err := json.Unmarshal([]byte(jsonData), &items); err != nil {
		log.Fatalf("Error unmarshalling JSON: %v", err)
	}

	return items
}

func GetMockSummaryResponse() *models.SummaryResponse {
	return &models.SummaryResponse{
		OverallSummary: "Today has been a significant day in the LocalLLaMA community with major developments in model optimization, tooling, and infrastructure. The community continues to push boundaries in local LLM deployment and practical applications.",
		KeyDevelopments: []models.KeyDevelopment{
			{
				Text:   "A user achieved 5000 tokens per second processing speed using 2x3090 GPUs with Qwen2.5-7B model, demonstrating significant improvements in local model inference speed through careful optimization of batch sizes and quantization.",
				ItemID: "t3_1k0tkca",
			},
			{
				Text:   "LocalAI v2.28.0 and LocalAGI were released, providing a comprehensive stack for running AI agents locally with existing LLM setups. This represents a major step forward in local AI agent deployment capabilities.",
				ItemID: "t3_1k0haqw",
			},
			{
				Text:   "A community member demonstrated achieving 160GB VRAM for approximately $1000 using MI50 GPUs, showing the continued innovation in cost-effective local LLM deployment solutions.",
				ItemID: "t3_1k0b8wx",
			},
		},
	}
}
