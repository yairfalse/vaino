package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AtomicWriter provides atomic file operations with backup/recovery
type AtomicWriter struct {
	mu       sync.RWMutex
	locks    map[string]*sync.RWMutex // per-file locks
	locksMu  sync.Mutex               // protects the locks map
	backupDir string
}

// NewAtomicWriter creates a new atomic writer
func NewAtomicWriter(backupDir string) *AtomicWriter {
	return &AtomicWriter{
		locks:     make(map[string]*sync.RWMutex),
		backupDir: backupDir,
	}
}

// WriteFile writes data to a file atomically with backup
func (w *AtomicWriter) WriteFile(filename string, data []byte, perm os.FileMode) error {
	// Get or create file-specific lock
	fileLock := w.getFileLock(filename)
	fileLock.Lock()
	defer fileLock.Unlock()

	// Create backup if file exists
	if err := w.createBackup(filename); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Write to temporary file first
	tempFile := filename + ".tmp." + generateTempSuffix()
	
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write data to temp file
	if err := os.WriteFile(tempFile, data, perm); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Verify written data integrity
	if err := w.verifyFileIntegrity(tempFile, data); err != nil {
		os.Remove(tempFile) // Clean up on verification failure
		return fmt.Errorf("file integrity check failed: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempFile, filename); err != nil {
		os.Remove(tempFile) // Clean up on rename failure
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// ReadFile reads a file with corruption detection and recovery
func (w *AtomicWriter) ReadFile(filename string) ([]byte, error) {
	fileLock := w.getFileLock(filename)
	fileLock.RLock()
	defer fileLock.RUnlock()

	data, err := os.ReadFile(filename)
	if err != nil {
		// Try to recover from backup if main file is missing/corrupted
		if os.IsNotExist(err) {
			return w.recoverFromBackup(filename)
		}
		return nil, err
	}

	// Basic corruption check - ensure file is not empty and has valid content
	if len(data) == 0 {
		return w.recoverFromBackup(filename)
	}

	return data, nil
}

// createBackup creates a backup of the existing file
func (w *AtomicWriter) createBackup(filename string) error {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil // No file to backup
	}

	if w.backupDir == "" {
		return nil // Backups disabled
	}

	// Ensure backup directory exists
	if err := os.MkdirAll(w.backupDir, 0755); err != nil {
		return err
	}

	// Create backup filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	backupName := fmt.Sprintf("%s.%s.backup", filepath.Base(filename), timestamp)
	backupPath := filepath.Join(w.backupDir, backupName)

	// Copy file to backup location
	return w.copyFile(filename, backupPath)
}

// recoverFromBackup attempts to recover a file from its most recent backup
func (w *AtomicWriter) recoverFromBackup(filename string) ([]byte, error) {
	if w.backupDir == "" {
		return nil, fmt.Errorf("no backup directory configured")
	}

	// Find most recent backup
	pattern := filepath.Join(w.backupDir, filepath.Base(filename)+".*.backup")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return nil, fmt.Errorf("no backup found for %s", filename)
	}

	// Get the most recent backup (assumes sorted by timestamp)
	var mostRecent string
	var mostRecentTime time.Time
	
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		if info.ModTime().After(mostRecentTime) {
			mostRecent = match
			mostRecentTime = info.ModTime()
		}
	}

	if mostRecent == "" {
		return nil, fmt.Errorf("no valid backup found")
	}

	// Read backup data
	data, err := os.ReadFile(mostRecent)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup: %w", err)
	}

	// Restore the backup to original location
	if err := w.WriteFile(filename, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to restore backup: %w", err)
	}

	return data, nil
}

// getFileLock gets or creates a lock for a specific file
func (w *AtomicWriter) getFileLock(filename string) *sync.RWMutex {
	w.locksMu.Lock()
	defer w.locksMu.Unlock()

	if lock, exists := w.locks[filename]; exists {
		return lock
	}

	lock := &sync.RWMutex{}
	w.locks[filename] = lock
	return lock
}

// verifyFileIntegrity verifies that written data matches expected data
func (w *AtomicWriter) verifyFileIntegrity(filename string, expectedData []byte) error {
	actualData, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	expectedHash := sha256.Sum256(expectedData)
	actualHash := sha256.Sum256(actualData)

	if expectedHash != actualHash {
		return fmt.Errorf("file integrity check failed: hash mismatch")
	}

	return nil
}

// copyFile copies a file from src to dst
func (w *AtomicWriter) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// generateTempSuffix generates a unique suffix for temporary files
func generateTempSuffix() string {
	timestamp := time.Now().UnixNano()
	hash := sha256.Sum256([]byte(fmt.Sprintf("%d", timestamp)))
	return hex.EncodeToString(hash[:4]) // Use first 4 bytes for brevity
}

