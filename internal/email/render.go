package email

import (
	"bytes"
	"fmt"
	"text/template"

	"embed"

	"github.com/bakkerme/ai-news-processor/internal/models"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

type EmailData struct {
	Summary *models.SummaryResponse
	Items   []models.Item
}

func RenderEmail(items []models.Item, summary *models.SummaryResponse) (string, error) {
	tmplContent, err := templateFS.ReadFile("templates/email_template.tmpl")
	if err != nil {
		return "", fmt.Errorf("failed to read template: %w", err)
	}

	// Create and parse the template
	tmpl, err := template.New("email").Parse(string(tmplContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	data := EmailData{
		Summary: summary,
		Items:   items,
	}

	// Execute the template into a buffer
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		panic(err)
	}

	// Get the result as a string
	result := buf.String()

	return result, nil
}
