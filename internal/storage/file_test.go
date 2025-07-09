package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yairfalse/wgo/pkg/types"
)

func TestFileStorage_SaveAndLoadSnapshot(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "wgo-storage-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Create test snapshot
	snapshot := &types.Snapshot{
		ID:        "test-snapshot-1",
		Timestamp: time.Now(),
		Provider:  "aws",
		Resources: []types.Resource{
			{
				ID:       "i-1234567890abcdef0",
				Type:     "ec2:instance",
				Provider: "aws",
				Name:     "test-instance",
				Region:   "us-west-2",
			},
		},
		Metadata: types.SnapshotMetadata{
			CollectorVersion: "1.0.0",
			CollectionTime:   time.Second * 5,
			ResourceCount:    1,
		},
	}

	// Save snapshot
	err = storage.SaveSnapshot(snapshot)
	if err != nil {
		t.Fatalf("Failed to save snapshot: %v", err)
	}

	// Load snapshot
	loaded, err := storage.LoadSnapshot("test-snapshot-1")
	if err != nil {
		t.Fatalf("Failed to load snapshot: %v", err)
	}

	// Verify data
	if loaded.ID != snapshot.ID {
		t.Errorf("Expected ID %s, got %s", snapshot.ID, loaded.ID)
	}
	if loaded.Provider != snapshot.Provider {
		t.Errorf("Expected provider %s, got %s", snapshot.Provider, loaded.Provider)
	}
	if len(loaded.Resources) != len(snapshot.Resources) {
		t.Errorf("Expected %d resources, got %d", len(snapshot.Resources), len(loaded.Resources))
	}
}

func TestFileStorage_ListSnapshots(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "wgo-storage-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Create multiple snapshots
	snapshots := []*types.Snapshot{
		{
			ID:        "snapshot-1",
			Timestamp: time.Now(),
			Provider:  "aws",
			Resources: []types.Resource{},
			Metadata:  types.SnapshotMetadata{},
		},
		{
			ID:        "snapshot-2",
			Timestamp: time.Now(),
			Provider:  "kubernetes",
			Resources: []types.Resource{},
			Metadata:  types.SnapshotMetadata{},
		},
	}

	// Save snapshots
	for _, snapshot := range snapshots {
		err = storage.SaveSnapshot(snapshot)
		if err != nil {
			t.Fatalf("Failed to save snapshot %s: %v", snapshot.ID, err)
		}
	}

	// List snapshots
	listed, err := storage.ListSnapshots()
	if err != nil {
		t.Fatalf("Failed to list snapshots: %v", err)
	}

	if len(listed) != 2 {
		t.Errorf("Expected 2 snapshots, got %d", len(listed))
	}

	// Check that both snapshots are present
	foundIDs := make(map[string]bool)
	for _, snapshot := range listed {
		foundIDs[snapshot.ID] = true
	}

	if !foundIDs["snapshot-1"] || !foundIDs["snapshot-2"] {
		t.Error("Not all snapshots were found in list")
	}
}

func TestFileStorage_DeleteSnapshot(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "wgo-storage-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Create and save snapshot
	snapshot := &types.Snapshot{
		ID:        "delete-test",
		Timestamp: time.Now(),
		Provider:  "test",
		Resources: []types.Resource{},
		Metadata:  types.SnapshotMetadata{},
	}

	err = storage.SaveSnapshot(snapshot)
	if err != nil {
		t.Fatalf("Failed to save snapshot: %v", err)
	}

	// Verify it exists
	_, err = storage.LoadSnapshot("delete-test")
	if err != nil {
		t.Fatalf("Snapshot should exist before deletion: %v", err)
	}

	// Delete snapshot
	err = storage.DeleteSnapshot("delete-test")
	if err != nil {
		t.Fatalf("Failed to delete snapshot: %v", err)
	}

	// Verify it's gone
	_, err = storage.LoadSnapshot("delete-test")
	if err == nil {
		t.Error("Snapshot should not exist after deletion")
	}
}

func TestFileStorage_SaveAndLoadBaseline(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "wgo-storage-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Create test baseline
	baseline := &types.Baseline{
		ID:          "baseline-1",
		Name:        "Production Baseline",
		Description: "Baseline for production environment",
		SnapshotID:  "snapshot-123",
		CreatedAt:   time.Now(),
		Tags: map[string]string{
			"environment": "production",
			"version":     "1.0",
		},
		Version: "1.0.0",
	}

	// Save baseline
	err = storage.SaveBaseline(baseline)
	if err != nil {
		t.Fatalf("Failed to save baseline: %v", err)
	}

	// Load baseline
	loaded, err := storage.LoadBaseline("baseline-1")
	if err != nil {
		t.Fatalf("Failed to load baseline: %v", err)
	}

	// Verify data
	if loaded.ID != baseline.ID {
		t.Errorf("Expected ID %s, got %s", baseline.ID, loaded.ID)
	}
	if loaded.Name != baseline.Name {
		t.Errorf("Expected name %s, got %s", baseline.Name, loaded.Name)
	}
	if loaded.SnapshotID != baseline.SnapshotID {
		t.Errorf("Expected snapshot ID %s, got %s", baseline.SnapshotID, loaded.SnapshotID)
	}
}

func TestFileStorage_DirectoryCreation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "wgo-storage-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Use a subdirectory that doesn't exist yet
	storageDir := filepath.Join(tempDir, "new-storage")

	storage, err := NewFileStorage(storageDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Check that directories were created
	snapshotsDir := filepath.Join(storageDir, "snapshots")
	baselinesDir := filepath.Join(storageDir, "baselines")

	if _, err := os.Stat(snapshotsDir); os.IsNotExist(err) {
		t.Error("Snapshots directory was not created")
	}

	if _, err := os.Stat(baselinesDir); os.IsNotExist(err) {
		t.Error("Baselines directory was not created")
	}

	// Test that we can use the storage
	snapshot := &types.Snapshot{
		ID:        "test",
		Timestamp: time.Now(),
		Provider:  "test",
		Resources: []types.Resource{},
		Metadata:  types.SnapshotMetadata{},
	}

	err = storage.SaveSnapshot(snapshot)
	if err != nil {
		t.Fatalf("Failed to save snapshot in new directory: %v", err)
	}
}
