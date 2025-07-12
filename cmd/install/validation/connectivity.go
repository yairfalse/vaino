package validation

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/yairfalse/wgo/cmd/install/installer"
)

// ConnectivityChecker validates network connectivity
type ConnectivityChecker struct {
	config  *installer.Config
	mirrors []string
}

// NewConnectivityChecker creates a new connectivity checker
func NewConnectivityChecker(config *installer.Config) *ConnectivityChecker {
	mirrors := config.Mirrors
	if len(mirrors) == 0 {
		// Default mirrors
		mirrors = []string{
			"https://releases.tapio.io",
			"https://github.com/tapio/releases",
		}
	}

	return &ConnectivityChecker{
		config:  config,
		mirrors: mirrors,
	}
}

// CheckConnectivity performs comprehensive connectivity checks
func (c *ConnectivityChecker) CheckConnectivity(ctx context.Context) installer.ValidationResult {
	checks := []installer.ValidationCheck{
		c.checkDNSResolution(ctx),
		c.checkHTTPSConnectivity(ctx),
		c.checkMirrorAccess(ctx),
		c.checkProxyConfiguration(ctx),
		c.checkTLSConfiguration(ctx),
	}

	allSuccess := true
	for _, check := range checks {
		if !check.Success {
			allSuccess = false
		}
	}

	return installer.ValidationResult{
		Success: allSuccess,
		Checks:  checks,
		Summary: fmt.Sprintf("Connectivity: %d/%d checks passed", countSuccessful(checks), len(checks)),
	}
}

func (c *ConnectivityChecker) checkDNSResolution(ctx context.Context) installer.ValidationCheck {
	start := time.Now()

	// Test DNS resolution for mirror domains
	domains := make(map[string][]string)
	var lastErr error
	success := true

	for _, mirror := range c.mirrors {
		u, err := url.Parse(mirror)
		if err != nil {
			continue
		}

		resolver := &net.Resolver{}
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		addrs, err := resolver.LookupHost(ctx, u.Hostname())
		if err != nil {
			lastErr = err
			success = false
		} else {
			domains[u.Hostname()] = addrs
		}
	}

	return installer.ValidationCheck{
		Name:        "DNS resolution",
		Description: "Check DNS resolution for download servers",
		Success:     success,
		Error:       lastErr,
		Duration:    time.Since(start),
		Metadata: map[string]interface{}{
			"resolved": domains,
		},
	}
}

func (c *ConnectivityChecker) checkHTTPSConnectivity(ctx context.Context) installer.ValidationCheck {
	start := time.Now()

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		},
	}

	// Test HTTPS connectivity
	testURL := "https://1.1.1.1"
	req, _ := http.NewRequestWithContext(ctx, "HEAD", testURL, nil)

	resp, err := client.Do(req)
	success := err == nil

	if resp != nil {
		resp.Body.Close()
	}

	return installer.ValidationCheck{
		Name:        "HTTPS connectivity",
		Description: "Check HTTPS connectivity",
		Success:     success,
		Error:       err,
		Duration:    time.Since(start),
		Metadata: map[string]interface{}{
			"testURL": testURL,
		},
	}
}

func (c *ConnectivityChecker) checkMirrorAccess(ctx context.Context) installer.ValidationCheck {
	start := time.Now()

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil // Allow redirects
		},
	}

	accessible := make(map[string]bool)
	var lastErr error
	anySuccess := false

	for _, mirror := range c.mirrors {
		req, err := http.NewRequestWithContext(ctx, "HEAD", mirror, nil)
		if err != nil {
			accessible[mirror] = false
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			accessible[mirror] = false
			lastErr = err
		} else {
			accessible[mirror] = resp.StatusCode < 400
			if accessible[mirror] {
				anySuccess = true
			}
			resp.Body.Close()
		}
	}

	return installer.ValidationCheck{
		Name:        "Mirror access",
		Description: "Check access to download mirrors",
		Success:     anySuccess,
		Error:       lastErr,
		Duration:    time.Since(start),
		Metadata: map[string]interface{}{
			"mirrors": accessible,
		},
	}
}

func (c *ConnectivityChecker) checkProxyConfiguration(ctx context.Context) installer.ValidationCheck {
	start := time.Now()

	// Check proxy environment variables
	httpProxy := os.Getenv("HTTP_PROXY")
	if httpProxy == "" {
		httpProxy = os.Getenv("http_proxy")
	}

	httpsProxy := os.Getenv("HTTPS_PROXY")
	if httpsProxy == "" {
		httpsProxy = os.Getenv("https_proxy")
	}

	noProxy := os.Getenv("NO_PROXY")
	if noProxy == "" {
		noProxy = os.Getenv("no_proxy")
	}

	hasProxy := httpProxy != "" || httpsProxy != ""

	// If proxy is configured, test it
	success := true
	var checkErr error

	if hasProxy {
		proxyURL, err := url.Parse(httpsProxy)
		if err != nil {
			success = false
			checkErr = fmt.Errorf("invalid proxy URL: %w", err)
		} else {
			// Test proxy connectivity
			conn, err := net.DialTimeout("tcp", proxyURL.Host, 5*time.Second)
			if err != nil {
				success = false
				checkErr = fmt.Errorf("cannot connect to proxy: %w", err)
			} else {
				conn.Close()
			}
		}
	}

	return installer.ValidationCheck{
		Name:        "Proxy configuration",
		Description: "Check proxy settings if configured",
		Success:     success,
		Error:       checkErr,
		Duration:    time.Since(start),
		Metadata: map[string]interface{}{
			"http_proxy":  httpProxy,
			"https_proxy": httpsProxy,
			"no_proxy":    noProxy,
			"has_proxy":   hasProxy,
		},
	}
}

func (c *ConnectivityChecker) checkTLSConfiguration(ctx context.Context) installer.ValidationCheck {
	start := time.Now()

	// Check TLS configuration by connecting to a mirror
	if len(c.mirrors) == 0 {
		return installer.ValidationCheck{
			Name:        "TLS configuration",
			Description: "Check TLS/SSL configuration",
			Success:     true,
			Duration:    time.Since(start),
			Metadata: map[string]interface{}{
				"skipped": "no mirrors configured",
			},
		}
	}

	testURL := c.mirrors[0]
	u, err := url.Parse(testURL)
	if err != nil {
		return installer.ValidationCheck{
			Name:        "TLS configuration",
			Description: "Check TLS/SSL configuration",
			Success:     false,
			Error:       err,
			Duration:    time.Since(start),
		}
	}

	// Connect and check TLS
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 5 * time.Second}, "tcp", u.Host+":443", &tls.Config{
		ServerName: u.Hostname(),
		MinVersion: tls.VersionTLS12,
	})

	if err != nil {
		return installer.ValidationCheck{
			Name:        "TLS configuration",
			Description: "Check TLS/SSL configuration",
			Success:     false,
			Error:       err,
			Duration:    time.Since(start),
		}
	}
	defer conn.Close()

	state := conn.ConnectionState()

	return installer.ValidationCheck{
		Name:        "TLS configuration",
		Description: "Check TLS/SSL configuration",
		Success:     true,
		Duration:    time.Since(start),
		Metadata: map[string]interface{}{
			"server":      u.Host,
			"tlsVersion":  fmt.Sprintf("0x%04x", state.Version),
			"cipherSuite": fmt.Sprintf("0x%04x", state.CipherSuite),
			"negotiated":  state.NegotiatedProtocol,
		},
	}
}
