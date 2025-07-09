package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yairfalse/wgo/pkg/types"
)

func TestNewLocalStorage(t *testing.T) {
	tmpDir := t.TempDir()

	config := Config{BaseDir: tmpDir}
	storage, err := NewLocalStorage(config)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	if storage.baseDir != tmpDir {
		t.Errorf("expected baseDir %s, got %s", tmpDir, storage.baseDir)
	}

	// Check that directories were created
	dirs := []string{
		filepath.Join(tmpDir, "baselines"),
		filepath.Join(tmpDir, "snapshots"),
		filepath.Join(tmpDir, "history", "drift-reports"),
		filepath.Join(tmpDir, "cache"),
	}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("directory %s was not created", dir)
		}
	}
}

func TestLocalStorage_SnapshotOperations(t *testing.T) {
	tmpDir := t.TempDir()
	config := Config{BaseDir: tmpDir}
	storage, err := NewLocalStorage(config)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Create test snapshot
	snapshot := &types.Snapshot{
		ID:        "test-snapshot-1",
		Timestamp: time.Now(),
		Provider:  "terraform",
		Resources: []types.Resource{
			{
				ID:       "test-resource-1",
				Type:     "aws_instance",
				Name:     "web-server",
				Provider: "aws",
				Configuration: map[string]interface{}{
					"instance_type": "t3.micro",
				},
			},
		},
		Metadata: types.SnapshotMetadata{
			CollectorVersion: "1.0.0",
			ResourceCount:    1,
		},
	}

	// Test Save
	err = storage.SaveSnapshot(snapshot)
	if err != nil {
		t.Fatalf("failed to save snapshot: %v", err)
	}

	// Test Load
	loadedSnapshot, err := storage.LoadSnapshot("test-snapshot-1")
	if err != nil {
		t.Fatalf("failed to load snapshot: %v", err)
	}

	if loadedSnapshot.ID != snapshot.ID {
		t.Errorf("expected ID %s, got %s", snapshot.ID, loadedSnapshot.ID)
	}

	if len(loadedSnapshot.Resources) != len(snapshot.Resources) {
		t.Errorf("expected %d resources, got %d", len(snapshot.Resources), len(loadedSnapshot.Resources))
	}

	// Test List
	snapshots, err := storage.ListSnapshots()
	if err != nil {
		t.Fatalf("failed to list snapshots: %v", err)
	}

	if len(snapshots) != 1 {
		t.Errorf("expected 1 snapshot, got %d", len(snapshots))
	}

	if snapshots[0].ID != "test-snapshot-1" {
		t.Errorf("expected snapshot ID test-snapshot-1, got %s", snapshots[0].ID)
	}

	// Test Delete
	err = storage.DeleteSnapshot("test-snapshot-1")
	if err != nil {
		t.Fatalf("failed to delete snapshot: %v", err)
	}

	// Verify deletion
	_, err = storage.LoadSnapshot("test-snapshot-1")
	if err == nil {
		t.Error("expected error loading deleted snapshot")
	}
}

