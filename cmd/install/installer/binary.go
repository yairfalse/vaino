package installer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// BinaryInstaller implements the binary installation strategy
type BinaryInstaller struct {
	config         *Config
	stateManager   StateManager
	progressChan   chan Progress
	circuitBreaker CircuitBreaker
	httpClient     HTTPClient
	steps          []Step[Binary]
	middleware     []Middleware[Binary]
	mu             sync.RWMutex
	state          State
	downloadPool   *sync.Pool
}

// NewBinaryInstaller creates a new binary installer
func NewBinaryInstaller(config *Config, options ...Option) (*BinaryInstaller, error) {
	bi := &BinaryInstaller{
		config:       config,
		progressChan: make(chan Progress, 100),
		steps:        make([]Step[Binary], 0),
		middleware:   make([]Middleware[Binary], 0),
		downloadPool: &sync.Pool{
			New: func() interface{} {
				return make([]byte, 32*1024) // 32KB buffer
			},
		},
	}

	// Apply options
	for _, opt := range options {
		if err := opt(bi.config); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	// Initialize default components if not provided
	if bi.httpClient == nil {
		bi.httpClient = NewDefaultHTTPClient()
	}
	if bi.circuitBreaker == nil {
		bi.circuitBreaker = NewCircuitBreaker(5, 1*time.Minute)
	}
	if bi.stateManager == nil {
		bi.stateManager = NewFileStateManager()
	}

	// Initialize installation steps
	bi.initializeSteps()

	return bi, nil
}

func (bi *BinaryInstaller) initializeSteps() {
	// Create the installation pipeline steps
	bi.steps = []Step[Binary]{
		&downloadStep{installer: bi},
		&verifyStep{installer: bi},
		&extractStep{installer: bi},
		&installStep{installer: bi},
		&pathStep{installer: bi},
		&permissionsStep{installer: bi},
	}
}

// Install performs the binary installation
func (bi *BinaryInstaller) Install(ctx context.Context) error {
	bi.mu.Lock()
	bi.state = State{
		ID:        generateInstallID(),
		Method:    "binary",
		Version:   bi.config.Version,
		StartTime: time.Now(),
		Steps:     make([]StepResult, 0),
	}
	bi.mu.Unlock()

	// Save initial state
	if err := bi.stateManager.SaveState(ctx, bi.state); err != nil {
		return fmt.Errorf("failed to save initial state: %w", err)
	}

	// Create binary target
	binary := Binary{
		Platform: runtime.GOOS,
		Arch:     runtime.GOARCH,
		Version:  bi.config.Version,
		Path:     filepath.Join(bi.config.InstallDir, "tapio"),
	}

	// Execute each step in the pipeline
	totalSteps := len(bi.steps)
	for i, step := range bi.steps {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Apply middleware
		wrappedStep := bi.applyMiddleware(step)

		// Send progress update
		bi.sendProgress(Progress{
			CurrentStep:    wrappedStep.Name(),
			TotalSteps:     totalSteps,
			CompletedSteps: i,
			Message:        fmt.Sprintf("Running: %s", wrappedStep.Name()),
		})

		// Execute step
		startTime := time.Now()
		err := wrappedStep.Execute(ctx, binary)
		endTime := time.Now()

		// Record step result
		result := StepResult{
			StepName:  wrappedStep.Name(),
			Success:   err == nil,
			Error:     err,
			StartTime: startTime,
			EndTime:   endTime,
		}

		bi.mu.Lock()
		bi.state.Steps = append(bi.state.Steps, result)
		bi.state.LastUpdate = time.Now()
		bi.mu.Unlock()

		// Save state after each step
		if saveErr := bi.stateManager.SaveState(ctx, bi.state); saveErr != nil {
			// Log but don't fail installation
			if bi.config.Debug {
				fmt.Printf("Warning: failed to save state: %v\n", saveErr)
			}
		}

		if err != nil {
			return fmt.Errorf("step %s failed: %w", wrappedStep.Name(), err)
		}
	}

	// Clear state on success
	if err := bi.stateManager.ClearState(ctx); err != nil && bi.config.Debug {
		fmt.Printf("Warning: failed to clear state: %v\n", err)
	}

	close(bi.progressChan)
	return nil
}

// Rollback reverses the binary installation
func (bi *BinaryInstaller) Rollback(ctx context.Context) error {
	bi.mu.RLock()
	state := bi.state
	bi.mu.RUnlock()

	// Rollback in reverse order
	for i := len(state.Steps) - 1; i >= 0; i-- {
		stepResult := state.Steps[i]
		if !stepResult.Success {
			continue
		}

		// Find the corresponding step
		var step Step[Binary]
		for _, s := range bi.steps {
			if s.Name() == stepResult.StepName {
				step = s
				break
			}
		}

		if step != nil {
			binary := Binary{
				Platform: runtime.GOOS,
				Arch:     runtime.GOARCH,
				Version:  bi.config.Version,
				Path:     filepath.Join(bi.config.InstallDir, "tapio"),
			}

			if err := step.Rollback(ctx, binary); err != nil {
				return fmt.Errorf("failed to rollback step %s: %w", stepResult.StepName, err)
			}
		}
	}

	return bi.stateManager.ClearState(ctx)
}

// Validate checks the binary installation
func (bi *BinaryInstaller) Validate(ctx context.Context) error {
	binary := Binary{
		Platform: runtime.GOOS,
		Arch:     runtime.GOARCH,
		Version:  bi.config.Version,
		Path:     filepath.Join(bi.config.InstallDir, "tapio"),
	}

	for _, step := range bi.steps {
		if err := step.Validate(ctx, binary); err != nil {
			return fmt.Errorf("validation failed for %s: %w", step.Name(), err)
		}
	}

	return nil
}

// Progress returns the progress channel
func (bi *BinaryInstaller) Progress() <-chan Progress {
	return bi.progressChan
}

func (bi *BinaryInstaller) sendProgress(progress Progress) {
	select {
	case bi.progressChan <- progress:
	default:
		// Don't block if channel is full
	}
}

func (bi *BinaryInstaller) applyMiddleware(step Step[Binary]) Step[Binary] {
	result := step
	for _, mw := range bi.middleware {
		result = mw(result)
	}
	return result
}

// downloadStep handles binary download with resumption support
type downloadStep struct {
	installer *BinaryInstaller
}

func (s *downloadStep) Name() string {
	return "Download"
}

func (s *downloadStep) Execute(ctx context.Context, binary Binary) error {
	url := s.buildDownloadURL(binary)
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("tapio-%s-%s-%s.tmp", binary.Version, binary.Platform, binary.Arch))

	// Check if we can resume a previous download
	var startByte int64
	if stat, err := os.Stat(tempFile); err == nil {
		startByte = stat.Size()
	}

	// Create download request
	req := &Request{
		URL:        url,
		Method:     "GET",
		RangeStart: startByte,
	}

	// Execute with circuit breaker
	var resp *Response
	err := s.installer.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = s.installer.httpClient.Do(ctx, req)
		return err
	})
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	// Open file for writing
	file, err := os.OpenFile(tempFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer file.Close()

	// Create progress tracking reader
	progressReader := &progressReader{
		reader:     resp.Body,
		installer:  s.installer,
		totalBytes: resp.ContentLength,
		startByte:  startByte,
	}

	// Copy with progress tracking
	buf := s.installer.downloadPool.Get().([]byte)
	defer s.installer.downloadPool.Put(buf)

	_, err = io.CopyBuffer(file, progressReader, buf)
	if err != nil {
		return fmt.Errorf("download interrupted: %w", err)
	}

	// Update download state
	s.installer.mu.Lock()
	s.installer.state.DownloadState = DownloadState{
		URL:             url,
		LocalPath:       tempFile,
		BytesDownloaded: progressReader.bytesRead + startByte,
		TotalBytes:      resp.ContentLength,
	}
	s.installer.mu.Unlock()

	return nil
}

