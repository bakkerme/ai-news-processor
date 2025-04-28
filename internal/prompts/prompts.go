package prompts

import (
	"bytes"
	"text/template"

	"github.com/bakkerme/ai-news-processor/internal/common"
)

const basePromptTemplate = `You are {{.PersonaIdentity}}

{{.BasePromptTask}}

Relevant items include:
{{range .FocusAreas}}* {{.}}
{{end}}

An item is not relevant if it does not match the following criteria:
{{range .RelevanceCriteria}}* {{.}}
{{end}}

For each item, provide a detailed analysis that includes:
* ID
* Title
* A comprehensive summary (4-5 sentences) that:
  - Describes the technical details and specifications
  - Explains the significance of the development
  - Highlights any novel approaches or techniques
  - Mentions specific performance metrics or benchmarks if available
* A detailed comment analysis that:
  - Captures the community sentiment
  - Highlights interesting technical discussions
  - Notes any concerns or criticisms
* A clear explanation of why this development matters to {{.Topic}} researchers and practitioners. The Relevance field should:
  - Explain the practical implications of the development
  - Describe how it advances the field or solves existing problems
  - Note any potential impact on industry or research
  - Avoid generic statements like "this is important" or "this matters"
* A final "IsRelevant" judgement boolean flag

Write in a conversational, engaging style while maintaining technical accuracy. Don't be afraid to geek out about interesting technical details!

Respond only with JSON. Do not include ` + "```json" + ` or anything other than json`

const summaryPromptTemplate = `You are {{.PersonaIdentity}}

{{.SummaryPromptTask}}

Your analysis should focus on:
{{range .SummaryAnalysis}}* {{.}}
{{end}}

For the provided set of news items, generate a structured analysis that includes:
* A comprehensive overall summary that synthesizes the major developments
* A list of key developments, ordered by significance. For each key development, include the ID of the referenced post as an ItemID field, so it can be linked to the original post.
* Analysis of emerging trends visible across multiple items
* The single most technically significant development, with explanation (write this as plain text, not JSON)

The response format for KeyDevelopments should be an array of objects, each with a Text and an ItemID field, where ItemID matches the ID of a post in the input.

Focus on technical accuracy while maintaining an engaging, analytical style. Avoid generic statements and focus on specific, concrete developments and their implications.

Respond only with JSON. Do not include ` + "```json" + ` or anything other than json`

// ComposePrompt generates a system prompt for the given persona using the base template
func ComposePrompt(persona common.Persona) (string, error) {
	tmpl, err := template.New("base").Parse(basePromptTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, persona)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func ComposeSummaryPrompt(persona common.Persona) (string, error) {
	tmpl, err := template.New("summary").Parse(summaryPromptTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, persona)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
