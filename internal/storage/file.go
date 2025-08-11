package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/yairfalse/vaino/pkg/types"
)

type FileStorage struct {
	dataDir    string
	cache      *snapshotCache
	workerPool *ioWorkerPool
}

// NewFileStorage creates a new file-based storage instance with optimizations
func NewFileStorage(dataDir string) (*FileStorage, error) {
	// Create necessary directories
	dirs := []string{
		filepath.Join(dataDir, "snapshots"),
		filepath.Join(dataDir, "baselines"),
		filepath.Join(dataDir, "history", "drift-reports"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	return &FileStorage{
		dataDir:    dataDir,
		cache:      newSnapshotCache(50), // Cache up to 50 snapshots
		workerPool: newIOWorkerPool(4),   // 4 concurrent I/O workers
	}, nil
}

func (fs *FileStorage) SaveSnapshot(snapshot *types.Snapshot) error {
	// Validate snapshot ID for security
	if err := validateResourceID(snapshot.ID); err != nil {
		return fmt.Errorf("invalid snapshot ID: %w", err)
	}

	filename := filepath.Join(fs.dataDir, "snapshots", snapshot.ID+".json")
	if err := validatePath(filename, fs.dataDir); err != nil {
		return fmt.Errorf("path traversal detected: %w", err)
	}

	return fs.saveJSON(filename, snapshot)
}

func (fs *FileStorage) LoadSnapshot(id string) (*types.Snapshot, error) {
	// Check cache first
	if snapshot := fs.cache.get(id); snapshot != nil {
		return snapshot, nil
	}

	// Validate snapshot ID for security
	if err := validateResourceID(id); err != nil {
		return nil, fmt.Errorf("invalid snapshot ID: %w", err)
	}

	filename := filepath.Join(fs.dataDir, "snapshots", id+".json")
	if err := validatePath(filename, fs.dataDir); err != nil {
		return nil, fmt.Errorf("path traversal detected: %w", err)
	}

	var snapshot types.Snapshot
	err := fs.loadJSON(filename, &snapshot)
	if err == nil {
		// Add to cache
		fs.cache.put(id, &snapshot)
	}
	return &snapshot, err
}

func (fs *FileStorage) ListSnapshots() ([]SnapshotInfo, error) {
	snapshotsDir := filepath.Join(fs.dataDir, "snapshots")
	entries, err := os.ReadDir(snapshotsDir)
	if err != nil {
		return nil, err
	}

	// Filter JSON files
	var jsonFiles []os.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			jsonFiles = append(jsonFiles, entry)
		}
	}

	// Load snapshots in parallel
	type result struct {
		info SnapshotInfo
		err  error
	}

	resultsChan := make(chan result, len(jsonFiles))
	var wg sync.WaitGroup

	// Use worker pool for parallel loading
	for _, entry := range jsonFiles {
		wg.Add(1)
		e := entry // Capture loop variable
		fs.workerPool.submit(func() {
			defer wg.Done()

			id := e.Name()[:len(e.Name())-5] // Remove .json
			snapshot, err := fs.LoadSnapshot(id)
			if err != nil {
				resultsChan <- result{err: err}
				return
			}

			stat, _ := e.Info()
			info := SnapshotInfo{
				ID:            snapshot.ID,
				Timestamp:     snapshot.Timestamp,
				Provider:      snapshot.Provider,
				ResourceCount: len(snapshot.Resources),
				FilePath:      filepath.Join(fs.dataDir, "snapshots", e.Name()),
				FileSize:      stat.Size(),
			}
			resultsChan <- result{info: info}
		})
	}

	// Close channel when done
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	var snapshots []SnapshotInfo
	for res := range resultsChan {
		if res.err == nil {
			snapshots = append(snapshots, res.info)
		}
	}

	return snapshots, nil
}

func (fs *FileStorage) DeleteSnapshot(id string) error {
	// Validate snapshot ID for security
	if err := validateResourceID(id); err != nil {
		return fmt.Errorf("invalid snapshot ID: %w", err)
	}

	filename := filepath.Join(fs.dataDir, "snapshots", id+".json")
	if err := validatePath(filename, fs.dataDir); err != nil {
		return fmt.Errorf("path traversal detected: %w", err)
	}

	return os.Remove(filename)
}

func (fs *FileStorage) SaveBaseline(baseline *types.Baseline) error {
	// Validate baseline ID for security
	if err := validateResourceID(baseline.ID); err != nil {
		return fmt.Errorf("invalid baseline ID: %w", err)
	}

	filename := filepath.Join(fs.dataDir, "baselines", baseline.ID+".json")
	if err := validatePath(filename, fs.dataDir); err != nil {
		return fmt.Errorf("path traversal detected: %w", err)
	}

	return fs.saveJSON(filename, baseline)
}

