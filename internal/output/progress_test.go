package output

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestProgressBar_Advanced(t *testing.T) {
	var buf bytes.Buffer
	config := ProgressBarConfig{
		Title:       "Advanced Test",
		Total:       100,
		Width:       20,
		ShowPercent: true,
		ShowETA:     false,
		NoColor:     true,
		Writer:      &buf,
	}

	bar := NewProgressBar(config)

	// Test initial state
	bar.Update(0)
	if bar.current != 0 {
		t.Errorf("Expected current to be 0, got %d", bar.current)
	}

	// Test normal updates
	bar.Update(25)
	if bar.current != 25 {
		t.Errorf("Expected current to be 25, got %d", bar.current)
	}

	bar.Update(50)
	if bar.current != 50 {
		t.Errorf("Expected current to be 50, got %d", bar.current)
	}

	// Test completion
	bar.Update(100)
	if !bar.isComplete {
		t.Error("Expected progress bar to be complete")
	}

	// Test over-completion (should cap at total)
	bar.Update(150)
	if bar.current != bar.total {
		t.Errorf("Expected current to be capped at total (%d), got %d", bar.total, bar.current)
	}
}

func TestProgressBar_Increment(t *testing.T) {
	var buf bytes.Buffer
	config := ProgressBarConfig{
		Title:   "Increment Test",
		Total:   100,
		Width:   10,
		NoColor: true,
		Writer:  &buf,
	}

	bar := NewProgressBar(config)

	// Test increments
	bar.Increment(30)
	if bar.current != 30 {
		t.Errorf("Expected current to be 30, got %d", bar.current)
	}

	bar.Increment(40)
	if bar.current != 70 {
		t.Errorf("Expected current to be 70, got %d", bar.current)
	}

	// Test increment that would exceed total
	bar.Increment(50)
	if bar.current != bar.total {
		t.Errorf("Expected current to be capped at total (%d), got %d", bar.total, bar.current)
	}

	if !bar.isComplete {
		t.Error("Expected progress bar to be complete after exceeding total")
	}
}

func TestProgressBar_Finish(t *testing.T) {
	var buf bytes.Buffer
	config := ProgressBarConfig{
		Title:   "Finish Test",
		Total:   100,
		Width:   10,
		NoColor: true,
		Writer:  &buf,
	}

	bar := NewProgressBar(config)

	bar.Update(50)
	bar.Finish()

	if bar.current != bar.total {
		t.Errorf("Expected current to be total (%d) after finish, got %d", bar.total, bar.current)
	}

	if !bar.isComplete {
		t.Error("Expected progress bar to be complete after finish")
	}
}

func TestProgressBar_Output(t *testing.T) {
	var buf bytes.Buffer
	config := ProgressBarConfig{
		Title:       "Output Test",
		Total:       10,
		Width:       10,
		ShowPercent: true,
		NoColor:     true,
		Writer:      &buf,
	}

	bar := NewProgressBar(config)
	bar.Update(5)

	output := buf.String()

	// Check for expected elements
	if !strings.Contains(output, "Output Test") {
		t.Error("Output should contain title")
	}

	if !strings.Contains(output, "50.0%") {
		t.Error("Output should contain percentage")
	}

	if !strings.Contains(output, "5/10") {
		t.Error("Output should contain current/total")
	}

	if !strings.Contains(output, "[") || !strings.Contains(output, "]") {
		t.Error("Output should contain progress bar brackets")
	}
}

func TestProgressBar_ConcurrentAccess(t *testing.T) {
	var buf bytes.Buffer
	config := ProgressBarConfig{
		Title:   "Concurrent Test",
		Total:   1000,
		Width:   20,
		NoColor: true,
		Writer:  &buf,
	}

	bar := NewProgressBar(config)

	// Test concurrent updates
	var wg sync.WaitGroup
	numGoroutines := 10
	incrementsPerGoroutine := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				bar.Increment(1)
				time.Sleep(time.Microsecond) // Small delay to encourage race conditions
			}
		}()
	}

	wg.Wait()

	expectedTotal := int64(numGoroutines * incrementsPerGoroutine)
	if bar.current != expectedTotal {
		t.Errorf("Expected current to be %d after concurrent updates, got %d", expectedTotal, bar.current)
	}
}

func TestSpinner_Advanced(t *testing.T) {
	spinner := NewSpinner("Advanced Testing...", true)

	if spinner.active {
		t.Error("Spinner should not be active initially")
	}

	spinner.Start()
	if !spinner.active {
		t.Error("Spinner should be active after start")
	}

	// Let it spin briefly
	time.Sleep(50 * time.Millisecond)

	spinner.Update("Still testing...")
	if spinner.title != "Still testing..." {
		t.Error("Spinner title should be updated")
	}

	spinner.Stop()
	if spinner.active {
		t.Error("Spinner should not be active after stop")
	}

	// Test double start/stop (should be safe)
	spinner.Start()
	spinner.Start() // Should be safe
	spinner.Stop()
	spinner.Stop() // Should be safe
}

func TestSpinner_Concurrent(t *testing.T) {
	spinner := NewSpinner("Concurrent test", true)

	var wg sync.WaitGroup

	// Start multiple goroutines trying to start/stop/update
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			spinner.Start()
			time.Sleep(10 * time.Millisecond)
			spinner.Update(fmt.Sprintf("Goroutine %d", id))
			time.Sleep(10 * time.Millisecond)
			spinner.Stop()
		}(i)
	}

	wg.Wait()

	// Final state should be consistent
	if spinner.active {
		t.Error("Spinner should not be active after all goroutines complete")
	}
}

