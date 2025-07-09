package output

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

// ProgressBar represents a progress bar with customizable appearance
type ProgressBar struct {
	mu          sync.Mutex
	writer      io.Writer
	title       string
	total       int64
	current     int64
	width       int
	showPercent bool
	showETA     bool
	startTime   time.Time
	isComplete  bool
	noColor     bool
}

// ProgressBarConfig configures a progress bar
type ProgressBarConfig struct {
	Title       string
	Total       int64
	Width       int
	ShowPercent bool
	ShowETA     bool
	NoColor     bool
	Writer      io.Writer
}

// NewProgressBar creates a new progress bar
func NewProgressBar(config ProgressBarConfig) *ProgressBar {
	if config.Width == 0 {
		config.Width = 40
	}
	if config.Writer == nil {
		config.Writer = os.Stderr
	}

	return &ProgressBar{
		writer:      config.Writer,
		title:       config.Title,
		total:       config.Total,
		width:       config.Width,
		showPercent: config.ShowPercent,
		showETA:     config.ShowETA,
		startTime:   time.Now(),
		noColor:     config.NoColor,
	}
}

// Update updates the progress bar with new progress
func (p *ProgressBar) Update(current int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current = current
	if current >= p.total {
		p.isComplete = true
	}
	p.render()
}

// Increment increments the progress by the given amount
func (p *ProgressBar) Increment(delta int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current += delta
	if p.current >= p.total {
		p.current = p.total
		p.isComplete = true
	}
	p.render()
}

// Finish completes the progress bar
func (p *ProgressBar) Finish() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current = p.total
	p.isComplete = true
	p.render()
	fmt.Fprintln(p.writer) // Add newline after completion
}

// render renders the progress bar
func (p *ProgressBar) render() {
	if p.total == 0 {
		return
	}

	percentage := float64(p.current) / float64(p.total)
	if percentage > 1.0 {
		percentage = 1.0
	}

	// Calculate bar components
	filled := int(percentage * float64(p.width))
	if filled > p.width {
		filled = p.width
	}

	// Build progress bar
	var bar strings.Builder

	// Add title
	if p.title != "" {
		bar.WriteString(p.colorize(p.title+" ", color.FgCyan))
	}

	// Add progress bar
	bar.WriteString("[")

	// Filled portion
	if filled > 0 {
		bar.WriteString(p.colorize(strings.Repeat("█", filled), color.FgGreen))
	}

	// Empty portion
	if filled < p.width {
		bar.WriteString(strings.Repeat("░", p.width-filled))
	}

	bar.WriteString("]")

	// Add percentage
	if p.showPercent {
		bar.WriteString(fmt.Sprintf(" %s", p.colorize(fmt.Sprintf("%.1f%%", percentage*100), color.FgWhite, color.Bold)))
	}

	// Add current/total
	bar.WriteString(fmt.Sprintf(" %s/%s",
		p.colorize(formatNumber(p.current), color.FgWhite, color.Bold),
		p.colorize(formatNumber(p.total), color.FgWhite)))

	// Add ETA
	if p.showETA && !p.isComplete && p.current > 0 {
		elapsed := time.Since(p.startTime)
		rate := float64(p.current) / elapsed.Seconds()
		remaining := float64(p.total-p.current) / rate
		eta := time.Duration(remaining) * time.Second

		bar.WriteString(fmt.Sprintf(" ETA: %s", p.colorize(formatDuration(eta), color.FgYellow)))
	}

	// Add completion indicator
	if p.isComplete {
		bar.WriteString(" ")
		bar.WriteString(p.colorize("✅", color.FgGreen))
	}

	// Clear line and print
	fmt.Fprintf(p.writer, "\r%s", strings.Repeat(" ", 100)) // Clear line
	fmt.Fprintf(p.writer, "\r%s", bar.String())
}

// colorize applies color if colors are enabled
func (p *ProgressBar) colorize(text string, attrs ...color.Attribute) string {
	if p.noColor {
		return text
	}
	return color.New(attrs...).Sprint(text)
}

// Spinner represents a spinning progress indicator
type Spinner struct {
	mu      sync.Mutex
	writer  io.Writer
	title   string
	chars   []string
	index   int
	active  bool
	ticker  *time.Ticker
	done    chan bool
	noColor bool
}

// NewSpinner creates a new spinner
func NewSpinner(title string, noColor bool) *Spinner {
	return &Spinner{
		writer:  os.Stderr,
		title:   title,
		chars:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		done:    make(chan bool),
		noColor: noColor,
	}
}

// Start starts the spinner
func (s *Spinner) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.active {
		return
	}

	s.active = true
	s.ticker = time.NewTicker(100 * time.Millisecond)

	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.render()
			case <-s.done:
				return
			}
		}
	}()
}

