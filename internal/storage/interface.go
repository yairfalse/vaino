package storage

import (
	"time"

	"github.com/yairfalse/wgo/pkg/types"
)

// Storage defines the interface for persisting baselines and snapshots
type Storage interface {
	// Baseline operations
	SaveBaseline(baseline *types.Baseline) error
	LoadBaseline(id string) (*types.Baseline, error)
	ListBaselines() ([]BaselineInfo, error)
	DeleteBaseline(id string) error

	// Snapshot operations
	SaveSnapshot(snapshot *types.Snapshot) error
	LoadSnapshot(id string) (*types.Snapshot, error)
	ListSnapshots() ([]SnapshotInfo, error)
	DeleteSnapshot(id string) error

	// History operations
	SaveDriftReport(report *types.DriftReport) error
	LoadDriftReport(id string) (*types.DriftReport, error)
	ListDriftReports() ([]DriftReportInfo, error)
	DeleteDriftReport(id string) error
}

// BaselineInfo provides metadata about a stored baseline
type BaselineInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	SnapshotID  string            `json:"snapshot_id"`
	CreatedAt   time.Time         `json:"created_at"`
	Tags        map[string]string `json:"tags,omitempty"`
	Version     string            `json:"version"`
	FilePath    string            `json:"file_path"`
	FileSize    int64             `json:"file_size"`
}

// SnapshotInfo provides metadata about a stored snapshot
type SnapshotInfo struct {
	ID            string            `json:"id"`
	Timestamp     time.Time         `json:"timestamp"`
	Provider      string            `json:"provider"`
	ResourceCount int               `json:"resource_count"`
	Tags          map[string]string `json:"tags,omitempty"`
	FilePath      string            `json:"file_path"`
	FileSize      int64             `json:"file_size"`
}

// DriftReportInfo provides metadata about a stored drift report
type DriftReportInfo struct {
	ID          string            `json:"id"`
	BaselineID  string            `json:"baseline_id"`
	SnapshotID  string            `json:"snapshot_id"`
	CreatedAt   time.Time         `json:"created_at"`
	ChangeCount int               `json:"change_count"`
	Tags        map[string]string `json:"tags,omitempty"`
	FilePath    string            `json:"file_path"`
	FileSize    int64             `json:"file_size"`
}

// Config holds storage configuration
type Config struct {
	BaseDir    string `json:"base_dir"`
	MaxHistory int    `json:"max_history,omitempty"`
	Compress   bool   `json:"compress,omitempty"`
}
