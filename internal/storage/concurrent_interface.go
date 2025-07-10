package storage

import (
	"context"

	"github.com/yairfalse/vaino/pkg/types"
)

// ConcurrentStorageInterface extends Storage with concurrent operations
type ConcurrentStorageInterface interface {
	Storage

	// Concurrent snapshot operations
	ListSnapshotsConcurrent(ctx context.Context) ([]SnapshotInfo, error)
	SaveSnapshotsConcurrent(ctx context.Context, snapshots []*types.Snapshot) error
	LoadSnapshotsConcurrent(ctx context.Context, ids []string) ([]*types.Snapshot, error)

	// Streaming operations for large files
	ProcessLargeSnapshot(ctx context.Context, path string, processFunc func(resource *types.Resource) error) error
}