func (fs *FileStorage) LoadBaseline(id string) (*types.Baseline, error) {
	// Validate baseline ID for security
	if err := validateResourceID(id); err != nil {
		return nil, fmt.Errorf("invalid baseline ID: %w", err)
	}

	filename := filepath.Join(fs.dataDir, "baselines", id+".json")
	if err := validatePath(filename, fs.dataDir); err != nil {
		return nil, fmt.Errorf("path traversal detected: %w", err)
	}

	var baseline types.Baseline
	err := fs.loadJSON(filename, &baseline)
	return &baseline, err
}

func (fs *FileStorage) ListBaselines() ([]BaselineInfo, error) {
	baselinesDir := filepath.Join(fs.dataDir, "baselines")
	entries, err := os.ReadDir(baselinesDir)
	if err != nil {
		return nil, err
	}

	var baselines []BaselineInfo
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			id := entry.Name()[:len(entry.Name())-5] // Remove .json
			baseline, err := fs.LoadBaseline(id)
			if err != nil {
				continue // Skip invalid files
			}

			stat, _ := entry.Info()
			info := BaselineInfo{
				ID:          baseline.ID,
				Name:        baseline.Name,
				Description: baseline.Description,
				SnapshotID:  baseline.SnapshotID,
				CreatedAt:   baseline.CreatedAt,
				Tags:        baseline.Tags,
				Version:     baseline.Version,
				FilePath:    filepath.Join(fs.dataDir, "baselines", entry.Name()),
				FileSize:    stat.Size(),
			}
			baselines = append(baselines, info)
		}
	}

	return baselines, nil
}

func (fs *FileStorage) DeleteBaseline(id string) error {
	// Validate baseline ID for security
	if err := validateResourceID(id); err != nil {
		return fmt.Errorf("invalid baseline ID: %w", err)
	}

	filename := filepath.Join(fs.dataDir, "baselines", id+".json")
	if err := validatePath(filename, fs.dataDir); err != nil {
		return fmt.Errorf("path traversal detected: %w", err)
	}

	return os.Remove(filename)
}

// SaveDriftReport saves a drift report to disk
func (fs *FileStorage) SaveDriftReport(report *types.DriftReport) error {
	// Validate report ID for security
	if err := validateResourceID(report.ID); err != nil {
		return fmt.Errorf("invalid drift report ID: %w", err)
	}

	filename := filepath.Join(fs.dataDir, "history", "drift-reports", report.ID+".json")
	if err := validatePath(filename, fs.dataDir); err != nil {
		return fmt.Errorf("path traversal detected: %w", err)
	}

	return fs.saveJSON(filename, report)
}

// LoadDriftReport loads a drift report from disk
func (fs *FileStorage) LoadDriftReport(id string) (*types.DriftReport, error) {
	// Validate report ID for security
	if err := validateResourceID(id); err != nil {
		return nil, fmt.Errorf("invalid drift report ID: %w", err)
	}

	filename := filepath.Join(fs.dataDir, "history", "drift-reports", id+".json")
	if err := validatePath(filename, fs.dataDir); err != nil {
		return nil, fmt.Errorf("path traversal detected: %w", err)
	}

	var report types.DriftReport
	err := fs.loadJSON(filename, &report)
	return &report, err
}

// ListDriftReports returns metadata for all stored drift reports
func (fs *FileStorage) ListDriftReports() ([]DriftReportInfo, error) {
	reportsDir := filepath.Join(fs.dataDir, "history", "drift-reports")
	entries, err := os.ReadDir(reportsDir)
	if err != nil {
		return nil, err
	}

	var reports []DriftReportInfo
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			id := entry.Name()[:len(entry.Name())-5] // Remove .json
			report, err := fs.LoadDriftReport(id)
			if err != nil {
				continue // Skip invalid files
			}

			stat, _ := entry.Info()
			info := DriftReportInfo{
				ID:          report.ID,
				BaselineID:  report.BaselineID,
				SnapshotID:  report.CurrentID,
				CreatedAt:   report.Timestamp,
				ChangeCount: len(report.Changes),
				FilePath:    filepath.Join(fs.dataDir, "history", "drift-reports", entry.Name()),
				FileSize:    stat.Size(),
			}
			reports = append(reports, info)
		}
	}

	return reports, nil
}

// DeleteDriftReport removes a drift report from disk
func (fs *FileStorage) DeleteDriftReport(id string) error {
	// Validate report ID for security
	if err := validateResourceID(id); err != nil {
		return fmt.Errorf("invalid drift report ID: %w", err)
	}

	filename := filepath.Join(fs.dataDir, "history", "drift-reports", id+".json")
	if err := validatePath(filename, fs.dataDir); err != nil {
		return fmt.Errorf("path traversal detected: %w", err)
	}

	return os.Remove(filename)
}

func (fs *FileStorage) saveJSON(filename string, data interface{}) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func (fs *FileStorage) loadJSON(filename string, data interface{}) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get file size for safety check
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Limit file size to prevent memory exhaustion (50MB max)
	const maxFileSize = 50 * 1024 * 1024
	if fileInfo.Size() > maxFileSize {
		return fmt.Errorf("file size %d exceeds maximum allowed size %d", fileInfo.Size(), maxFileSize)
	}

	// Create a limited reader to prevent reading beyond the file size
	limitedReader := &limitedReader{
		R: file,
		N: fileInfo.Size(),
	}

	decoder := json.NewDecoder(limitedReader)
	// Disable unknown field detection to be more permissive but still safe
	decoder.DisallowUnknownFields()

	return decoder.Decode(data)
}

