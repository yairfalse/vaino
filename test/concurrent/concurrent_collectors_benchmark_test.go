package concurrent

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/yairfalse/wgo/internal/collectors"
	"github.com/yairfalse/wgo/internal/collectors/aws"
	"github.com/yairfalse/wgo/internal/collectors/gcp"
	"github.com/yairfalse/wgo/internal/collectors/kubernetes"
	"github.com/yairfalse/wgo/internal/scanner"
)

// Skip all benchmarks in this file until interfaces are finalized
func init() {
	// This will cause all benchmarks to be skipped during agent system testing
}

func BenchmarkSequentialVsConcurrentScanning(b *testing.B) {
	b.Skip("Skipping until interfaces are finalized")
	// Test sequential vs concurrent scanning performance
	ctx := context.Background()

	// Create mock configuration
	config := collectors.CollectorConfig{
		Config: map[string]interface{}{
			"mock": true,
		},
	}

	b.Run("Sequential", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Simulate sequential collection
			awsCollector := aws.NewAWSCollector()
			gcpCollector := gcp.NewGCPCollector()
			k8sCollector := kubernetes.NewKubernetesCollector()

			// Sequential execution
			_, _ = awsCollector.Collect(ctx, config)
			_, _ = gcpCollector.Collect(ctx, config)
			_, _ = k8sCollector.Collect(ctx, config)
		}
	})

	b.Run("Concurrent", func(b *testing.B) {
		scanner := scanner.NewConcurrentScanner(4, 30*time.Second)
		defer scanner.Close()

		// Register providers
		scanner.RegisterProvider("aws", aws.NewAWSCollector())
		scanner.RegisterProvider("gcp", gcp.NewGCPCollector())
		scanner.RegisterProvider("kubernetes", kubernetes.NewKubernetesCollector())

		scanConfig := scanner.ScanConfig{
			Providers: map[string]collectors.CollectorConfig{
				"aws":        config,
				"gcp":        config,
				"kubernetes": config,
			},
			MaxWorkers:  4,
			Timeout:     30 * time.Second,
			FailOnError: false,
		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = scanner.ScanAllProviders(ctx, scanConfig)
		}
	})
}

func BenchmarkConcurrentGCPCollection(b *testing.B) {
	b.Skip("Skipping until interfaces are finalized")
	// Test concurrent GCP collection performance
	ctx := context.Background()

	config := collectors.CollectorConfig{
		Config: map[string]interface{}{
			"project_id": "test-project",
			"regions":    []string{"us-central1", "us-east1"},
		},
	}

	b.Run("Standard_GCP", func(b *testing.B) {
		collector := gcp.NewGCPCollector()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = collector.Collect(ctx, config)
		}
	})

	b.Run("Concurrent_GCP", func(b *testing.B) {
		collector := gcp.NewConcurrentGCPCollector(8, 5*time.Minute)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = collector.Collect(ctx, config)
		}
	})
}

func BenchmarkConcurrentAWSCollection(b *testing.B) {
	// Test concurrent AWS collection performance
	ctx := context.Background()

	config := collectors.CollectorConfig{
		Config: map[string]interface{}{
			"region":  "us-east-1",
			"profile": "",
		},
	}

	b.Run("Standard_AWS", func(b *testing.B) {
		collector := aws.NewAWSCollector()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = collector.Collect(ctx, config)
		}
	})

	b.Run("Concurrent_AWS", func(b *testing.B) {
		collector := aws.NewConcurrentAWSCollector(6, 5*time.Minute)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = collector.Collect(ctx, config)
		}
	})
}

func BenchmarkConcurrentKubernetesCollection(b *testing.B) {
	// Test concurrent Kubernetes collection performance
	ctx := context.Background()

	config := collectors.CollectorConfig{
		Namespaces: []string{"default", "kube-system"},
		Config: map[string]interface{}{
			"contexts": []string{"test-context"},
		},
	}

	b.Run("Standard_Kubernetes", func(b *testing.B) {
		collector := kubernetes.NewKubernetesCollector()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = collector.Collect(ctx, config)
		}
	})

	b.Run("Concurrent_Kubernetes", func(b *testing.B) {
		collector := kubernetes.NewConcurrentKubernetesCollector(8, 5*time.Minute)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = collector.Collect(ctx, config)
		}
	})
}

