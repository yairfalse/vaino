package systemd

import (
	"context"
	"runtime"
	"testing"

	"github.com/yairfalse/vaino/internal/collectors"
)

func TestCollectorInterface(t *testing.T) {
	// Skip if not on Linux
	if runtime.GOOS != "linux" {
		t.Skip("systemd collector tests require Linux")
	}

	// This test just ensures the collector implements the interface correctly
	var _ collectors.EnhancedCollector = (*Collector)(nil)
}

func TestCollectorName(t *testing.T) {
	// This test can run on any platform
	c := &Collector{}
	if got := c.Name(); got != "systemd" {
		t.Errorf("Name() = %v, want %v", got, "systemd")
	}
}

func TestCollectorValidate(t *testing.T) {
	c := &Collector{}

	tests := []struct {
		name    string
		config  collectors.CollectorConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: collectors.CollectorConfig{
				Config: map[string]interface{}{
					"filters":    []interface{}{"state:active", "type:service"},
					"rate_limit": 1000,
				},
			},
			wantErr: runtime.GOOS != "linux",
		},
		{
			name: "invalid filter format",
			config: collectors.CollectorConfig{
				Config: map[string]interface{}{
					"filters": []interface{}{"invalid-filter"},
				},
			},
			wantErr: true,
		},
		{
			name: "rate limit too low",
			config: collectors.CollectorConfig{
				Config: map[string]interface{}{
					"rate_limit": 50,
				},
			},
			wantErr: true,
		},
		{
			name: "rate limit too high",
			config: collectors.CollectorConfig{
				Config: map[string]interface{}{
					"rate_limit": 20000,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := c.Validate(tt.config); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSupportedRegions(t *testing.T) {
	c := &Collector{}
	regions := c.SupportedRegions()

	if len(regions) != 1 || regions[0] != "local" {
		t.Errorf("SupportedRegions() = %v, want [local]", regions)
	}
}

func TestAutoDiscover(t *testing.T) {
	c := &Collector{}
	config, err := c.AutoDiscover()

	if runtime.GOOS != "linux" {
		if err == nil {
			t.Error("AutoDiscover() should fail on non-Linux systems")
		}
		return
	}

	// On Linux, it might succeed or fail depending on systemd availability
	if err == nil {
		// Verify default config
		if config.Config["monitor_restarts"] != true {
			t.Error("AutoDiscover() should set monitor_restarts to true")
		}
		if config.Config["rate_limit"] != 1000 {
			t.Error("AutoDiscover() should set rate_limit to 1000")
		}
	}
}

func TestRestartPatternAnalysis(t *testing.T) {
	rp := NewRestartPattern()

	// Test with no restarts
	analysis := rp.GetAnalysis()
	if analysis.Pattern != "stable" {
		t.Errorf("Empty pattern should be stable, got %s", analysis.Pattern)
	}

	// Add some restarts
	now := context.Background()
	_ = now // Use a proper time in real tests
}
