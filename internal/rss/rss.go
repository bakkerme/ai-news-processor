package rss

import (
	"encoding/xml"
	"fmt"
	"github.com/grokify/html-strip-tags-go"
	"strings"
)

type Feed struct {
	Entries []Entry `xml:"entry"`
}

type Entry struct {
	Title     string `xml:"title"`
	Link      Link   `xml:"link"`
	ID        string `xml:"id"`
	Published string `xml:"published"`
	Content   string `xml:"content"`
	Comments  []EntryComments
}

type CommentFeed struct {
	Entries []EntryComments `xml:"entry"`
}

type EntryComments struct {
	Content string `xml:"content"`
}

type Link struct {
	Href string `xml:"href,attr"`
}

func (e *Entry) String() string {
	var s strings.Builder
	s.WriteString(fmt.Sprintf("Title: %s\nID: %s\nSummary: %s\n",
		strings.Trim(e.Title, " "),
		e.ID,
		cleanContent(e.Content, 1200),
	))

	for _, comment := range e.Comments {
		s.WriteString(fmt.Sprintf("Comment: %s\n", cleanContent(comment.Content, 600)))
	}

	return s.String()
}

func (e *Entry) GetCommentRSSURL() string {
	return fmt.Sprintf("%s.rss?depth=1", e.Link.Href)
}

func ProcessRSSFeed(input string) (*Feed, error) {
	var feed Feed
	if err := xml.Unmarshal([]byte(input), &feed); err != nil {
		return nil, err
	}

	return &feed, nil
}

func ProcessCommentsRSSFeed(input string) (*CommentFeed, error) {
	var commentFeed CommentFeed
	if err := xml.Unmarshal([]byte(input), &commentFeed); err != nil {
		return nil, err
	}

	return &commentFeed, nil
}

func cleanContent(s string, maxLen int) string {
	stripped := strip.StripTags(s)
	stripped = strings.ReplaceAll(stripped, "&#39;", "'")
	stripped = strings.ReplaceAll(stripped, "&#32;", " ")
	stripped = strings.ReplaceAll(stripped, "&quot;", "\"")

	lenToUse := maxLen
	strLen := len(stripped)

	if strLen < lenToUse {
		lenToUse = strLen
	}

	truncated := stripped[0:lenToUse]

	// Tack a ... on the end to signify it's truncated to the llm
	if lenToUse != strLen {
		truncated += "..."
	}

	return truncated
}
