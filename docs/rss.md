# Feature: RSS Module

## Overview
The RSS module handles the fetching, parsing, and enrichment of RSS feed data and comments. It supports both live HTTP retrieval and mock feeds for testing, along with utilities for cleaning and truncating content.

## Features
- Parse RSS and comment feed XML into Go structs
- Fetch feed content over HTTP
- Enrich entries with comments from separate RSS feeds
- Dump raw RSS content to disk for debugging
- Clean HTML tags and entities, with optional truncation
- Mock feed retrieval for running tests without network dependencies
- Custom XML unmarshalling for parsing publication timestamps

## Directory Structure
```plaintext
internal/rss/
  ├─ rss.go         # Core types and parsing logic
  ├─ processor.go   # High-level feed fetching, processing, and enrichment
  └─ mocks.go       # Mock data retrieval for testing
```

## Notable Types
- `Feed`: Represents an RSS feed with a slice of `Entry`.
- `Entry`: Represents a single RSS item, including title, link, ID, publication time, content, and associated comments.
- `CommentFeed`: Represents an RSS feed containing comment entries.
- `EntryComments`: Represents a comment associated with an `Entry`.
- `Link`: Represents the `href` attribute of an RSS link element.

## Notable Functions

### rss.go
- `ProcessRSSFeed(input string) (*Feed, error)`: Unmarshal raw RSS XML into `Feed` structs.
- `ProcessCommentsRSSFeed(input string) (*CommentFeed, error)`: Unmarshal comment feed XML into `CommentFeed`.
- `GetFeeds(urls []string) ([]*Feed, error)`: Fetch and process multiple RSS feeds from URLs.
- `GetMockFeeds(personaName string) []*Feed`: Load mock RSS feed data for a specific persona.
- `FetchRSS(url string) (string, error)`: HTTP GET request to retrieve RSS XML as a string.
- `cleanContent(s string, maxLen int, disableTruncation bool) string`: Strip HTML tags, replace HTML entities, and truncate content.
- `(e *Entry) String(disableTruncation bool) string`: Format an entry and its comments as text, with optional truncation.
- `(e *Entry) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error`: Custom XML unmarshalling for parsing the `published` timestamp.

### processor.go
- `FetchAndProcessFeed(feedURL string, mockRSS bool, personaName string, debugRssDump bool) ([]Entry, error)`: Fetch or mock an RSS feed, parse it, and return entries.
- `FetchAndEnrichWithComments(entries []Entry, mockRSS bool, debugRssDump bool, personaName string) ([]Entry, error)`: Enrich each entry with comments from its comment RSS feed.
- `getCommentRSS(entry Entry) (string, error)`: Internal helper to fetch the comment RSS for an entry.
- `FindEntryByID(id string, entries []Entry) *Entry`: Locate an `Entry` by its ID in a slice.
- `DumpRSS(feedURL, content, personaName, itemName string) error`: Save raw RSS XML content to disk under `feed_mocks/` for debugging.

### mocks.go
- `ReturnFakeRSS(personaName string) string`: Read a mock RSS file for a persona from disk.
- `ReturnFakeCommentRSS(personaName, id string) string`: Read a mock comment RSS file for a specific entry. 