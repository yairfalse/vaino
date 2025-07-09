package watcher

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/yairfalse/wgo/internal/analyzer"
	"github.com/yairfalse/wgo/internal/differ"
)

func TestWatcher_SendWebhook(t *testing.T) {
	// Create test server
	var receivedPayload WebhookPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected application/json content type, got %s", r.Header.Get("Content-Type"))
		}

		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		if err != nil {
			t.Errorf("Failed to decode webhook payload: %v", err)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create watcher with webhook URL
	config := WatcherConfig{
		Providers:  []string{"kubernetes"},
		Interval:   10 * time.Second,
		WebhookURL: server.URL,
		Quiet:      true,
	}

	watcher, err := NewWatcher(config)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}

	// Create test event
	event := &WatchEvent{
		Timestamp: time.Now(),
		CorrelatedGroups: []analyzer.ChangeGroup{
			{
				Title:       "Test Scaling",
				Description: "Scaled from 3 to 5 replicas",
				Confidence:  "high",
				Changes: []differ.SimpleChange{
					{
						Type:         "modified",
						ResourceType: "deployment",
						ResourceName: "test-app",
						Namespace:    "default",
						Timestamp:    time.Now(),
					},
				},
			},
		},
		Summary: differ.ChangeSummary{
			Total:    1,
			Modified: 1,
		},
		Source: "wgo-watch",
	}

	// Send webhook
	err = watcher.sendWebhook(event)
	if err != nil {
		t.Errorf("sendWebhook failed: %v", err)
	}

	// Verify received payload
	if receivedPayload.Source != "wgo-watch" {
		t.Errorf("Expected source 'wgo-watch', got '%s'", receivedPayload.Source)
	}

	if receivedPayload.Summary.Total != 1 {
		t.Errorf("Expected total 1, got %d", receivedPayload.Summary.Total)
	}

	if receivedPayload.Summary.Modified != 1 {
		t.Errorf("Expected modified 1, got %d", receivedPayload.Summary.Modified)
	}

	if receivedPayload.Summary.HighConf != 1 {
		t.Errorf("Expected high confidence 1, got %d", receivedPayload.Summary.HighConf)
	}

	if len(receivedPayload.Groups) != 1 {
		t.Errorf("Expected 1 group, got %d", len(receivedPayload.Groups))
	}

	group := receivedPayload.Groups[0]
	if group.Title != "Test Scaling" {
		t.Errorf("Expected title 'Test Scaling', got '%s'", group.Title)
	}

	if group.Confidence != "high" {
		t.Errorf("Expected confidence 'high', got '%s'", group.Confidence)
	}

	if group.ChangeCount != 1 {
		t.Errorf("Expected change count 1, got %d", group.ChangeCount)
	}
}

func TestWatcher_SendWebhook_ServerError(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := WatcherConfig{
		Providers:  []string{"kubernetes"},
		Interval:   10 * time.Second,
		WebhookURL: server.URL,
		Quiet:      true,
	}

	watcher, err := NewWatcher(config)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}

	event := &WatchEvent{
		Timestamp: time.Now(),
		Summary:   differ.ChangeSummary{Total: 1},
		Source:    "test",
	}

	// Should return error for server error
	err = watcher.sendWebhook(event)
	if err == nil {
		t.Error("Expected error for server error response")
	}
}

func TestWatcher_SendWebhook_NoURL(t *testing.T) {
	config := WatcherConfig{
		Providers: []string{"kubernetes"},
		Interval:  10 * time.Second,
		// No webhook URL
		Quiet: true,
	}

	watcher, err := NewWatcher(config)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}

	event := &WatchEvent{
		Timestamp: time.Now(),
		Summary:   differ.ChangeSummary{Total: 1},
		Source:    "test",
	}

	// Should not send webhook if no URL configured
	err = watcher.sendWebhook(event)
	if err != nil {
		t.Errorf("Unexpected error when no webhook URL: %v", err)
	}
}

