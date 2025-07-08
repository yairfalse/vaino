package performance

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/yairfalse/wgo/internal/collectors"
	"github.com/yairfalse/wgo/internal/collectors/terraform"
	"github.com/yairfalse/wgo/internal/differ"
	"github.com/yairfalse/wgo/internal/storage"
	"github.com/yairfalse/wgo/pkg/types"
)

// MemorySnapshot represents memory usage at a point in time
type MemorySnapshot struct {
	Timestamp    time.Time
	HeapAlloc    uint64
	HeapSys      uint64
	HeapIdle     uint64
	HeapInuse    uint64
	StackInuse   uint64
	GCCycles     uint32
	Operation    string
	ResourceCount int
}

// MemoryProfiler tracks memory usage during operations
type MemoryProfiler struct {
	snapshots []MemorySnapshot
	startTime time.Time
}

func NewMemoryProfiler() *MemoryProfiler {
	return &MemoryProfiler{
		snapshots: make([]MemorySnapshot, 0),
		startTime: time.Now(),
	}
}

func (mp *MemoryProfiler) TakeSnapshot(operation string, resourceCount int) {
	runtime.GC() // Force GC for accurate measurement
	runtime.GC() // Double GC to ensure cleanup
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	snapshot := MemorySnapshot{
		Timestamp:     time.Now(),
		HeapAlloc:     m.HeapAlloc,
		HeapSys:       m.HeapSys,
		HeapIdle:      m.HeapIdle,
		HeapInuse:     m.HeapInuse,
		StackInuse:    m.StackInuse,
		GCCycles:      m.NumGC,
		Operation:     operation,
		ResourceCount: resourceCount,
	}
	
	mp.snapshots = append(mp.snapshots, snapshot)
}

func (mp *MemoryProfiler) GetReport() string {
	if len(mp.snapshots) == 0 {
		return "No memory snapshots recorded"
	}
	
	report := "Memory Usage Analysis Report\n"
	report += "============================\n\n"
	
	for i, snapshot := range mp.snapshots {
		duration := snapshot.Timestamp.Sub(mp.startTime)
		report += fmt.Sprintf("Snapshot %d: %s (at %v)\n", i+1, snapshot.Operation, duration)
		report += fmt.Sprintf("  Heap Allocated: %.2f MB\n", float64(snapshot.HeapAlloc)/(1024*1024))
		report += fmt.Sprintf("  Heap System: %.2f MB\n", float64(snapshot.HeapSys)/(1024*1024))
		report += fmt.Sprintf("  Heap In Use: %.2f MB\n", float64(snapshot.HeapInuse)/(1024*1024))
		report += fmt.Sprintf("  Stack In Use: %.2f MB\n", float64(snapshot.StackInuse)/(1024*1024))
		report += fmt.Sprintf("  GC Cycles: %d\n", snapshot.GCCycles)
		if snapshot.ResourceCount > 0 {
			report += fmt.Sprintf("  Resource Count: %d\n", snapshot.ResourceCount)
			report += fmt.Sprintf("  Memory per Resource: %.2f KB\n", float64(snapshot.HeapAlloc)/(1024*float64(snapshot.ResourceCount)))
		}
		report += "\n"
	}
	
	// Memory growth analysis
	if len(mp.snapshots) > 1 {
		report += "Memory Growth Analysis:\n"
		baseline := mp.snapshots[0]
		peak := mp.snapshots[0]
		
		for _, snapshot := range mp.snapshots[1:] {
			if snapshot.HeapAlloc > peak.HeapAlloc {
				peak = snapshot
			}
		}
		
		growth := float64(peak.HeapAlloc-baseline.HeapAlloc) / (1024 * 1024)
		report += fmt.Sprintf("  Peak memory growth: %.2f MB\n", growth)
		report += fmt.Sprintf("  Peak operation: %s\n", peak.Operation)
		
		// Check for memory leaks
		final := mp.snapshots[len(mp.snapshots)-1]
		retained := float64(final.HeapAlloc-baseline.HeapAlloc) / (1024 * 1024)
		report += fmt.Sprintf("  Retained memory: %.2f MB\n", retained)
		
		if retained > growth*0.5 {
			report += "  WARNING: Potential memory leak detected!\n"
		}
	}
	
	return report
}

