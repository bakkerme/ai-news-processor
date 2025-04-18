package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// Item represents the structure of the JSON object
type Item struct {
	Title                   string `json:"Title"`
	Summary                 string `json:"Summary"`
	Link                    string `json:"Link"`
	ShouldThisBeIncluded    bool   `json:"Should this be included"`
	ReasonWhyThisIsRelevant string `json:"Reason why this is relevant"`
}

func (i *Item) FormatItem() string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("Title: %s\n", i.Title))
	builder.WriteString(fmt.Sprintf("Summary: %s\n", i.Summary))
	builder.WriteString(fmt.Sprintf("Link: %s\n", i.Link))
	builder.WriteString(fmt.Sprintf("Should this be included: %v\n", i.ShouldThisBeIncluded))
	builder.WriteString(fmt.Sprintf("Reason why this is relevant: %s\n", i.ReasonWhyThisIsRelevant))

	return builder.String()
}

func llmResponseToItem(jsonData string) Item {
	// Unmarshal the JSON string into an Item struct
	var item Item
	if err := json.Unmarshal([]byte(jsonData), &item); err != nil {
		log.Fatalf("Error unmarshalling JSON: %v", err)
	}

	return item
}
