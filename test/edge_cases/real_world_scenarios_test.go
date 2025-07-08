package edgecases

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestCloudProviderMaintenanceWindows tests scenarios during cloud provider maintenance
func TestCloudProviderMaintenanceWindows(t *testing.T) {
	// Mock server simulating various maintenance scenarios
	maintenanceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/maintenance/503":
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Header().Set("Retry-After", "1800") // 30 minutes
			w.Write([]byte(`{"error": "Service temporarily unavailable for maintenance"}`))
		case "/maintenance/502":
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(`{"error": "Bad Gateway - upstream maintenance"}`))
		case "/maintenance/intermittent":
			// Randomly return 503 or 200
			if time.Now().UnixNano()%2 == 0 {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte(`{"error": "Intermittent maintenance"}`))
			} else {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"instances": [{"id": "i-partial"}]}`))
			}
		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"instances": []}`))
		}
	}))
	defer maintenanceServer.Close()

	tests := []struct {
		name           string
		endpoint       string
		expectError    bool
		retryAttempts  int
		description    string
	}{
		{
			name:          "planned_maintenance_503",
			endpoint:      "/maintenance/503",
			expectError:   true,
			retryAttempts: 3,
			description:   "Planned maintenance with clear Retry-After header",
		},
		{
			name:          "gateway_maintenance_502",
			endpoint:      "/maintenance/502",
			expectError:   true,
			retryAttempts: 3,
			description:   "Gateway maintenance affecting API access",
		},
		{
			name:          "intermittent_maintenance",
			endpoint:      "/maintenance/intermittent",
			expectError:   false, // Should eventually succeed
			retryAttempts: 10,
			description:   "Intermittent maintenance with partial availability",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{Timeout: 5 * time.Second}
			
			var lastErr error
			successCount := 0
			
			for attempt := 0; attempt < tt.retryAttempts; attempt++ {
				resp, err := client.Get(maintenanceServer.URL + tt.endpoint)
				if err != nil {
					lastErr = err
					continue
				}
				
				if resp.StatusCode == http.StatusOK {
					successCount++
				} else {
					lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
				}
				resp.Body.Close()
				
				// Brief delay between retries
				time.Sleep(100 * time.Millisecond)
			}
			
			if tt.expectError && successCount == tt.retryAttempts {
				t.Errorf("Expected some failures for %s but all succeeded", tt.description)
			} else if !tt.expectError && successCount == 0 {
				t.Errorf("Expected some successes for %s but all failed. Last error: %v", tt.description, lastErr)
			}
			
			t.Logf("%s: %d/%d requests succeeded", tt.description, successCount, tt.retryAttempts)
		})
	}
}

// TestMassiveInfrastructureScans tests scenarios with very large infrastructure
func TestMassiveInfrastructureScans(t *testing.T) {
	// Mock server returning large amounts of data
	largeDataServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate large response based on query parameters
		sizeParam := r.URL.Query().Get("size")
		size := 100 // default
		
		switch sizeParam {
		case "small":
			size = 10
		case "medium":
			size = 1000
		case "large":
			size = 10000
		case "xlarge":
			size = 50000
		}
		
		instances := make([]map[string]interface{}, size)
		for i := 0; i < size; i++ {
			instances[i] = map[string]interface{}{
				"id":           fmt.Sprintf("i-%08d", i),
				"type":         "t3.micro",
				"state":        "running",
				"launch_time":  time.Now().Add(-time.Duration(i)*time.Hour).Format(time.RFC3339),
				"tags":         map[string]string{"Name": fmt.Sprintf("instance-%d", i)},
				"vpc_id":       fmt.Sprintf("vpc-%08d", i%100),
				"subnet_id":    fmt.Sprintf("subnet-%08d", i%1000),
				"security_groups": []string{fmt.Sprintf("sg-%08d", i%50)},
			}
		}
		
		response := map[string]interface{}{
			"instances": instances,
			"count":     size,
		}
		
		json.NewEncoder(w).Encode(response)
	}))
	defer largeDataServer.Close()

	tests := []struct {
		name        string
		size        string
		timeout     time.Duration
		expectError bool
		description string
	}{
		{
			name:        "small_infrastructure",
			size:        "small",
			timeout:     5 * time.Second,
			expectError: false,
			description: "Small infrastructure scan (10 resources)",
		},
		{
			name:        "medium_infrastructure",
			size:        "medium", 
			timeout:     10 * time.Second,
			expectError: false,
			description: "Medium infrastructure scan (1K resources)",
		},
		{
			name:        "large_infrastructure",
			size:        "large",
			timeout:     30 * time.Second,
			expectError: false,
			description: "Large infrastructure scan (10K resources)",
		},
		{
			name:        "xlarge_infrastructure_short_timeout",
			size:        "xlarge",
			timeout:     5 * time.Second,
			expectError: true,
			description: "XLarge infrastructure with insufficient timeout",
		},
		{
			name:        "xlarge_infrastructure_long_timeout",
			size:        "xlarge",
			timeout:     60 * time.Second,
			expectError: false,
			description: "XLarge infrastructure with sufficient timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()
			
			client := &http.Client{Timeout: tt.timeout}
			
			url := fmt.Sprintf("%s?size=%s", largeDataServer.URL, tt.size)
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			
			start := time.Now()
			resp, err := client.Do(req)
			duration := time.Since(start)
			
			if tt.expectError && err == nil {
				resp.Body.Close()
				t.Errorf("Expected timeout error for %s but request succeeded", tt.description)
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected success for %s but got error: %v", tt.description, err)
			}
			
			if resp != nil {
				resp.Body.Close()
			}
			
			t.Logf("%s completed in %v", tt.description, duration)
		})
	}
}

