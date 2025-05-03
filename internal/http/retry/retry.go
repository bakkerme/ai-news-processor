package retry

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// RetryConfig holds configuration for retry behavior
type RetryConfig struct {
	MaxRetries      int           // Maximum number of retry attempts
	InitialBackoff  time.Duration // Initial backoff duration
	MaxBackoff      time.Duration // Maximum backoff duration
	BackoffFactor   float64       // Multiplier for exponential backoff
	MaxTotalTimeout time.Duration // Maximum total time across all retries (0 means no timeout)
}

// DefaultRetryConfig provides sensible default values for retry behavior
var DefaultRetryConfig = RetryConfig{
	MaxRetries:      3,
	InitialBackoff:  1 * time.Second,
	MaxBackoff:      30 * time.Second,
	BackoffFactor:   2.0,
	MaxTotalTimeout: 2 * time.Minute, // Global timeout for all retries
}

// RetryableFunc is a function that can be retried. The function should:
// - Accept a context.Context for cancellation
// - Return a generic type T and an error
// - Be idempotent (safe to execute multiple times)
// - Handle its own internal state management
//
// Example implementation:
//
//	func makeAPICall[T](ctx context.Context) (T, error) {
//	    result, err := client.Call(ctx)
//	    if err != nil {
//	        return zero, fmt.Errorf("api call failed: %w", err)
//	    }
//	    return result, nil
//	}
type RetryableFunc[T any] func(ctx context.Context) (T, error)

// ShouldRetry is a function that determines if a retry should be attempted based on the error.
// It should return:
// - true if the error is transient and the operation should be retried
// - false if the error is permanent and retrying would not help
//
// Example implementation:
//
//	func shouldRetry(err error) bool {
//	    // Check for network errors
//	    var netErr net.Error
//	    if errors.As(err, &netErr) {
//	        return netErr.Temporary()
//	    }
//
//	    // Check for rate limiting
//	    if errors.Is(err, ErrRateLimit) {
//	        return true
//	    }
//
//	    return false
//	}
type ShouldRetry func(err error) bool

// RetryWithBackoff executes the given function with exponential backoff retry logic.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - config: RetryConfig specifying retry behavior (use DefaultRetryConfig for sensible defaults)
//   - fn: The RetryableFunc to execute
//   - shouldRetry: Function determining if an error should trigger a retry
//
// The function implements the following retry strategy:
//  1. Executes the provided function
//  2. If successful (no error), returns immediately
//  3. If error occurs and shouldRetry returns true:
//     - Waits for backoff duration (exponentially increasing)
//     - Retries up to MaxRetries times
//  4. Respects context cancellation and MaxTotalTimeout
//
// Example usage:
//
//	result, err := RetryWithBackoff(
//	    ctx,
//	    DefaultRetryConfig,
//	    func(ctx context.Context) (MyType, error) {
//	        return makeAPICall(ctx)
//	    },
//	    func(err error) bool {
//	        return IsTransientError(err)
//	    },
//	)
func RetryWithBackoff[T any](
	ctx context.Context,
	config RetryConfig,
	fn RetryableFunc[T],
	shouldRetry ShouldRetry,
) (T, error) {
	var zero T
	var lastErr error
	currentBackoff := config.InitialBackoff
	startTime := time.Now()

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Check context cancellation
		if ctx.Err() != nil {
			return zero, ctx.Err()
		}

		// Check total timeout if set
		if config.MaxTotalTimeout > 0 && time.Since(startTime) > config.MaxTotalTimeout {
			if lastErr != nil {
				return zero, fmt.Errorf("exceeded maximum total timeout of %v: %w",
					config.MaxTotalTimeout, lastErr)
			}
			return zero, fmt.Errorf("exceeded maximum total timeout of %v",
				config.MaxTotalTimeout)
		}

		// Execute the retryable function
		result, err := fn(ctx)
		if err == nil {
			return result, nil
		}

		lastErr = err
		if !shouldRetry(err) || attempt == config.MaxRetries {
			break
		}

		// Wait before next attempt
		timer := time.NewTimer(currentBackoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return zero, ctx.Err()
		case <-timer.C:
		}

		// Calculate next backoff
		nextBackoff := time.Duration(float64(currentBackoff) * config.BackoffFactor)
		if nextBackoff > config.MaxBackoff {
			nextBackoff = config.MaxBackoff
		}
		currentBackoff = nextBackoff
	}

	return zero, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// IsRateLimitError checks if the error is a rate limit error
func IsRateLimitError(resp *http.Response) bool {
	return resp != nil && resp.StatusCode == http.StatusTooManyRequests
}

// GetRetryAfterDuration gets the retry after duration from response headers
func GetRetryAfterDuration(resp *http.Response) time.Duration {
	if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
		if seconds, err := time.ParseDuration(retryAfter + "s"); err == nil {
			return seconds
		}
	}
	return 0
}