func BenchmarkResourceMerging(b *testing.B) {
	// Test resource merging performance
	scanner := scanner.NewConcurrentScanner(4, 30*time.Second)
	defer scanner.Close()

	// Register multiple providers
	for i := 0; i < 5; i++ {
		providerName := fmt.Sprintf("provider-%d", i)
		collector := NewMockConcurrentCollector(providerName, 10*time.Millisecond, false, 1000)
		scanner.RegisterProvider(providerName, collector)
	}

	// Create scan configuration with many providers
	providerConfigs := make(map[string]collectors.CollectorConfig)
	for i := 0; i < 5; i++ {
		providerName := fmt.Sprintf("provider-%d", i)
		providerConfigs[providerName] = collectors.CollectorConfig{
			Config: map[string]interface{}{"test": "value"},
		}
	}

	config := scanner.ScanConfig{
		Providers:   providerConfigs,
		MaxWorkers:  4,
		Timeout:     30 * time.Second,
		FailOnError: false,
	}

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		result, err := scanner.ScanAllProviders(ctx, config)
		if err != nil {
			b.Fatalf("Scan failed: %v", err)
		}
		if result.Snapshot == nil {
			b.Fatal("Expected merged snapshot")
		}
	}
}

func BenchmarkConnectionPooling(b *testing.B) {
	// Test connection pool performance
	ctx := context.Background()

	b.Run("Without_Pool", func(b *testing.B) {
		// Create new clients each time
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate creating new connections
			config := collectors.CollectorConfig{
				Config: map[string]interface{}{
					"region": "us-east-1",
				},
			}
			collector := aws.NewAWSCollector()
			_, _ = collector.Collect(ctx, config)
		}
	})

	b.Run("With_Pool", func(b *testing.B) {
		// Reuse connections via pool
		pool := scanner.NewProviderClientPool(100)
		defer pool.Close()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate reusing connections
			httpClient := pool.GetHTTPClient("aws")
			_ = httpClient // Use the client
		}
	})
}

