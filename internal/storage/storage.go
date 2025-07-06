package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yairfalse/wgo/pkg/types"
)

// Storage interface for persisting data
type Storage interface {
	SaveSnapshot(snapshot *types.Snapshot) error
	LoadSnapshot(id string) (*types.Snapshot, error)
	ListSnapshots() ([]*types.Snapshot, error)
	DeleteSnapshot(id string) error
	
	SaveBaseline(baseline *types.Baseline) error
	LoadBaseline(id string) (*types.Baseline, error)
	ListBaselines() ([]*types.Baseline, error)
	DeleteBaseline(id string) error
}

// NewLocal creates a new local file storage instance
func NewLocal(dataDir string) (Storage, error) {
	return NewFileStorage(dataDir)
}

// NewFileStorage creates a new file-based storage instance
func NewFileStorage(dataDir string) (Storage, error) {
	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}
	
	// Create subdirectories
	for _, subdir := range []string{"snapshots", "baselines"} {
		if err := os.MkdirAll(filepath.Join(dataDir, subdir), 0755); err != nil {
			return nil, fmt.Errorf("failed to create %s directory: %w", subdir, err)
		}
	}
	
	return &FileStorage{dataDir: dataDir}, nil
}