// TestMemoryUsagePatterns analyzes memory usage patterns during different operations
func TestMemoryUsagePatterns(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory usage test in short mode")
	}

	profiler := NewMemoryProfiler()
	tmpDir := t.TempDir()
	
	profiler.TakeSnapshot("baseline", 0)

	// Test 1: Collection memory pattern
	t.Log("Testing collection memory patterns...")
	
	resourceCounts := []int{1000, 5000, 10000, 25000}
	for _, resourceCount := range resourceCounts {
		stateFile := filepath.Join(tmpDir, fmt.Sprintf("memory-test-%d.tfstate", resourceCount))
		
		state := createMegaTestState(resourceCount)
		data, err := json.MarshalIndent(state, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal state: %v", err)
		}
		
		if err := os.WriteFile(stateFile, data, 0644); err != nil {
			t.Fatalf("Failed to write state file: %v", err)
		}
		
		collector := terraform.NewTerraformCollector()
		config := collectors.CollectorConfig{
			StatePaths: []string{stateFile},
			Tags:       map[string]string{"memory-test": "true"},
		}
		
		_, err = collector.Collect(context.Background(), config)
		if err != nil {
			t.Fatalf("Collection failed: %v", err)
		}
		
		profiler.TakeSnapshot(fmt.Sprintf("collection_%d", resourceCount), resourceCount)
	}

	// Test 2: Diff memory pattern
	t.Log("Testing diff memory patterns...")
	
	baseline := createLargeSnapshot("baseline", 10000)
	modified := createLargeSnapshotWithChanges("modified", 10000, 5)
	
	profiler.TakeSnapshot("before_diff", 10000)
	
	differ := differ.NewSimpleDiffer()
	_, err := differ.Compare(baseline, modified)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}
	
	profiler.TakeSnapshot("after_diff", 10000)

	// Test 3: Storage memory pattern
	t.Log("Testing storage memory patterns...")
	
	storageConfig := storage.Config{BaseDir: tmpDir}
	storageEngine, err := storage.NewLocalStorage(storageConfig)
	if err != nil {
		t.Fatalf("Failed to create storage engine: %v", err)
	}
	
	for i := 0; i < 5; i++ {
		snapshot := createLargeSnapshot(fmt.Sprintf("storage-test-%d", i), 2000)
		err := storageEngine.SaveSnapshot(snapshot)
		if err != nil {
			t.Fatalf("Storage failed: %v", err)
		}
	}
	
	profiler.TakeSnapshot("after_storage", 10000)

	// Generate and log report
	report := profiler.GetReport()
	t.Log("\n" + report)
	
	// Write detailed report to file
	reportFile := filepath.Join(tmpDir, "memory-analysis-report.txt")
	if err := os.WriteFile(reportFile, []byte(report), 0644); err != nil {
		t.Logf("Warning: Could not write memory report: %v", err)
	} else {
		t.Logf("Detailed memory report written to: %s", reportFile)
	}
}

// TestMemoryLeakDetection tests for memory leaks during repeated operations
func TestMemoryLeakDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	profiler := NewMemoryProfiler()
	tmpDir := t.TempDir()
	
	profiler.TakeSnapshot("baseline", 0)

	// Create test data
	stateFile := filepath.Join(tmpDir, "leak-test.tfstate")
	state := createMegaTestState(5000)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal state: %v", err)
	}
	
	if err := os.WriteFile(stateFile, data, 0644); err != nil {
		t.Fatalf("Failed to write state file: %v", err)
	}

	// Perform repeated operations to detect leaks
	iterations := 50
	t.Logf("Performing %d iterations to detect memory leaks...", iterations)

	for i := 0; i < iterations; i++ {
		// Collection operation
		collector := terraform.NewTerraformCollector()
		config := collectors.CollectorConfig{
			StatePaths: []string{stateFile},
			Tags:       map[string]string{"iteration": fmt.Sprintf("%d", i)},
		}
		
		snapshot, err := collector.Collect(context.Background(), config)
		if err != nil {
			t.Fatalf("Collection failed at iteration %d: %v", i, err)
		}
		
		// Diff operation
		baseline := createLargeSnapshot("baseline", 5000)
		differ := differ.NewSimpleDiffer()
		_, err = differ.Compare(baseline, snapshot)
		if err != nil {
			t.Fatalf("Diff failed at iteration %d: %v", i, err)
		}
		
		// Take memory snapshots every 10 iterations
		if i%10 == 0 {
			profiler.TakeSnapshot(fmt.Sprintf("iteration_%d", i), 5000)
		}
	}

	profiler.TakeSnapshot("final", 5000)

	// Analyze for memory leaks
	report := profiler.GetReport()
	t.Log("\n" + report)

	// Check memory growth pattern
	snapshots := profiler.snapshots
	if len(snapshots) < 3 {
		t.Fatal("Not enough snapshots for leak analysis")
	}

	baseline := snapshots[0].HeapAlloc
	final := snapshots[len(snapshots)-1].HeapAlloc
	growth := float64(final-baseline) / (1024 * 1024)

	t.Logf("Memory leak analysis:")
	t.Logf("  Baseline memory: %.2f MB", float64(baseline)/(1024*1024))
	t.Logf("  Final memory: %.2f MB", float64(final)/(1024*1024))
	t.Logf("  Total growth: %.2f MB", growth)

	// Memory growth should be minimal for repeated operations
	maxAllowedGrowthMB := 50.0 // Allow up to 50MB growth
	if growth > maxAllowedGrowthMB {
		t.Errorf("Potential memory leak detected: %.2f MB growth (max allowed: %.2f MB)", 
			growth, maxAllowedGrowthMB)
	} else {
		t.Logf("✓ No significant memory leak detected (%.2f MB growth)", growth)
	}
}

