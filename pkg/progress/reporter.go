package progress

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Reporter provides real-time progress updates for long operations
type Reporter struct {
	output      io.Writer
	mu          sync.RWMutex
	trackers    map[string]*Tracker
	showSpinner bool
	updateRate  time.Duration
}

// NewReporter creates a new progress reporter
func NewReporter() *Reporter {
	return &Reporter{
		output:      os.Stdout,
		trackers:    make(map[string]*Tracker),
		showSpinner: true,
		updateRate:  100 * time.Millisecond,
	}
}

// NewQuietReporter creates a reporter that only shows summary output
func NewQuietReporter() *Reporter {
	return &Reporter{
		output:      os.Stdout,
		trackers:    make(map[string]*Tracker),
		showSpinner: false,
		updateRate:  time.Second,
	}
}

// Tracker tracks progress for a specific operation
type Tracker struct {
	name        string
	total       int64
	current     int64
	startTime   time.Time
	lastUpdate  time.Time
	status      string
	subTrackers map[string]*SubTracker
	mu          sync.RWMutex
	done        chan struct{}
}

// SubTracker tracks progress for sub-operations
type SubTracker struct {
	name    string
	current int64
	total   int64
	status  string
}

// StartOperation begins tracking a new operation
func (r *Reporter) StartOperation(name string, total int64) *Tracker {
	r.mu.Lock()
	defer r.mu.Unlock()

	tracker := &Tracker{
		name:        name,
		total:       total,
		startTime:   time.Now(),
		lastUpdate:  time.Now(),
		subTrackers: make(map[string]*SubTracker),
		done:        make(chan struct{}),
	}

	r.trackers[name] = tracker

	// Start progress display goroutine
	if r.showSpinner {
		go r.displayProgress(tracker)
	}

	return tracker
}

// displayProgress shows real-time progress updates
func (r *Reporter) displayProgress(tracker *Tracker) {
	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	spinnerIndex := 0
	ticker := time.NewTicker(r.updateRate)
	defer ticker.Stop()

	for {
		select {
		case <-tracker.done:
			// Final update
			r.printProgress(tracker, "", true)
			return
		case <-ticker.C:
			spinnerChar := spinner[spinnerIndex%len(spinner)]
			spinnerIndex++
			r.printProgress(tracker, spinnerChar, false)
		}
	}
}

// printProgress outputs the current progress status
func (r *Reporter) printProgress(tracker *Tracker, spinner string, final bool) {
	tracker.mu.RLock()
	defer tracker.mu.RUnlock()

	current := atomic.LoadInt64(&tracker.current)
	total := tracker.total
	elapsed := time.Since(tracker.startTime)

	if total > 0 {
		percentage := float64(current) / float64(total) * 100
		rate := float64(current) / elapsed.Seconds()

		// Estimate time remaining
		var etaStr string
		if current > 0 && current < total {
			remaining := float64(total-current) / rate
			eta := time.Duration(remaining * float64(time.Second))
			etaStr = fmt.Sprintf(" | ETA: %s", formatDuration(eta))
		}

		// Build progress bar
		barWidth := 30
		filled := int(percentage * float64(barWidth) / 100)
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

		if final {
			fmt.Fprintf(r.output, "\r%s: [%s] 100%% (%d/%d) | Duration: %s ✓\n",
				tracker.name, strings.Repeat("█", barWidth), total, total, formatDuration(elapsed))
		} else {
			fmt.Fprintf(r.output, "\r%s %s: [%s] %.1f%% (%d/%d) | Rate: %.1f/s%s",
				spinner, tracker.name, bar, percentage, current, total, rate, etaStr)
		}
	} else if tracker.status != "" {
		if final {
			fmt.Fprintf(r.output, "\r%s: %s | Duration: %s ✓\n",
				tracker.name, tracker.status, formatDuration(elapsed))
		} else {
			fmt.Fprintf(r.output, "\r%s %s: %s | Elapsed: %s",
				spinner, tracker.name, tracker.status, formatDuration(elapsed))
		}
	} else {
		if final {
			fmt.Fprintf(r.output, "\r%s: Completed %d items | Duration: %s ✓\n",
				tracker.name, current, formatDuration(elapsed))
		} else {
			fmt.Fprintf(r.output, "\r%s %s: Processing... (%d items) | Elapsed: %s",
				spinner, tracker.name, current, formatDuration(elapsed))
		}
	}
}

// Increment increases the progress counter
func (t *Tracker) Increment(delta int64) {
	atomic.AddInt64(&t.current, delta)
	t.mu.Lock()
	t.lastUpdate = time.Now()
	t.mu.Unlock()
}

// SetStatus updates the status message
func (t *Tracker) SetStatus(status string) {
	t.mu.Lock()
	t.status = status
	t.lastUpdate = time.Now()
	t.mu.Unlock()
}

// AddSubTracker creates a sub-tracker for nested operations
func (t *Tracker) AddSubTracker(name string, total int64) *SubTracker {
	t.mu.Lock()
	defer t.mu.Unlock()

	subTracker := &SubTracker{
		name:  name,
		total: total,
	}
	t.subTrackers[name] = subTracker
	return subTracker
}

// UpdateSubTracker updates a sub-tracker's progress
func (t *Tracker) UpdateSubTracker(name string, current int64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if sub, exists := t.subTrackers[name]; exists {
		sub.current = current
		t.lastUpdate = time.Now()
	}
}

// Complete marks the operation as complete
func (t *Tracker) Complete() {
	atomic.StoreInt64(&t.current, t.total)
	close(t.done)
}

