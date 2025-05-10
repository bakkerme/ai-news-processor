package fetcher_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bakkerme/ai-news-processor/internal/fetcher"
	"github.com/bakkerme/ai-news-processor/internal/http/retry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a test server with a configurable handler
func setupTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	// No need to defer server.Close() here, as each test that uses it should manage its lifecycle
	// or it can be closed in a t.Cleanup() if used across multiple sub-tests in a table.
	return server
}

func TestHTTPFetcher_Fetch_Successful(t *testing.T) {
	t.Parallel()

	expectedBody := "Hello, world!"
	expectedUserAgent := "test-agent/1.0"
	var userAgentReceived string

	handler := func(w http.ResponseWriter, r *http.Request) {
		userAgentReceived = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(expectedBody))
		assert.NoError(t, err)
	}
	server := setupTestServer(t, handler)
	defer server.Close()

	client := server.Client() // Use the test server's client
	retryCfg := retry.DefaultRetryConfig
	retryCfg.MaxRetries = 1 // No need for many retries on success

	f := fetcher.NewHTTPFetcher(client, retryCfg, expectedUserAgent)

	resp, err := f.Fetch(context.Background(), server.URL)
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, expectedBody, string(bodyBytes))
	assert.Equal(t, expectedUserAgent, userAgentReceived, "User-Agent header did not match")
}

func TestHTTPFetcher_Fetch_DefaultUserAgent(t *testing.T) {
	t.Parallel()
	var userAgentReceived string
	handler := func(w http.ResponseWriter, r *http.Request) {
		userAgentReceived = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}
	server := setupTestServer(t, handler)
	defer server.Close()

	f := fetcher.NewHTTPFetcher(server.Client(), retry.DefaultRetryConfig, "") // Empty user agent

	resp, err := f.Fetch(context.Background(), server.URL)
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer resp.Body.Close()

	assert.Equal(t, fetcher.DefaultUserAgent, userAgentReceived)
}

func TestHTTPFetcher_Fetch_ClientError_NonRetryable(t *testing.T) {
	t.Parallel()

	var requestCount int32
	handler := func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		http.Error(w, "Not Found", http.StatusNotFound)
	}
	server := setupTestServer(t, handler)
	defer server.Close()

	client := server.Client()
	retryCfg := retry.DefaultRetryConfig
	// Make retry attempts very fast for this test if it were to retry
	retryCfg.InitialBackoff = 1 * time.Millisecond
	retryCfg.MaxBackoff = 5 * time.Millisecond
	retryCfg.MaxRetries = 2

	f := fetcher.NewHTTPFetcher(client, retryCfg, "test-agent/1.0")

	resp, err := f.Fetch(context.Background(), server.URL)
	require.Error(t, err) // Expect an error
	require.NotNil(t, resp, "Response should not be nil even on HTTPError")
	defer resp.Body.Close()

	var httpErr *fetcher.HTTPError
	require.ErrorAs(t, err, &httpErr, "Error should be of type fetcher.HTTPError")
	assert.Equal(t, http.StatusNotFound, httpErr.StatusCode)
	assert.Equal(t, int32(1), atomic.LoadInt32(&requestCount), "Should only make 1 attempt for non-retryable 404")

	// Verify response body can be read (even though it's an error)
	bodyBytes, readErr := io.ReadAll(resp.Body)
	require.NoError(t, readErr)
	assert.True(t, strings.Contains(string(bodyBytes), "Not Found"), "Body should contain error message from server")
}