// TestConcurrentCollectorFailures tests scenarios where multiple collectors fail simultaneously
func TestConcurrentCollectorFailures(t *testing.T) {
	// Mock servers with different failure patterns
	servers := make([]*httptest.Server, 3)
	
	// Server 1: Always fails with 500
	servers[0] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal server error"}`))
	}))
	
	// Server 2: Fails with auth error
	servers[1] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Authentication failed"}`))
	}))
	
	// Server 3: Times out (slow response)
	servers[2] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second) // Will timeout
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"instances": []}`))
	}))
	
	defer func() {
		for _, server := range servers {
			server.Close()
		}
	}()

	t.Run("concurrent_collector_failures", func(t *testing.T) {
		var wg sync.WaitGroup
		errors := make(chan error, len(servers))
		
		// Start concurrent requests to each failing server
		for i, server := range servers {
			wg.Add(1)
			go func(serverIndex int, serverURL string) {
				defer wg.Done()
				
				client := &http.Client{Timeout: 3 * time.Second}
				resp, err := client.Get(serverURL)
				
				if err != nil {
					errors <- fmt.Errorf("server %d: %v", serverIndex, err)
					return
				}
				defer resp.Body.Close()
				
				if resp.StatusCode != http.StatusOK {
					errors <- fmt.Errorf("server %d: HTTP %d", serverIndex, resp.StatusCode)
					return
				}
				
				errors <- nil // Success
			}(i, server.URL)
		}
		
		// Wait for all requests to complete
		wg.Wait()
		close(errors)
		
		// Collect all errors
		var allErrors []error
		for err := range errors {
			if err != nil {
				allErrors = append(allErrors, err)
			}
		}
		
		// We expect all requests to fail in different ways
		if len(allErrors) != len(servers) {
			t.Errorf("Expected %d errors but got %d: %v", len(servers), len(allErrors), allErrors)
		}
		
		t.Logf("Concurrent failures: %v", allErrors)
	})
}

// TestResourcesChangingDuringScan tests scenarios where infrastructure changes during scan
func TestResourcesChangingDuringScan(t *testing.T) {
	resourceCounter := 0
	var mutex sync.Mutex
	
	dynamicServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		resourceCounter++
		currentCount := resourceCounter
		mutex.Unlock()
		
		// Simulate resources being added/removed during scan
		instances := make([]map[string]interface{}, currentCount%10)
		for i := 0; i < len(instances); i++ {
			instances[i] = map[string]interface{}{
				"id":    fmt.Sprintf("dynamic-i-%d-%d", currentCount, i),
				"type":  "t3.micro",
				"state": "running",
			}
		}
		
		response := map[string]interface{}{
			"instances": instances,
			"timestamp": time.Now().Unix(),
			"scan_id":   currentCount,
		}
		
		json.NewEncoder(w).Encode(response)
	}))
	defer dynamicServer.Close()

	t.Run("resources_changing_during_scan", func(t *testing.T) {
		client := &http.Client{Timeout: 5 * time.Second}
		
		// Make multiple requests simulating a scan
		var results []map[string]interface{}
		for i := 0; i < 5; i++ {
			resp, err := client.Get(dynamicServer.URL)
			if err != nil {
				t.Errorf("Request %d failed: %v", i, err)
				continue
			}
			
			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Errorf("Failed to decode response %d: %v", i, err)
				resp.Body.Close()
				continue
			}
			resp.Body.Close()
			
			results = append(results, result)
			
			// Brief delay between requests
			time.Sleep(100 * time.Millisecond)
		}
		
		// Verify that we got different results (resources changed)
		if len(results) < 2 {
			t.Error("Not enough results to compare")
			return
		}
		
		changesDetected := false
		for i := 1; i < len(results); i++ {
			if results[i]["scan_id"] != results[i-1]["scan_id"] {
				changesDetected = true
				break
			}
		}
		
		if !changesDetected {
			t.Error("Expected to detect changes in resources during scan")
		} else {
			t.Log("Successfully detected resources changing during scan")
		}
	})
}

// TestMemoryAndCPUExhaustion tests scenarios under resource pressure
func TestMemoryAndCPUExhaustion(t *testing.T) {
	tests := []struct {
		name        string
		scenario    func(t *testing.T)
		description string
	}{
		{
			name: "large_json_parsing",
			scenario: func(t *testing.T) {
				// Create very large JSON structure
				largeData := make(map[string]interface{})
				for i := 0; i < 100000; i++ {
					largeData[fmt.Sprintf("resource_%d", i)] = map[string]interface{}{
						"id":            fmt.Sprintf("id-%d", i),
						"type":          "large_resource",
						"configuration": strings.Repeat("data", 100),
						"metadata":      make(map[string]string),
					}
				}
				
				// Marshal and unmarshal to simulate real workload
				jsonData, err := json.Marshal(largeData)
				if err != nil {
					t.Errorf("Failed to marshal large data: %v", err)
					return
				}
				
				var parsed map[string]interface{}
				if err := json.Unmarshal(jsonData, &parsed); err != nil {
					t.Errorf("Failed to unmarshal large data: %v", err)
					return
				}
				
				t.Logf("Successfully processed large JSON: %d bytes", len(jsonData))
			},
			description: "Large JSON parsing under memory pressure",
		},
		{
			name: "concurrent_heavy_operations",
			scenario: func(t *testing.T) {
				var wg sync.WaitGroup
				errors := make(chan error, 10)
				
				// Start multiple CPU-intensive operations
				for i := 0; i < 10; i++ {
					wg.Add(1)
					go func(id int) {
						defer wg.Done()
						
						// CPU-intensive task
						data := make([]string, 10000)
						for j := range data {
							data[j] = fmt.Sprintf("heavy-computation-%d-%d", id, j)
						}
						
						// Simulate JSON processing
						jsonData, err := json.Marshal(data)
						if err != nil {
							errors <- fmt.Errorf("goroutine %d marshal error: %v", id, err)
							return
						}
						
						var parsed []string
						if err := json.Unmarshal(jsonData, &parsed); err != nil {
							errors <- fmt.Errorf("goroutine %d unmarshal error: %v", id, err)
							return
						}
						
						errors <- nil
					}(i)
				}
				
				wg.Wait()
				close(errors)
				
				// Check for errors
				var allErrors []error
				for err := range errors {
					if err != nil {
						allErrors = append(allErrors, err)
					}
				}
				
				if len(allErrors) > 0 {
					t.Errorf("Got %d errors during concurrent operations: %v", len(allErrors), allErrors)
				} else {
					t.Log("All concurrent operations completed successfully")
				}
			},
			description: "Concurrent heavy operations under CPU pressure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			tt.scenario(t)
			duration := time.Since(start)
			t.Logf("%s completed in %v", tt.description, duration)
		})
	}
}

// TestCorruptedCloudResponses tests scenarios with malformed API responses
func TestCorruptedCloudResponses(t *testing.T) {
	corruptedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/incomplete-json":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"instances": [{"id": "i-123", "type"`)) // Incomplete JSON
		case "/mixed-encoding":
			w.WriteHeader(http.StatusOK)
			// Mix of valid UTF-8 and invalid bytes
			w.Write([]byte(`{"instances": [{"id": "i-123", "name": "test\xFF\xFE"}]}`))
		case "/unexpected-format":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<xml><instances><instance id="i-123"/></instances></xml>`)) // XML instead of JSON
		case "/empty-response":
			w.WriteHeader(http.StatusOK)
			// Empty response body
		case "/huge-response":
			w.WriteHeader(http.StatusOK)
			// Start valid JSON but send gigabytes of data
			w.Write([]byte(`{"instances": [`))
			for i := 0; i < 1000000; i++ {
				fmt.Fprintf(w, `{"id": "i-%d", "data": "%s"},`, i, strings.Repeat("x", 1000))
			}
			w.Write([]byte(`]}`))
		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"instances": []}`))
		}
	}))
	defer corruptedServer.Close()

	tests := []struct {
		name        string
		endpoint    string
		expectError bool
		description string
	}{
		{
			name:        "incomplete_json",
			endpoint:    "/incomplete-json",
			expectError: true,
			description: "Incomplete JSON response from cloud API",
		},
		{
			name:        "mixed_encoding",
			endpoint:    "/mixed-encoding",
			expectError: true,
			description: "Response with invalid UTF-8 sequences",
		},
		{
			name:        "unexpected_format",
			endpoint:    "/unexpected-format",
			expectError: true,
			description: "XML response when JSON expected",
		},
		{
			name:        "empty_response",
			endpoint:    "/empty-response",
			expectError: true,
			description: "Empty response body",
		},
		{
			name:        "huge_response",
			endpoint:    "/huge-response",
			expectError: false, // Should handle large responses gracefully
			description: "Extremely large response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{Timeout: 10 * time.Second}
			
			resp, err := client.Get(corruptedServer.URL + tt.endpoint)
			if err != nil {
				if !tt.expectError {
					t.Errorf("Unexpected network error for %s: %v", tt.description, err)
				}
				return
			}
			defer resp.Body.Close()
			
			// Try to parse response as JSON
			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected parsing error for %s but got none", tt.description)
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected successful parsing for %s but got: %v", tt.description, err)
			}
			
			if err != nil {
				t.Logf("Got expected error for %s: %v", tt.description, err)
			}
		})
	}
}

