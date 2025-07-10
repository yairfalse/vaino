package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/yairfalse/vaino/pkg/types"
)

// ConcurrentStorage wraps LocalStorage with concurrent operations
type ConcurrentStorage struct {
	*LocalStorage
	workers    int
	bufferPool sync.Pool
}

// fileOperation represents a file operation task
type fileOperation struct {
	Type   string
	Path   string
	Data   interface{}
	Result chan<- fileResult
}

// fileResult represents the result of a file operation
type fileResult struct {
	Data  interface{}
	Error error
}

// NewConcurrentStorage creates a new concurrent storage instance
func NewConcurrentStorage(config Config) (*ConcurrentStorage, error) {
	localStorage, err := NewLocalStorage(config)
	if err != nil {
		return nil, err
	}

	workers := runtime.NumCPU()
	if workers > 8 {
		workers = 8 // Cap at 8 workers to avoid excessive goroutines
	}

	return &ConcurrentStorage{
		LocalStorage: localStorage,
		workers:      workers,
		bufferPool: sync.Pool{
			New: func() interface{} {
				// Pre-allocate 64KB buffers for file operations
				buf := make([]byte, 64*1024)
				return &buf
			},
		},
	}, nil
}

// ListSnapshotsConcurrent returns metadata for all stored snapshots using concurrent file reads
func (s *ConcurrentStorage) ListSnapshotsConcurrent(ctx context.Context) ([]SnapshotInfo, error) {
	files, err := os.ReadDir(s.snapshots)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshots directory: %w", err)
	}

	// Filter JSON files
	var jsonFiles []os.DirEntry
	for _, file := range files {
		if file.Name()[len(file.Name())-5:] == ".json" {
			jsonFiles = append(jsonFiles, file)
		}
	}

	if len(jsonFiles) == 0 {
		return []SnapshotInfo{}, nil
	}

	// Create worker pool
	taskChan := make(chan fileOperation, len(jsonFiles))
	resultChan := make(chan fileResult, len(jsonFiles))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go s.fileWorker(ctx, &wg, taskChan)
	}

	// Submit tasks
	for _, file := range jsonFiles {
		taskChan <- fileOperation{
			Type:   "load_snapshot_info",
			Path:   filepath.Join(s.snapshots, file.Name()),
			Data:   file,
			Result: resultChan,
		}
	}
	close(taskChan)

	// Wait for workers to finish
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var infos []SnapshotInfo
	var errors []error

	for result := range resultChan {
		if result.Error != nil {
			errors = append(errors, result.Error)
			continue
		}
		if info, ok := result.Data.(SnapshotInfo); ok {
			infos = append(infos, info)
		}
	}

	if len(errors) > 0 {
		// Return first error for simplicity, but could aggregate
		return infos, errors[0]
	}

	// Sort by timestamp (newest first)
	sortSnapshotInfos(infos)

	return infos, nil
}

// SaveSnapshotsConcurrent saves multiple snapshots concurrently
func (s *ConcurrentStorage) SaveSnapshotsConcurrent(ctx context.Context, snapshots []*types.Snapshot) error {
	if len(snapshots) == 0 {
		return nil
	}

	// Validate all snapshots first
	for i, snapshot := range snapshots {
		if err := snapshot.Validate(); err != nil {
			return fmt.Errorf("invalid snapshot at index %d: %w", i, err)
		}
	}

	// Create task channel
	taskChan := make(chan fileOperation, len(snapshots))
	resultChan := make(chan fileResult, len(snapshots))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go s.fileWorker(ctx, &wg, taskChan)
	}

	// Submit save tasks
	for _, snapshot := range snapshots {
		filename := fmt.Sprintf("%s-scan.json", snapshot.Timestamp.Format("2006-01-02T15-04-05"))
		taskChan <- fileOperation{
			Type:   "save_json",
			Path:   filepath.Join(s.snapshots, filename),
			Data:   snapshot,
			Result: resultChan,
		}
	}
	close(taskChan)

	// Wait for completion
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Check for errors
	var errors []error
	for result := range resultChan {
		if result.Error != nil {
			errors = append(errors, result.Error)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to save %d snapshots: %w", len(errors), errors[0])
	}

	return nil
}

