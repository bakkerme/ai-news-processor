package rss

import (
	"encoding/xml"
	"fmt"
	"github.com/grokify/html-strip-tags-go"
)

type Feed struct {
	Entries []Entry `xml:"entry"`
}

type Entry struct {
	Title   string `xml:"title"`
	Link    Link   `xml:"link"`
	ID      string `xml:"id"`
	Updated string `xml:"updated"`
	Content string `xml:"content"`
}

type Link struct {
	Href string `xml:"href,attr"`
}

func (e *Entry) String() string {
	return fmt.Sprintf("Title: %s\nID: %s\nDate: %s\nSummary: %s\n\n",
		e.Title,
		e.ID,
		e.Updated,
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
	return strip.StripTags(s)
}
