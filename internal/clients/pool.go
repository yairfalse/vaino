package clients

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// HTTPClientPool manages a pool of HTTP clients with connection reuse
type HTTPClientPool struct {
	clients    map[string]*http.Client
	mu         sync.RWMutex
	maxConns   int
	timeout    time.Duration
	keepAlive  time.Duration
	maxRetries int
}

// NewHTTPClientPool creates a new HTTP client pool with optimized settings
func NewHTTPClientPool() *HTTPClientPool {
	return &HTTPClientPool{
		clients:    make(map[string]*http.Client),
		maxConns:   100,
		timeout:    30 * time.Second,
		keepAlive:  30 * time.Second,
		maxRetries: 3,
	}
}

// GetClient returns an HTTP client optimized for the specified provider
func (p *HTTPClientPool) GetClient(provider string) *http.Client {
	p.mu.RLock()
	if client, exists := p.clients[provider]; exists {
		p.mu.RUnlock()
		return client
	}
	p.mu.RUnlock()

	// Create new client if not exists
	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if client, exists := p.clients[provider]; exists {
		return client
	}

	// Create optimized transport with connection pooling
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: p.keepAlive,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          p.maxConns,
		MaxIdleConnsPerHost:   10,
		MaxConnsPerHost:       25,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    false,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   p.timeout,
	}

	p.clients[provider] = client
	return client
}

// CircuitBreaker implements circuit breaker pattern for API calls
type CircuitBreaker struct {
	maxFailures  int
	resetTimeout time.Duration
	failures     int
	lastFailTime time.Time
	state        circuitState
	mu           sync.RWMutex
}

type circuitState int

const (
	stateClosed circuitState = iota
	stateOpen
	stateHalfOpen
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        stateClosed,
	}
}

// Call executes the function with circuit breaker protection
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.RLock()
	state := cb.state
	cb.mu.RUnlock()

	if state == stateOpen {
		cb.mu.RLock()
		if time.Since(cb.lastFailTime) > cb.resetTimeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			cb.state = stateHalfOpen
			cb.mu.Unlock()
		} else {
			cb.mu.RUnlock()
			return fmt.Errorf("circuit breaker is open")
		}
	}

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failures++
		cb.lastFailTime = time.Now()

		if cb.failures >= cb.maxFailures {
			cb.state = stateOpen
			return fmt.Errorf("circuit breaker opened after %d failures: %w", cb.failures, err)
		}
		return err
	}

	// Success - reset failures
	if cb.state == stateHalfOpen {
		cb.state = stateClosed
	}
	cb.failures = 0
	return nil
}

// RetryClient wraps HTTP client with retry logic
type RetryClient struct {
	client     *http.Client
	maxRetries int
	backoff    time.Duration
}

// NewRetryClient creates a new client with retry capabilities
func NewRetryClient(client *http.Client, maxRetries int) *RetryClient {
	return &RetryClient{
		client:     client,
		maxRetries: maxRetries,
		backoff:    time.Second,
	}
}

// Do performs an HTTP request with automatic retries
func (rc *RetryClient) Do(req *http.Request) (*http.Response, error) {
	var lastErr error
	backoff := rc.backoff

	for attempt := 0; attempt <= rc.maxRetries; attempt++ {
		if attempt > 0 {
			// Clone request for retry
			reqCopy := req.Clone(req.Context())
			if req.Body != nil {
				// Body needs to be reset for retry
				if req.GetBody != nil {
					body, err := req.GetBody()
					if err != nil {
						return nil, fmt.Errorf("failed to get request body for retry: %w", err)
					}
					reqCopy.Body = body
				}
			}
			req = reqCopy

			// Exponential backoff
			time.Sleep(backoff)
			backoff *= 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
		}

		resp, err := rc.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		// Check if response indicates we should retry
		if resp.StatusCode >= 500 || resp.StatusCode == 429 {
			resp.Body.Close()
			lastErr = fmt.Errorf("server returned status %d", resp.StatusCode)
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", rc.maxRetries, lastErr)
}

// ConnectionManager manages connections across all providers
type ConnectionManager struct {
	pool            *HTTPClientPool
	circuitBreakers map[string]*CircuitBreaker
	mu              sync.RWMutex
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		pool:            NewHTTPClientPool(),
		circuitBreakers: make(map[string]*CircuitBreaker),
	}
}

// GetClientWithBreaker returns an HTTP client with circuit breaker for a provider
func (cm *ConnectionManager) GetClientWithBreaker(provider string) (*http.Client, *CircuitBreaker) {
	client := cm.pool.GetClient(provider)

	cm.mu.RLock()
	breaker, exists := cm.circuitBreakers[provider]
	cm.mu.RUnlock()

	if !exists {
		cm.mu.Lock()
		breaker = NewCircuitBreaker(5, 30*time.Second)
		cm.circuitBreakers[provider] = breaker
		cm.mu.Unlock()
	}

	return client, breaker
}

// ExecuteWithRetry executes a function with retry and circuit breaker
func (cm *ConnectionManager) ExecuteWithRetry(ctx context.Context, provider string, fn func() error) error {
	_, breaker := cm.GetClientWithBreaker(provider)

	return breaker.Call(func() error {
		// Add context timeout
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		errChan := make(chan error, 1)
		go func() {
			errChan <- fn()
		}()

		select {
		case <-ctx.Done():
			return fmt.Errorf("operation timed out: %w", ctx.Err())
		case err := <-errChan:
			return err
		}
	})
}

// RateLimiter implements rate limiting for API calls
type RateLimiter struct {
	tokens     chan struct{}
	ticker     *time.Ticker
	maxTokens  int
	refillRate time.Duration
	stop       chan struct{}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxTokens int, refillRate time.Duration) *RateLimiter {
	rl := &RateLimiter{
		tokens:     make(chan struct{}, maxTokens),
		ticker:     time.NewTicker(refillRate),
		maxTokens:  maxTokens,
		refillRate: refillRate,
		stop:       make(chan struct{}),
	}

	// Fill initial tokens
	for i := 0; i < maxTokens; i++ {
		rl.tokens <- struct{}{}
	}

	// Start refill goroutine
	go rl.refill()

	return rl
}

// refill adds tokens at the specified rate
func (rl *RateLimiter) refill() {
	for {
		select {
		case <-rl.ticker.C:
			select {
			case rl.tokens <- struct{}{}:
				// Token added
			default:
				// Bucket full
			}
		case <-rl.stop:
			rl.ticker.Stop()
			return
		}
	}
}

// Wait blocks until a token is available
func (rl *RateLimiter) Wait(ctx context.Context) error {
	select {
	case <-rl.tokens:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stop stops the rate limiter
func (rl *RateLimiter) Stop() {
	close(rl.stop)
}