// TestMemoryProfileDuringStorageOperations tests memory usage during storage operations
func TestMemoryProfileDuringStorageOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping storage memory test in short mode")
	}

	profiler := NewMemoryProfiler()
	tmpDir := t.TempDir()
	
	profiler.TakeSnapshot("storage_baseline", 0)

	// Setup storage
	storageConfig := storage.Config{BaseDir: tmpDir}
	storageEngine, err := storage.NewLocalStorage(storageConfig)
	if err != nil {
		t.Fatalf("Failed to create storage engine: %v", err)
	}
	
	profiler.TakeSnapshot("storage_start", 0)

	// Create and store many snapshots
	snapshotCount := 50
	t.Logf("Creating and storing %d snapshots...", snapshotCount)

	for i := 0; i < snapshotCount; i++ {
		snapshot := createLargeSnapshot(fmt.Sprintf("storage-memory-test-%d", i), 1000)
		
		err := storageEngine.SaveSnapshot(snapshot)
		if err != nil {
			t.Fatalf("Failed to store snapshot %d: %v", i, err)
		}

		if i%10 == 0 {
			profiler.TakeSnapshot(fmt.Sprintf("storage_save_%d", i), i*1000)
		}
	}

	profiler.TakeSnapshot("storage_after_saves", snapshotCount*1000)

	// Load all snapshots
	t.Logf("Loading %d snapshots...", snapshotCount)
	for i := 0; i < snapshotCount; i++ {
		snapshotID := fmt.Sprintf("storage-memory-test-%d", i)
		_, err := storageEngine.LoadSnapshot(snapshotID)
		if err != nil {
			t.Fatalf("Failed to load snapshot %d: %v", i, err)
		}

		if i%10 == 0 {
			profiler.TakeSnapshot(fmt.Sprintf("storage_load_%d", i), snapshotCount*1000)
		}
	}

	profiler.TakeSnapshot("storage_after_loads", snapshotCount*1000)

	// Analyze storage memory usage
	report := profiler.GetReport()
	t.Log("\n" + report)

	// Check for memory growth during storage operations
	snapshots := profiler.snapshots
	startSnapshot := snapshots[1] // storage_start
	endSnapshot := snapshots[len(snapshots)-1] // storage_after_loads

	growth := float64(endSnapshot.HeapAlloc-startSnapshot.HeapAlloc) / (1024 * 1024)
	t.Logf("Storage operations memory analysis:")
	t.Logf("  Start memory: %.2f MB", float64(startSnapshot.HeapAlloc)/(1024*1024))
	t.Logf("  End memory: %.2f MB", float64(endSnapshot.HeapAlloc)/(1024*1024))
	t.Logf("  Memory growth: %.2f MB", growth)
	t.Logf("  Snapshots processed: %d", snapshotCount)

	// Storage should not accumulate too much memory
	maxStorageGrowthMB := 100.0 // Allow up to 100MB growth for 50 snapshots
	if growth > maxStorageGrowthMB {
		t.Errorf("Storage memory growth too high: %.2f MB (max: %.2f MB)", 
			growth, maxStorageGrowthMB)
	} else {
		t.Logf("✓ Storage memory usage acceptable (%.2f MB growth)", growth)
	}
}

