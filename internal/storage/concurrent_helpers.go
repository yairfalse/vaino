package storage

import (
	"sort"
	"time"
)

// sortSnapshotInfos sorts snapshot infos by timestamp (newest first)
func sortSnapshotInfos(infos []SnapshotInfo) {
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Timestamp.After(infos[j].Timestamp)
	})
}

// parseTimestamp parses various timestamp formats
func parseTimestamp(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, nil
}