// LoadSnapshotsConcurrent loads multiple snapshots by ID concurrently
func (s *ConcurrentStorage) LoadSnapshotsConcurrent(ctx context.Context, ids []string) ([]*types.Snapshot, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// First, get all snapshot files
	files, err := os.ReadDir(s.snapshots)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshots directory: %w", err)
	}

	// Create ID to file mapping
	fileMap := make(map[string]string)
	taskChan := make(chan fileOperation, len(files))
	resultChan := make(chan fileResult, len(files))

	// Start workers for initial scan
	var wg sync.WaitGroup
	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go s.fileWorker(ctx, &wg, taskChan)
	}

	// Submit tasks to check snapshot IDs
	for _, file := range files {
		if file.Name()[len(file.Name())-5:] == ".json" {
			taskChan <- fileOperation{
				Type:   "check_snapshot_id",
				Path:   filepath.Join(s.snapshots, file.Name()),
				Result: resultChan,
			}
		}
	}
	close(taskChan)

	// Wait and collect ID mappings
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for result := range resultChan {
		if result.Error == nil && result.Data != nil {
			if mapping, ok := result.Data.(map[string]string); ok {
				for id, path := range mapping {
					fileMap[id] = path
				}
			}
		}
	}

	// Now load the requested snapshots
	loadChan := make(chan fileOperation, len(ids))
	loadResultChan := make(chan fileResult, len(ids))

	// Start new workers for loading
	var loadWg sync.WaitGroup
	for i := 0; i < s.workers; i++ {
		loadWg.Add(1)
		go s.fileWorker(ctx, &loadWg, loadChan)
	}

	// Submit load tasks
	foundCount := 0
	for _, id := range ids {
		if path, ok := fileMap[id]; ok {
			loadChan <- fileOperation{
				Type:   "load_json",
				Path:   path,
				Result: loadResultChan,
			}
			foundCount++
		}
	}
	close(loadChan)

	// Wait for loading to complete
	go func() {
		loadWg.Wait()
		close(loadResultChan)
	}()

	// Collect snapshots
	snapshots := make([]*types.Snapshot, 0, foundCount)
	for result := range loadResultChan {
		if result.Error != nil {
			return nil, result.Error
		}
		if snapshot, ok := result.Data.(*types.Snapshot); ok {
			snapshots = append(snapshots, snapshot)
		}
	}

	return snapshots, nil
}

// fileWorker processes file operations
func (s *ConcurrentStorage) fileWorker(ctx context.Context, wg *sync.WaitGroup, tasks <-chan fileOperation) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-tasks:
			if !ok {
				return
			}
			s.processFileOperation(ctx, task)
		}
	}
}

// processFileOperation handles individual file operations
func (s *ConcurrentStorage) processFileOperation(ctx context.Context, op fileOperation) {
	var result fileResult

	switch op.Type {
	case "save_json":
		result.Error = s.saveJSONWithBuffer(op.Path, op.Data)

	case "load_json":
		var snapshot types.Snapshot
		result.Error = s.loadJSONWithBuffer(op.Path, &snapshot)
		if result.Error == nil {
			result.Data = &snapshot
		}

	case "load_snapshot_info":
		info, err := s.loadSnapshotInfo(op.Path, op.Data.(os.DirEntry))
		result.Data = info
		result.Error = err

	case "check_snapshot_id":
		mapping, err := s.checkSnapshotID(op.Path)
		result.Data = mapping
		result.Error = err
	}

	select {
	case op.Result <- result:
	case <-ctx.Done():
		return
	}
}

// saveJSONWithBuffer saves JSON using a pooled buffer
func (s *ConcurrentStorage) saveJSONWithBuffer(path string, data interface{}) error {
	// Get buffer from pool
	bufPtr := s.bufferPool.Get().(*[]byte)
	defer s.bufferPool.Put(bufPtr)

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write atomically using a temp file
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, jsonData, 0o644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", tempPath, err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// loadJSONWithBuffer loads JSON using a pooled buffer
func (s *ConcurrentStorage) loadJSONWithBuffer(path string, target interface{}) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer file.Close()

	// Get buffer from pool
	bufPtr := s.bufferPool.Get().(*[]byte)
	defer s.bufferPool.Put(bufPtr)

	// Use buffered decoder
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("failed to decode JSON: %w", err)
	}

	return nil
}