func TestBuildWebhookPayload(t *testing.T) {
	config := WatcherConfig{
		Providers: []string{"kubernetes", "terraform"},
		Interval:  30 * time.Second,
		Quiet:     true,
	}

	watcher, err := NewWatcher(config)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}

	now := time.Now()
	event := &WatchEvent{
		Timestamp: now,
		CorrelatedGroups: []analyzer.ChangeGroup{
			{
				Title:       "Frontend Scaling",
				Description: "Scaled from 3 to 5 replicas",
				Confidence:  "high",
				Reason:      "Deployment scaling detected",
				Changes: []differ.SimpleChange{
					{
						Type:         "modified",
						ResourceType: "deployment",
						ResourceName: "frontend",
						Namespace:    "web",
						Timestamp:    now,
					},
				},
			},
			{
				Title:       "Config Update",
				Description: "Configuration changed",
				Confidence:  "medium",
				Reason:      "Configuration update",
				Changes: []differ.SimpleChange{
					{
						Type:         "modified",
						ResourceType: "configmap",
						ResourceName: "app-config",
						Namespace:    "default",
						Timestamp:    now,
					},
				},
			},
		},
		RawChanges: []differ.SimpleChange{
			{
				Type:         "added",
				ResourceType: "service",
				ResourceName: "new-service",
				Namespace:    "default",
				Timestamp:    now,
			},
		},
		Summary: differ.ChangeSummary{
			Total:    3,
			Added:    1,
			Modified: 2,
		},
		Source: "wgo-watch",
	}

	payload := watcher.buildWebhookPayload(event)

	// Test basic fields
	if payload.Source != "wgo-watch" {
		t.Errorf("Expected source 'wgo-watch', got '%s'", payload.Source)
	}

	if !payload.Timestamp.Equal(now) {
		t.Errorf("Expected timestamp %v, got %v", now, payload.Timestamp)
	}

	// Test summary
	if payload.Summary.Total != 3 {
		t.Errorf("Expected total 3, got %d", payload.Summary.Total)
	}

	if payload.Summary.Added != 1 {
		t.Errorf("Expected added 1, got %d", payload.Summary.Added)
	}

	if payload.Summary.Modified != 2 {
		t.Errorf("Expected modified 2, got %d", payload.Summary.Modified)
	}

	if payload.Summary.HighConf != 1 {
		t.Errorf("Expected high confidence 1, got %d", payload.Summary.HighConf)
	}

	if payload.Summary.MediumConf != 1 {
		t.Errorf("Expected medium confidence 1, got %d", payload.Summary.MediumConf)
	}

	// Test groups
	if len(payload.Groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(payload.Groups))
	}

	group1 := payload.Groups[0]
	if group1.Title != "Frontend Scaling" {
		t.Errorf("Expected title 'Frontend Scaling', got '%s'", group1.Title)
	}

	if group1.Confidence != "high" {
		t.Errorf("Expected confidence 'high', got '%s'", group1.Confidence)
	}

	if group1.ChangeCount != 1 {
		t.Errorf("Expected change count 1, got %d", group1.ChangeCount)
	}

	if len(group1.Changes) != 1 {
		t.Errorf("Expected 1 change in group, got %d", len(group1.Changes))
	}

	// Test raw changes
	if len(payload.RawChanges) != 1 {
		t.Errorf("Expected 1 raw change, got %d", len(payload.RawChanges))
	}

	rawChange := payload.RawChanges[0]
	if rawChange.Type != "added" {
		t.Errorf("Expected type 'added', got '%s'", rawChange.Type)
	}

	if rawChange.ResourceName != "new-service" {
		t.Errorf("Expected resource name 'new-service', got '%s'", rawChange.ResourceName)
	}

	// Test metadata
	if payload.Metadata.WatchInterval != "30s" {
		t.Errorf("Expected watch interval '30s', got '%s'", payload.Metadata.WatchInterval)
	}

	if len(payload.Metadata.Providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(payload.Metadata.Providers))
	}
}

func TestGetSlackColor(t *testing.T) {
	watcher := &Watcher{}

	tests := []struct {
		name          string
		event         *WatchEvent
		expectedColor string
	}{
		{
			name: "high confidence changes",
			event: &WatchEvent{
				CorrelatedGroups: []analyzer.ChangeGroup{
					{Confidence: "high"},
				},
				Summary: differ.ChangeSummary{Modified: 1},
			},
			expectedColor: "danger",
		},
		{
			name: "removed resources",
			event: &WatchEvent{
				CorrelatedGroups: []analyzer.ChangeGroup{
					{Confidence: "medium"},
				},
				Summary: differ.ChangeSummary{Removed: 1},
			},
			expectedColor: "danger",
		},
		{
			name: "added resources",
			event: &WatchEvent{
				CorrelatedGroups: []analyzer.ChangeGroup{
					{Confidence: "medium"},
				},
				Summary: differ.ChangeSummary{Added: 1},
			},
			expectedColor: "good",
		},
		{
			name: "modified resources only",
			event: &WatchEvent{
				CorrelatedGroups: []analyzer.ChangeGroup{
					{Confidence: "medium"},
				},
				Summary: differ.ChangeSummary{Modified: 1},
			},
			expectedColor: "warning",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			color := watcher.getSlackColor(tt.event)
			if color != tt.expectedColor {
				t.Errorf("Expected color '%s', got '%s'", tt.expectedColor, color)
			}
		})
	}
}

func BenchmarkBuildWebhookPayload(b *testing.B) {
	config := WatcherConfig{
		Providers: []string{"kubernetes"},
		Interval:  10 * time.Second,
		Quiet:     true,
	}

	watcher, err := NewWatcher(config)
	if err != nil {
		b.Fatalf("NewWatcher failed: %v", err)
	}

	// Create large event for benchmarking
	now := time.Now()
	changes := make([]differ.SimpleChange, 100)
	for i := 0; i < 100; i++ {
		changes[i] = differ.SimpleChange{
			Type:         "modified",
			ResourceType: "deployment",
			ResourceName: fmt.Sprintf("app-%d", i),
			Namespace:    "default",
			Timestamp:    now,
		}
	}

	event := &WatchEvent{
		Timestamp: now,
		CorrelatedGroups: []analyzer.ChangeGroup{
			{
				Title:      "Mass Scaling",
				Confidence: "high",
				Changes:    changes,
			},
		},
		RawChanges: changes,
		Summary: differ.ChangeSummary{
			Total:    100,
			Modified: 100,
		},
		Source: "benchmark",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		payload := watcher.buildWebhookPayload(event)
		_ = payload // Prevent optimization
	}
}
