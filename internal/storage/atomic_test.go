package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestAtomicWriter_WriteFile(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "vaino-atomic-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create backup directory
	backupDir := filepath.Join(tempDir, "backups")
	writer := NewAtomicWriter(backupDir)

	testFile := filepath.Join(tempDir, "test.txt")
	testData := []byte("Hello, World!")

	// Test writing a new file
	err = writer.WriteFile(testFile, testData, 0644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Verify file exists and has correct content
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(data) != string(testData) {
		t.Errorf("File content mismatch. Expected: %s, Got: %s", testData, data)
	}

	// Test overwriting existing file (should create backup)
	newData := []byte("Updated content")
	err = writer.WriteFile(testFile, newData, 0644)
	if err != nil {
		t.Fatalf("Failed to overwrite file: %v", err)
	}

	// Verify new content
	data, err = os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read updated file: %v", err)
	}

	if string(data) != string(newData) {
		t.Errorf("Updated file content mismatch. Expected: %s, Got: %s", newData, data)
	}

	// Verify backup was created
	backupFiles, err := filepath.Glob(filepath.Join(backupDir, "test.txt.*.backup"))
	if err != nil {
		t.Fatalf("Failed to check for backup files: %v", err)
	}

	if len(backupFiles) == 0 {
		t.Error("Expected backup file to be created")
	}
}

func TestAtomicWriter_ConcurrentWrites(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vaino-concurrent-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	writer := NewAtomicWriter("")
	testFile := filepath.Join(tempDir, "concurrent.txt")

	// Perform concurrent writes
	const numGoroutines = 10
	const numWrites = 5

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numWrites)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numWrites; j++ {
				data := []byte(fmt.Sprintf("Writer %d - Write %d", id, j))
				if err := writer.WriteFile(testFile, data, 0644); err != nil {
					errors <- err
					return
				}
				time.Sleep(time.Millisecond) // Small delay to encourage race conditions
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent write error: %v", err)
	}

	// Verify file exists and is readable
	if _, err := os.ReadFile(testFile); err != nil {
		t.Errorf("Failed to read file after concurrent writes: %v", err)
	}
}

func TestAtomicWriter_ReadFileWithRecovery(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vaino-recovery-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	backupDir := filepath.Join(tempDir, "backups")
	writer := NewAtomicWriter(backupDir)

	testFile := filepath.Join(tempDir, "recovery-test.txt")
	originalData := []byte("Original data")

	// Write original file and create backup
	err = writer.WriteFile(testFile, originalData, 0644)
	if err != nil {
		t.Fatalf("Failed to write original file: %v", err)
	}

	// Write updated file to create backup
	updatedData := []byte("Updated data")
	err = writer.WriteFile(testFile, updatedData, 0644)
	if err != nil {
		t.Fatalf("Failed to write updated file: %v", err)
	}

	// Corrupt the main file
	err = os.WriteFile(testFile, []byte{}, 0644) // Empty file
	if err != nil {
		t.Fatalf("Failed to corrupt file: %v", err)
	}

	// Try to read - should recover from backup
	data, err := writer.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read/recover file: %v", err)
	}

	if string(data) != string(updatedData) {
		t.Errorf("Recovery failed. Expected: %s, Got: %s", updatedData, data)
	}
}

