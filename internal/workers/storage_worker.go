package workers

import (
	"compress/gzip"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yairfalse/wgo/pkg/types"
)

// StorageOperation represents a storage operation
type StorageOperation struct {
	Type     StorageOperationType
	Data     interface{}
	FilePath string
	Options  StorageOptions
}

// StorageOperationType defines the type of storage operation
type StorageOperationType int

const (
	StorageOpSave StorageOperationType = iota
	StorageOpLoad
	StorageOpDelete
	StorageOpCompress
	StorageOpCleanup
	StorageOpValidate
)

// StorageOptions configures storage behavior
type StorageOptions struct {
	Compression   bool
	Encryption    bool
	Validation    bool
	Backup        bool
	Checksum      bool
	Timeout       time.Duration
	RetryCount    int
	BufferSize    int
	ChunkSize     int64
}

// StorageResult holds the result of a storage operation
type StorageResult struct {
	Operation   StorageOperation
	Success     bool
	Error       error
	BytesRead   int64
	BytesWritten int64
	Duration    time.Duration
	WorkerID    int
	Checksum    string
}

// StorageJob represents a storage job
type StorageJob struct {
	Operation  StorageOperation
	ResultChan chan<- StorageResult
	Priority   int
}

// ConcurrentStorageManager manages concurrent storage operations
type ConcurrentStorageManager struct {
	workerCount     int
	jobChan         chan StorageJob
	workers         []*storageWorker
	
	// Configuration
	bufferSize      int
	operationTimeout time.Duration
	maxFileSize     int64
	
	// State management
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	
	// Metrics
	totalOperations int64
	totalBytes      int64
	totalErrors     int64
	totalTime       time.Duration
	mu              sync.RWMutex
	
	// Storage components
	compressor      *FileCompressor
	encryptor       *FileEncryptor
	validator       *FileValidator
	cleaner         *StorageCleaner
	
	// Performance optimization
	fileCache       *FileCache
	operationPool   *OperationPool
	bufferPool      *BufferPool
}

// storageWorker represents a single storage worker
type storageWorker struct {
	id       int
	manager  *ConcurrentStorageManager
	stats    storageWorkerStats
	executor *OperationExecutor
}

// storageWorkerStats holds statistics for a storage worker
type storageWorkerStats struct {
	operationsCompleted int64
	bytesProcessed      int64
	errors              int64
	totalTime           time.Duration
	lastActive          time.Time
	mu                  sync.RWMutex
}

// OperationExecutor executes storage operations
type OperationExecutor struct {
	compressor *FileCompressor
	encryptor  *FileEncryptor
	validator  *FileValidator
	bufferPool *BufferPool
}

// FileCompressor handles file compression
type FileCompressor struct {
	compressionLevel int
	bufferSize       int
	pool             sync.Pool
}

// FileEncryptor handles file encryption
type FileEncryptor struct {
	keyFile    string
	algorithm  string
	bufferSize int
}

// FileValidator validates file integrity
type FileValidator struct {
	checksumCache map[string]string
	mu            sync.RWMutex
}

// StorageCleaner handles background cleanup
type StorageCleaner struct {
	cleanupInterval time.Duration
	retentionPeriod time.Duration
	ticker          *time.Ticker
	stopChan        chan struct{}
}

// FileCache caches file metadata
type FileCache struct {
	cache   map[string]FileCacheEntry
	mu      sync.RWMutex
	maxSize int
	ttl     time.Duration
}

// FileCacheEntry represents a cached file entry
type FileCacheEntry struct {
	Size      int64
	ModTime   time.Time
	Checksum  string
	CacheTime time.Time
}

// OperationPool pools operation objects
type OperationPool struct {
	pool sync.Pool
}

// BufferPool manages byte buffers
type BufferPool struct {
	pool sync.Pool
	size int
}

