package common

// Item represents the structure of the JSON object
type Item struct {
	Title          string `json:"Title" jsonschema_description:"Title of the post" jsonschema:"required"`
	ID             string `json:"ID" jsonschema_description:"Post ID" jsonschema:"required"`
	Summary        string `json:"Summary" jsonschema_description:"Provide a summary of the post content" jsonschema:"required"`
	CommentSummary string `json:"Comment Summary" jsonschema_description:"Provide a summary and semtiment of the comments" jsonschema:"required"`
	Link           string `json:"Link" jsonschema_description:"A link to the post" jsonschema:"required"`
	Relevance      string `json:"Relevance" jsonschema_description:"Why is this relevant?" jsonschema:"required"`
	IsRelevant     bool   `json:"IsRelevant" jsonschema_description:"Should this be included?" jsonschema:"required"`
}
