package email

import (
	"errors"
	"fmt"
	"net/smtp"
	"strings"
)

// EmailSender defines the interface for sending emails
type EmailSender interface {
	Send(recipient string, subject string, htmlContent string) error
}

// Client represents an SMTP email client
type Client struct {
	host     string
	port     string
	username string
	password string
	sender   string
}

// New creates a new SMTP email client
func New(host, port, username, password, sender string) (*Client, error) {
	if host == "" || port == "" || username == "" || password == "" || sender == "" {
		return nil, errors.New("all fields (host, port, username, password, sender) are required")
	}

	return &Client{
		host:     host,
		port:     port,
		username: username,
		password: password,
		sender:   sender,
	}, nil
}

// Send sends an HTML email to the specified recipient
func (c *Client) Send(recipient string, subject string, htmlContent string) error {
	if recipient == "" {
		return errors.New("recipient email cannot be empty")
	}

	// Validate recipient contains @
	if !strings.Contains(recipient, "@") {
		return errors.New("invalid recipient email format")
	}

	// Set up authentication
	auth := smtp.PlainAuth("", c.username, c.password, c.host)

	// Construct MIME headers
	headers := make(map[string]string)
	headers["From"] = c.sender
	headers["To"] = recipient
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=\"UTF-8\""

	// Build message from headers
	var message strings.Builder
	for k, v := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	message.WriteString("\r\n" + htmlContent)

	// Send email
	err := smtp.SendMail(
		fmt.Sprintf("%s:%s", c.host, c.port),
		auth,
		c.sender,
		[]string{recipient},
		[]byte(message.String()),
	)

	return err
}
