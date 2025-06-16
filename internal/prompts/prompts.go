package prompts

import (
	"bytes"
	"errors"
	"fmt"
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

For each item, provide a newsletter-style explanation that includes:
* "ID"
* "Title"
* "Overview"
	* A quick, concise overview of the post content in 2-3 bullet points or sentences
	* Designed to help readers quickly decide if they want to read the full post
	* Should highlight the most important aspects without going into deep technical detail
* "Summary"
	* 1 - 2 paragraphs, extracting key points of interest from the post, image description, factoring in relevant, factual information from the comments
	* Extrapolate on the details of these key points of interest
	* Provide highly detailed technical analysis, if applicable
	* If this development matters, explain why
* "CommentSummary"
  * 1 - 2 paragraphs that
    * Captures the community sentiment
    * Highlights interesting discussions
    * Notes any concerns or criticisms
* "IsRelevant"
  * A final judgement boolean flag. If the item matches any of the exclusion criteria, IsRelevant should be false.

Write in a conversational, engaging style while maintaining technical accuracy. Don't be afraid to geek out about interesting technical details!

Do not start with 'This post...' or 'This item...'.

Keep responses concise but comprehensive. Aim for:
* Summary: 2-3 sentences per paragraph (500-800 words total)
* CommentSummary: 2-3 sentences per paragraph (300-600 words total)

Respond only with valid JSON. Put JSON in ` + "```json" + ` tags.
Use the following JSON structure:
{{.ItemJSONExample}}
`

const summaryPromptTemplate = `You are {{.PersonaIdentity}}

{{.SummaryPromptTask}}

Your analysis should focus on:
{{range .SummaryAnalysis}}* {{.}}
{{end}}

For the provided set of news items, generate a structured analysis that includes:
* KeyDevelopments
  * A list of key developments, ordered by significance. For each key development, include the ID of the referenced post as an ItemID field, so it can be linked to the original post.

The response format for KeyDevelopments should be an array of objects, each with a Text and an ItemID field, where ItemID matches the ID of a post in the input.

Focus on technical accuracy while maintaining an engaging, analytical style. Avoid generic statements and focus on specific, concrete developments and their implications. This is a newsletter.

Keep the response concise but informative. Aim for 2-3 key developments with 1-2 sentences each.

Respond only with valid JSON. Put JSON in ` + "```json" + ` tags.
{{.SummaryJSONExample}}
`

const imagePromptTemplate = `You are {{.PersonaIdentity}}

Your task is to analyze the provided image and generate a detailed description.

The image is from a post titled: "{{.Title}}"

Describe what is shown in the image (people, objects, text, UI elements, charts, etc.), within 400 words.

Keep your description concise but comprehensive, focusing on the most important and technically relevant details.

Respond with a concise but comprehensive description focusing on technical and factual details. If something is not in English, is blurry or not clear, do not describe it.`

// ComposePrompt generates a system prompt for the given persona using the base template
func ComposePrompt(p persona.Persona, imageDescription string) (string, error) {
	tmpl, err := template.New("base").Parse(basePromptTemplate)
	if err != nil {
		return "", err
	}

	// Generate JSON example automatically from real struct
	itemJSONExample, err := GetRealItemJSONExample()
	if err != nil {
		return "", fmt.Errorf("failed to generate item JSON example: %w", err)
	}

	// Create a data structure for the template that includes the image description and generated JSON example
	data := struct {
		persona.Persona
		ImageDescription string
		ItemJSONExample  string
	}{
		Persona:          p,
		ImageDescription: imageDescription,
		ItemJSONExample:  itemJSONExample,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func ComposeSummaryPrompt(p persona.Persona) (string, error) {
	if p.PersonaIdentity == "" {
		return "", errors.New("persona identity is empty")
	}

	tmpl, err := template.New("summary").Parse(summaryPromptTemplate)
	if err != nil {
		return "", err
	}

	// Generate JSON example automatically from real struct
	summaryJSONExample, err := GetRealSummaryResponseJSONExample()
	if err != nil {
		return "", fmt.Errorf("failed to generate summary JSON example: %w", err)
	}

	// Create a data structure for the template that includes the generated JSON example
	data := struct {
		persona.Persona
		SummaryJSONExample string
	}{
		Persona:            p,
		SummaryJSONExample: summaryJSONExample,
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
