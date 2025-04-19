package main

import (
	"encoding/json"
	"fmt"
	"github.com/bakkerme/ai-news-processor/common"
)

func llmResponseToItem(jsonData string) ([]common.Item, error) {
	// Unmarshal the JSON string into an Item struct
	var item []common.Item
	if err := json.Unmarshal([]byte(jsonData), &item); err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	return item, nil
}
