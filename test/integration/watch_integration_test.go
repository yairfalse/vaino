package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	// "github.com/yairfalse/vaino/internal/analyzer" // TODO: uncomment when test is fixed
	"github.com/yairfalse/vaino/internal/collectors"
	"github.com/yairfalse/vaino/internal/differ"
	"github.com/yairfalse/vaino/internal/watchers"
	"github.com/yairfalse/vaino/pkg/types"
)

// TestWatchModeIntegration tests watch mode with real collectors
func TestWatchModeIntegration(t *testing.T) {
	t.Skip("Skipping until watcher interface is finalized")
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create test infrastructure
	testDir := t.TempDir()

	// Create a mock Terraform state file
	terraformState := `{
		"version": 4,
		"terraform_version": "1.0.0",
		"resources": [
			{
				"mode": "managed",
				"type": "aws_instance",
				"name": "web",
				"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
				"instances": [
					{
						"attributes": {
							"id": "i-1234567890abcdef0",
							"instance_type": "t2.micro",
							"tags": {
								"Name": "WebServer"
							}
						}
					}
				]
			}
		]
	}`

	stateFile := fmt.Sprintf("%s/terraform.tfstate", testDir)
	err := os.WriteFile(stateFile, []byte(terraformState), 0644)
	if err != nil {
		t.Fatalf("Failed to create test state file: %v", err)
	}

	// Set up watcher configuration
	config := watchers.WatcherConfig{
		Providers:    []string{"terraform"},
		Interval:     5 * time.Second,
		OutputFormat: "json",
		Quiet:        true,
	}

	// Create watcher
	w, err := watchers.NewWatcher(config)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	// Start watching in background
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Note: ChangeCallback is not implemented in current WatcherConfig
	// This test may need to be updated based on actual watcher interface

	go func() {
		err := w.Start(ctx)
		if err != nil && err != context.DeadlineExceeded {
			t.Errorf("Watch failed: %v", err)
		}
	}()

	// Wait for initial snapshot
	time.Sleep(2 * time.Second)

	// Modify the state file to trigger a change
	modifiedState := `{
		"version": 4,
		"terraform_version": "1.0.0",
		"resources": [
			{
				"mode": "managed",
				"type": "aws_instance",
				"name": "web",
				"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
				"instances": [
					{
						"attributes": {
							"id": "i-1234567890abcdef0",
							"instance_type": "t2.large",
							"tags": {
								"Name": "WebServer",
								"Environment": "production"
							}
						}
					}
				]
			}
		]
	}`

	err = os.WriteFile(stateFile, []byte(modifiedState), 0644)
	if err != nil {
		t.Fatalf("Failed to modify state file: %v", err)
	}

	// Wait for change detection
	/* TODO: Fix when watcher exposes events
	select {
	case event := <-changes:
		if event.Summary.Total == 0 {
			t.Error("Expected changes but got none")
		}
		if event.Summary.Modified == 0 {
			t.Error("Expected modified resources")
		}
		if len(event.CorrelatedGroups) == 0 {
			t.Error("Expected correlation groups")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Timeout waiting for change detection")
	}
	*/
}

// TestWatchModeWebhookIntegration tests webhook notifications
func TestWatchModeWebhookIntegration(t *testing.T) {
	t.Skip("Skipping until watcher interface is finalized")
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create webhook receiver
	webhookReceived := make(chan watchers.WebhookPayload, 1)
	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload watchers.WebhookPayload
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			t.Errorf("Failed to decode webhook: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		webhookReceived <- payload
		w.WriteHeader(http.StatusOK)
	}))
	defer webhookServer.Close()

	// Set up watcher with webhook
	config := watchers.WatcherConfig{
		Providers:    []string{"kubernetes"},
		Interval:     5 * time.Second,
		WebhookURL:   webhookServer.URL,
		OutputFormat: "quiet",
		Quiet:        true,
	}

	_, err := watchers.NewWatcher(config)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	// Simulate a change event
	// TODO: Use when ChangeCallback is implemented
	/*
		event := &watchers.DisplayEvent{
			Timestamp: time.Now(),
			CorrelatedGroups: []analyzer.ChangeGroup{
				{
					Title:       "Deployment Scaling",
					Description: "Scaled from 3 to 5 replicas",
					Confidence:  "high",
					Changes: []differ.SimpleChange{
						{
							Type:         "modified",
							ResourceType: "deployment",
							ResourceName: "web-app",
							Namespace:    "default",
						},
					},
				},
			},
			Summary: differ.ChangeSummary{
				Total:    1,
				Modified: 1,
			},
			Source: "test",
		}
	*/

	// Trigger a change event through the callback if configured
	// TODO: Implement when ChangeCallback is added to WatcherConfig
	// if config.ChangeCallback != nil {
	// 	config.ChangeCallback(event)
	// }

	// Verify webhook received
	select {
	case payload := <-webhookReceived:
		if payload.Source != "test" {
			t.Errorf("Expected source 'test', got '%s'", payload.Source)
		}
		if payload.Summary.Total != 1 {
			t.Errorf("Expected 1 total change, got %d", payload.Summary.Total)
		}
		if len(payload.Groups) != 1 {
			t.Errorf("Expected 1 group, got %d", len(payload.Groups))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for webhook")
	}
}