// NewConcurrentStorageManager creates a new concurrent storage manager
func NewConcurrentStorageManager(opts ...StorageManagerOption) *ConcurrentStorageManager {
	csm := &ConcurrentStorageManager{
		workerCount:      runtime.NumCPU(),
		bufferSize:       100,
		operationTimeout: 60 * time.Second,
		maxFileSize:      1024 * 1024 * 1024, // 1GB
		compressor:       NewFileCompressor(),
		encryptor:        NewFileEncryptor(),
		validator:        NewFileValidator(),
		cleaner:          NewStorageCleaner(),
		fileCache:        NewFileCache(1000, 10*time.Minute),
		operationPool:    NewOperationPool(),
		bufferPool:       NewBufferPool(64 * 1024), // 64KB buffers
	}
	
	// Apply options
	for _, opt := range opts {
		opt(csm)
	}
	
	// Initialize channels
	csm.jobChan = make(chan StorageJob, csm.bufferSize)
	
	// Create context
	csm.ctx, csm.cancel = context.WithCancel(context.Background())
	
	// Initialize workers
	csm.workers = make([]*storageWorker, csm.workerCount)
	for i := 0; i < csm.workerCount; i++ {
		csm.workers[i] = &storageWorker{
			id:      i,
			manager: csm,
			executor: &OperationExecutor{
				compressor: csm.compressor,
				encryptor:  csm.encryptor,
				validator:  csm.validator,
				bufferPool: csm.bufferPool,
			},
		}
	}
	
	return csm
}

// StorageManagerOption configures the storage manager
type StorageManagerOption func(*ConcurrentStorageManager)

// WithStorageWorkerCount sets the number of worker goroutines
func WithStorageWorkerCount(count int) StorageManagerOption {
	return func(csm *ConcurrentStorageManager) {
		if count > 0 {
			csm.workerCount = count
		}
	}
}

// WithStorageBufferSize sets the buffer size for job channels
func WithStorageBufferSize(size int) StorageManagerOption {
	return func(csm *ConcurrentStorageManager) {
		if size > 0 {
			csm.bufferSize = size
		}
	}
}

// WithStorageTimeout sets the operation timeout
func WithStorageTimeout(timeout time.Duration) StorageManagerOption {
	return func(csm *ConcurrentStorageManager) {
		csm.operationTimeout = timeout
	}
}

// WithStorageMaxFileSize sets the maximum file size
func WithStorageMaxFileSize(size int64) StorageManagerOption {
	return func(csm *ConcurrentStorageManager) {
		csm.maxFileSize = size
	}
}

// SaveSnapshotConcurrent saves a snapshot concurrently
func (csm *ConcurrentStorageManager) SaveSnapshotConcurrent(snapshot *types.Snapshot) error {
	if snapshot == nil {
		return fmt.Errorf("snapshot cannot be nil")
	}
	
	// Start manager if not already running
	if err := csm.start(); err != nil {
		return fmt.Errorf("failed to start storage manager: %w", err)
	}
	
	// Create save operation
	filePath := filepath.Join("snapshots", fmt.Sprintf("%s.json", snapshot.ID))
	operation := StorageOperation{
		Type:     StorageOpSave,
		Data:     snapshot,
		FilePath: filePath,
		Options: StorageOptions{
			Compression: true,
			Validation:  true,
			Backup:      true,
			Checksum:    true,
			Timeout:     csm.operationTimeout,
			RetryCount:  2,
			BufferSize:  64 * 1024,
			ChunkSize:   1024 * 1024,
		},
	}
	
	// Execute operation
	result, err := csm.executeOperation(operation)
	if err != nil {
		return fmt.Errorf("failed to save snapshot: %w", err)
	}
	
	if !result.Success {
		return fmt.Errorf("save operation failed: %v", result.Error)
	}
	
	// Start background cleanup
	csm.scheduleCleanup()
	
	return nil
}

// LoadSnapshotConcurrent loads a snapshot concurrently
func (csm *ConcurrentStorageManager) LoadSnapshotConcurrent(snapshotID string) (*types.Snapshot, error) {
	if snapshotID == "" {
		return nil, fmt.Errorf("snapshot ID cannot be empty")
	}
	
	// Start manager if not already running
	if err := csm.start(); err != nil {
		return nil, fmt.Errorf("failed to start storage manager: %w", err)
	}
	
	// Create load operation
	filePath := filepath.Join("snapshots", fmt.Sprintf("%s.json", snapshotID))
	operation := StorageOperation{
		Type:     StorageOpLoad,
		FilePath: filePath,
		Options: StorageOptions{
			Compression: true,
			Validation:  true,
			Timeout:     csm.operationTimeout,
			RetryCount:  2,
			BufferSize:  64 * 1024,
		},
	}
	
	// Execute operation
	result, err := csm.executeOperation(operation)
	if err != nil {
		return nil, fmt.Errorf("failed to load snapshot: %w", err)
	}
	
	if !result.Success {
		return nil, fmt.Errorf("load operation failed: %v", result.Error)
	}
	
	// Extract snapshot from result
	snapshot, ok := result.Operation.Data.(*types.Snapshot)
	if !ok {
		return nil, fmt.Errorf("invalid snapshot data type")
	}
	
	return snapshot, nil
}

