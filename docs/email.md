# Feature: Email Module

## Overview
The email module handles the rendering and delivery of HTML emails containing processed AI news content. It provides SMTP-based email delivery, HTML templating, and debugging utilities for the application.

## Features
- SMTP client for sending HTML emails
- HTML templating for email rendering
- Debug mode for writing emails to disk instead of sending
- Embeds templates directly in the application binary
- Support for personalized email content with item summaries
- Responsive HTML design for various email clients and screen sizes

## Directory Structure
```plaintext
internal/email/
  ├─ email.go       # Core email client and SMTP sending functionality
  ├─ service.go     # Higher-level email service with rendering and configuration
  ├─ render.go      # HTML template rendering
  └─ templates/     # HTML email templates
     └─ email_template.tmpl # Main email template with responsive design
```

## Notable Types

### email.go
- `EmailSender`: Interface defining the contract for sending emails
- `Client`: SMTP email client implementing the `EmailSender` interface

### service.go
- `Service`: Manages email rendering and delivery with configuration options

### render.go
- `EmailData`: Data structure passed to email templates for rendering

## Notable Functions

### email.go
- `New(host, port, username, password, sender string) (*Client, error)`: Creates a new SMTP email client
- `(c *Client) Send(recipient string, subject string, htmlContent string) error`: Sends an HTML email to a specified recipient

### service.go
- `NewService(config *specification.Specification) (*Service, error)`: Creates a new email service with configuration
- `(s *Service) RenderAndSend(items []common.Item, summary *common.SummaryResponse) error`: Renders and sends an email with news items and summary
- `writeEmailToDisk(content string) error`: Debug utility to write email content to a file instead of sending

### render.go
- `RenderEmail(items []common.Item, summary *common.SummaryResponse) (string, error)`: Renders items and summary into HTML using templates

## Templates
The email module uses Go's `text/template` package with embedded templates via Go's embed package. The main template (`email_template.tmpl`) provides:

- Responsive HTML layout with media queries
- Sections for overall summary, key developments, and emerging trends
- Individual news item rendering with titles, summaries, and links
- Styling for different content types and highlights
- Header and footer sections

## Usage Example
The email module is typically used through the `Service` type:

```go
// Create email service
emailService, err := email.NewService(config)
if err != nil {
    log.Fatalf("Failed to create email service: %v", err)
}

// Send news items with summary
err = emailService.RenderAndSend(processedItems, summary)
if err != nil {
    log.Errorf("Failed to send email: %v", err)
}
```

When `DebugSkipEmail` is enabled in the configuration, emails will be written to disk rather than sent via SMTP. 