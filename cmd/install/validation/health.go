package validation

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/yairfalse/vaino/cmd/install/installer"
)

// HealthChecker performs system health validation
type HealthChecker struct {
	config *installer.Config
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(config *installer.Config) *HealthChecker {
	return &HealthChecker{
		config: config,
	}
}

// CheckSystemHealth performs comprehensive system health checks
func (h *HealthChecker) CheckSystemHealth(ctx context.Context) installer.ValidationResult {
	checks := []installer.ValidationCheck{
		h.checkSystemResources(ctx),
		h.checkNetworkConnectivity(ctx),
		h.checkEnvironmentVariables(ctx),
		h.checkSystemTime(ctx),
		h.checkLocale(ctx),
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
		Summary: fmt.Sprintf("System health: %d/%d checks passed", countSuccessful(checks), len(checks)),
	}
}

func (h *HealthChecker) checkSystemResources(ctx context.Context) installer.ValidationCheck {
	start := time.Now()

	// Get system memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Check available memory (simplified)
	availableMemory := m.Sys - m.Alloc
	requiredMemory := uint64(100 * 1024 * 1024) // 100MB minimum

	success := availableMemory >= requiredMemory

	return installer.ValidationCheck{
		Name:        "System resources",
		Description: "Check system has adequate resources",
		Success:     success,
		Duration:    time.Since(start),
		Metadata: map[string]interface{}{
			"availableMemory": formatBytes(int64(availableMemory)),
			"requiredMemory":  formatBytes(int64(requiredMemory)),
			"numCPU":          runtime.NumCPU(),
			"goVersion":       runtime.Version(),
			"platform":        fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		},
	}
}

func (h *HealthChecker) checkNetworkConnectivity(ctx context.Context) installer.ValidationCheck {
	start := time.Now()

	// Test connectivity to common endpoints
	endpoints := []string{
		"8.8.8.8:53", // Google DNS
		"1.1.1.1:53", // Cloudflare DNS
	}

	var lastErr error
	connected := false

	for _, endpoint := range endpoints {
		conn, err := net.DialTimeout("tcp", endpoint, 5*time.Second)
		if err == nil {
			conn.Close()
			connected = true
			break
		}
		lastErr = err
	}

	// Test HTTP connectivity if configured with mirrors
	if connected && len(h.config.Mirrors) > 0 {
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(h.config.Mirrors[0])
		if err != nil {
			lastErr = err
			connected = false
		} else {
			resp.Body.Close()
		}
	}

	return installer.ValidationCheck{
		Name:        "Network connectivity",
		Description: "Check network connectivity for downloads",
		Success:     connected,
		Error:       lastErr,
		Duration:    time.Since(start),
		Metadata: map[string]interface{}{
			"tested": endpoints,
		},
	}
}

func (h *HealthChecker) checkEnvironmentVariables(ctx context.Context) installer.ValidationCheck {
	start := time.Now()

	// Check critical environment variables
	required := []string{
		"PATH",
		"HOME",
	}

	if runtime.GOOS == "windows" {
		required = append(required, "USERPROFILE", "TEMP")
	} else {
		required = append(required, "USER")
	}

	missing := []string{}
	for _, env := range required {
		if os.Getenv(env) == "" {
			missing = append(missing, env)
		}
	}

	success := len(missing) == 0
	var checkErr error
	if !success {
		checkErr = fmt.Errorf("missing environment variables: %v", missing)
	}

	return installer.ValidationCheck{
		Name:        "Environment variables",
		Description: "Check required environment variables",
		Success:     success,
		Error:       checkErr,
		Duration:    time.Since(start),
		Metadata: map[string]interface{}{
			"required": required,
			"missing":  missing,
		},
	}
}

func (h *HealthChecker) checkSystemTime(ctx context.Context) installer.ValidationCheck {
	start := time.Now()

	// Check if system time seems reasonable
	now := time.Now()
	year := now.Year()

	// Check if year is reasonable (between 2020 and 2030)
	success := year >= 2020 && year <= 2030

	var checkErr error
	if !success {
		checkErr = fmt.Errorf("system time appears incorrect: %v", now)
	}

	return installer.ValidationCheck{
		Name:        "System time",
		Description: "Check system time is reasonable",
		Success:     success,
		Error:       checkErr,
		Duration:    time.Since(start),
		Metadata: map[string]interface{}{
			"systemTime": now.Format(time.RFC3339),
			"timezone":   now.Location().String(),
		},
	}
}

func (h *HealthChecker) checkLocale(ctx context.Context) installer.ValidationCheck {
	start := time.Now()

	// Check locale settings
	locale := os.Getenv("LANG")
	if locale == "" {
		locale = os.Getenv("LC_ALL")
	}

	success := locale != ""

	var checkErr error
	if !success {
		checkErr = fmt.Errorf("no locale configured")
	}

	return installer.ValidationCheck{
		Name:        "Locale settings",
		Description: "Check system locale configuration",
		Success:     success,
		Error:       checkErr,
		Duration:    time.Since(start),
		Metadata: map[string]interface{}{
			"locale": locale,
			"lang":   os.Getenv("LANG"),
			"lc_all": os.Getenv("LC_ALL"),
		},
	}
}

func countSuccessful(checks []installer.ValidationCheck) int {
	count := 0
	for _, check := range checks {
		if check.Success {
			count++
		}
	}
	return count
}
