package installer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// DefaultHTTPClient implements HTTPClient with circuit breaker support
type DefaultHTTPClient struct {
	client *http.Client
}

// NewDefaultHTTPClient creates a new HTTP client
func NewDefaultHTTPClient() HTTPClient {
	return &DefaultHTTPClient{
		client: &http.Client{
			Timeout: 5 * time.Minute,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				DisableCompression:  false,
				TLSHandshakeTimeout: 10 * time.Second,
			},
		},
	}
}

// Do executes an HTTP request
func (c *DefaultHTTPClient) Do(ctx context.Context, req *Request) (*Response, error) {
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, req.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// Set range header for resumable downloads
	if req.RangeStart > 0 {
		if req.RangeEnd > 0 {
			httpReq.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", req.RangeStart, req.RangeEnd))
		} else {
			httpReq.Header.Set("Range", fmt.Sprintf("bytes=%d-", req.RangeStart))
		}
	}

	// Execute request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Check status code
	if resp.StatusCode >= 400 {
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Build response
	headers := make(map[string]string)
	for k := range resp.Header {
		headers[k] = resp.Header.Get(k)
	}

	return &Response{
		StatusCode:    resp.StatusCode,
		Headers:       headers,
		Body:          resp.Body,
		ContentLength: resp.ContentLength,
	}, nil
}

// circuitBreaker implements the CircuitBreaker interface
type circuitBreaker struct {
	maxFailures      int32
	failureCount     int32
	successCount     int32
	lastFailureTime  int64 // Unix timestamp
	state            int32 // CircuitState
	halfOpenAttempts int32
	resetTimeout     time.Duration
	mu               sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) CircuitBreaker {
	return &circuitBreaker{
		maxFailures:  int32(maxFailures),
		resetTimeout: resetTimeout,
		state:        int32(CircuitClosed),
	}
}

// Execute runs a function with circuit breaker protection
func (cb *circuitBreaker) Execute(ctx context.Context, fn func() error) error {
	state := cb.getState()

	switch state {
	case CircuitOpen:
		// Check if we should transition to half-open
		lastFailure := atomic.LoadInt64(&cb.lastFailureTime)
		if time.Since(time.Unix(lastFailure, 0)) > cb.resetTimeout {
			cb.setState(CircuitHalfOpen)
			atomic.StoreInt32(&cb.halfOpenAttempts, 0)
		} else {
			return fmt.Errorf("circuit breaker is open")
		}

	case CircuitHalfOpen:
		// Allow limited attempts in half-open state
		attempts := atomic.AddInt32(&cb.halfOpenAttempts, 1)
		if attempts > 3 {
			cb.setState(CircuitOpen)
			return fmt.Errorf("circuit breaker is open (half-open limit exceeded)")
		}
	}

	// Execute the function
	err := fn()

	if err != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}

	return err
}

// State returns the current circuit state
func (cb *circuitBreaker) State() CircuitState {
	return CircuitState(atomic.LoadInt32(&cb.state))
}

// Reset resets the circuit breaker
func (cb *circuitBreaker) Reset() {
	atomic.StoreInt32(&cb.failureCount, 0)
	atomic.StoreInt32(&cb.successCount, 0)
	atomic.StoreInt32(&cb.halfOpenAttempts, 0)
	cb.setState(CircuitClosed)
}

func (cb *circuitBreaker) getState() CircuitState {
	return CircuitState(atomic.LoadInt32(&cb.state))
}

func (cb *circuitBreaker) setState(state CircuitState) {
	atomic.StoreInt32(&cb.state, int32(state))
}

func (cb *circuitBreaker) recordFailure() {
	failures := atomic.AddInt32(&cb.failureCount, 1)
	atomic.StoreInt64(&cb.lastFailureTime, time.Now().Unix())

	if failures >= cb.maxFailures {
		cb.setState(CircuitOpen)
		atomic.StoreInt32(&cb.successCount, 0)
	}
}

func (cb *circuitBreaker) recordSuccess() {
	state := cb.getState()

	if state == CircuitHalfOpen {
		successes := atomic.AddInt32(&cb.successCount, 1)
		if successes >= 3 {
			// Transition back to closed after 3 successful attempts
			cb.Reset()
		}
	} else {
		// Reset failure count on success in closed state
		atomic.StoreInt32(&cb.failureCount, 0)
	}
}

// RetryableHTTPClient wraps an HTTP client with retry logic
type RetryableHTTPClient struct {
	client         HTTPClient
	maxRetries     int
	retryBackoff   time.Duration
	circuitBreaker CircuitBreaker
}

// NewRetryableHTTPClient creates a new retryable HTTP client
func NewRetryableHTTPClient(client HTTPClient, maxRetries int, backoff time.Duration) HTTPClient {
	return &RetryableHTTPClient{
		client:         client,
		maxRetries:     maxRetries,
		retryBackoff:   backoff,
		circuitBreaker: NewCircuitBreaker(5, 1*time.Minute),
	}
}

// Do executes an HTTP request with retry logic
func (c *RetryableHTTPClient) Do(ctx context.Context, req *Request) (*Response, error) {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff with jitter
			backoff := time.Duration(attempt) * c.retryBackoff
			jitter := time.Duration(float64(backoff) * 0.1 * (0.5 - float64(time.Now().UnixNano()%100)/100))
			select {
			case <-time.After(backoff + jitter):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		err := c.circuitBreaker.Execute(ctx, func() error {
			resp, err := c.client.Do(ctx, req)
			if err != nil {
				lastErr = err
				return err
			}

			// Don't retry on success
			lastErr = nil
			return nil
		})

		if err == nil && lastErr == nil {
			return c.client.Do(ctx, req)
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("request failed after %d retries: %w", c.maxRetries, lastErr)
	}
	return nil, fmt.Errorf("circuit breaker prevented request execution")
}
