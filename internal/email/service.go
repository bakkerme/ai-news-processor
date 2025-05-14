package email

import (
	"fmt"
	"os"
	"time"

	"github.com/bakkerme/ai-news-processor/internal/models"
	"github.com/bakkerme/ai-news-processor/internal/specification"
)

// Service handles email rendering and delivery
type Service struct {
	emailer *Client
	config  *specification.Specification
}

// NewService creates a new email service
func NewService(config *specification.Specification) (*Service, error) {
	emailer, err := New(
		config.EmailHost,
		config.EmailPort,
		config.EmailUsername,
		config.EmailPassword,
		config.EmailFrom,
	)
	if err != nil {
		return nil, fmt.Errorf("could not set up emailer: %w", err)
	}

	return &Service{
		emailer: emailer,
		config:  config,
	}, nil
}

// RenderAndSend handles rendering and sending an email with the specified items and summary
func (s *Service) RenderAndSend(items []models.Item, summary *models.SummaryResponse, personaName string) error {
	email, err := RenderEmail(items, summary, personaName)
	if err != nil {
		return fmt.Errorf("could not render email: %w", err)
	}

	if !s.config.DebugSkipEmail {
		fmt.Printf("Sending email to %s\n", s.config.EmailTo)
		return s.emailer.Send(s.config.EmailTo, fmt.Sprintf("%s News", personaName), email)
	}

	// If in debug mode, write to disk instead
	return writeEmailToDisk(email)
}

// writeEmailToDisk writes the email content to a file for debugging
func writeEmailToDisk(content string) error {
	filename := fmt.Sprintf("email_%s.html", time.Now().Format("2006-01-02_15-04-05"))
	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("could not write email to disk: %w", err)
	}
	fmt.Printf("Email written to %s\n", filename)
	return nil
}