// start initializes and starts the storage manager
func (csm *ConcurrentStorageManager) start() error {
	// Start workers
	for i := 0; i < csm.workerCount; i++ {
		csm.wg.Add(1)
		go csm.worker(i)
	}
	
	// Start background cleaner
	csm.cleaner.Start(csm.ctx)
	
	return nil
}

// stop gracefully shuts down the storage manager
func (csm *ConcurrentStorageManager) stop() {
	// Cancel context
	csm.cancel()
	
	// Close job channel
	close(csm.jobChan)
	
	// Wait for workers to finish
	done := make(chan struct{})
	go func() {
		csm.wg.Wait()
		close(done)
	}()
	
	// Wait for completion or timeout
	select {
	case <-done:
		// Normal shutdown
	case <-time.After(10 * time.Second):
		// Force shutdown
		fmt.Println("Warning: Storage manager shutdown timed out")
	}
	
	// Stop cleaner
	csm.cleaner.Stop()
}

// worker processes storage jobs
func (csm *ConcurrentStorageManager) worker(workerID int) {
	defer csm.wg.Done()
	
	worker := csm.workers[workerID]
	
	for {
		select {
		case <-csm.ctx.Done():
			return
		case job, ok := <-csm.jobChan:
			if !ok {
				return
			}
			
			// Process the job
			result := csm.processStorageJob(job, workerID)
			
			// Update worker stats
			worker.updateStats(result)
			
			// Send result
			select {
			case job.ResultChan <- result:
			case <-csm.ctx.Done():
				return
			}
		}
	}
}

// processStorageJob processes a single storage job
func (csm *ConcurrentStorageManager) processStorageJob(job StorageJob, workerID int) StorageResult {
	startTime := time.Now()
	
	result := StorageResult{
		Operation: job.Operation,
		WorkerID:  workerID,
		Duration:  0,
	}
	
	// Execute operation with timeout
	ctx, cancel := context.WithTimeout(csm.ctx, job.Operation.Options.Timeout)
	defer cancel()
	
	worker := csm.workers[workerID]
	success, bytesRead, bytesWritten, checksum, err := worker.executor.Execute(ctx, job.Operation)
	
	result.Duration = time.Since(startTime)
	result.Success = success
	result.BytesRead = bytesRead
	result.BytesWritten = bytesWritten
	result.Checksum = checksum
	result.Error = err
	
	// Update metrics
	atomic.AddInt64(&csm.totalOperations, 1)
	atomic.AddInt64(&csm.totalBytes, bytesRead+bytesWritten)
	if err != nil {
		atomic.AddInt64(&csm.totalErrors, 1)
	}
	
	return result
}

// executeOperation executes a storage operation
func (csm *ConcurrentStorageManager) executeOperation(operation StorageOperation) (StorageResult, error) {
	resultChan := make(chan StorageResult, 1)
	
	job := StorageJob{
		Operation:  operation,
		ResultChan: resultChan,
		Priority:   100,
	}
	
	// Submit job
	select {
	case csm.jobChan <- job:
	case <-csm.ctx.Done():
		return StorageResult{}, fmt.Errorf("storage manager context cancelled")
	}
	
	// Wait for result
	select {
	case result := <-resultChan:
		return result, nil
	case <-time.After(csm.operationTimeout * 2):
		return StorageResult{}, fmt.Errorf("timeout waiting for operation result")
	}
}

// scheduleCleanup schedules background cleanup
func (csm *ConcurrentStorageManager) scheduleCleanup() {
	go func() {
		cleanupOp := StorageOperation{
			Type: StorageOpCleanup,
			Options: StorageOptions{
				Timeout: 30 * time.Second,
			},
		}
		
		resultChan := make(chan StorageResult, 1)
		job := StorageJob{
			Operation:  cleanupOp,
			ResultChan: resultChan,
			Priority:   10, // Low priority
		}
		
		select {
		case csm.jobChan <- job:
		case <-csm.ctx.Done():
			return
		}
		
		// Wait for cleanup to complete
		<-resultChan
	}()
}

// Execute executes a storage operation
func (oe *OperationExecutor) Execute(ctx context.Context, operation StorageOperation) (bool, int64, int64, string, error) {
	switch operation.Type {
	case StorageOpSave:
		return oe.executeSave(ctx, operation)
	case StorageOpLoad:
		return oe.executeLoad(ctx, operation)
	case StorageOpDelete:
		return oe.executeDelete(ctx, operation)
	case StorageOpCompress:
		return oe.executeCompress(ctx, operation)
	case StorageOpCleanup:
		return oe.executeCleanup(ctx, operation)
	case StorageOpValidate:
		return oe.executeValidate(ctx, operation)
	default:
		return false, 0, 0, "", fmt.Errorf("unknown operation type: %v", operation.Type)
	}
}

