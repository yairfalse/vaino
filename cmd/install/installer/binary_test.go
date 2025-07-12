package installer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestBinaryInstaller_Install(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest/tapio-linux-amd64":
			w.Header().Set("Content-Length", "1024")
			w.WriteHeader(http.StatusOK)
			// Write test binary data
			data := make([]byte, 1024)
			for i := range data {
				data[i] = byte(i % 256)
			}
			w.Write(data)
		case "/latest/tapio-linux-amd64.sha256":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tempDir := t.TempDir()
	config := &Config{
		Method:          "binary",
		InstallDir:      tempDir,
		Version:         "latest",
		Mirrors:         []string{server.URL},
		Timeout:         30 * time.Second,
		RetryAttempts:   3,
		ValidationLevel: "basic",
	}

	installer, err := NewBinaryInstaller(config)
	if err != nil {
		t.Fatalf("Failed to create installer: %v", err)
	}

	ctx := context.Background()
	if err := installer.Install(ctx); err != nil {
		t.Fatalf("Installation failed: %v", err)
	}

	// Verify binary was installed
	binaryPath := filepath.Join(tempDir, "tapio")
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Errorf("Binary not found at expected path: %s", binaryPath)
	}
}

func TestBinaryInstaller_Rollback(t *testing.T) {
	tempDir := t.TempDir()
	config := &Config{
		Method:     "binary",
		InstallDir: tempDir,
		Version:    "latest",
	}

	installer, err := NewBinaryInstaller(config)
	if err != nil {
		t.Fatalf("Failed to create installer: %v", err)
	}

	// Simulate a partial installation
	binaryPath := filepath.Join(tempDir, "tapio")
	if err := os.WriteFile(binaryPath, []byte("test"), 0755); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Perform rollback
	ctx := context.Background()
	if err := installer.Rollback(ctx); err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Verify binary was removed
	if _, err := os.Stat(binaryPath); !os.IsNotExist(err) {
		t.Errorf("Binary still exists after rollback: %s", binaryPath)
	}
}

func TestProgressReader(t *testing.T) {
	data := make([]byte, 1024*1024) // 1MB
	for i := range data {
		data[i] = byte(i % 256)
	}

	installer := &BinaryInstaller{
		progressChan: make(chan Progress, 100),
	}

	pr := &progressReader{
		reader:     bytes.NewReader(data),
		installer:  installer,
		totalBytes: int64(len(data)),
		lastUpdate: time.Now().Add(-1 * time.Second), // Force immediate update
	}

	// Read all data
	buf := make([]byte, 4096)
	totalRead := 0
	progressUpdates := 0

	go func() {
		for range installer.progressChan {
			progressUpdates++
		}
	}()

	for {
		n, err := pr.Read(buf)
		totalRead += n
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	}

	// Wait for progress updates
	time.Sleep(200 * time.Millisecond)
	close(installer.progressChan)

	if totalRead != len(data) {
		t.Errorf("Expected to read %d bytes, got %d", len(data), totalRead)
	}

	if progressUpdates == 0 {
		t.Error("Expected progress updates, got none")
	}
}

func TestCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(3, 100*time.Millisecond)

	// Test successful execution
	err := cb.Execute(context.Background(), func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}

	// Test circuit opening after failures
	failCount := 0
	for i := 0; i < 5; i++ {
		err := cb.Execute(context.Background(), func() error {
			failCount++
			return fmt.Errorf("test error")
		})
		if err == nil {
			t.Error("Expected error, got success")
		}
	}

	// Circuit should be open after 3 failures
	if failCount != 3 {
		t.Errorf("Expected 3 executions before circuit opens, got %d", failCount)
	}

	// Test circuit reset after timeout
	time.Sleep(150 * time.Millisecond)

	err = cb.Execute(context.Background(), func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected success after reset, got error: %v", err)
	}
}

func TestConcurrentDownloads(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent download test in short mode")
	}

	// Create test server with delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Simulate network delay
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test data"))
	}))
	defer server.Close()

	client := NewDefaultHTTPClient()

	// Test concurrent requests
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			req := &Request{
				URL:    server.URL,
				Method: "GET",
			}

			resp, err := client.Do(context.Background(), req)
			if err != nil {
				errors <- err
				return
			}
			resp.Body.Close()
		}()
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent request failed: %v", err)
	}
}

// Benchmarks

func BenchmarkProgressReader(b *testing.B) {
	data := make([]byte, 10*1024*1024) // 10MB
	for i := range data {
		data[i] = byte(i % 256)
	}

	installer := &BinaryInstaller{
		progressChan: make(chan Progress, 100),
	}

	// Drain progress channel
	go func() {
		for range installer.progressChan {
		}
	}()

	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		pr := &progressReader{
			reader:     bytes.NewReader(data),
			installer:  installer,
			totalBytes: int64(len(data)),
		}

		io.Copy(io.Discard, pr)
	}
}

func BenchmarkCircuitBreaker(b *testing.B) {
	cb := NewCircuitBreaker(5, 1*time.Minute)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cb.Execute(context.Background(), func() error {
				return nil
			})
		}
	})
}

func BenchmarkChecksumCalculation(b *testing.B) {
	// Create test file
	tempFile := filepath.Join(b.TempDir(), "test.bin")
	data := make([]byte, 1024*1024) // 1MB
	for i := range data {
		data[i] = byte(i % 256)
	}

	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	verifier := &verifyStep{}

	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		_, err := verifier.calculateChecksum(tempFile)
		if err != nil {
			b.Fatalf("Checksum calculation failed: %v", err)
		}
	}
}

func BenchmarkStateManager(b *testing.B) {
	sm := NewMemoryStateManager()
	state := State{
		ID:        "test-install",
		Method:    "binary",
		Version:   "1.0.0",
		StartTime: time.Now(),
		Steps: []StepResult{
			{StepName: "download", Success: true},
			{StepName: "verify", Success: true},
			{StepName: "install", Success: true},
		},
	}

	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := sm.SaveState(ctx, state); err != nil {
				b.Fatalf("SaveState failed: %v", err)
			}

			if _, err := sm.LoadState(ctx); err != nil {
				b.Fatalf("LoadState failed: %v", err)
			}
		}
	})
}