func TestHTTPFetcher_Fetch_RetryScenarios(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name                 string
		handlerSetup         func(t *testing.T) (http.HandlerFunc, *int32)
		retryCfg             retry.RetryConfig
		expectError          bool
		expectedStatusCode   int // Expected status code on final attempt (success or last error)
		expectedAttempts     int32
		expectedBodyContains string // For successful responses or error bodies
	}

	testTable := []testCase{
		{
			name: "Server error 500, succeeds on 2nd attempt",
			handlerSetup: func(t *testing.T) (http.HandlerFunc, *int32) {
				var attempts int32
				return func(w http.ResponseWriter, r *http.Request) {
					currentAttempt := atomic.AddInt32(&attempts, 1)
					if currentAttempt == 1 {
						http.Error(w, "Internal Server Error", http.StatusInternalServerError)
						return
					}
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte("Success on 2nd try"))
					assert.NoError(t, err)
				}, &attempts
			},
			retryCfg: func() retry.RetryConfig {
				cfg := retry.DefaultRetryConfig
				cfg.MaxRetries = 2
				cfg.InitialBackoff = 1 * time.Millisecond // Fast retries for testing
				cfg.MaxBackoff = 5 * time.Millisecond
				return cfg
			}(),
			expectError:          false,
			expectedStatusCode:   http.StatusOK,
			expectedAttempts:     2,
			expectedBodyContains: "Success on 2nd try",
		},
		{
			name: "Server error 503, succeeds on 3rd attempt",
			handlerSetup: func(t *testing.T) (http.HandlerFunc, *int32) {
				var attempts int32
				return func(w http.ResponseWriter, r *http.Request) {
					currentAttempt := atomic.AddInt32(&attempts, 1)
					if currentAttempt < 3 {
						http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
						return
					}
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte("Finally available"))
					assert.NoError(t, err)
				}, &attempts
			},
			retryCfg: func() retry.RetryConfig {
				cfg := retry.DefaultRetryConfig
				cfg.MaxRetries = 3
				cfg.InitialBackoff = 1 * time.Millisecond
				cfg.MaxBackoff = 5 * time.Millisecond
				return cfg
			}(),
			expectError:          false,
			expectedStatusCode:   http.StatusOK,
			expectedAttempts:     3,
			expectedBodyContains: "Finally available",
		},
		{
			name: "Server error 500, always fails, exhausts retries",
			handlerSetup: func(t *testing.T) (http.HandlerFunc, *int32) {
				var attempts int32
				return func(w http.ResponseWriter, r *http.Request) {
					atomic.AddInt32(&attempts, 1)
					http.Error(w, "Persistent Error", http.StatusInternalServerError)
				}, &attempts
			},
			retryCfg: func() retry.RetryConfig {
				cfg := retry.DefaultRetryConfig
				cfg.MaxRetries = 2
				cfg.InitialBackoff = 1 * time.Millisecond
				cfg.MaxBackoff = 5 * time.Millisecond
				return cfg
			}(),
			expectError:          true,
			expectedStatusCode:   http.StatusInternalServerError, // Last error status
			expectedAttempts:     3,                              // Initial + 2 retries
			expectedBodyContains: "Persistent Error",
		},
		{
			name: "Rate limit 429, succeeds on 2nd attempt",
			handlerSetup: func(t *testing.T) (http.HandlerFunc, *int32) {
				var attempts int32
				return func(w http.ResponseWriter, r *http.Request) {
					currentAttempt := atomic.AddInt32(&attempts, 1)
					if currentAttempt == 1 {
						http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
						return
					}
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte("Rate limit cleared"))
					assert.NoError(t, err)
				}, &attempts
			},
			retryCfg: func() retry.RetryConfig {
				cfg := retry.DefaultRetryConfig
				cfg.MaxRetries = 2
				cfg.InitialBackoff = 1 * time.Millisecond
				cfg.MaxBackoff = 5 * time.Millisecond
				return cfg
			}(),
			expectError:          false,
			expectedStatusCode:   http.StatusOK,
			expectedAttempts:     2,
			expectedBodyContains: "Rate limit cleared",
		},
		{
			name: "Rate limit 429, always fails, exhausts retries",
			handlerSetup: func(t *testing.T) (http.HandlerFunc, *int32) {
				var attempts int32
				return func(w http.ResponseWriter, r *http.Request) {
					atomic.AddInt32(&attempts, 1)
					http.Error(w, "Still rate limited", http.StatusTooManyRequests)
				}, &attempts
			},
			retryCfg: func() retry.RetryConfig {
				cfg := retry.DefaultRetryConfig
				cfg.MaxRetries = 1
				cfg.InitialBackoff = 1 * time.Millisecond
				cfg.MaxBackoff = 5 * time.Millisecond
				return cfg
			}(),
			expectError:          true,
			expectedStatusCode:   http.StatusTooManyRequests,
			expectedAttempts:     2, // Initial + 1 retry
			expectedBodyContains: "Still rate limited",
		},
	}

	for _, tc := range testTable {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			handler, attemptsCounter := tc.handlerSetup(t)
			server := setupTestServer(t, handler)
			defer server.Close()

			f := fetcher.NewHTTPFetcher(server.Client(), tc.retryCfg, "test-agent-retry/1.0")

			resp, err := f.Fetch(context.Background(), server.URL)

			if tc.expectError {
				require.Error(t, err, "Expected an error")

				var httpErr *fetcher.HTTPError
				if errors.As(err, &httpErr) {
					// If the error is an HTTPError, our fetcher's retryableFunc should have returned the response with it.
					require.NotNil(t, resp, "Response should not be nil if error is HTTPError")
					assert.Equal(t, tc.expectedStatusCode, httpErr.StatusCode, "Final status code mismatch for HTTPError")
					if resp != nil { // Should always be true if httpErr is not nil
						defer resp.Body.Close()
					}
				} else {
					// If it's not an HTTPError (e.g. direct network error on last try), resp might be nil.
					t.Logf("Received non-HTTPError: %v", err)
					// The expectedStatusCode might not be directly comparable if it's a network error without a response.
					// However, if a response IS available from the last attempt, check its status.
					if resp != nil {
						defer resp.Body.Close()
						assert.Equal(t, tc.expectedStatusCode, resp.StatusCode, "Final response status code mismatch on non-HTTPError")
					}
				}
			} else {
				require.NoError(t, err, "Did not expect an error")
				require.NotNil(t, resp, "Response should not be nil on success")
				defer resp.Body.Close()
				assert.Equal(t, tc.expectedStatusCode, resp.StatusCode, "Successful status code mismatch")
			}

			assert.Equal(t, tc.expectedAttempts, atomic.LoadInt32(attemptsCounter), "Number of attempts mismatch")

			if resp != nil && resp.Body != nil {
				bodyBytes, readErr := io.ReadAll(resp.Body)
				require.NoError(t, readErr, "Failed to read response body")
				assert.True(t, strings.Contains(string(bodyBytes), tc.expectedBodyContains),
					"Response body does not contain expected string. Body: %s", string(bodyBytes))
			} else if tc.expectedBodyContains != "" {
				t.Errorf("Response or response body was nil, but expected body content: %s", tc.expectedBodyContains)
			}
		})
	}
}

