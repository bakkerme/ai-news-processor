package rss

import (
	"encoding/xml"
	"fmt"
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
	Title     string    `xml:"title"`
	Link      Link      `xml:"link"`
	ID        string    `xml:"id"`
	Published time.Time `xml:"published"`
	Content   string    `xml:"content"`
	Comments  []EntryComments
}

type EntryComments struct {
	Content string `xml:"content"`
}

type Link struct {
	Href string `xml:"href,attr"`
}

func (e *Entry) String(disableTruncation bool) string {
	var s strings.Builder
	s.WriteString(fmt.Sprintf("Title: %s\nID: %s\nSummary: %s\n",
		strings.Trim(e.Title, " "),
		e.ID,
		cleanContent(e.Content, 1200, disableTruncation),
	))

	for _, comment := range e.Comments {
		s.WriteString(fmt.Sprintf("Comment: %s\n", cleanContent(comment.Content, 600, disableTruncation)))
	}

	return s.String()
}

func (e *Entry) GetCommentRSSURL() string {
	return fmt.Sprintf("%s.rss?depth=1", e.Link.Href)
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
