package prompts

import (
	"bytes"
	"text/template"

	"github.com/bakkerme/ai-news-processor/internal/persona"
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

{{if .ImageDescription}}
The following image description was generated from the post:
{{.ImageDescription}}
{{end}}

If an item matches any of the exclusion criteria, set the IsRelevant field to false.

For each item, provide a detailed analysis that includes:
* "ID"
* "Title"
* "Summmary"
  * A comprehensive summary (4-5 sentences) that 
    * Provides in detail the content of the item
    * Mentions specific performance metrics or benchmarks if applicable
    * Takes into account the image description if it is present
* "CommentSummary"
  * A detailed comment analysis that:
    * Captures the community sentiment
    * Highlights interesting discussions
    * Notes any concerns or criticisms
* "IsRelevant" A final judgement boolean flag. If the item matches any of the exclusion criteria, IsRelevant should be false.

This is a newsletter. Write in a conversational, engaging style while maintaining technical accuracy. Don't be afraid to geek out about interesting technical details!

Respond only with JSON. Put JSON in ` + "```json" + ` tags.`

// * Relevance
//   * Explain in detail (4-5 sentences) if and how the item is relevant to the {{.Topic}}. Tell me why should I care about this.

// When analyzing images:
// * Describe what is shown in the image if it's relevant to understanding the content
// * Mention any charts, diagrams, or UI elements that provide additional information
// * If code or text is visible in the image, summarize what it shows
// * Relate the image content back to the main topic when applicable

const summaryPromptTemplate = `You are {{.PersonaIdentity}}

{{.SummaryPromptTask}}

Your analysis should focus on:
{{range .SummaryAnalysis}}* {{.}}
{{end}}

For the provided set of news items, generate a structured analysis that includes:
* OverallSummary
  * A comprehensive overall summary that synthesizes the major developments
* KeyDevelopments
  * A list of key developments, ordered by significance. For each key development, include the ID of the referenced post as an ItemID field, so it can be linked to the original post.
* EmergingTrends
  * List of 3-5 emerging trends visible across multiple items
* TechnicalHighlight
  * The single most technically significant development, with explanation (write this as plain text, not JSON)

The response format for KeyDevelopments should be an array of objects, each with a Text and an ItemID field, where ItemID matches the ID of a post in the input.

Focus on technical accuracy while maintaining an engaging, analytical style. Avoid generic statements and focus on specific, concrete developments and their implications. This is a newsletter.

Respond only with JSON. Put JSON in ` + "```json" + ` tags.`

const imagePromptTemplate = `You are {{.PersonaIdentity}}

Your task is to analyze the provided image and generate a detailed description.

The image is from a post titled: "{{.Title}}"

Describe:
* What is shown in the image (people, objects, text, UI elements, charts, etc.)
* Any technical details visible in the image
* How the image relates to the post title
* Key insights that can be gained from the image

Respond with a concise but comprehensive description focusing on technical and factual details.`

// ComposePrompt generates a system prompt for the given persona using the base template
func ComposePrompt(p persona.Persona, imageDescription string) (string, error) {
	tmpl, err := template.New("base").Parse(basePromptTemplate)
	if err != nil {
		return "", err
	}

	// Create a data structure for the template that includes the image description
	data := struct {
		persona.Persona
		ImageDescription string
	}{
		Persona:          p,
		ImageDescription: imageDescription,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func ComposeSummaryPrompt(p persona.Persona) (string, error) {
	tmpl, err := template.New("summary").Parse(summaryPromptTemplate)
	if err != nil {
		return "", err
	}

	// Create a data structure for the template that includes the image descriptions
	data := struct {
		persona.Persona
	}{
		Persona: p,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ComposeImagePrompt generates a system prompt for image description
func ComposeImagePrompt(p persona.Persona, title string) (string, error) {
	tmpl, err := template.New("image").Parse(imagePromptTemplate)
	if err != nil {
		return "", err
	}

	// Create a data structure for the template
	data := struct {
		PersonaIdentity string
		Title           string
	}{
		PersonaIdentity: p.PersonaIdentity,
		Title:           title,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
