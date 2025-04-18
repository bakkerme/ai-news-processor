package main

import (
	"encoding/xml"
	"fmt"
	"strings"
)

type Feed struct {
	Entries []Entry `xml:"entry"`
}

type Entry struct {
	Title   string `xml:"title"`
	Link    string `xml:"link>href"`
	Updated string `xml:"updated"`
	Content string `xml:"content"`
}

func processRSSFeed(input string) (*Feed, error) {
	var feed Feed
	if err := xml.Unmarshal([]byte(input), &feed); err != nil {
		return nil, err
	}

	return &feed, nil
}

func entryToString(entry Entry) string {
	return fmt.Sprintf("Title: %s\nDate: %s\nSummary: %s\n\n",
		entry.Title,
		entry.Updated,
		cleanContent(entry.Content),
	)
}

func cleanContent(s string) string {
	s = strings.ReplaceAll(s, "<!-- SC_ON -->", "")
	s = strings.ReplaceAll(s, "<!-- SC_OFF -->", "")
	s = strings.ReplaceAll(s, "&amp;#32;", " ")
	s = strings.ReplaceAll(s, "```json", " ")
	s = strings.ReplaceAll(s, "```", " ")
	return strings.Join(strings.Fields(s), " ")
}