func (s *downloadStep) Rollback(ctx context.Context, binary Binary) error {
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("tapio-%s-%s-%s.tmp", binary.Version, binary.Platform, binary.Arch))
	return os.RemoveAll(tempFile)
}

func (s *downloadStep) Validate(ctx context.Context, binary Binary) error {
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("tapio-%s-%s-%s.tmp", binary.Version, binary.Platform, binary.Arch))
	_, err := os.Stat(tempFile)
	return err
}

func (s *downloadStep) buildDownloadURL(binary Binary) string {
	// Use mirror if available
	baseURL := "https://releases.tapio.io"
	if len(s.installer.config.Mirrors) > 0 {
		baseURL = s.installer.config.Mirrors[0]
	}
	return fmt.Sprintf("%s/%s/tapio-%s-%s", baseURL, binary.Version, binary.Platform, binary.Arch)
}

// progressReader tracks download progress
type progressReader struct {
	reader     io.Reader
	installer  *BinaryInstaller
	totalBytes int64
	bytesRead  int64
	lastUpdate time.Time
	lastBytes  int64
	startByte  int64
}

func (pr *progressReader) Read(p []byte) (n int, err error) {
	n, err = pr.reader.Read(p)
	atomic.AddInt64(&pr.bytesRead, int64(n))

	// Update progress every 100ms
	now := time.Now()
	if now.Sub(pr.lastUpdate) >= 100*time.Millisecond {
		bytesRead := atomic.LoadInt64(&pr.bytesRead)
		totalRead := bytesRead + pr.startByte

		// Calculate speed
		duration := now.Sub(pr.lastUpdate).Seconds()
		bytesInPeriod := bytesRead - pr.lastBytes
		speed := float64(bytesInPeriod) / duration

		// Estimate remaining time
		var estimatedTime time.Duration
		if speed > 0 && pr.totalBytes > 0 {
			remaining := pr.totalBytes - totalRead
			estimatedTime = time.Duration(float64(remaining)/speed) * time.Second
		}

		pr.installer.sendProgress(Progress{
			CurrentStep:     "Download",
			BytesDownloaded: totalRead,
			TotalBytes:      pr.totalBytes,
			Speed:           speed,
			EstimatedTime:   estimatedTime,
			Message:         fmt.Sprintf("Downloading: %s/s", formatBytes(int64(speed))),
		})

		pr.lastUpdate = now
		pr.lastBytes = bytesRead
	}

	return n, err
}

