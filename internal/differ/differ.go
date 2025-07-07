package differ

import (
	"fmt"

	"github.com/yairfalse/wgo/pkg/types"
)

// Differ compares two infrastructure snapshots
type Differ interface {
	Compare(baseline, current *types.Snapshot) (*types.DriftReport, error)
}

// StandardDiffer implements the Differ interface
type StandardDiffer struct{}

// NewStandardDiffer creates a new StandardDiffer
func NewStandardDiffer() *StandardDiffer {
	return &StandardDiffer{}
}

// Compare compares two snapshots and returns a drift report
func (d *StandardDiffer) Compare(baseline, current *types.Snapshot) (*types.DriftReport, error) {
	if baseline == nil || current == nil {
		return nil, fmt.Errorf("both baseline and current snapshots are required")
	}

	// TODO: Implement actual comparison logic
	report := &types.DriftReport{
		ID:         fmt.Sprintf("drift-%d", current.Timestamp.Unix()),
		Timestamp:  current.Timestamp,
		BaselineID: baseline.ID,
		CurrentID:  current.ID,
		Changes:    []types.Change{},
		Summary: types.DriftSummary{
			TotalChanges:     0,
			AddedResources:   0,
			DeletedResources: 0,
			ModifiedResources: 0,
		},
	}

	return report, nil
}