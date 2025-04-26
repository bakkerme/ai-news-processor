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

// SummaryResponse represents an overall summary of multiple relevant AI news items
type SummaryResponse struct {
	OverallSummary     string   `json:"OverallSummary" jsonschema_description:"A high-level summary of the major AI developments and trends" jsonschema:"required"`
	KeyDevelopments    []string `json:"KeyDevelopments" jsonschema_description:"List of the most significant developments" jsonschema:"required"`
	EmergingTrends     string   `json:"EmergingTrends" jsonschema_description:"Analysis of emerging trends across the articles" jsonschema:"required"`
	TechnicalHighlight string   `json:"TechnicalHighlight" jsonschema_description:"Most technically significant development" jsonschema:"required"`
}
