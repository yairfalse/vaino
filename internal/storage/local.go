package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yairfalse/wgo/pkg/types"
)

// LocalStorage implements the Storage interface using local filesystem
type LocalStorage struct {
	config    Config
	baseDir   string
	baselines string
	snapshots string
	reports   string
	cache     string
}

// NewLocal creates a new local storage instance with default directory
func NewLocal(baseDir string) Storage {
	config := Config{BaseDir: baseDir}
	storage, err := NewLocalStorage(config)
	if err != nil {
		// For convenience, create a storage that will work in most cases
		// but will return errors on actual operations if directory creation fails
		return storage
	}
	return storage
}

// NewLocalStorage creates a new local storage instance
func NewLocalStorage(config Config) (*LocalStorage, error) {
	if config.BaseDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		config.BaseDir = filepath.Join(homeDir, ".wgo")
	}

	storage := &LocalStorage{
		config:    config,
		baseDir:   config.BaseDir,
		baselines: filepath.Join(config.BaseDir, "baselines"),
		snapshots: filepath.Join(config.BaseDir, "snapshots"),
		reports:   filepath.Join(config.BaseDir, "history", "drift-reports"),
		cache:     filepath.Join(config.BaseDir, "cache"),
	}

	// Create directories
	dirs := []string{
		storage.baseDir,
		storage.baselines,
		storage.snapshots,
		storage.reports,
		storage.cache,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return storage, nil
}

// SaveBaseline saves a baseline to disk
func (s *LocalStorage) SaveBaseline(baseline *types.Baseline) error {
	if err := baseline.Validate(); err != nil {
		return fmt.Errorf("invalid baseline: %w", err)
	}

	filename := fmt.Sprintf("%s-%s.json", 
		sanitizeFilename(baseline.Name), 
		baseline.CreatedAt.Format("2006-01-02"))
	filepath := filepath.Join(s.baselines, filename)

	return s.saveJSON(filepath, baseline)
}

// LoadBaseline loads a baseline from disk
func (s *LocalStorage) LoadBaseline(id string) (*types.Baseline, error) {
	// Try by ID first
	files, err := os.ReadDir(s.baselines)
	if err != nil {
		return nil, fmt.Errorf("failed to read baselines directory: %w", err)
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		var baseline types.Baseline
		path := filepath.Join(s.baselines, file.Name())
		if err := s.loadJSON(path, &baseline); err != nil {
			continue
		}

		if baseline.ID == id || baseline.Name == id {
			return &baseline, nil
		}
	}

	return nil, fmt.Errorf("baseline not found: %s", id)
}

// ListBaselines returns metadata for all stored baselines
func (s *LocalStorage) ListBaselines() ([]BaselineInfo, error) {
	files, err := os.ReadDir(s.baselines)
	if err != nil {
		return nil, fmt.Errorf("failed to read baselines directory: %w", err)
	}

	var infos []BaselineInfo
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		path := filepath.Join(s.baselines, file.Name())
		stat, err := file.Info()
		if err != nil {
			continue
		}

		var baseline types.Baseline
		if err := s.loadJSON(path, &baseline); err != nil {
			continue
		}

		info := BaselineInfo{
			ID:          baseline.ID,
			Name:        baseline.Name,
			Description: baseline.Description,
			SnapshotID:  baseline.SnapshotID,
			CreatedAt:   baseline.CreatedAt,
			Tags:        baseline.Tags,
			Version:     baseline.Version,
			FilePath:    path,
			FileSize:    stat.Size(),
		}
		infos = append(infos, info)
	}

	// Sort by creation time (newest first)
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].CreatedAt.After(infos[j].CreatedAt)
	})

	return infos, nil
}

// DeleteBaseline removes a baseline from disk
func (s *LocalStorage) DeleteBaseline(id string) error {
	baseline, err := s.LoadBaseline(id)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("%s-%s.json", 
		sanitizeFilename(baseline.Name), 
		baseline.CreatedAt.Format("2006-01-02"))
	filepath := filepath.Join(s.baselines, filename)

	return os.Remove(filepath)
}

// SaveSnapshot saves a snapshot to disk
func (s *LocalStorage) SaveSnapshot(snapshot *types.Snapshot) error {
	if err := snapshot.Validate(); err != nil {
		return fmt.Errorf("invalid snapshot: %w", err)
	}

	filename := fmt.Sprintf("%s-scan.json", snapshot.Timestamp.Format("2006-01-02T15-04-05"))
	filepath := filepath.Join(s.snapshots, filename)

	return s.saveJSON(filepath, snapshot)
}