func TestLocalStorage_BaselineOperations(t *testing.T) {
	tmpDir := t.TempDir()
	config := Config{BaseDir: tmpDir}
	storage, err := NewLocalStorage(config)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Create test baseline
	baseline := &types.Baseline{
		ID:          "test-baseline-1",
		Name:        "Production Baseline",
		Description: "Main production environment baseline",
		SnapshotID:  "snapshot-123",
		CreatedAt:   time.Now(),
		Tags: map[string]string{
			"environment": "production",
			"team":        "platform",
		},
		Version: "1.0.0",
	}

	// Test Save
	err = storage.SaveBaseline(baseline)
	if err != nil {
		t.Fatalf("failed to save baseline: %v", err)
	}

	// Test Load
	loadedBaseline, err := storage.LoadBaseline("test-baseline-1")
	if err != nil {
		t.Fatalf("failed to load baseline: %v", err)
	}

	if loadedBaseline.ID != baseline.ID {
		t.Errorf("expected ID %s, got %s", baseline.ID, loadedBaseline.ID)
	}

	if loadedBaseline.Name != baseline.Name {
		t.Errorf("expected name %s, got %s", baseline.Name, loadedBaseline.Name)
	}

	// Test List
	baselines, err := storage.ListBaselines()
	if err != nil {
		t.Fatalf("failed to list baselines: %v", err)
	}

	if len(baselines) != 1 {
		t.Errorf("expected 1 baseline, got %d", len(baselines))
	}

	if baselines[0].ID != "test-baseline-1" {
		t.Errorf("expected baseline ID test-baseline-1, got %s", baselines[0].ID)
	}

	// Test Delete
	err = storage.DeleteBaseline("test-baseline-1")
	if err != nil {
		t.Fatalf("failed to delete baseline: %v", err)
	}

	// Verify deletion
	_, err = storage.LoadBaseline("test-baseline-1")
	if err == nil {
		t.Error("expected error loading deleted baseline")
	}
}

func TestLocalStorage_DriftReportOperations(t *testing.T) {
	tmpDir := t.TempDir()
	config := Config{BaseDir: tmpDir}
	storage, err := NewLocalStorage(config)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Create test drift report
	report := &types.DriftReport{
		ID:         "drift-report-1",
		Timestamp:  time.Now(),
		BaselineID: "baseline-123",
		CurrentID:  "snapshot-456",
		Changes: []types.Change{
			{
				Field:       "instance_type",
				OldValue:    "t3.micro",
				NewValue:    "t3.medium",
				Severity:    "medium",
				Path:        "resources[0].configuration.instance_type",
				Description: "Instance type changed",
			},
		},
		Summary: types.DriftSummary{
			TotalChanges:      1,
			AddedResources:    0,
			DeletedResources:  0,
			ModifiedResources: 1,
			RiskScore:         0.5,
		},
	}

	// Test Save
	err = storage.SaveDriftReport(report)
	if err != nil {
		t.Fatalf("failed to save drift report: %v", err)
	}

	// Test Load
	loadedReport, err := storage.LoadDriftReport("drift-report-1")
	if err != nil {
		t.Fatalf("failed to load drift report: %v", err)
	}

	if loadedReport.ID != report.ID {
		t.Errorf("expected ID %s, got %s", report.ID, loadedReport.ID)
	}

	if len(loadedReport.Changes) != len(report.Changes) {
		t.Errorf("expected %d changes, got %d", len(report.Changes), len(loadedReport.Changes))
	}

	// Test List
	reports, err := storage.ListDriftReports()
	if err != nil {
		t.Fatalf("failed to list drift reports: %v", err)
	}

	if len(reports) != 1 {
		t.Errorf("expected 1 drift report, got %d", len(reports))
	}

	if reports[0].ID != "drift-report-1" {
		t.Errorf("expected drift report ID drift-report-1, got %s", reports[0].ID)
	}
}

func TestLocalStorage_ErrorHandling(t *testing.T) {
	// Test with invalid directory
	config := Config{BaseDir: "/invalid/path/that/cannot/be/created"}
	_, err := NewLocalStorage(config)
	if err == nil {
		t.Error("expected error creating storage with invalid path")
	}

	// Test loading non-existent items
	tmpDir := t.TempDir()
	config = Config{BaseDir: tmpDir}
	storage, err := NewLocalStorage(config)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	// Test loading non-existent snapshot
	_, err = storage.LoadSnapshot("non-existent")
	if err == nil {
		t.Error("expected error loading non-existent snapshot")
	}

	// Test loading non-existent baseline
	_, err = storage.LoadBaseline("non-existent")
	if err == nil {
		t.Error("expected error loading non-existent baseline")
	}

	// Test loading non-existent drift report
	_, err = storage.LoadDriftReport("non-existent")
	if err == nil {
		t.Error("expected error loading non-existent drift report")
	}
}