// CompleteWithCount marks the operation as complete with a final count
func (t *Tracker) CompleteWithCount(finalCount int64) {
	atomic.StoreInt64(&t.current, finalCount)
	atomic.StoreInt64(&t.total, finalCount)
	close(t.done)
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		min := int(d.Minutes())
		sec := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", min, sec)
	}
	hour := int(d.Hours())
	min := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", hour, min)
}

// MultiProgress manages multiple concurrent progress trackers
type MultiProgress struct {
	mu       sync.RWMutex
	trackers []*ConcurrentTracker
	output   io.Writer
	done     chan struct{}
}

// ConcurrentTracker tracks progress for concurrent operations
type ConcurrentTracker struct {
	ID         string
	Name       string
	Current    int64
	Total      int64
	Status     string
	StartTime  time.Time
	LastUpdate time.Time
	completed  bool
}

// NewMultiProgress creates a manager for multiple progress trackers
func NewMultiProgress() *MultiProgress {
	mp := &MultiProgress{
		output:   os.Stdout,
		trackers: make([]*ConcurrentTracker, 0),
		done:     make(chan struct{}),
	}

	// Start display goroutine
	go mp.display()

	return mp
}

// AddTracker adds a new concurrent tracker
func (mp *MultiProgress) AddTracker(id, name string, total int64) *ConcurrentTracker {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	tracker := &ConcurrentTracker{
		ID:        id,
		Name:      name,
		Total:     total,
		StartTime: time.Now(),
	}

	mp.trackers = append(mp.trackers, tracker)
	return tracker
}

// Update updates a tracker's progress
func (mp *MultiProgress) Update(id string, current int64, status string) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	for _, tracker := range mp.trackers {
		if tracker.ID == id {
			atomic.StoreInt64(&tracker.Current, current)
			tracker.Status = status
			tracker.LastUpdate = time.Now()
			break
		}
	}
}

// Complete marks a tracker as complete
func (mp *MultiProgress) Complete(id string) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	for _, tracker := range mp.trackers {
		if tracker.ID == id {
			tracker.Current = tracker.Total
			tracker.completed = true
			break
		}
	}
}

// display shows all concurrent progress trackers
func (mp *MultiProgress) display() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-mp.done:
			mp.render(true)
			return
		case <-ticker.C:
			mp.render(false)
		}
	}
}

// render displays the current state of all trackers
func (mp *MultiProgress) render(final bool) {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	// Clear previous lines
	if !final {
		fmt.Fprintf(mp.output, "\033[%dA\033[K", len(mp.trackers))
	}

	for _, tracker := range mp.trackers {
		current := atomic.LoadInt64(&tracker.Current)
		if tracker.Total > 0 {
			percentage := float64(current) / float64(tracker.Total) * 100
			barWidth := 20
			filled := int(percentage * float64(barWidth) / 100)
			bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

			status := ""
			if tracker.completed {
				status = " ✓"
			} else if tracker.Status != "" {
				status = fmt.Sprintf(" - %s", tracker.Status)
			}

			fmt.Fprintf(mp.output, "%s: [%s] %.1f%% (%d/%d)%s\n",
				tracker.Name, bar, percentage, current, tracker.Total, status)
		} else {
			status := tracker.Status
			if tracker.completed {
				status = "Complete ✓"
			} else if status == "" {
				status = "Processing..."
			}
			fmt.Fprintf(mp.output, "%s: %s (%d items)\n",
				tracker.Name, status, current)
		}
	}
}

// Stop stops the multi-progress display
func (mp *MultiProgress) Stop() {
	close(mp.done)
}

// SimpleProgress provides a simple progress indicator for basic operations
type SimpleProgress struct {
	message string
	spinner []string
	index   int
	done    chan struct{}
	mu      sync.Mutex
}

// NewSimpleProgress creates a simple progress indicator
func NewSimpleProgress(message string) *SimpleProgress {
	sp := &SimpleProgress{
		message: message,
		spinner: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		done:    make(chan struct{}),
	}

	go sp.spin()
	return sp
}

// spin displays the spinning animation
func (sp *SimpleProgress) spin() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-sp.done:
			fmt.Printf("\r%s... Done ✓\n", sp.message)
			return
		case <-ticker.C:
			sp.mu.Lock()
			fmt.Printf("\r%s %s...", sp.spinner[sp.index%len(sp.spinner)], sp.message)
			sp.index++
			sp.mu.Unlock()
		}
	}
}

// Stop stops the progress indicator
func (sp *SimpleProgress) Stop() {
	close(sp.done)
	time.Sleep(100 * time.Millisecond) // Allow final render
}

// ProgressContext provides progress tracking through context
type ProgressContext struct {
	ctx      context.Context
	reporter *Reporter
}

// WithProgress adds progress tracking to a context
func WithProgress(ctx context.Context, reporter *Reporter) context.Context {
	return context.WithValue(ctx, progressKey{}, reporter)
}

// GetProgress retrieves the progress reporter from context
func GetProgress(ctx context.Context) *Reporter {
	if reporter, ok := ctx.Value(progressKey{}).(*Reporter); ok {
		return reporter
	}
	return nil
}

type progressKey struct{}

// TrackOperation is a helper to track an operation with automatic cleanup
func TrackOperation(ctx context.Context, name string, total int64, fn func(*Tracker) error) error {
	reporter := GetProgress(ctx)
	if reporter == nil {
		reporter = NewQuietReporter()
	}

	tracker := reporter.StartOperation(name, total)
	defer tracker.Complete()

	return fn(tracker)
}