func TestStepProgress_Advanced(t *testing.T) {
	steps := []string{
		"Initialize",
		"Process",
		"Validate",
		"Complete",
	}

	progress := NewStepProgress("Advanced Test Steps", steps, true)

	// Test initial state
	if progress.current != 0 {
		t.Errorf("Expected initial current to be 0, got %d", progress.current)
	}

	// Test stepping through
	progress.NextStep()
	if progress.current != 1 {
		t.Errorf("Expected current to be 1 after first step, got %d", progress.current)
	}

	progress.NextStep()
	if progress.current != 2 {
		t.Errorf("Expected current to be 2 after second step, got %d", progress.current)
	}

	// Test SetStep
	progress.SetStep(3)
	if progress.current != 3 {
		t.Errorf("Expected current to be 3 after SetStep(3), got %d", progress.current)
	}

	// Test bounds checking
	progress.SetStep(10) // Beyond total
	if progress.current != 4 {
		t.Errorf("Expected current to be capped at total (4), got %d", progress.current)
	}

	progress.SetStep(-1) // Negative
	if progress.current != 4 {
		t.Errorf("Expected current to remain 4 after invalid SetStep(-1), got %d", progress.current)
	}
}

func TestStepProgress_NextStepBounds(t *testing.T) {
	steps := []string{"Step1", "Step2"}
	progress := NewStepProgress("Bounds Test", steps, true)

	// Step through normally
	progress.NextStep() // current = 1
	progress.NextStep() // current = 2 (at total)

	// Additional NextStep calls should not exceed total
	progress.NextStep() // should remain at 2
	if progress.current != 2 {
		t.Errorf("Expected current to remain at total (2), got %d", progress.current)
	}
}

func TestStepProgress_Finish(t *testing.T) {
	steps := []string{"Step1", "Step2", "Step3"}
	progress := NewStepProgress("Finish Test", steps, true)

	progress.NextStep() // current = 1
	progress.Finish()   // should set current = total

	if progress.current != progress.total {
		t.Errorf("Expected current to be total (%d) after finish, got %d", progress.total, progress.current)
	}
}

func TestMultiProgressBar_Basic(t *testing.T) {
	multi := NewMultiProgressBar(true)

	// Add multiple bars
	bar1 := multi.AddBar(ProgressBarConfig{
		Title: "Task 1",
		Total: 100,
		Width: 10,
	})

	bar2 := multi.AddBar(ProgressBarConfig{
		Title: "Task 2",
		Total: 50,
		Width: 10,
	})

	if len(multi.bars) != 2 {
		t.Errorf("Expected 2 bars, got %d", len(multi.bars))
	}

	// Update bars
	bar1.Update(50)
	bar2.Update(25)

	if bar1.current != 50 {
		t.Errorf("Bar1 current should be 50, got %d", bar1.current)
	}

	if bar2.current != 25 {
		t.Errorf("Bar2 current should be 25, got %d", bar2.current)
	}

	// Finish all
	multi.Finish()

	if !bar1.isComplete || !bar2.isComplete {
		t.Error("All bars should be complete after multi.Finish()")
	}
}

func TestProgressBar_ETA(t *testing.T) {
	var buf bytes.Buffer
	config := ProgressBarConfig{
		Title:   "ETA Test",
		Total:   100,
		Width:   10,
		ShowETA: true,
		NoColor: true,
		Writer:  &buf,
	}

	bar := NewProgressBar(config)

	// Need some progress and time for ETA calculation
	bar.Update(10)
	time.Sleep(10 * time.Millisecond)
	bar.Update(20)

	output := buf.String()

	// Should contain ETA information when there's progress
	if bar.current > 0 && !bar.isComplete {
		if !strings.Contains(output, "ETA:") {
			t.Error("Output should contain ETA when ShowETA is true and bar is not complete")
		}
	}
}

func TestProgressBar_ZeroTotal(t *testing.T) {
	var buf bytes.Buffer
	config := ProgressBarConfig{
		Title:   "Zero Total",
		Total:   0, // Zero total
		Width:   10,
		NoColor: true,
		Writer:  &buf,
	}

	bar := NewProgressBar(config)
	bar.Update(10) // Should not panic

	// With zero total, render should handle gracefully
	if buf.Len() > 0 {
		// If it renders anything, it should not crash
		output := buf.String()
		t.Logf("Zero total output: %s", output)
	}
}

func TestFormatHelpers(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{500, "500"},
		{1500, "1.5K"},
		{1500000, "1.5M"},
		{999, "999"},
		{1000, "1.0K"},
		{1000000, "1.0M"},
	}

	for _, test := range tests {
		result := formatNumber(test.input)
		if result != test.expected {
			t.Errorf("formatNumber(%d) = %s, expected %s", test.input, result, test.expected)
		}
	}

	// Test duration formatting
	durationTests := []struct {
		input    time.Duration
		contains string
	}{
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m"},
		{3661 * time.Second, "1h"},
	}

	for _, test := range durationTests {
		result := formatDuration(test.input)
		if !strings.Contains(result, test.contains) {
			t.Errorf("formatDuration(%v) = %s, should contain %s", test.input, result, test.contains)
		}
	}
}

// Benchmark tests
func BenchmarkProgressBar_Update(b *testing.B) {
	var buf bytes.Buffer
	config := ProgressBarConfig{
		Title:   "Benchmark",
		Total:   int64(b.N),
		Width:   20,
		NoColor: true,
		Writer:  &buf,
	}

	bar := NewProgressBar(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bar.Update(int64(i))
	}
}

func BenchmarkProgressBar_Increment(b *testing.B) {
	var buf bytes.Buffer
	config := ProgressBarConfig{
		Title:   "Benchmark Increment",
		Total:   int64(b.N),
		Width:   20,
		NoColor: true,
		Writer:  &buf,
	}

	bar := NewProgressBar(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bar.Increment(1)
	}
}