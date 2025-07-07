package types

import (
	"time"
)

type Snapshot struct {
	ID        string                 `json:"id"`
	Timestamp time.Time             `json:"timestamp"`
	Provider  string                `json:"provider"`
	Data      map[string]interface{} `json:"data"`
	Metadata  map[string]string     `json:"metadata"`
}

type DriftResult struct {
	SnapshotID string                 `json:"snapshot_id"`
	Changes    []Change              `json:"changes"`
	Summary    string                `json:"summary"`
	Timestamp  time.Time             `json:"timestamp"`
}

type Change struct {
	Type        string      `json:"type"`
	Resource    string      `json:"resource"`
	Field       string      `json:"field"`
	OldValue    interface{} `json:"old_value"`
	NewValue    interface{} `json:"new_value"`
	Severity    string      `json:"severity"`
	Description string      `json:"description"`
}