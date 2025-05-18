package prompts

import (
	"bytes"
	"errors"
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
	* 1 - 2 paragraphs, extracting key points of interest from the post, image description, factoring in relevant, factual information from the comments
	* Extrapolate on the details of these key points of interest
	* Provide highly detailed technical analysis, if applicable
	* If this development matters, explain why
* "CommentOverview"
  * 1 - 2 paragraphs that
    * Captures the community sentiment
    * Highlights interesting discussions
    * Notes any concerns or criticisms
* "IsRelevant"
  * A final judgement boolean flag. If the item matches any of the exclusion criteria, IsRelevant should be false.

Write in a conversational, engaging style while maintaining technical accuracy. Don't be afraid to geek out about interesting technical details!

Do not start with 'This post...' or 'This item...'.

Respond only with valid JSON. Put JSON in ` + "```json" + ` tags.
Use the following JSON structure:
{
  "id": "t3_1keo3te",
  "title": "",
  "overview": "",
  "comment_overview": "",
  "is_relevant": true
}
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

Respond only with valid JSON. Put JSON in ` + "```json" + ` tags.
{
  "key_developments": [
    {
      "text": "",
      "item_id": ""
    }
  ]
}
`

const imagePromptTemplate = `You are {{.PersonaIdentity}}

Your task is to analyze the provided image and generate a detailed description.

The image is from a post titled: "{{.Title}}"

Describe what is shown in the image (people, objects, text, UI elements, charts, etc.), within 400 words.

Respond with a concise but comprehensive description focusing on technical and factual details. If something is not in English, is blurry or not clear, do not describe it.`

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
	if p.PersonaIdentity == "" {
		return "", errors.New("persona identity is empty")
	}

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