// verifyStep verifies the downloaded binary
type verifyStep struct {
	installer *BinaryInstaller
}

func (s *verifyStep) Name() string {
	return "Verify"
}

func (s *verifyStep) Execute(ctx context.Context, binary Binary) error {
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("tapio-%s-%s-%s.tmp", binary.Version, binary.Platform, binary.Arch))

	// Download checksum file
	checksumURL := s.buildChecksumURL(binary)
	expectedChecksum, err := s.downloadChecksum(ctx, checksumURL)
	if err != nil {
		return fmt.Errorf("failed to download checksum: %w", err)
	}

	// Calculate actual checksum
	actualChecksum, err := s.calculateChecksum(tempFile)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// Verify
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}

func (s *verifyStep) Rollback(ctx context.Context, binary Binary) error {
	return nil // Nothing to rollback
}

func (s *verifyStep) Validate(ctx context.Context, binary Binary) error {
	return nil // Validation happens during execution
}

func (s *verifyStep) buildChecksumURL(binary Binary) string {
	url := s.installer.steps[0].(*downloadStep).buildDownloadURL(binary)
	return url + ".sha256"
}

func (s *verifyStep) downloadChecksum(ctx context.Context, url string) (string, error) {
	req := &Request{
		URL:    url,
		Method: "GET",
	}

	var resp *Response
	err := s.installer.circuitBreaker.Execute(ctx, func() error {
		var err error
		resp, err = s.installer.httpClient.Do(ctx, req)
		return err
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	checksumBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(checksumBytes[:64]), nil // SHA256 is 64 hex characters
}

func (s *verifyStep) calculateChecksum(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// Additional steps would be implemented similarly...

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

func generateInstallID() string {
	return fmt.Sprintf("install-%d", time.Now().Unix())
}
