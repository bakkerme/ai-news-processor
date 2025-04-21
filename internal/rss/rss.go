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
}

type Link struct {
	Href string `xml:"href,attr"`
}

func (e *Entry) String() string {
	return fmt.Sprintf("Title: %s\nID: %s\nSummary: %s\n\n",
		strings.Trim(e.Title, " "),
		e.ID,
		cleanContent(e.Content),
	)
}

func ProcessRSSFeed(input string) (*Feed, error) {
	var feed Feed
	if err := xml.Unmarshal([]byte(input), &feed); err != nil {
		return nil, err
	}

	return &feed, nil
}

func cleanContent(s string) string {
	stripped := strip.StripTags(s)
	stripped = strings.ReplaceAll(stripped, "&#39;", "'")
	stripped = strings.ReplaceAll(stripped, "&#32;", " ")
	stripped = strings.ReplaceAll(stripped, "&quot;", "\"")

	lenToUse := 500
	maxLen := len(stripped)

	if maxLen < lenToUse {
		lenToUse = maxLen
	}

	truncated := stripped[0:lenToUse]

	return truncated
}