// TestMultiRegionFailures tests scenarios where some regions are accessible and others are not
func TestMultiRegionFailures(t *testing.T) {
	regions := []string{"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1"}
	
	// Mock server that simulates region-specific issues
	regionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		region := r.Header.Get("X-Region")
		
		switch region {
		case "us-east-1":
			// Working region
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"instances": [{"id": "i-east-1", "region": "us-east-1"}]}`))
		case "us-west-2":
			// Region with rate limiting
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": "Rate limit exceeded for us-west-2"}`))
		case "eu-west-1":
			// Region with auth issues
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error": "Access denied for eu-west-1"}`))
		case "ap-southeast-1":
			// Region that's down
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error": "Region ap-southeast-1 is temporarily unavailable"}`))
		default:
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "Unknown region"}`))
		}
	}))
	defer regionServer.Close()

	t.Run("multi_region_partial_failures", func(t *testing.T) {
		client := &http.Client{Timeout: 5 * time.Second}
		
		var results []struct {
			region string
			err    error
			data   map[string]interface{}
		}
		
		for _, region := range regions {
			req, err := http.NewRequest("GET", regionServer.URL, nil)
			if err != nil {
				t.Fatalf("Failed to create request for region %s: %v", region, err)
			}
			req.Header.Set("X-Region", region)
			
			resp, err := client.Do(req)
			result := struct {
				region string
				err    error
				data   map[string]interface{}
			}{region: region}
			
			if err != nil {
				result.err = err
			} else {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					var data map[string]interface{}
					if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
						result.err = err
					} else {
						result.data = data
					}
				} else {
					result.err = fmt.Errorf("HTTP %d", resp.StatusCode)
				}
			}
			
			results = append(results, result)
		}
		
		// Analyze results
		successCount := 0
		failureCount := 0
		
		for _, result := range results {
			if result.err == nil {
				successCount++
				t.Logf("Region %s: SUCCESS - %v", result.region, result.data)
			} else {
				failureCount++
				t.Logf("Region %s: FAILED - %v", result.region, result.err)
			}
		}
		
		// We expect mixed results - some regions working, some failing
		if successCount == 0 {
			t.Error("Expected at least one region to succeed")
		}
		if failureCount == 0 {
			t.Error("Expected at least one region to fail")
		}
		
		t.Logf("Multi-region scan: %d successful, %d failed", successCount, failureCount)
	})
}

// TestRealWorldWeirdScenarios tests truly bizarre edge cases that might happen in production
func TestRealWorldWeirdScenarios(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		scenario func(t *testing.T)
	}{
		{
			name: "config_file_replaced_during_read",
			scenario: func(t *testing.T) {
				configFile := filepath.Join(tempDir, "config.yaml")
				
				// Start with one config
				original := `storage:
  base_path: /tmp/original`
				os.WriteFile(configFile, []byte(original), 0644)
				
				// Read file in background
				go func() {
					time.Sleep(50 * time.Millisecond)
					// Replace config during read
					replacement := `storage:
  base_path: /tmp/replaced`
					os.WriteFile(configFile, []byte(replacement), 0644)
				}()
				
				// This might catch the file mid-replacement
				content, err := os.ReadFile(configFile)
				if err != nil {
					t.Errorf("Failed to read config during replacement: %v", err)
				} else {
					t.Logf("Read config during replacement: %s", content)
				}
			},
		},
		{
			name: "system_clock_changes_during_scan",
			scenario: func(t *testing.T) {
				// Simulate time-based operations during clock changes
				timestamps := make([]time.Time, 10)
				
				for i := range timestamps {
					timestamps[i] = time.Now()
					time.Sleep(10 * time.Millisecond)
				}
				
				// Verify timestamps are monotonic
				for i := 1; i < len(timestamps); i++ {
					if timestamps[i].Before(timestamps[i-1]) {
						t.Logf("Non-monotonic time detected: %v -> %v", timestamps[i-1], timestamps[i])
					}
				}
			},
		},
		{
			name: "extremely_deep_directory_structure",
			scenario: func(t *testing.T) {
				// Create deeply nested directory structure
				deepPath := tempDir
				for i := 0; i < 100; i++ {
					deepPath = filepath.Join(deepPath, fmt.Sprintf("level%d", i))
				}
				
				err := os.MkdirAll(deepPath, 0755)
				if err != nil {
					t.Logf("Failed to create extremely deep directory (expected on some systems): %v", err)
				} else {
					// Try to create a file in the deep directory
					deepFile := filepath.Join(deepPath, "deep-config.yaml")
					err = os.WriteFile(deepFile, []byte("test: deep"), 0644)
					if err != nil {
						t.Logf("Failed to write to deep directory: %v", err)
					} else {
						t.Logf("Successfully created file at depth 100: %s", deepFile)
					}
				}
			},
		},
		{
			name: "unicode_nightmare_in_resource_names",
			scenario: func(t *testing.T) {
				// Test various Unicode edge cases
				weirdNames := []string{
					"normal-name",
					"name-with-émojis-🚀",
					"right-to-left-اختبار",
					"zero-width-spaces\u200B\u200C\u200D",
					"combining-characters-é́́́́",
					"surrogate-pairs-𝕳𝖊𝖑𝖑𝖔",
					"normalization-café-vs-cafe\u0301",
					"control-chars-\x7F\x80\x81",
					strings.Repeat("very-long-name-", 1000),
				}
				
				for _, name := range weirdNames {
					// Test JSON marshal/unmarshal with weird names
					data := map[string]interface{}{
						"name": name,
						"id":   "test-id",
					}
					
					jsonData, err := json.Marshal(data)
					if err != nil {
						t.Logf("Failed to marshal name '%s': %v", name, err)
						continue
					}
					
					var parsed map[string]interface{}
					if err := json.Unmarshal(jsonData, &parsed); err != nil {
						t.Logf("Failed to unmarshal name '%s': %v", name, err)
						continue
					}
					
					if parsed["name"] != name {
						t.Logf("Name changed during JSON round-trip: '%s' -> '%s'", name, parsed["name"])
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.scenario(t)
		})
	}
}