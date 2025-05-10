# External URL Summarization

## Purpose

The External URL Summarization feature is designed to enhance RSS feed items by automatically fetching, extracting, and summarizing content from external URLs found within the feed item's content. This provides users with a concise overview of linked resources without needing to visit each URL manually.

Note: For runtime reasons, this system is set to only summarise the first URL in the feed.

## Mechanism

The process of summarizing external URLs is orchestrated within the `internal/llm/Processor` and involves several steps:

1.  **Enablement**: The feature is active if the `URLSummaryEnabled` flag in the `EntryProcessConfig` (defined in `internal/llm/processor_types.go`) is set to `true`.

2.  **URL Extraction**:
    *   When processing an RSS entry, the `Processor.processExternalURLs` method is invoked.
    *   It utilizes an `urlextraction.Extractor` (specifically, `urlextraction.RedditExtractor` found in `internal/urlextraction/extractor.go`) to parse the HTML content of an `rss.Entry`.
    *   The `RedditExtractor` identifies and extracts all hyperlinks, filtering out any URLs belonging to Reddit domains (e.g., `reddit.com`, `redd.it`).

3.  **Content Fetching**:
    *   For each valid external URL, the system uses an HTTP fetcher (`internal/fetcher/fetcher.go`) to retrieve the content of the linked page.

4.  **Article Extraction**:
    *   The fetched HTML content is then processed by `contentextractor.ExtractArticle` (from `internal/contentextractor/extractor.go`).
    *   This function leverages the `go-readability` library to isolate the main article text from surrounding clutter like navigation menus, ads, and footers, providing clean text for summarization.

5.  **Summarization**:
    *   The cleaned article text, along with its title and original URL, is passed to the `Processor.summarizeWebSite` method.
    *   This method, in turn, calls `Processor.chatCompletionForWebSummary`, which makes a request to a Language Model (LLM) to generate a concise summary of the provided content.
    *   The LLM is prompted to act as a concise summarizer for web content.

6.  **Storage**:
    *   The generated summary for each external URL is stored in the `ExternalURLSummaries` field of the `rss.Entry` object (defined in `internal/rss/types.go`). This field is a map where keys are the external URLs and values are their corresponding summaries.

## Configuration

The primary configuration for this feature is the `URLSummaryEnabled` boolean flag. This is a global setting for the application, typically configured via an environment variable `ANP_LLM_URL_SUMMARY_ENABLED`.

It is defined within the `internal/specification/Specification` struct:

```go
// internal/specification/specification.go
type Specification struct {
    // ... other fields
    LlmUrlSummaryEnabled bool `split_words:"true" default:"true"`
    // ... other fields
}
```

And also mirrored in the `EntryProcessConfig` struct, which is passed down to the processor:

```go
// internal/llm/processor_types.go
type EntryProcessConfig struct {
    // ... other fields
    URLSummaryEnabled    bool // Whether URL summarization is enabled
}
```

Setting this to `true` enables the feature, while `false` disables it.

## Output

The summaries of external URLs are stored directly within the `rss.Entry` struct in the `ExternalURLSummaries` map. This allows downstream processes to access these summaries. For instance, they can be included in the main text summarization phase or presented as additional information alongside the primary RSS entry content.

## Key Components

-   **`internal/llm/processor.go`**: Contains the main orchestration logic in `Processor.processExternalURLs` and `Processor.summarizeWebSite`.
-   **`internal/llm/processor_types.go`**: Defines `EntryProcessConfig` including the `URLSummaryEnabled` flag.
-   **`internal/urlextraction/extractor.go`**: Provides the `RedditExtractor` for identifying and extracting external URLs.
-   **`internal/contentextractor/extractor.go`**: Implements `ExtractArticle` for cleaning HTML content.
-   **`internal/rss/types.go`**: Defines the `Entry` struct, which includes the `ExternalURLSummaries` map to store the results.
-   **`internal/fetcher/fetcher.go`**: Handles the HTTP fetching of URL content. 