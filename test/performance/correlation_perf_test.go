package performance

import (
	"fmt"
	"math/rand"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/yairfalse/wgo/internal/analyzer"
	"github.com/yairfalse/wgo/internal/differ"
	"github.com/yairfalse/wgo/pkg/types"
)

// BenchmarkCorrelationEngine benchmarks the correlation engine with different change set sizes
func BenchmarkCorrelationEngine(b *testing.B) {
	sizes := []int{10, 50, 100, 500, 1000}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("changes_%d", size), func(b *testing.B) {
			changes := generateRandomChanges(size)
			correlator := analyzer.NewCorrelator()
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = correlator.GroupChanges(changes)
			}
		})
	}
}

// BenchmarkCorrelationMemory tests memory usage of correlation engine
func BenchmarkCorrelationMemory(b *testing.B) {
	changes := generateRandomChanges(1000)
	correlator := analyzer.NewCorrelator()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		groups := correlator.GroupChanges(changes)
		_ = groups // Prevent optimization
	}
}

// BenchmarkTimelineFormatting benchmarks timeline formatting
func BenchmarkTimelineFormatting(b *testing.B) {
	sizes := []int{10, 50, 100}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("groups_%d", size), func(b *testing.B) {
			groups := generateRandomGroups(size)
			duration := 1 * time.Hour
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = analyzer.FormatChangeTimeline(groups, duration)
			}
		})
	}
}

// BenchmarkCorrelatedFormatting benchmarks correlated changes formatting
func BenchmarkCorrelatedFormatting(b *testing.B) {
	groups := generateRandomGroups(50)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_ = analyzer.FormatCorrelatedChanges(groups)
	}
}

// BenchmarkSimpleDiffer benchmarks the simple differ
func BenchmarkSimpleDiffer(b *testing.B) {
	sizes := []int{100, 500, 1000, 5000}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("resources_%d", size), func(b *testing.B) {
			from, to := generateTestSnapshots(size)
			differ := differ.NewSimpleDiffer()
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := differ.Compare(from, to)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkEndToEndCorrelation benchmarks the complete correlation workflow
func BenchmarkEndToEndCorrelation(b *testing.B) {
	from, to := generateTestSnapshots(1000)
	simpleDiffer := differ.NewSimpleDiffer()
	correlator := analyzer.NewCorrelator()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Full workflow: diff -> correlate -> format
		report, err := simpleDiffer.Compare(from, to)
		if err != nil {
			b.Fatal(err)
		}
		
		groups := correlator.GroupChanges(report.Changes)
		_ = analyzer.FormatCorrelatedChanges(groups)
	}
}

// TestCorrelationPerformanceRequirements tests performance requirements
func TestCorrelationPerformanceRequirements(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}
	
	tests := []struct {
		name         string
		changeCount  int
		maxDuration  time.Duration
		description  string
	}{
		{
			name:         "small_changes",
			changeCount:  50,
			maxDuration:  10 * time.Millisecond,
			description:  "Small change sets should be very fast",
		},
		{
			name:         "medium_changes",
			changeCount:  500,
			maxDuration:  100 * time.Millisecond,
			description:  "Medium change sets should be fast",
		},
		{
			name:         "large_changes",
			changeCount:  2000,
			maxDuration:  1 * time.Second,
			description:  "Large change sets should complete in reasonable time",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := generateRandomChanges(tt.changeCount)
			correlator := analyzer.NewCorrelator()
			
			start := time.Now()
			groups := correlator.GroupChanges(changes)
			duration := time.Since(start)
			
			if duration > tt.maxDuration {
				t.Errorf("%s: took %v, expected < %v", tt.description, duration, tt.maxDuration)
			}
			
			// Sanity check that correlation produced results
			if len(groups) == 0 {
				t.Error("Correlation produced no groups")
			}
			
			t.Logf("Processed %d changes in %v, produced %d groups", 
				tt.changeCount, duration, len(groups))
		})
	}
}

