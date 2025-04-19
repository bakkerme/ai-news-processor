package main

import (
	"encoding/json"
	"github.com/bakkerme/ai-news-processor/common"
	"log"
	"os"
)

func returnFakeLLMResponse() []common.Item {
	// Assuming localllama.rss is located in the same directory as this file
	jsonData, err := os.ReadFile("./llmresponse.json")
	if err != nil {
		panic(err)
	}

	// Unmarshal the JSON string into an Item struct
	var items []common.Item
	if err := json.Unmarshal([]byte(jsonData), &items); err != nil {
		log.Fatalf("Error unmarshalling JSON: %v", err)
	}

	return items
}
