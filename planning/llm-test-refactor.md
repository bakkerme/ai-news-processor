Here's a list of such calls from `internal/llm/llm.go` and how they could be extracted into interfaces:

1.  **`httputil.FetchImageAsBase64(imgURL)`**
    *   **Location:** `processImageWithRetry` method.
    *   **Reasoning:** This function likely performs an HTTP GET request to fetch an image and then encodes it. This external network call makes unit testing `processImageWithRetry` difficult without actually hitting a network, which can lead to slow and flaky tests.
    *   **Extraction Strategy:**
        1.  **Define an `ImageFetcher` interface:**
            ```go
            package llm // Or a more general package like 'internal/httpio'

            type ImageFetcher interface {
                FetchAsBase64(imageURL string) (string, error)
            }
            ```
        2.  **Add `ImageFetcher` to `Processor`:**
            Modify the `Processor` struct to include a field of this new interface type.
            ```go
            type Processor struct {
                // ... other fields
                imageFetcher ImageFetcher
                // ...
            }
            ```
        3.  **Update `NewProcessor`:**
            The `NewProcessor` function should be updated to accept an `ImageFetcher` instance or to initialize a default concrete implementation.
            *   A concrete type, say `httputil.DefaultImageFetcher`, would implement this interface, and its `FetchAsBase64` method would contain the original logic from `httputil.FetchImageAsBase64`.
            ```go
            // In NewProcessor:
            // processor.imageFetcher = &httputil.DefaultImageFetcher{} // or accept as parameter
            ```
        4.  **Modify the Call Site:**
            In `processImageWithRetry`, change the direct call to use the interface method:
            ```go
            // dataURI := httputil.FetchImageAsBase64(imgURL) // Old
            dataURI, err := p.imageFetcher.FetchAsBase64(imgURL) // New
            // Handle err appropriately
            if err != nil {
                return "", fmt.Errorf("could not fetch image using imageFetcher from URL %s: %w", imgURL, err)
            }
            // The original code checked if dataURI was empty, which might now be signified by an error.
            ```
        5.  **Mocking in Tests:** In your tests, you can then provide a mock implementation of `ImageFetcher` that returns predefined data or errors without making network calls.

2.  **`contentextractor.ExtractArticle(resp.Body, parsedURL)`**
    *   **Location:** `processExternalURLs` method.
    *   **Reasoning:** While the `resp.Body` comes from a potentially mocked `Fetcher`, the `ExtractArticle` function itself is a direct call. If this function is complex, has its own dependencies, or performs significant processing, abstracting it can simplify testing `processExternalURLs` by allowing you to focus solely on the logic within `processExternalURLs` and mock the article extraction behavior.
    *   **Extraction Strategy (Similar to `ImageFetcher`):**
        1.  **Define an `ArticleExtractor` interface:**
            ```go
            package llm // Or 'internal/contentextractor'

            import (
                "io"
                "net/url"
                "github.com/bakkerme/ai-news-processor/internal/contentextractor" // For ArticleData type
            )

            type ArticleExtractor interface {
                Extract(body io.Reader, sourceURL *url.URL) (*contentextractor.ArticleData, error)
            }
            ```
        2.  **Add `ArticleExtractor` to `Processor`:**
            ```go
            type Processor struct {
                // ... other fields
                articleExtractor ArticleExtractor
                // ...
            }
            ```
        3.  **Update `NewProcessor`:**
            Initialize with a concrete implementation (e.g., `contentextractor.DefaultArticleExtractor` which wraps the original `ExtractArticle` function) or accept it as a parameter.
            ```go
            // In NewProcessor:
            // processor.articleExtractor = &contentextractor.DefaultArticleExtractor{} // or accept as parameter
            ```
        4.  **Modify the Call Site:**
            In `processExternalURLs`, use the interface method:
            ```go
            // articleData, err := contentextractor.ExtractArticle(resp.Body, parsedURL) // Old
            articleData, err := p.articleExtractor.Extract(resp.Body, parsedURL) // New
            if err != nil {
                // ...
            }
            ```
        5.  **Mocking in Tests:** Provide a mock `ArticleExtractor` in your tests.

Calls to standard library functions like `url.Parse` (in `processExternalURLs`) or `json.Unmarshal` (in `llmResponseToItems`) are generally not prime candidates for this kind of abstraction unless they present very specific testing challenges. Their behavior is well-defined and can be controlled by the inputs you provide in tests.

By making these changes, you'll significantly improve the unit testability of the `Processor`'s methods, as you can now easily mock out these external interactions and complex processing steps.