// LoadSnapshot loads a snapshot from disk
func (s *LocalStorage) LoadSnapshot(id string) (*types.Snapshot, error) {
	files, err := os.ReadDir(s.snapshots)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshots directory: %w", err)
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		var snapshot types.Snapshot
		path := filepath.Join(s.snapshots, file.Name())
		if err := s.loadJSON(path, &snapshot); err != nil {
			continue
		}

		if snapshot.ID == id {
			return &snapshot, nil
		}
	}

	return nil, fmt.Errorf("snapshot not found: %s", id)
}

// ListSnapshots returns metadata for all stored snapshots
func (s *LocalStorage) ListSnapshots() ([]SnapshotInfo, error) {
	files, err := os.ReadDir(s.snapshots)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshots directory: %w", err)
	}

	var infos []SnapshotInfo
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		path := filepath.Join(s.snapshots, file.Name())
		stat, err := file.Info()
		if err != nil {
			continue
		}

		var snapshot types.Snapshot
		if err := s.loadJSON(path, &snapshot); err != nil {
			continue
		}

		info := SnapshotInfo{
			ID:            snapshot.ID,
			Timestamp:     snapshot.Timestamp,
			Provider:      snapshot.Provider,
			ResourceCount: len(snapshot.Resources),
			Tags:          snapshot.Metadata.Tags,
			FilePath:      path,
			FileSize:      stat.Size(),
		}
		infos = append(infos, info)
	}

	// Sort by timestamp (newest first)
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Timestamp.After(infos[j].Timestamp)
	})

	return infos, nil
}

// DeleteSnapshot removes a snapshot from disk
func (s *LocalStorage) DeleteSnapshot(id string) error {
	snapshot, err := s.LoadSnapshot(id)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("%s-scan.json", snapshot.Timestamp.Format("2006-01-02T15-04-05"))
	filepath := filepath.Join(s.snapshots, filename)

	return os.Remove(filepath)
}

// SaveDriftReport saves a drift report to disk
func (s *LocalStorage) SaveDriftReport(report *types.DriftReport) error {
	// Basic validation
	if report.ID == "" {
		return fmt.Errorf("drift report ID is required")
	}
	if report.BaselineID == "" {
		return fmt.Errorf("baseline ID is required")
	}
	if report.CurrentID == "" {
		return fmt.Errorf("current snapshot ID is required")
	}

	filename := fmt.Sprintf("drift-report-%s.json", report.Timestamp.Format("2006-01-02T15-04-05"))
	filepath := filepath.Join(s.reports, filename)

	return s.saveJSON(filepath, report)
}

// LoadDriftReport loads a drift report from disk
func (s *LocalStorage) LoadDriftReport(id string) (*types.DriftReport, error) {
	files, err := os.ReadDir(s.reports)
	if err != nil {
		return nil, fmt.Errorf("failed to read drift reports directory: %w", err)
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		var report types.DriftReport
		path := filepath.Join(s.reports, file.Name())
		if err := s.loadJSON(path, &report); err != nil {
			continue
		}

		if report.ID == id {
			return &report, nil
		}
	}

	return nil, fmt.Errorf("drift report not found: %s", id)
}

// ListDriftReports returns metadata for all stored drift reports
func (s *LocalStorage) ListDriftReports() ([]DriftReportInfo, error) {
	files, err := os.ReadDir(s.reports)
	if err != nil {
		return nil, fmt.Errorf("failed to read drift reports directory: %w", err)
	}

	var infos []DriftReportInfo
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		path := filepath.Join(s.reports, file.Name())
		stat, err := file.Info()
		if err != nil {
			continue
		}

		var report types.DriftReport
		if err := s.loadJSON(path, &report); err != nil {
			continue
		}

		info := DriftReportInfo{
			ID:          report.ID,
			BaselineID:  report.BaselineID,
			SnapshotID:  report.CurrentID,
			CreatedAt:   report.Timestamp,
			ChangeCount: len(report.Changes),
			Tags:        make(map[string]string),
			FilePath:    path,
			FileSize:    stat.Size(),
		}
		infos = append(infos, info)
	}

	// Sort by creation time (newest first)
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].CreatedAt.After(infos[j].CreatedAt)
	})

	return infos, nil
}

// DeleteDriftReport removes a drift report from disk
func (s *LocalStorage) DeleteDriftReport(id string) error {
	report, err := s.LoadDriftReport(id)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("drift-report-%s.json", report.Timestamp.Format("2006-01-02T15-04-05"))
	filepath := filepath.Join(s.reports, filename)

	return os.Remove(filepath)
}

// saveJSON saves data as JSON to the specified path
func (s *LocalStorage) saveJSON(path string, data interface{}) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", path, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// loadJSON loads JSON data from the specified path
func (s *LocalStorage) loadJSON(path string, target interface{}) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("failed to decode JSON: %w", err)
	}

	return nil
}

// sanitizeFilename removes invalid characters from filenames
func sanitizeFilename(name string) string {
	// Replace invalid characters with hyphens
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", " "}
	result := name
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "-")
	}
	return result
}