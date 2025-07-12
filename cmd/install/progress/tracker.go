package progress

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yairfalse/wgo/cmd/install/installer"
)

// Tracker tracks installation progress
type Tracker interface {
	Start(totalSteps int)
	Update(progress installer.Progress)
	Complete(step string)
	Fail(step string, err error)
	Finish()
}

// TerminalTracker provides terminal-based progress tracking
type TerminalTracker struct {
	writer      io.Writer
	totalSteps  int
	currentStep int32
	startTime   time.Time
	mu          sync.Mutex
	lastUpdate  time.Time
	lastLine    string
}

// NewTerminalTracker creates a new terminal progress tracker
func NewTerminalTracker(writer io.Writer) Tracker {
	return &TerminalTracker{
		writer:     writer,
		lastUpdate: time.Now(),
	}
}

func (t *TerminalTracker) Start(totalSteps int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.totalSteps = totalSteps
	t.startTime = time.Now()
	atomic.StoreInt32(&t.currentStep, 0)
}

func (t *TerminalTracker) Update(progress installer.Progress) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Throttle updates to avoid terminal spam
	if time.Since(t.lastUpdate) < 100*time.Millisecond {
		return
	}
	t.lastUpdate = time.Now()

	// Clear previous line
	t.clearLine()

	// Format progress message
	var message string
	if progress.TotalBytes > 0 {
		// Download progress
		percentage := float64(progress.BytesDownloaded) / float64(progress.TotalBytes) * 100
		message = fmt.Sprintf("[%d/%d] %s: %.1f%% (%s/%s) %s %s",
			progress.CompletedSteps+1,
			t.totalSteps,
			progress.CurrentStep,
			percentage,
			formatBytes(progress.BytesDownloaded),
			formatBytes(progress.TotalBytes),
			formatSpeed(progress.Speed),
			formatDuration(progress.EstimatedTime),
		)
	} else {
		// General progress
		elapsed := time.Since(t.startTime)
		message = fmt.Sprintf("[%d/%d] %s... %s",
			progress.CompletedSteps+1,
			t.totalSteps,
			progress.CurrentStep,
			formatDuration(elapsed),
		)
	}

	// Print progress
	fmt.Fprint(t.writer, "\r"+message)
	t.lastLine = message
}

func (t *TerminalTracker) Complete(step string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	atomic.AddInt32(&t.currentStep, 1)
	t.clearLine()

	currentStep := atomic.LoadInt32(&t.currentStep)
	fmt.Fprintf(t.writer, "✓ [%d/%d] %s\n", currentStep, t.totalSteps, step)
}

func (t *TerminalTracker) Fail(step string, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.clearLine()

	currentStep := atomic.LoadInt32(&t.currentStep) + 1
	fmt.Fprintf(t.writer, "✗ [%d/%d] %s: %v\n", currentStep, t.totalSteps, step, err)
}

func (t *TerminalTracker) Finish() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.clearLine()

	duration := time.Since(t.startTime)
	fmt.Fprintf(t.writer, "\nInstallation completed in %s\n", formatDuration(duration))
}

func (t *TerminalTracker) clearLine() {
	if t.lastLine != "" {
		// Clear the line
		fmt.Fprintf(t.writer, "\r%s\r", strings.Repeat(" ", len(t.lastLine)))
		t.lastLine = ""
	}
}

// SilentTracker provides no-op progress tracking
type SilentTracker struct{}

func NewSilentTracker() Tracker {
	return &SilentTracker{}
}

func (s *SilentTracker) Start(totalSteps int)               {}
func (s *SilentTracker) Update(progress installer.Progress) {}
func (s *SilentTracker) Complete(step string)               {}
func (s *SilentTracker) Fail(step string, err error)        {}
func (s *SilentTracker) Finish()                            {}

// Helper functions
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatSpeed(bytesPerSecond float64) string {
	if bytesPerSecond <= 0 {
		return ""
	}
	return fmt.Sprintf("@ %s/s", formatBytes(int64(bytesPerSecond)))
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		return "calculating..."
	}

	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