func TestHTTPFetcher_Fetch_NetworkTimeout(t *testing.T) {
	t.Parallel()

	var requestCount int32
	handler := func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		time.Sleep(100 * time.Millisecond) // Sleep longer than client timeout
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Should not reach here"))
	}
	server := setupTestServer(t, handler)
	defer server.Close()

	// Client with a very short timeout
	shortTimeoutClient := &http.Client{
		Timeout: 10 * time.Millisecond,
	}

	retryCfg := retry.DefaultRetryConfig
	retryCfg.MaxRetries = 2
	retryCfg.InitialBackoff = 1 * time.Millisecond
	retryCfg.MaxBackoff = 5 * time.Millisecond
	retryCfg.MaxTotalTimeout = 50 * time.Millisecond // Overall timeout for retries

	f := fetcher.NewHTTPFetcher(shortTimeoutClient, retryCfg, "test-agent-timeout/1.0")

	startTime := time.Now()
	resp, err := f.Fetch(context.Background(), server.URL)
	duration := time.Since(startTime)

	require.Error(t, err, "Expected a timeout error")
	// On timeout with http.Client, resp might be nil or non-nil depending on when timeout occurs.
	// The error is the important part.
	if resp != nil {
		defer resp.Body.Close()
	}

	t.Logf("Error received: %v", err)
	assert.Contains(t, err.Error(), "context deadline exceeded", "Error message should indicate a timeout")
	// It could also be i/o timeout directly from net/http if the retry's own context expires first.

	// MaxRetries is 2, so initial + 2 retries = 3 attempts if all timeout
	// However, MaxTotalTimeout might cut it short.
	// Given the short backoffs and MaxTotalTimeout, it's likely to hit MaxTotalTimeout.
	assert.LessOrEqual(t, atomic.LoadInt32(&requestCount), int32(3), "Should attempt at most MaxRetries + 1 times")
	assert.True(t, duration < 100*time.Millisecond, "Total duration should be less than server sleep due to timeouts")

	// Check if error indicates it's due to MaxTotalTimeout from retry logic
	// This can be tricky as the underlying error from client.Do might also be context.DeadlineExceeded
	if strings.Contains(err.Error(), "exceeded maximum total timeout") {
		t.Log("Retry logic MaxTotalTimeout was exceeded.")
	} else {
		t.Log("Timeout likely occurred within an http.Client.Do call.")
	}
}