// CleanupBackups removes old backup files beyond the retention limit
func (w *AtomicWriter) CleanupBackups(maxAge time.Duration, maxCount int) error {
	if w.backupDir == "" {
		return nil
	}

	entries, err := os.ReadDir(w.backupDir)
	if err != nil {
		return err
	}

	// Group backups by original filename
	backupGroups := make(map[string][]os.DirEntry)
	for _, entry := range entries {
		if !entry.Type().IsRegular() || filepath.Ext(entry.Name()) != ".backup" {
			continue
		}

		// Extract original filename from backup name (filename.timestamp.backup)
		name := entry.Name()
		parts := filepath.SplitList(name[:len(name)-7]) // Remove .backup
		if len(parts) >= 2 {
			originalName := parts[0]
			backupGroups[originalName] = append(backupGroups[originalName], entry)
		}
	}

	// Clean up each group
	for _, backups := range backupGroups {
		if err := w.cleanupBackupGroup(backups, maxAge, maxCount); err != nil {
			return err
		}
	}

	return nil
}

// cleanupBackupGroup cleans up backups for a specific file
func (w *AtomicWriter) cleanupBackupGroup(backups []os.DirEntry, maxAge time.Duration, maxCount int) error {
	now := time.Now()
	
	// Sort by modification time (newest first)
	// Note: This is a simplified sort - in production, you'd want proper timestamp parsing
	
	var toDelete []string
	
	for i, backup := range backups {
		backupPath := filepath.Join(w.backupDir, backup.Name())
		info, err := backup.Info()
		if err != nil {
			continue
		}

		// Delete if too old
		if maxAge > 0 && now.Sub(info.ModTime()) > maxAge {
			toDelete = append(toDelete, backupPath)
			continue
		}

		// Delete if exceeding count limit (keep maxCount newest)
		if maxCount > 0 && i >= maxCount {
			toDelete = append(toDelete, backupPath)
		}
	}

	// Remove files marked for deletion
	for _, path := range toDelete {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to remove backup %s: %w", path, err)
		}
	}

	return nil
}

// StorageSpaceManager manages storage space and cleanup
type StorageSpaceManager struct {
	writer    *AtomicWriter
	threshold int64 // bytes
}

// NewStorageSpaceManager creates a new storage space manager
func NewStorageSpaceManager(writer *AtomicWriter, threshold int64) *StorageSpaceManager {
	return &StorageSpaceManager{
		writer:    writer,
		threshold: threshold,
	}
}

// CheckAndCleanup checks storage space and performs cleanup if needed
func (s *StorageSpaceManager) CheckAndCleanup(dataDir string) error {
	// Get directory size
	size, err := s.getDirSize(dataDir)
	if err != nil {
		return err
	}

	if size > s.threshold {
		// Perform cleanup
		return s.performCleanup(dataDir, size)
	}

	return nil
}

// getDirSize calculates the total size of a directory
func (s *StorageSpaceManager) getDirSize(dir string) (int64, error) {
	var size int64
	
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	
	return size, err
}

// performCleanup performs storage cleanup operations
func (s *StorageSpaceManager) performCleanup(dataDir string, currentSize int64) error {
	// Strategy: Remove old snapshots first, then old backups
	
	// Clean up old backups first
	if err := s.writer.CleanupBackups(30*24*time.Hour, 10); err != nil {
		return fmt.Errorf("backup cleanup failed: %w", err)
	}

	// If still over threshold, remove old snapshots
	newSize, err := s.getDirSize(dataDir)
	if err != nil {
		return err
	}

	if newSize > s.threshold {
		return s.cleanupOldSnapshots(dataDir)
	}

	return nil
}

// cleanupOldSnapshots removes old snapshot files
func (s *StorageSpaceManager) cleanupOldSnapshots(dataDir string) error {
	snapshotDir := filepath.Join(dataDir, "snapshots")
	
	entries, err := os.ReadDir(snapshotDir)
	if err != nil {
		return err
	}

	// Sort by modification time and remove oldest files
	// This is a simplified implementation
	var oldFiles []string
	
	for _, entry := range entries {
		if entry.Type().IsRegular() {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			
			// Remove files older than 7 days
			if time.Since(info.ModTime()) > 7*24*time.Hour {
				oldFiles = append(oldFiles, filepath.Join(snapshotDir, entry.Name()))
			}
		}
	}

	// Remove old files
	for _, file := range oldFiles {
		if err := os.Remove(file); err != nil {
			return fmt.Errorf("failed to remove old snapshot %s: %w", file, err)
		}
	}

	return nil
}