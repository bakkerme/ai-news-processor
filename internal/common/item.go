package common

// Item represents the structure of the JSON object
type Item struct {
	Title          string `json:"Title" jsonschema_description:"Title of the post"`
	ID             string `json:"ID" jsonschema_description:"Post ID"`
	Summary        string `json:"Summary" jsonschema_description:"Provide a summary of the post content"`
	CommentSummary string `json:"Comment Summary" jsonschema_description:"Provide a summary and semtiment of the comments"`
	Link           string `json:"Link" jsonschema_description:"A link to the post"`
	Relevance      string `json:"Relevance" jsonschema_description:"Why is this relevant?"`
	IsRelevant     bool   `json:"IsRelevant" jsonschema_description:"Should this be included?"`
}
