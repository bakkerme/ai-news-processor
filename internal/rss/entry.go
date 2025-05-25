package rss

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Feedlike is an interface that can be used to represent any type that has a FeedString method, i.e. Feed and CommentFeed
type Feedlike interface {
	FeedString() string
}

// Feed and Comment Feed are used as intermediate types for RSS feeds
type Feed struct {
	Entries []Entry `xml:"entry"`
	RawRSS  string  // Added field to store raw RSS data
}

func (f *Feed) FeedString() string {
	return f.RawRSS // Method to return the raw RSS data
}

type CommentFeed struct {
	Entries []EntryComments `xml:"entry"`
	RawRSS  string          // Added field to store raw RSS data
}

func (cf *CommentFeed) FeedString() string {
	return cf.RawRSS // Method to return the raw RSS data
}

// Entry and EntryComments are used throughout the codebase for RSS feeds
type Entry struct {
	Title               string            `xml:"title" json:"title"`
	Link                Link              `xml:"link" json:"link"`
	ID                  string            `xml:"id" json:"id"`
	Published           time.Time         `xml:"published" json:"published"`
	Content             string            `xml:"content" json:"content"`
	Comments            []EntryComments   `xml:"comments" json:"comments"`
	ExternalURLs        []url.URL         `json:"externalURLs"`                                                 // New field to store external URLs found in content
	ImageURLs           []url.URL         `json:"imageURLs"`                                                    // New field to store extracted image URLs
	MediaThumbnail      MediaThumbnail    `xml:"http://search.yahoo.com/mrss/ thumbnail" json:"mediaThumbnail"` // Field to store thumbnail information from media namespace
	ImageDescription    string            `json:"imageDescription"`                                             // Field to store image descriptions from dedicated image processing
	WebContentSummaries map[string]string `json:"webContentSummaries"`                                          // New field to store summaries of external URLs found in content
}

type EntryComments struct {
	Content string `xml:"content"`
}

type Link struct {
	Href string `xml:"href,attr"`
}

// MediaThumbnail represents the media:thumbnail element in RSS feeds
type MediaThumbnail struct {
	URL string `xml:"url,attr"`
}

func (e *Entry) String(disableTruncation bool) string {
	var s strings.Builder
	s.WriteString(fmt.Sprintf("Title: %s\nID: %s\nSummary: %s\nImageDescription: %s\n",
		strings.Trim(e.Title, " "),
		e.ID,
		cleanContent(e.Content, 1200, disableTruncation),
		e.ImageDescription,
	))

	if len(e.ExternalURLs) > 0 {
		s.WriteString("\nExternal URLs:\n")
		for _, url := range e.ExternalURLs {
			s.WriteString(fmt.Sprintf("- %s\n", url.String()))
		}
	}

	if len(e.WebContentSummaries) > 0 {
		s.WriteString("\nExternal URL Summaries:\n")
		for url, summary := range e.WebContentSummaries {
			s.WriteString(fmt.Sprintf("- %s: %s\n", url, summary))
		}
	}

	for _, comment := range e.Comments {
		s.WriteString(fmt.Sprintf("Comment: %s\n", cleanContent(comment.Content, 600, disableTruncation)))
	}

	return s.String()
}

func (e *Entry) GetCommentRSSURL() string {
	return fmt.Sprintf("%s.rss?depth=1", e.Link.Href)
}

// GetID returns the Entry's ID, implementing the ContentProvider interface
func (e Entry) GetID() string {
	return e.ID
}

// GetContent returns the Entry's Content, implementing the ContentProvider interface
func (e Entry) GetContent() string {
	return e.Content
}

// UnmarshalXML implements xml.Unmarshaler for custom time parsing
func (e *Entry) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type Alias Entry
	aux := &struct {
		Published string `xml:"published"`
		*Alias
	}{
		Alias: (*Alias)(e),
	}
	if err := d.DecodeElement(aux, &start); err != nil {
		return err
	}

	// Parse the time string
	if aux.Published != "" {
		t, err := time.Parse(time.RFC3339, aux.Published)
		if err != nil {
			return fmt.Errorf("failed to parse published time: %w", err)
		}
		e.Published = t
	}
	return nil
}