// TestMemoryUsage tests memory usage doesn't grow unexpectedly
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}
	
	// Test with incrementally larger datasets
	correlator := analyzer.NewCorrelator()
	
	sizes := []int{100, 500, 1000, 2000}
	var prevMemory int64
	
	for _, size := range sizes {
		changes := generateRandomChanges(size)
		
		// Force GC before measurement
		runtime.GC()
		runtime.GC()
		
		var beforeMem runtime.MemStats
		runtime.ReadMemStats(&beforeMem)
		
		groups := correlator.GroupChanges(changes)
		_ = groups
		
		var afterMem runtime.MemStats
		runtime.ReadMemStats(&afterMem)
		
		memUsed := int64(afterMem.HeapAlloc - beforeMem.HeapAlloc)
		
		t.Logf("Size %d: used %d bytes (%d KB)", size, memUsed, memUsed/1024)
		
		// Memory should grow somewhat linearly with input size
		if prevMemory > 0 {
			growth := float64(memUsed) / float64(prevMemory)
			if growth > 10.0 { // More than 10x growth is suspicious
				t.Errorf("Memory usage grew by %0.1fx from %d to %d items", 
					growth, sizes[len(sizes)-2], size)
			}
		}
		
		prevMemory = memUsed
	}
}

// TestCorrelationScaling tests how correlation quality scales with size
func TestCorrelationScaling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping scaling test in short mode")
	}
	
	correlator := analyzer.NewCorrelator()
	
	sizes := []int{50, 200, 1000}
	
	for _, size := range sizes {
		t.Run(fmt.Sprintf("size_%d", size), func(t *testing.T) {
			// Generate changes with known correlation patterns
			changes := generateKnownPatternChanges(size)
			
			start := time.Now()
			groups := correlator.GroupChanges(changes)
			duration := time.Since(start)
			
			// Should find the scaling patterns we embedded
			scalingGroups := 0
			for _, group := range groups {
				if strings.Contains(group.Title, "Scaling") {
					scalingGroups++
				}
			}
			
			expectedScalingGroups := size / 20 // We embed 1 scaling pattern per 20 changes
			
			if scalingGroups < expectedScalingGroups/2 {
				t.Errorf("Found %d scaling groups, expected at least %d", 
					scalingGroups, expectedScalingGroups/2)
			}
			
			t.Logf("Size %d: %v duration, %d groups, %d scaling groups", 
				size, duration, len(groups), scalingGroups)
		})
	}
}

// Helper functions

func generateRandomChanges(count int) []differ.SimpleChange {
	changes := make([]differ.SimpleChange, count)
	now := time.Now()
	
	changeTypes := []string{"added", "modified", "removed"}
	resourceTypes := []string{"deployment", "service", "configmap", "secret", "pod"}
	namespaces := []string{"default", "kube-system", "app-1", "app-2"}
	
	for i := 0; i < count; i++ {
		changeType := changeTypes[rand.Intn(len(changeTypes))]
		resourceType := resourceTypes[rand.Intn(len(resourceTypes))]
		namespace := namespaces[rand.Intn(len(namespaces))]
		
		change := differ.SimpleChange{
			Type:         changeType,
			ResourceID:   fmt.Sprintf("%s/resource-%d", resourceType, i),
			ResourceType: resourceType,
			ResourceName: fmt.Sprintf("resource-%d", i),
			Namespace:    namespace,
			Timestamp:    now.Add(time.Duration(rand.Intn(300)) * time.Second),
		}
		
		if changeType == "modified" {
			change.Details = []differ.SimpleFieldChange{
				{
					Field:    "field-" + fmt.Sprintf("%d", rand.Intn(10)),
					OldValue: fmt.Sprintf("old-%d", rand.Intn(100)),
					NewValue: fmt.Sprintf("new-%d", rand.Intn(100)),
				},
			}
		}
		
		changes[i] = change
	}
	
	return changes
}