func TestHTTPFetcher_Fetch_ContextCancellation(t *testing.T) {
	t.Parallel()

	var requestStarted atomic.Bool
	serverDone := make(chan struct{})

	handler := func(w http.ResponseWriter, r *http.Request) {
		requestStarted.Store(true)
		<-serverDone // Wait until test signals to complete
		w.WriteHeader(http.StatusOK)
	}
	server := setupTestServer(t, handler)
	defer server.Close()
	defer close(serverDone) // Ensure handler can exit

	retryCfg := retry.DefaultRetryConfig
	retryCfg.InitialBackoff = 100 * time.Millisecond // Give some time for cancellation to occur
	retryCfg.MaxRetries = 1

	f := fetcher.NewHTTPFetcher(server.Client(), retryCfg, "test-agent-cancel/1.0")

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		// Wait a moment for the request to potentially start, then cancel
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	resp, err := f.Fetch(ctx, server.URL)

	require.Error(t, err, "Expected an error due to context cancellation")
	if resp != nil {
		defer resp.Body.Close()
	}
	assert.ErrorIs(t, err, context.Canceled, "Error should be context.Canceled")
	// Check that the request didn't just complete successfully before cancellation had effect
	if requestStarted.Load() && resp != nil && resp.StatusCode == http.StatusOK {
		t.Error("Request completed successfully despite context cancellation")
	}
}

func TestHTTPFetcher_Fetch_RetryAfterHeader(t *testing.T) {
	t.Parallel()

	var attempts int32
	var firstAttemptTime, secondAttemptTime time.Time
	serverDone := make(chan struct{}) // To signal handler completion

	handler := func(w http.ResponseWriter, r *http.Request) {
		currentAttempt := atomic.AddInt32(&attempts, 1)
		if currentAttempt == 1 {
			firstAttemptTime = time.Now()
			w.Header().Set("Retry-After", "1") // 1 second
			http.Error(w, "Too Many Requests, try again later", http.StatusTooManyRequests)
			return
		}
		secondAttemptTime = time.Now()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Success after Retry-After"))
		close(serverDone) // Signal that the second request has been processed
	}
	server := setupTestServer(t, handler)
	defer server.Close()

	retryCfg := retry.DefaultRetryConfig
	retryCfg.MaxRetries = 1 // Allow one retry
	// Set very short backoffs to ensure Retry-After is the dominant delay
	retryCfg.InitialBackoff = 1 * time.Millisecond
	retryCfg.MaxBackoff = 5 * time.Millisecond

	f := fetcher.NewHTTPFetcher(server.Client(), retryCfg, "test-agent-retry-after/1.0")

	resp, err := f.Fetch(context.Background(), server.URL)

	// Wait for the server to have processed the second request if it occurred
	select {
	case <-serverDone:
		// Proceed with assertions
	case <-time.After(3 * time.Second): // Timeout for the test guard
		t.Fatal("Test timed out waiting for server to complete second request")
	}

	require.NoError(t, err, "Expected no error after successful retry")
	require.NotNil(t, resp, "Response should not be nil")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status OK on the second attempt")
	assert.Equal(t, int32(2), atomic.LoadInt32(&attempts), "Expected exactly two attempts")

	require.False(t, firstAttemptTime.IsZero(), "First attempt time not recorded")
	require.False(t, secondAttemptTime.IsZero(), "Second attempt time not recorded")

	delay := secondAttemptTime.Sub(firstAttemptTime)
	t.Logf("Measured delay between 1st and 2nd attempt: %v", delay)

	// Check if the delay is approximately 1 second (Retry-After value)
	// Allow some leeway for processing and network overhead.
	assert.GreaterOrEqual(t, delay, time.Second-100*time.Millisecond, "Delay too short, Retry-After likely not respected")
	assert.LessOrEqual(t, delay, time.Second+500*time.Millisecond, "Delay too long, something else might be causing a wait")
}