func TestAtomicWriter_FileIntegrityCheck(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vaino-integrity-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	writer := NewAtomicWriter("")
	testFile := filepath.Join(tempDir, "integrity-test.txt")
	testData := []byte("Integrity test data")

	// Normal write should pass integrity check
	err = writer.WriteFile(testFile, testData, 0644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Verify integrity check
	err = writer.verifyFileIntegrity(testFile, testData)
	if err != nil {
		t.Errorf("Integrity check failed for valid file: %v", err)
	}

	// Test with mismatched data
	wrongData := []byte("Wrong data")
	err = writer.verifyFileIntegrity(testFile, wrongData)
	if err == nil {
		t.Error("Expected integrity check to fail with wrong data")
	}
}

func TestStorageSpaceManager_CheckAndCleanup(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vaino-space-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	backupDir := filepath.Join(tempDir, "backups")
	writer := NewAtomicWriter(backupDir)

	// Create space manager with very small threshold (1 KB)
	threshold := int64(1024)
	spaceManager := NewStorageSpaceManager(writer, threshold)

	// Create some large files to exceed threshold
	dataDir := filepath.Join(tempDir, "data")
	snapshotDir := filepath.Join(dataDir, "snapshots")
	err = os.MkdirAll(snapshotDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create snapshot dir: %v", err)
	}

	// Create files that exceed threshold
	largeData := make([]byte, 2048) // 2KB
	for i := 0; i < 3; i++ {
		filename := filepath.Join(snapshotDir, fmt.Sprintf("large-file-%d.json", i))
		err = os.WriteFile(filename, largeData, 0644)
		if err != nil {
			t.Fatalf("Failed to create large file: %v", err)
		}
		// Make some files old
		if i < 2 {
			oldTime := time.Now().Add(-10 * 24 * time.Hour) // 10 days old
			os.Chtimes(filename, oldTime, oldTime)
		}
	}

	// Check initial size
	initialSize, err := spaceManager.getDirSize(dataDir)
	if err != nil {
		t.Fatalf("Failed to get directory size: %v", err)
	}

	if initialSize <= threshold {
		t.Errorf("Expected directory size (%d) to exceed threshold (%d)", initialSize, threshold)
	}

	// Run cleanup
	err = spaceManager.CheckAndCleanup(dataDir)
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Check final size
	finalSize, err := spaceManager.getDirSize(dataDir)
	if err != nil {
		t.Fatalf("Failed to get directory size after cleanup: %v", err)
	}

	if finalSize >= initialSize {
		t.Error("Expected cleanup to reduce directory size")
	}
}

func TestAtomicWriter_CleanupBackups(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vaino-backup-cleanup-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	backupDir := filepath.Join(tempDir, "backups")
	writer := NewAtomicWriter(backupDir)

	testFile := filepath.Join(tempDir, "cleanup-test.txt")

	// Create multiple backup versions by writing multiple times
	for i := 0; i < 5; i++ {
		data := []byte(fmt.Sprintf("Version %d", i))
		err = writer.WriteFile(testFile, data, 0644)
		if err != nil {
			t.Fatalf("Failed to write version %d: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Check backup files were created
	backupFiles, err := filepath.Glob(filepath.Join(backupDir, "*.backup"))
	if err != nil {
		t.Fatalf("Failed to list backup files: %v", err)
	}

	initialBackupCount := len(backupFiles)
	if initialBackupCount == 0 {
		t.Fatal("No backup files were created")
	}

	// Cleanup with max count of 2
	err = writer.CleanupBackups(24*time.Hour, 2)
	if err != nil {
		t.Fatalf("Backup cleanup failed: %v", err)
	}

	// Check remaining backups
	remainingFiles, err := filepath.Glob(filepath.Join(backupDir, "*.backup"))
	if err != nil {
		t.Fatalf("Failed to list remaining backup files: %v", err)
	}

	if len(remainingFiles) > 2 {
		t.Errorf("Expected at most 2 backup files, got %d", len(remainingFiles))
	}
}

func TestAtomicWriter_ErrorHandling(t *testing.T) {
	writer := NewAtomicWriter("")

	// Test writing to invalid path
	err := writer.WriteFile("/invalid/path/file.txt", []byte("test"), 0644)
	if err == nil {
		t.Error("Expected error when writing to invalid path")
	}

	// Test reading non-existent file with no backup
	_, err = writer.ReadFile("/non/existent/file.txt")
	if err == nil {
		t.Error("Expected error when reading non-existent file")
	}
}

// Benchmark tests
func BenchmarkAtomicWriter_WriteFile(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "vaino-benchmark")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	writer := NewAtomicWriter("")
	testData := []byte("Benchmark test data")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testFile := filepath.Join(tempDir, fmt.Sprintf("bench-%d.txt", i))
		err := writer.WriteFile(testFile, testData, 0644)
		if err != nil {
			b.Fatalf("Write failed: %v", err)
		}
	}
}

func BenchmarkAtomicWriter_ReadFile(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "vaino-benchmark")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	writer := NewAtomicWriter("")
	testFile := filepath.Join(tempDir, "bench-read.txt")
	testData := []byte("Benchmark read test data")

	// Create file once
	err = writer.WriteFile(testFile, testData, 0644)
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := writer.ReadFile(testFile)
		if err != nil {
			b.Fatalf("Read failed: %v", err)
		}
	}
}