func generateKnownPatternChanges(count int) []differ.SimpleChange {
	changes := make([]differ.SimpleChange, 0, count)
	now := time.Now()
	
	// Generate scaling patterns every 20 changes
	for i := 0; i < count; i += 20 {
		appName := fmt.Sprintf("app-%d", i/20)
		
		// Deployment scaling
		deploymentChange := differ.SimpleChange{
			Type:         "modified",
			ResourceID:   fmt.Sprintf("deployment/%s", appName),
			ResourceType: "deployment",
			ResourceName: appName,
			Namespace:    "default",
			Timestamp:    now.Add(time.Duration(i) * time.Second),
			Details: []differ.SimpleFieldChange{
				{Field: "replicas", OldValue: 3, NewValue: 5},
			},
		}
		changes = append(changes, deploymentChange)
		
		// Related pod changes
		for j := 0; j < 2; j++ {
			podChange := differ.SimpleChange{
				Type:         "added",
				ResourceID:   fmt.Sprintf("pod/%s-%d", appName, j),
				ResourceType: "pod",
				ResourceName: fmt.Sprintf("%s-%d", appName, j),
				Namespace:    "default",
				Timestamp:    now.Add(time.Duration(i+j+1) * time.Second),
			}
			changes = append(changes, podChange)
		}
		
		// Fill remaining slots with random changes
		remaining := 20 - 3 // 3 changes used for pattern
		for j := 0; j < remaining; j++ {
			randomChange := differ.SimpleChange{
				Type:         "modified",
				ResourceID:   fmt.Sprintf("configmap/random-%d-%d", i, j),
				ResourceType: "configmap",
				ResourceName: fmt.Sprintf("random-%d-%d", i, j),
				Namespace:    "default",
				Timestamp:    now.Add(time.Duration(i+j+10) * time.Second),
			}
			changes = append(changes, randomChange)
		}
	}
	
	return changes[:count]
}

func generateRandomGroups(count int) []analyzer.ChangeGroup {
	groups := make([]analyzer.ChangeGroup, count)
	now := time.Now()
	
	groupTypes := []string{"Scaling", "Config Update", "Service Deployment", "Other Changes"}
	confidences := []string{"high", "medium", "low"}
	
	for i := 0; i < count; i++ {
		groupType := groupTypes[rand.Intn(len(groupTypes))]
		confidence := confidences[rand.Intn(len(confidences))]
		
		changeCount := rand.Intn(5) + 1
		changes := make([]differ.SimpleChange, changeCount)
		
		for j := 0; j < changeCount; j++ {
			changes[j] = differ.SimpleChange{
				Type:         "modified",
				ResourceID:   fmt.Sprintf("resource-%d-%d", i, j),
				ResourceType: "deployment",
				ResourceName: fmt.Sprintf("resource-%d-%d", i, j),
				Timestamp:    now.Add(time.Duration(i*60) * time.Second),
			}
		}
		
		groups[i] = analyzer.ChangeGroup{
			Timestamp:   now.Add(time.Duration(i*60) * time.Second),
			Title:       fmt.Sprintf("%s %d", groupType, i),
			Description: fmt.Sprintf("Test %s description", groupType),
			Changes:     changes,
			Confidence:  confidence,
			Reason:      fmt.Sprintf("Test reason for %s", groupType),
		}
	}
	
	return groups
}

func generateTestSnapshots(resourceCount int) (*types.Snapshot, *types.Snapshot) {
	now := time.Now()
	
	// Create baseline snapshot
	baselineResources := make([]types.Resource, resourceCount)
	for i := 0; i < resourceCount; i++ {
		baselineResources[i] = types.Resource{
			ID:       fmt.Sprintf("deployment/app-%d", i),
			Type:     "deployment",
			Name:     fmt.Sprintf("app-%d", i),
			Provider: "kubernetes",
			Namespace: fmt.Sprintf("namespace-%d", i%10),
			Configuration: map[string]interface{}{
				"replicas": 3,
				"image":    fmt.Sprintf("app:v1.%d", i),
			},
			Metadata: types.ResourceMetadata{
				Version: fmt.Sprintf("%d", 100+i),
			},
		}
	}
	
	baseline := &types.Snapshot{
		ID:        "baseline-perf",
		Timestamp: now,
		Provider:  "kubernetes",
		Resources: baselineResources,
	}
	
	// Create modified snapshot with 10% of resources changed
	modifiedResources := make([]types.Resource, resourceCount)
	copy(modifiedResources, baselineResources)
	
	changeCount := resourceCount / 10
	for i := 0; i < changeCount; i++ {
		idx := rand.Intn(resourceCount)
		config := modifiedResources[idx].Configuration
		config["replicas"] = 5 // Scale up
		modifiedResources[idx].Metadata.Version = fmt.Sprintf("%d", 200+idx)
	}
	
	modified := &types.Snapshot{
		ID:        "modified-perf",
		Timestamp: now.Add(5 * time.Minute),
		Provider:  "kubernetes",
		Resources: modifiedResources,
	}
	
	return baseline, modified
}