// executeSave executes a save operation
func (oe *OperationExecutor) executeSave(ctx context.Context, operation StorageOperation) (bool, int64, int64, string, error) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(operation.FilePath), 0755); err != nil {
		return false, 0, 0, "", fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Create temporary file
	tempFile := operation.FilePath + ".tmp"
	file, err := os.Create(tempFile)
	if err != nil {
		return false, 0, 0, "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer file.Close()
	
	var writer io.Writer = file
	var bytesWritten int64
	
	// Apply compression if requested
	if operation.Options.Compression {
		gzipWriter := gzip.NewWriter(file)
		defer gzipWriter.Close()
		writer = gzipWriter
	}
	
	// Serialize data
	encoder := json.NewEncoder(writer)
	if err := encoder.Encode(operation.Data); err != nil {
		os.Remove(tempFile)
		return false, 0, 0, "", fmt.Errorf("failed to encode data: %w", err)
	}
	
	// Get file size
	if info, err := file.Stat(); err == nil {
		bytesWritten = info.Size()
	}
	
	// Calculate checksum if requested
	var checksum string
	if operation.Options.Checksum {
		file.Seek(0, 0)
		hash := md5.New()
		if _, err := io.Copy(hash, file); err != nil {
			os.Remove(tempFile)
			return false, 0, 0, "", fmt.Errorf("failed to calculate checksum: %w", err)
		}
		checksum = fmt.Sprintf("%x", hash.Sum(nil))
	}
	
	// Close file
	file.Close()
	
	// Create backup if requested
	if operation.Options.Backup {
		if _, err := os.Stat(operation.FilePath); err == nil {
			backupPath := operation.FilePath + ".backup"
			if err := os.Rename(operation.FilePath, backupPath); err != nil {
				os.Remove(tempFile)
				return false, 0, 0, "", fmt.Errorf("failed to create backup: %w", err)
			}
		}
	}
	
	// Atomic rename
	if err := os.Rename(tempFile, operation.FilePath); err != nil {
		os.Remove(tempFile)
		return false, 0, 0, "", fmt.Errorf("failed to rename temp file: %w", err)
	}
	
	return true, 0, bytesWritten, checksum, nil
}

// executeLoad executes a load operation
func (oe *OperationExecutor) executeLoad(ctx context.Context, operation StorageOperation) (bool, int64, int64, string, error) {
	// Open file
	file, err := os.Open(operation.FilePath)
	if err != nil {
		return false, 0, 0, "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	
	var reader io.Reader = file
	var bytesRead int64
	
	// Get file size
	if info, err := file.Stat(); err == nil {
		bytesRead = info.Size()
	}
	
	// Handle compression
	if operation.Options.Compression {
		gzipReader, err := gzip.NewReader(file)
		if err != nil {
			return false, 0, 0, "", fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzipReader.Close()
		reader = gzipReader
	}
	
	// Deserialize data
	decoder := json.NewDecoder(reader)
	var snapshot types.Snapshot
	if err := decoder.Decode(&snapshot); err != nil {
		return false, 0, 0, "", fmt.Errorf("failed to decode data: %w", err)
	}
	
	// Store loaded data back in operation
	operation.Data = &snapshot
	
	return true, bytesRead, 0, "", nil
}

// executeDelete executes a delete operation
func (oe *OperationExecutor) executeDelete(ctx context.Context, operation StorageOperation) (bool, int64, int64, string, error) {
	if err := os.Remove(operation.FilePath); err != nil {
		return false, 0, 0, "", fmt.Errorf("failed to delete file: %w", err)
	}
	
	return true, 0, 0, "", nil
}

// executeCompress executes a compression operation
func (oe *OperationExecutor) executeCompress(ctx context.Context, operation StorageOperation) (bool, int64, int64, string, error) {
	// Implementation for compression
	return true, 0, 0, "", nil
}

// executeCleanup executes a cleanup operation
func (oe *OperationExecutor) executeCleanup(ctx context.Context, operation StorageOperation) (bool, int64, int64, string, error) {
	// Implementation for cleanup
	return true, 0, 0, "", nil
}

// executeValidate executes a validation operation
func (oe *OperationExecutor) executeValidate(ctx context.Context, operation StorageOperation) (bool, int64, int64, string, error) {
	// Implementation for validation
	return true, 0, 0, "", nil
}

// updateStats updates worker statistics
func (w *storageWorker) updateStats(result StorageResult) {
	w.stats.mu.Lock()
	defer w.stats.mu.Unlock()
	
	w.stats.lastActive = time.Now()
	w.stats.totalTime += result.Duration
	w.stats.operationsCompleted++
	w.stats.bytesProcessed += result.BytesRead + result.BytesWritten
	
	if result.Error != nil {
		w.stats.errors++
	}
}

// GetStats returns storage manager statistics
func (csm *ConcurrentStorageManager) GetStats() StorageManagerStats {
	csm.mu.RLock()
	defer csm.mu.RUnlock()
	
	stats := StorageManagerStats{
		TotalOperations: atomic.LoadInt64(&csm.totalOperations),
		TotalBytes:      atomic.LoadInt64(&csm.totalBytes),
		TotalErrors:     atomic.LoadInt64(&csm.totalErrors),
		WorkerCount:     csm.workerCount,
		WorkerStats:     make([]StorageWorkerStats, len(csm.workers)),
	}
	
	for i, worker := range csm.workers {
		worker.stats.mu.RLock()
		stats.WorkerStats[i] = StorageWorkerStats{
			WorkerID:            i,
			OperationsCompleted: worker.stats.operationsCompleted,
			BytesProcessed:      worker.stats.bytesProcessed,
			Errors:              worker.stats.errors,
			TotalTime:           worker.stats.totalTime,
			LastActive:          worker.stats.lastActive,
		}
		worker.stats.mu.RUnlock()
	}
	
	return stats
}

// StorageManagerStats holds storage manager statistics
type StorageManagerStats struct {
	TotalOperations int64
	TotalBytes      int64
	TotalErrors     int64
	WorkerCount     int
	WorkerStats     []StorageWorkerStats
}

// StorageWorkerStats holds individual worker statistics
type StorageWorkerStats struct {
	WorkerID            int
	OperationsCompleted int64
	BytesProcessed      int64
	Errors              int64
	TotalTime           time.Duration
	LastActive          time.Time
}

// NewFileCompressor creates a new file compressor
func NewFileCompressor() *FileCompressor {
	return &FileCompressor{
		compressionLevel: gzip.DefaultCompression,
		bufferSize:       64 * 1024,
	}
}

// NewFileEncryptor creates a new file encryptor
func NewFileEncryptor() *FileEncryptor {
	return &FileEncryptor{
		algorithm:  "AES-256-GCM",
		bufferSize: 64 * 1024,
	}
}

// NewFileValidator creates a new file validator
func NewFileValidator() *FileValidator {
	return &FileValidator{
		checksumCache: make(map[string]string),
	}
}

// NewStorageCleaner creates a new storage cleaner
func NewStorageCleaner() *StorageCleaner {
	return &StorageCleaner{
		cleanupInterval: 1 * time.Hour,
		retentionPeriod: 7 * 24 * time.Hour,
		stopChan:        make(chan struct{}),
	}
}

// Start starts the storage cleaner
func (sc *StorageCleaner) Start(ctx context.Context) {
	sc.ticker = time.NewTicker(sc.cleanupInterval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-sc.stopChan:
				return
			case <-sc.ticker.C:
				sc.cleanup()
			}
		}
	}()
}

// Stop stops the storage cleaner
func (sc *StorageCleaner) Stop() {
	if sc.ticker != nil {
		sc.ticker.Stop()
	}
	close(sc.stopChan)
}

// cleanup performs cleanup operations
func (sc *StorageCleaner) cleanup() {
	// Implementation for cleanup
}

// NewFileCache creates a new file cache
func NewFileCache(maxSize int, ttl time.Duration) *FileCache {
	return &FileCache{
		cache:   make(map[string]FileCacheEntry),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// NewOperationPool creates a new operation pool
func NewOperationPool() *OperationPool {
	return &OperationPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &StorageOperation{}
			},
		},
	}
}

// NewBufferPool creates a new buffer pool
func NewBufferPool(size int) *BufferPool {
	return &BufferPool{
		size: size,
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, size)
			},
		},
	}
}

// Get gets a buffer from the pool
func (bp *BufferPool) Get() []byte {
	return bp.pool.Get().([]byte)
}

// Put returns a buffer to the pool
func (bp *BufferPool) Put(buf []byte) {
	bp.pool.Put(buf)
}