// TestCPUProfileDuringIntensiveOperations profiles CPU usage
func TestCPUProfileDuringIntensiveOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CPU profile test in short mode")
	}

	tmpDir := t.TempDir()
	cpuProfileFile := filepath.Join(tmpDir, "cpu-profile.prof")

	// Start CPU profiling
	f, err := os.Create(cpuProfileFile)
	if err != nil {
		t.Fatalf("Could not create CPU profile file: %v", err)
	}
	defer f.Close()

	if err := pprof.StartCPUProfile(f); err != nil {
		t.Fatalf("Could not start CPU profile: %v", err)
	}
	defer pprof.StopCPUProfile()

	// Perform intensive operations
	t.Log("Performing intensive operations for CPU profiling...")

	// Create large dataset
	stateFile := filepath.Join(tmpDir, "cpu-profile-test.tfstate")
	state := createMegaTestState(25000)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal state: %v", err)
	}
	
	if err := os.WriteFile(stateFile, data, 0644); err != nil {
		t.Fatalf("Failed to write state file: %v", err)
	}

	startTime := time.Now()

	// Intensive collection
	for i := 0; i < 5; i++ {
		collector := terraform.NewTerraformCollector()
		config := collectors.CollectorConfig{
			StatePaths: []string{stateFile},
			Tags:       map[string]string{"cpu-profile": fmt.Sprintf("%d", i)},
		}
		
		snapshot, err := collector.Collect(context.Background(), config)
		if err != nil {
			t.Fatalf("Collection failed: %v", err)
		}
		
		// Intensive diff
		baseline := createLargeSnapshot("baseline", 25000)
		differ := differ.NewSimpleDiffer()
		_, err = differ.Compare(baseline, snapshot)
		if err != nil {
			t.Fatalf("Diff failed: %v", err)
		}
	}

	duration := time.Since(startTime)

	t.Logf("CPU profiling completed in %v", duration)
	t.Logf("CPU profile saved to: %s", cpuProfileFile)
	t.Logf("To analyze profile, run: go tool pprof %s", cpuProfileFile)
}

// TestHeapProfileDuringMemoryIntensiveOps profiles heap usage
func TestHeapProfileDuringMemoryIntensiveOps(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping heap profile test in short mode")
	}

	tmpDir := t.TempDir()
	
	// Perform memory-intensive operations
	t.Log("Performing memory-intensive operations for heap profiling...")

	// Create multiple large datasets
	resourceCounts := []int{5000, 10000, 15000, 20000}
	snapshots := make([]*types.Snapshot, len(resourceCounts))

	for i, resourceCount := range resourceCounts {
		snapshots[i] = createLargeSnapshot(fmt.Sprintf("heap-test-%d", i), resourceCount)
	}

	// Perform multiple diffs to stress heap
	differ := differ.NewSimpleDiffer()
	for i := 0; i < len(snapshots)-1; i++ {
		_, err := differ.Compare(snapshots[i], snapshots[i+1])
		if err != nil {
			t.Fatalf("Diff failed: %v", err)
		}
	}

	// Take heap profile
	heapProfileFile := filepath.Join(tmpDir, "heap-profile.prof")
	f, err := os.Create(heapProfileFile)
	if err != nil {
		t.Fatalf("Could not create heap profile file: %v", err)
	}
	defer f.Close()

	runtime.GC() // Force GC before heap profile
	if err := pprof.WriteHeapProfile(f); err != nil {
		t.Fatalf("Could not write heap profile: %v", err)
	}

	t.Logf("Heap profile saved to: %s", heapProfileFile)
	t.Logf("To analyze profile, run: go tool pprof %s", heapProfileFile)

	// Basic heap analysis
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	t.Logf("Heap analysis:")
	t.Logf("  Heap allocated: %.2f MB", float64(m.HeapAlloc)/(1024*1024))
	t.Logf("  Heap system: %.2f MB", float64(m.HeapSys)/(1024*1024))
	t.Logf("  GC cycles: %d", m.NumGC)
	t.Logf("  Total allocations: %.2f MB", float64(m.TotalAlloc)/(1024*1024))
}