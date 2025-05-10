package fetcher

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/bakkerme/ai-news-processor/internal/http/retry"
)

const DefaultUserAgent = "ai-news-processor-fetcher/1.0"

// HTTPError is a custom error type that wraps an HTTP response when the status code
// indicates an error, but no lower-level network error occurred.
type HTTPError struct {
	StatusCode int
	Status     string
	Response   *http.Response // Keep a reference to the original response
	RetryAfter *time.Duration // Added to store parsed Retry-After header
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("http error: status code %d %s", e.StatusCode, e.Status)
}

// Fetcher defines the interface for fetching HTTP content.
type Fetcher interface {
	Fetch(ctx context.Context, url string) (*http.Response, error)
}

// HTTPFetcher implements the Fetcher interface using a standard http.Client
// and integrates retry logic.
type HTTPFetcher struct {
	client      *http.Client
	retryConfig retry.RetryConfig
	userAgent   string // Added User-Agent field
}

// NewHTTPFetcher creates a new HTTPFetcher with a default http.Client,
// the provided retry configuration, and a custom user agent.
// If client is nil, a default client with a 30-second timeout will be used.
// If userAgent is an empty string, DefaultUserAgent will be used.
func NewHTTPFetcher(client *http.Client, cfg retry.RetryConfig, userAgent string) *HTTPFetcher {
	if client == nil {
		client = &http.Client{
			Timeout: 30 * time.Second, // Default timeout
		}
	}
	ua := userAgent
	if ua == "" {
		ua = DefaultUserAgent
	}
	return &HTTPFetcher{
		client:      client,
		retryConfig: cfg,
		userAgent:   ua, // Store the User-Agent
	}
}

// Fetch performs an HTTP GET request to the specified URL with retry logic.
// The caller is responsible for closing the response body if the error is nil.
func (hf *HTTPFetcher) Fetch(ctx context.Context, url string) (*http.Response, error) {
	retryableFunc := func(innerCtx context.Context) (*http.Response, error) {
		req, err := http.NewRequestWithContext(innerCtx, http.MethodGet, url, nil)
		if err != nil {
			// This error is likely non-retryable (e.g., malformed URL)
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Set the custom User-Agent header
		req.Header.Set("User-Agent", hf.userAgent)

		resp, err := hf.client.Do(req)
		if err != nil {
			// Network error or other error from client.Do
			// resp might be nil here, or might have partial info.
			// shouldRetryHTTP will inspect this error.
			return resp, err
		}

		// Check if the status code indicates an error that should be handled by retry logic
		if resp.StatusCode >= 400 {
			// Wrap the response in a custom error to pass it to shouldRetryHTTP
			// The original response is returned along with the error,
			// so if this is the last attempt, the caller can still inspect it.
			httpError := &HTTPError{
				StatusCode: resp.StatusCode,
				Status:     resp.Status,
				Response:   resp,
			}

			// Handle Retry-After header for 429 (Too Many Requests) and 503 (Service Unavailable)
			if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
				headerVal := resp.Header.Get("Retry-After")
				if headerVal != "" {
					var parsedDuration *time.Duration

					// Try parsing as delay-seconds
					if seconds, errConv := strconv.Atoi(headerVal); errConv == nil {
						if seconds >= 0 { // Non-negative seconds
							dur := time.Duration(seconds) * time.Second
							parsedDuration = &dur
						}
						// else: negative seconds, invalid, parsedDuration remains nil
					} else {
						// Try parsing as HTTP-date
						if date, errParseTime := http.ParseTime(headerVal); errParseTime == nil {
							// Calculate duration until the specified date
							dur := time.Until(date) // time.Until handles past dates by returning non-positive duration
							if dur < 0 {            // If date is in the past, treat as immediate retry (or very soon)
								dur = 0
							}
							parsedDuration = &dur
						}
						// else: not seconds and not a valid HTTP-date, parsedDuration remains nil
					}
					httpError.RetryAfter = parsedDuration
				}
			}
			return resp, httpError
		}

		// Success
		return resp, nil
	}

	// Use the refined shouldRetryHTTP function.
	// Note: retry.ShouldRetry expects `func(error) bool`. Our shouldRetryHTTP will fit this.
	return retry.RetryWithBackoff(ctx, hf.retryConfig, retryableFunc, shouldRetryHTTP)
}

// shouldRetryHTTP determines if an HTTP request should be retried based on the error.
func shouldRetryHTTP(err error) bool {
	if err == nil {
		return false // No error, no need to retry
	}

	// Non-retryable context errors
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		// Retry on 5xx server errors
		if httpErr.StatusCode >= 500 && httpErr.StatusCode <= 599 {
			return true
		}
		// Retry on 429 Too Many Requests
		if httpErr.StatusCode == http.StatusTooManyRequests {
			return true
		}
		// Do not retry other 4xx client errors by default
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return true
		}
	}

	// Check for known non-retryable errors from http.NewRequestWithContext if they weren't wrapped
	// (e.g. if http.NewRequestWithContext itself failed before client.Do was called)
	// This part might be redundant if NewRequestWithContext errors are not retryable by nature.
	// For now, we assume errors from NewRequestWithContext are not retryable unless specifically known.
	// Example: url.Error could be here if the URL is fundamentally invalid.

	// Default to not retrying if the error type is not recognized as transient
	// or a retryable HTTP status.
	return false
}