// Stop stops the spinner
func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return
	}

	s.active = false
	s.ticker.Stop()
	s.done <- true

	// Clear the spinner line
	fmt.Fprintf(s.writer, "\r%s\r", strings.Repeat(" ", 100))
}

// Update updates the spinner title
func (s *Spinner) Update(title string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.title = title
}

// render renders the spinner
func (s *Spinner) render() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return
	}

	char := s.chars[s.index]
	s.index = (s.index + 1) % len(s.chars)

	var output strings.Builder
	output.WriteString(s.colorize(char, color.FgCyan))
	output.WriteString(" ")
	output.WriteString(s.colorize(s.title, color.FgWhite))

	fmt.Fprintf(s.writer, "\r%s", output.String())
}

// colorize applies color if colors are enabled
func (s *Spinner) colorize(text string, attrs ...color.Attribute) string {
	if s.noColor {
		return text
	}
	return color.New(attrs...).Sprint(text)
}

// MultiProgressBar manages multiple progress bars
type MultiProgressBar struct {
	mu      sync.Mutex
	writer  io.Writer
	bars    []*ProgressBar
	lines   int
	noColor bool
}

// NewMultiProgressBar creates a new multi-progress bar manager
func NewMultiProgressBar(noColor bool) *MultiProgressBar {
	return &MultiProgressBar{
		writer:  os.Stderr,
		noColor: noColor,
	}
}

// AddBar adds a new progress bar
func (m *MultiProgressBar) AddBar(config ProgressBarConfig) *ProgressBar {
	m.mu.Lock()
	defer m.mu.Unlock()

	config.Writer = &lineWriter{parent: m, lineIndex: len(m.bars)}
	config.NoColor = m.noColor

	bar := NewProgressBar(config)
	m.bars = append(m.bars, bar)
	m.lines = len(m.bars)

	return bar
}

// Finish completes all progress bars
func (m *MultiProgressBar) Finish() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, bar := range m.bars {
		bar.Finish()
	}

	// Move cursor down past all bars
	for i := 0; i < m.lines; i++ {
		fmt.Fprintln(m.writer)
	}
}

// lineWriter writes to a specific line in the multi-progress display
type lineWriter struct {
	parent    *MultiProgressBar
	lineIndex int
}

// Write implements io.Writer for line-specific output
func (w *lineWriter) Write(p []byte) (n int, err error) {
	// This is a simplified implementation
	// In practice, you'd want proper terminal control
	return w.parent.writer.Write(p)
}

// StepProgress represents a step-based progress indicator
type StepProgress struct {
	mu      sync.Mutex
	writer  io.Writer
	title   string
	steps   []string
	current int
	total   int
	noColor bool
}

// NewStepProgress creates a new step progress indicator
func NewStepProgress(title string, steps []string, noColor bool) *StepProgress {
	return &StepProgress{
		writer:  os.Stderr,
		title:   title,
		steps:   steps,
		total:   len(steps),
		noColor: noColor,
	}
}

// NextStep advances to the next step
func (s *StepProgress) NextStep() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current < s.total {
		s.current++
	}
	s.render()
}

// SetStep sets the current step by index
func (s *StepProgress) SetStep(index int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if index >= 0 && index <= s.total {
		s.current = index
	}
	s.render()
}

// Finish completes the step progress
func (s *StepProgress) Finish() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.current = s.total
	s.render()
	fmt.Fprintln(s.writer)
}

// render renders the step progress
func (s *StepProgress) render() {
	var output strings.Builder

	if s.title != "" {
		output.WriteString(s.colorize(s.title+":\n", color.FgCyan, color.Bold))
	}

	for i, step := range s.steps {
		var icon, stepColor string

		if i < s.current {
			icon = "✅"
			stepColor = s.colorize(step, color.FgGreen)
		} else if i == s.current {
			icon = "🔄"
			stepColor = s.colorize(step, color.FgYellow, color.Bold)
		} else {
			icon = "⏳"
			stepColor = s.colorize(step, color.FgWhite)
		}

		output.WriteString(fmt.Sprintf("%s %s\n", icon, stepColor))
	}

	fmt.Fprintf(s.writer, "\r%s", output.String())
}

// colorize applies color if colors are enabled
func (s *StepProgress) colorize(text string, attrs ...color.Attribute) string {
	if s.noColor {
		return text
	}
	return color.New(attrs...).Sprint(text)
}

// Helper functions

// formatNumber formats a number with appropriate units
func formatNumber(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	} else if n < 1000000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	} else {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0fm%.0fs", d.Minutes(), d.Seconds()-60*d.Minutes())
	} else {
		return fmt.Sprintf("%.0fh%.0fm", d.Hours(), d.Minutes()-60*d.Hours())
	}
}
