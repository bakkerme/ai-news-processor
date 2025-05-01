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

An item must match the following criteria to be considered relevant:
{{range .RelevanceCriteria}}* {{.}}
{{end}}

An item is not relevant if it matches the following criteria:
{{range .ExclusionCriteria}}* {{.}}
{{end}}

If an item matches any of the exclusion criteria, set the IsRelevant field to false.

For each item, provide a detailed analysis that includes:
* ID
* Title
* A comprehensive summary (4-5 sentences) that 
  - Provides in detail the content of the item
  - Mentions specific performance metrics or benchmarks if applicable
* A detailed comment analysis that:
  - Captures the community sentiment
  - Highlights interesting discussions
  - Notes any concerns or criticisms
* Expalain in detail (4-5 sentences) if and how the item is relevant to to the {{.Topic}}. Tell me why should I care about this.
* A final "IsRelevant" judgement boolean flag. If the item matches any of the exclusion criteria, IsRelevant should be false.

This is a newlsetter. Write in a conversational, engaging style while maintaining technical accuracy. Don't be afraid to geek out about interesting technical details!

Respond only with JSON. Do not include ` + "```json" + ` or anything other than json. Return all input data in the response.`

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

Focus on technical accuracy while maintaining an engaging, analytical style. Avoid generic statements and focus on specific, concrete developments and their implications. This is a newsletter.

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
