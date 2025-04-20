package common

import (
	"fmt"
	"strings"
)

// Item represents the structure of the JSON object
type Item struct {
	Title         string `json:"Title"`
	Summary       string `json:"Summary"`
	Link          string `json:"Link"`
	Relevance     string `json:"Relevance"`
	ShouldInclude bool   `json:"ShouldInclude"`
}

func (i *Item) FormatItem() string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("Title: %s\n", i.Title))
	builder.WriteString(fmt.Sprintf("Summary: %s\n", i.Summary))
	builder.WriteString(fmt.Sprintf("Link: %s\n", i.Link))
	builder.WriteString(fmt.Sprintf("Reason why this is relevant: %s\n", i.Relevance))
	builder.WriteString(fmt.Sprintf("Should this be included: %v\n", i.ShouldInclude))

	return builder.String()
}