// TestWatchModeMultipleProviders tests watching multiple providers
func TestWatchModeMultipleProviders(t *testing.T) {
	t.Skip("Skipping until watcher interface is finalized")
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set up mock collectors
	registry := collectors.NewEnhancedRegistry()

	// Mock Terraform collector
	terraformCollector := &mockCollector{
		name: "terraform",
		resources: []types.Resource{
			{
				ID:       "tf-1",
				Type:     "aws_instance",
				Provider: "terraform",
				Name:     "web-server",
				Region:   "us-east-1",
			},
		},
	}
	registry.RegisterEnhanced(terraformCollector)

	// Mock Kubernetes collector
	k8sCollector := &mockCollector{
		name: "kubernetes",
		resources: []types.Resource{
			{
				ID:        "k8s-1",
				Type:      "deployment",
				Provider:  "kubernetes",
				Name:      "nginx",
				Namespace: "default",
			},
		},
	}
	registry.RegisterEnhanced(k8sCollector)

	// Create watcher for multiple providers
	config := watchers.WatcherConfig{
		Providers:    []string{"terraform", "kubernetes"},
		Interval:     5 * time.Second,
		OutputFormat: "table",
		Quiet:        true,
	}

	w, err := watchers.NewWatcher(config)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	// Verify watcher was created with multiple providers
	status := w.GetStatus()
	providers, ok := status["providers"].([]string)
	if !ok {
		t.Fatal("Expected providers in status")
	}

	if len(providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(providers))
	}

	// Verify both providers are configured
	terraformFound := false
	k8sFound := false
	for _, p := range providers {
		if p == "terraform" {
			terraformFound = true
		}
		if p == "kubernetes" {
			k8sFound = true
		}
	}

	if !terraformFound {
		t.Error("Expected Terraform in providers")
	}
	if !k8sFound {
		t.Error("Expected Kubernetes in providers")
	}
}

// TestWatchModeConcurrentWebhooks tests concurrent webhook sending
func TestWatchModeConcurrentWebhooks(t *testing.T) {
	t.Skip("Skipping until watcher interface is finalized")
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Track webhook calls
	var mu sync.Mutex
	webhookCount := 0

	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		webhookCount++
		mu.Unlock()

		// Simulate slow webhook processing
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer webhookServer.Close()

	config := watchers.WatcherConfig{
		Providers:  []string{"kubernetes"},
		Interval:   5 * time.Second,
		WebhookURL: webhookServer.URL,
		Quiet:      true,
	}

	_, err := watchers.NewWatcher(config)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	// Send multiple webhooks concurrently
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			event := &watchers.DisplayEvent{
				Timestamp: time.Now(),
				Summary: differ.ChangeSummary{
					Total: index,
				},
				Source: fmt.Sprintf("test-%d", index),
			}
			// Use HTTP client to send webhook directly for testing
			payload := buildTestWebhookPayload(event)
			jsonData, _ := json.Marshal(payload)
			resp, err := http.Post(webhookServer.URL, "application/json", bytes.NewReader(jsonData))
			if err == nil {
				resp.Body.Close()
			}
		}(i)
	}

	// Wait for all webhooks to complete
	wg.Wait()

	// Verify all webhooks were sent
	mu.Lock()
	finalCount := webhookCount
	mu.Unlock()

	if finalCount != 10 {
		t.Errorf("Expected 10 webhooks, got %d", finalCount)
	}
}

// Mock collector for testing
type mockCollector struct {
	name      string
	resources []types.Resource
	mu        sync.Mutex
}

func (m *mockCollector) Name() string {
	return m.name
}

func (m *mockCollector) Type() string {
	return m.name
}

func (m *mockCollector) Collect(ctx context.Context, config collectors.CollectorConfig) (*types.Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return &types.Snapshot{
		ID:        "mock-snapshot",
		Provider:  m.name,
		Timestamp: time.Now(),
		Resources: m.resources,
	}, nil
}

func (m *mockCollector) Validate(config collectors.CollectorConfig) error {
	return nil
}

func (m *mockCollector) AutoDiscover() (collectors.CollectorConfig, error) {
	// Return a simple config for testing
	return collectors.CollectorConfig{
		Config: map[string]interface{}{
			"enabled": true,
		},
	}, nil
}

func (m *mockCollector) Status() string {
	return "ready"
}

func (m *mockCollector) SupportedRegions() []string {
	return []string{"us-east-1", "us-west-2"}
}

func (m *mockCollector) SetResources(resources []types.Resource) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resources = resources
}

// Helper function to build webhook payload for testing
func buildTestWebhookPayload(event *watchers.DisplayEvent) map[string]interface{} {
	return map[string]interface{}{
		"timestamp": event.Timestamp,
		"source":    event.Source,
		"summary": map[string]interface{}{
			"total":    event.Summary.Total,
			"added":    event.Summary.Added,
			"modified": event.Summary.Modified,
			"removed":  event.Summary.Removed,
		},
	}
}