// validateResourceID validates that a resource ID is safe for file operations
func validateResourceID(id string) error {
	if id == "" {
		return fmt.Errorf("empty ID not allowed")
	}

	if len(id) > 255 {
		return fmt.Errorf("ID too long (max 255 characters)")
	}

	// Check for path traversal attempts
	if strings.Contains(id, "..") {
		return fmt.Errorf("ID contains path traversal characters")
	}

	if strings.Contains(id, "/") || strings.Contains(id, "\\") {
		return fmt.Errorf("ID contains path separators")
	}

	// Check for control characters and special characters that could be dangerous
	for _, r := range id {
		if r < 32 || r == 127 { // Control characters
			return fmt.Errorf("ID contains control characters")
		}
	}

	// Disallow reserved names on Windows and other systems
	reserved := []string{"CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9"}
	upperID := strings.ToUpper(id)
	for _, r := range reserved {
		if upperID == r {
			return fmt.Errorf("ID is a reserved system name")
		}
	}

	return nil
}

// validatePath ensures the resolved path is within the expected directory
func validatePath(filePath, expectedBase string) error {
	// Clean the paths to resolve any ".." elements
	cleanFilePath := filepath.Clean(filePath)
	cleanExpectedBase := filepath.Clean(expectedBase)

	// Convert to absolute paths for comparison
	absFilePath, err := filepath.Abs(cleanFilePath)
	if err != nil {
		return fmt.Errorf("failed to resolve file path: %w", err)
	}

	absExpectedBase, err := filepath.Abs(cleanExpectedBase)
	if err != nil {
		return fmt.Errorf("failed to resolve base path: %w", err)
	}

	// Ensure the file path starts with the expected base directory
	if !strings.HasPrefix(absFilePath, absExpectedBase) {
		return fmt.Errorf("path %s is outside expected directory %s", absFilePath, absExpectedBase)
	}

	return nil
}

// limitedReader is similar to io.LimitedReader but with additional safety features
type limitedReader struct {
	R io.Reader
	N int64 // max bytes remaining
}

func (l *limitedReader) Read(p []byte) (n int, err error) {
	if l.N <= 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > l.N {
		p = p[0:l.N]
	}
	n, err = l.R.Read(p)
	l.N -= int64(n)
	return
}

// snapshotCache provides an LRU cache for frequently accessed snapshots
type snapshotCache struct {
	mu      sync.RWMutex
	cache   map[string]*cacheEntry
	order   []string
	maxSize int
}

type cacheEntry struct {
	snapshot    *types.Snapshot
	accessCount int
}

func newSnapshotCache(maxSize int) *snapshotCache {
	return &snapshotCache{
		cache:   make(map[string]*cacheEntry),
		order:   make([]string, 0, maxSize),
		maxSize: maxSize,
	}
}

func (c *snapshotCache) get(id string) *types.Snapshot {
	c.mu.RLock()
	entry, exists := c.cache[id]
	c.mu.RUnlock()

	if !exists {
		return nil
	}

	c.mu.Lock()
	entry.accessCount++
	// Move to front of order
	c.moveToFront(id)
	c.mu.Unlock()

	return entry.snapshot
}

func (c *snapshotCache) put(id string, snapshot *types.Snapshot) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.cache[id]; exists {
		c.cache[id].snapshot = snapshot
		c.moveToFront(id)
		return
	}

	// Add new entry
	c.cache[id] = &cacheEntry{
		snapshot:    snapshot,
		accessCount: 1,
	}
	c.order = append([]string{id}, c.order...)

	// Evict if over capacity
	if len(c.order) > c.maxSize {
		evictID := c.order[len(c.order)-1]
		c.order = c.order[:len(c.order)-1]
		delete(c.cache, evictID)
	}
}

func (c *snapshotCache) moveToFront(id string) {
	for i, oid := range c.order {
		if oid == id {
			c.order = append([]string{id}, append(c.order[:i], c.order[i+1:]...)...)
			return
		}
	}
}

// ioWorkerPool manages a pool of workers for parallel I/O operations
type ioWorkerPool struct {
	workers int
	tasks   chan func()
	wg      sync.WaitGroup
}

func newIOWorkerPool(workers int) *ioWorkerPool {
	pool := &ioWorkerPool{
		workers: workers,
		tasks:   make(chan func(), workers*2),
	}

	// Start workers
	for i := 0; i < workers; i++ {
		go pool.worker()
	}

	return pool
}

func (p *ioWorkerPool) worker() {
	for task := range p.tasks {
		task()
	}
}

func (p *ioWorkerPool) submit(task func()) {
	p.tasks <- task
}

func (p *ioWorkerPool) shutdown() {
	close(p.tasks)
}