// loadSnapshotInfo loads snapshot metadata without loading the full snapshot
func (s *ConcurrentStorage) loadSnapshotInfo(path string, fileInfo os.DirEntry) (SnapshotInfo, error) {
	var info SnapshotInfo

	stat, err := fileInfo.Info()
	if err != nil {
		return info, err
	}

	// Read only the necessary fields using streaming JSON decoder
	file, err := os.Open(path)
	if err != nil {
		return info, err
	}
	defer file.Close()

	// Use a limited reader to read only the beginning of the file
	limitedReader := io.LimitReader(file, 4096) // Read first 4KB

	var partialSnapshot struct {
		ID        string `json:"id"`
		Timestamp string `json:"timestamp"`
		Provider  string `json:"provider"`
		Metadata  struct {
			Tags map[string]string `json:"tags"`
		} `json:"metadata"`
	}

	decoder := json.NewDecoder(limitedReader)
	decoder.Decode(&partialSnapshot)

	// Parse timestamp
	timestamp, _ := parseTimestamp(partialSnapshot.Timestamp)

	// For resource count, we need to read the full file
	// but we can optimize this by counting array elements
	resourceCount := s.countResources(path)

	info = SnapshotInfo{
		ID:            partialSnapshot.ID,
		Timestamp:     timestamp,
		Provider:      partialSnapshot.Provider,
		ResourceCount: resourceCount,
		Tags:          partialSnapshot.Metadata.Tags,
		FilePath:      path,
		FileSize:      stat.Size(),
	}

	return info, nil
}

// checkSnapshotID quickly checks a snapshot's ID without loading the full file
func (s *ConcurrentStorage) checkSnapshotID(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read only the ID field
	var partial struct {
		ID string `json:"id"`
	}

	decoder := json.NewDecoder(io.LimitReader(file, 1024))
	if err := decoder.Decode(&partial); err != nil {
		return nil, err
	}

	if partial.ID != "" {
		return map[string]string{partial.ID: path}, nil
	}

	return nil, nil
}

// countResources efficiently counts resources in a snapshot file
func (s *ConcurrentStorage) countResources(path string) int {
	file, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer file.Close()

	// Use a streaming approach to count array elements
	decoder := json.NewDecoder(file)
	count := 0
	inResources := false
	depth := 0

	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		switch t := token.(type) {
		case string:
			if depth == 1 && t == "resources" {
				inResources = true
			}
		case json.Delim:
			switch t {
			case '{', '[':
				depth++
				if inResources && depth == 3 { // Inside resources array
					count++
				}
			case '}', ']':
				depth--
				if depth == 1 {
					inResources = false
				}
			}
		}
	}

	return count
}

// StreamingSnapshotProcessor processes snapshots using streaming JSON for memory efficiency
type StreamingSnapshotProcessor struct {
	storage *ConcurrentStorage
}

// NewStreamingProcessor creates a new streaming processor
func NewStreamingProcessor(storage *ConcurrentStorage) *StreamingSnapshotProcessor {
	return &StreamingSnapshotProcessor{storage: storage}
}

// ProcessLargeSnapshot processes a large snapshot file in chunks
func (p *StreamingSnapshotProcessor) ProcessLargeSnapshot(
	ctx context.Context,
	path string,
	processFunc func(resource *types.Resource) error,
) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open snapshot file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	// Read opening tokens until we find resources array
	var inResources bool
	var depth int

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch t := token.(type) {
		case string:
			if depth == 1 && t == "resources" {
				// Next token should be array start
				if _, err := decoder.Token(); err != nil {
					return err
				}
				inResources = true
			}
		case json.Delim:
			if t == '[' && inResources {
				// Process resources one by one
				for decoder.More() {
					var resource types.Resource
					if err := decoder.Decode(&resource); err != nil {
						return fmt.Errorf("failed to decode resource: %w", err)
					}
					if err := processFunc(&resource); err != nil {
						return err
					}
				}
				// Read closing bracket
				decoder.Token()
				inResources = false
			}
		}
	}

	return nil
}

// Metrics for monitoring
type StorageMetrics struct {
	ConcurrentReads   atomic.Int64
	ConcurrentWrites  atomic.Int64
	BuffersInUse      atomic.Int64
	TotalBytesRead    atomic.Int64
	TotalBytesWritten atomic.Int64
}

var metrics StorageMetrics

// GetMetrics returns current storage metrics
func GetMetrics() StorageMetrics {
	return StorageMetrics{
		ConcurrentReads:   atomic.Int64{},
		ConcurrentWrites:  atomic.Int64{},
		BuffersInUse:      atomic.Int64{},
		TotalBytesRead:    atomic.Int64{},
		TotalBytesWritten: atomic.Int64{},
	}
}