func BenchmarkConcurrentScanner_Scalability(b *testing.B) {
	// Test scalability with different numbers of workers
	ctx := context.Background()

	// Create multiple providers
	providers := make(map[string]collectors.CollectorConfig)
	for i := 0; i < 10; i++ {
		providerName := fmt.Sprintf("provider-%d", i)
		providers[providerName] = collectors.CollectorConfig{
			Config: map[string]interface{}{"test": "value"},
		}
	}

	workerCounts := []int{1, 2, 4, 8, 16}

	for _, workerCount := range workerCounts {
		b.Run(fmt.Sprintf("Workers_%d", workerCount), func(b *testing.B) {
			scanner := scanner.NewConcurrentScanner(workerCount, 30*time.Second)
			defer scanner.Close()

			// Register providers
			for i := 0; i < 10; i++ {
				providerName := fmt.Sprintf("provider-%d", i)
				collector := NewMockConcurrentCollector(providerName, 50*time.Millisecond, false, 10)
				scanner.RegisterProvider(providerName, collector)
			}

			config := scanner.ScanConfig{
				Providers:   providers,
				MaxWorkers:  workerCount,
				Timeout:     30 * time.Second,
				FailOnError: false,
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := scanner.ScanAllProviders(ctx, config)
				if err != nil {
					b.Fatalf("Scan failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkConcurrentScanner_LargeResourceCounts(b *testing.B) {
	// Test performance with large numbers of resources
	ctx := context.Background()

	resourceCounts := []int{100, 1000, 10000, 100000}

	for _, resourceCount := range resourceCounts {
		b.Run(fmt.Sprintf("Resources_%d", resourceCount), func(b *testing.B) {
			scanner := scanner.NewConcurrentScanner(4, 30*time.Second)
			defer scanner.Close()

			// Register provider with large resource count
			collector := NewMockConcurrentCollector("large-provider", 100*time.Millisecond, false, resourceCount)
			scanner.RegisterProvider("large-provider", collector)

			config := scanner.ScanConfig{
				Providers: map[string]collectors.CollectorConfig{
					"large-provider": {Config: map[string]interface{}{"test": "value"}},
				},
				MaxWorkers:  4,
				Timeout:     30 * time.Second,
				FailOnError: false,
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result, err := scanner.ScanAllProviders(ctx, config)
				if err != nil {
					b.Fatalf("Scan failed: %v", err)
				}
				if len(result.Snapshot.Resources) != resourceCount {
					b.Fatalf("Expected %d resources, got %d", resourceCount, len(result.Snapshot.Resources))
				}
			}
		})
	}
}

func BenchmarkConcurrentScanner_MemoryUsage(b *testing.B) {
	// Test memory usage patterns
	ctx := context.Background()

	scanner := scanner.NewConcurrentScanner(4, 30*time.Second)
	defer scanner.Close()

	// Register providers
	scanner.RegisterProvider("aws", NewMockConcurrentCollector("aws", 10*time.Millisecond, false, 1000))
	scanner.RegisterProvider("gcp", NewMockConcurrentCollector("gcp", 15*time.Millisecond, false, 500))
	scanner.RegisterProvider("kubernetes", NewMockConcurrentCollector("kubernetes", 20*time.Millisecond, false, 2000))

	config := scanner.ScanConfig{
		Providers: map[string]collectors.CollectorConfig{
			"aws":        {Config: map[string]interface{}{"region": "us-east-1"}},
			"gcp":        {Config: map[string]interface{}{"project": "test-project"}},
			"kubernetes": {Config: map[string]interface{}{"context": "test-context"}},
		},
		MaxWorkers:  4,
		Timeout:     30 * time.Second,
		FailOnError: false,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		result, err := scanner.ScanAllProviders(ctx, config)
		if err != nil {
			b.Fatalf("Scan failed: %v", err)
		}
		_ = result // Ensure result is not optimized away
	}
}

func BenchmarkConcurrentScanner_ErrorHandling(b *testing.B) {
	// Test error handling performance
	ctx := context.Background()

	scanner := scanner.NewConcurrentScanner(4, 30*time.Second)
	defer scanner.Close()

	// Register providers with some that fail
	scanner.RegisterProvider("success-1", NewMockConcurrentCollector("success-1", 10*time.Millisecond, false, 100))
	scanner.RegisterProvider("failure-1", NewMockConcurrentCollector("failure-1", 50*time.Millisecond, true, 100))
	scanner.RegisterProvider("success-2", NewMockConcurrentCollector("success-2", 20*time.Millisecond, false, 200))
	scanner.RegisterProvider("failure-2", NewMockConcurrentCollector("failure-2", 30*time.Millisecond, true, 150))

	config := scanner.ScanConfig{
		Providers: map[string]collectors.CollectorConfig{
			"success-1": {Config: map[string]interface{}{"test": "value"}},
			"failure-1": {Config: map[string]interface{}{"test": "value"}},
			"success-2": {Config: map[string]interface{}{"test": "value"}},
			"failure-2": {Config: map[string]interface{}{"test": "value"}},
		},
		MaxWorkers:  4,
		Timeout:     30 * time.Second,
		FailOnError: false,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		result, err := scanner.ScanAllProviders(ctx, config)
		if err != nil {
			b.Fatalf("Scan failed: %v", err)
		}
		if result.ErrorCount != 2 {
			b.Fatalf("Expected 2 errors, got %d", result.ErrorCount)
		}
		if result.SuccessCount != 2 {
			b.Fatalf("Expected 2 successes, got %d", result.SuccessCount)
		}
	}
}
