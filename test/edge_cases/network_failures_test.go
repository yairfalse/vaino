package edgecases

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/yairfalse/wgo/internal/collectors"
	"github.com/yairfalse/wgo/internal/collectors/aws"
	wgoerrors "github.com/yairfalse/wgo/internal/errors"
)

// TestNetworkTimeouts tests various network timeout scenarios
func TestNetworkTimeouts(t *testing.T) {
	tests := []struct {
		name        string
		delay       time.Duration
		timeout     time.Duration
		expectError bool
		errorType   string
	}{
		{
			name:        "quick_timeout",
			delay:       5 * time.Second,
			timeout:     1 * time.Second,
			expectError: true,
			errorType:   "timeout",
		},
		{
			name:        "slow_response_within_timeout",
			delay:       2 * time.Second,
			timeout:     5 * time.Second,
			expectError: false,
		},
		{
			name:        "extremely_slow_response",
			delay:       30 * time.Second,
			timeout:     5 * time.Second,
			expectError: true,
			errorType:   "timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server that delays responses
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(tt.delay)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"instances": []}`))
			}))
			defer server.Close()

			// Test with AWS collector (modify endpoint for testing)
			collector := aws.NewAWSCollector()
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			config := collectors.CollectorConfig{
				Config: map[string]interface{}{
					"region":          "us-east-1",
					"custom_endpoint": server.URL, // For testing
				},
			}

			_, err := collector.Collect(ctx, config)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else {
					// Verify it's the right type of error
					if wgoErr, ok := err.(*wgoerrors.WGOError); ok {
						if wgoErr.Type != wgoerrors.ErrorTypeNetwork {
							t.Errorf("Expected network error, got %v", wgoErr.Type)
						}
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestNetworkConnectivityIssues tests various connectivity problems
func TestNetworkConnectivityIssues(t *testing.T) {
	tests := []struct {
		name        string
		endpoint    string
		expectError bool
		errorType   wgoerrors.ErrorType
	}{
		{
			name:        "dns_resolution_failure",
			endpoint:    "https://this-domain-does-not-exist-123456789.com",
			expectError: true,
			errorType:   wgoerrors.ErrorTypeNetwork,
		},
		{
			name:        "connection_refused",
			endpoint:    "http://localhost:99999",
			expectError: true,
			errorType:   wgoerrors.ErrorTypeNetwork,
		},
		{
			name:        "invalid_url",
			endpoint:    "not-a-valid-url",
			expectError: true,
			errorType:   wgoerrors.ErrorTypeValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test network connectivity with various bad endpoints
			client := &http.Client{
				Timeout: 5 * time.Second,
			}

			_, err := client.Get(tt.endpoint)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				// Verify error contains expected information
				if !isNetworkError(err) {
					t.Errorf("Expected network error, got: %v", err)
				}
			}
		})
	}
}

// TestRateLimitScenarios tests handling of API rate limits
func TestRateLimitScenarios(t *testing.T) {
	rateLimitCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rateLimitCount++
		if rateLimitCount <= 3 {
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": "rate limit exceeded"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"instances": []}`))
	}))
	defer server.Close()

	t.Run("rate_limit_handling", func(t *testing.T) {
		// Test that we handle rate limits gracefully
		client := &http.Client{Timeout: 10 * time.Second}

		for i := 0; i < 5; i++ {
			resp, err := client.Get(server.URL)
			if err != nil {
				t.Errorf("Request %d failed: %v", i, err)
				continue
			}

			if i < 3 && resp.StatusCode != http.StatusTooManyRequests {
				t.Errorf("Expected rate limit on request %d, got status %d", i, resp.StatusCode)
			}

			if i >= 3 && resp.StatusCode != http.StatusOK {
				t.Errorf("Expected success on request %d, got status %d", i, resp.StatusCode)
			}

			resp.Body.Close()
		}
	})
}

// TestProxyAndFirewallIssues tests scenarios with proxies and firewalls
func TestProxyAndFirewallIssues(t *testing.T) {
	originalProxy := os.Getenv("HTTP_PROXY")
	defer os.Setenv("HTTP_PROXY", originalProxy)

	tests := []struct {
		name      string
		proxyURL  string
		shouldErr bool
	}{
		{
			name:      "invalid_proxy",
			proxyURL:  "http://invalid-proxy:8080",
			shouldErr: true,
		},
		{
			name:      "malformed_proxy_url",
			proxyURL:  "not-a-url",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("HTTP_PROXY", tt.proxyURL)

			client := &http.Client{Timeout: 5 * time.Second}
			_, err := client.Get("https://httpbin.org/get")

			if tt.shouldErr && err == nil {
				t.Errorf("Expected error with proxy %s but got none", tt.proxyURL)
			}
		})
	}
}

// TestPartialNetworkFailures tests scenarios where some requests succeed and others fail
func TestPartialNetworkFailures(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		// Fail every other request
		if requestCount%2 == 0 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "internal server error"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"instances": [{"id": "i-123"}]}`))
	}))
	defer server.Close()

	t.Run("partial_failures", func(t *testing.T) {
		client := &http.Client{Timeout: 5 * time.Second}

		successCount := 0
		failureCount := 0

		for i := 0; i < 10; i++ {
			resp, err := client.Get(server.URL)
			if err != nil {
				failureCount++
				continue
			}

			if resp.StatusCode == http.StatusOK {
				successCount++
			} else {
				failureCount++
			}
			resp.Body.Close()
		}

		if successCount == 0 {
			t.Error("Expected some successful requests")
		}
		if failureCount == 0 {
			t.Error("Expected some failed requests")
		}

		t.Logf("Success: %d, Failures: %d", successCount, failureCount)
	})
}

// TestConcurrentNetworkFailures tests handling of multiple simultaneous network issues
func TestConcurrentNetworkFailures(t *testing.T) {
	t.Run("concurrent_timeouts", func(t *testing.T) {
		// Create multiple servers with different delay patterns
		servers := make([]*httptest.Server, 3)
		delays := []time.Duration{1 * time.Second, 5 * time.Second, 10 * time.Second}

		for i, delay := range delays {
			d := delay // Capture for closure
			servers[i] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(d)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"success": true}`))
			}))
			defer servers[i].Close()
		}

		// Make concurrent requests with short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		type result struct {
			index int
			err   error
		}

		results := make(chan result, len(servers))

		for i, server := range servers {
			go func(idx int, url string) {
				client := &http.Client{Timeout: 3 * time.Second}
				req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
				_, err := client.Do(req)
				results <- result{index: idx, err: err}
			}(i, server.URL)
		}

		timeoutCount := 0
		successCount := 0

		for i := 0; i < len(servers); i++ {
			res := <-results
			if res.err != nil {
				timeoutCount++
			} else {
				successCount++
			}
		}

		// We expect first server to succeed, others to timeout
		if successCount == 0 {
			t.Error("Expected at least one success")
		}
		if timeoutCount == 0 {
			t.Error("Expected at least one timeout")
		}
	})
}

// Helper function to check if error is network-related
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}

	// Check for common network error types
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout() || netErr.Temporary()
	}

	// Check for DNS errors
	if dnsErr, ok := err.(*net.DNSError); ok {
		return dnsErr.IsNotFound || dnsErr.IsTimeout
	}

	return false
}
