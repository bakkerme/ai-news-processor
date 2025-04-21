package common

import (
	"fmt"
	"strings"
)

// Item represents the structure of the JSON object
type Item struct {
	Title         string `json:"Title" jsonschema_description:"Title of the post"`
	ID            string `json:"ID" jsonschema_description:"Post ID"`
	Summary       string `json:"Summary" jsonschema_description:"Provide a summary of the post content"`
	Link          string `json:"Link" jsonschema_description:"A link to the post"`
	Relevance     string `json:"Relevance" jsonschema_description:"Why is this relevant?"`
	ShouldInclude bool   `json:"ShouldInclude" jsonschema_description:"Should this be included?"`
}

func (i *Item) FormatItem() string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("Title: %s\n", i.Title))
	builder.WriteString(fmt.Sprintf("ID: %s\n", i.ID))
	builder.WriteString(fmt.Sprintf("Summary: %s\n", i.Summary))
	builder.WriteString(fmt.Sprintf("Link: %s\n", i.Link))
	builder.WriteString(fmt.Sprintf("Reason why this is relevant: %s\n", i.Relevance))
	builder.WriteString(fmt.Sprintf("Should this be included: %v\n", i.ShouldInclude))

	return builder.String()
}
