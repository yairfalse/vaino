package storage

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/yairfalse/wgo/pkg/types"
)

type FileStorage struct {
	dataDir string
}

// NewFileStorage creates a new file-based storage instance
func NewFileStorage(dataDir string) (*FileStorage, error) {
	// Create necessary directories
	dirs := []string{
		filepath.Join(dataDir, "snapshots"),
		filepath.Join(dataDir, "baselines"),
		filepath.Join(dataDir, "history", "drift-reports"),
	}
	
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	
	return &FileStorage{dataDir: dataDir}, nil
}

func (fs *FileStorage) SaveSnapshot(snapshot *types.Snapshot) error {
	filename := filepath.Join(fs.dataDir, "snapshots", snapshot.ID+".json")
	return fs.saveJSON(filename, snapshot)
}

func (fs *FileStorage) LoadSnapshot(id string) (*types.Snapshot, error) {
	filename := filepath.Join(fs.dataDir, "snapshots", id+".json")
	var snapshot types.Snapshot
	err := fs.loadJSON(filename, &snapshot)
	return &snapshot, err
}

func (fs *FileStorage) ListSnapshots() ([]SnapshotInfo, error) {
	snapshotsDir := filepath.Join(fs.dataDir, "snapshots")
	entries, err := os.ReadDir(snapshotsDir)
	if err != nil {
		return nil, err
	}
	
	var snapshots []SnapshotInfo
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			id := entry.Name()[:len(entry.Name())-5] // Remove .json
			snapshot, err := fs.LoadSnapshot(id)
			if err != nil {
				continue // Skip invalid files
			}
			
			stat, _ := entry.Info()
			info := SnapshotInfo{
				ID:            snapshot.ID,
				Timestamp:     snapshot.Timestamp,
				Provider:      snapshot.Provider,
				ResourceCount: len(snapshot.Resources),
				FilePath:      filepath.Join(fs.dataDir, "snapshots", entry.Name()),
				FileSize:      stat.Size(),
			}
			snapshots = append(snapshots, info)
		}
	}
	
	return snapshots, nil
}

func (fs *FileStorage) DeleteSnapshot(id string) error {
	filename := filepath.Join(fs.dataDir, "snapshots", id+".json")
	return os.Remove(filename)
}

func (fs *FileStorage) SaveBaseline(baseline *types.Baseline) error {
	filename := filepath.Join(fs.dataDir, "baselines", baseline.ID+".json")
	return fs.saveJSON(filename, baseline)
}

func (fs *FileStorage) LoadBaseline(id string) (*types.Baseline, error) {
	filename := filepath.Join(fs.dataDir, "baselines", id+".json")
	var baseline types.Baseline
	err := fs.loadJSON(filename, &baseline)
	return &baseline, err
}

func (fs *FileStorage) ListBaselines() ([]BaselineInfo, error) {
	baselinesDir := filepath.Join(fs.dataDir, "baselines")
	entries, err := os.ReadDir(baselinesDir)
	if err != nil {
		return nil, err
	}
	
	var baselines []BaselineInfo
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			id := entry.Name()[:len(entry.Name())-5] // Remove .json
			baseline, err := fs.LoadBaseline(id)
			if err != nil {
				continue // Skip invalid files
			}
			
			stat, _ := entry.Info()
			info := BaselineInfo{
				ID:          baseline.ID,
				Name:        baseline.Name,
				Description: baseline.Description,
				SnapshotID:  baseline.SnapshotID,
				CreatedAt:   baseline.CreatedAt,
				Tags:        baseline.Tags,
				Version:     baseline.Version,
				FilePath:    filepath.Join(fs.dataDir, "baselines", entry.Name()),
				FileSize:    stat.Size(),
			}
			baselines = append(baselines, info)
		}
	}
	
	return baselines, nil
}

func (fs *FileStorage) DeleteBaseline(id string) error {
	filename := filepath.Join(fs.dataDir, "baselines", id+".json")
	return os.Remove(filename)
}

// SaveDriftReport saves a drift report to disk
func (fs *FileStorage) SaveDriftReport(report *types.DriftReport) error {
	filename := filepath.Join(fs.dataDir, "history", "drift-reports", report.ID+".json")
	return fs.saveJSON(filename, report)
}

// LoadDriftReport loads a drift report from disk
func (fs *FileStorage) LoadDriftReport(id string) (*types.DriftReport, error) {
	filename := filepath.Join(fs.dataDir, "history", "drift-reports", id+".json")
	var report types.DriftReport
	err := fs.loadJSON(filename, &report)
	return &report, err
}

// ListDriftReports returns metadata for all stored drift reports
func (fs *FileStorage) ListDriftReports() ([]DriftReportInfo, error) {
	reportsDir := filepath.Join(fs.dataDir, "history", "drift-reports")
	entries, err := os.ReadDir(reportsDir)
	if err != nil {
		return nil, err
	}
	
	var reports []DriftReportInfo
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			id := entry.Name()[:len(entry.Name())-5] // Remove .json
			report, err := fs.LoadDriftReport(id)
			if err != nil {
				continue // Skip invalid files
			}
			
			stat, _ := entry.Info()
			info := DriftReportInfo{
				ID:          report.ID,
				BaselineID:  report.BaselineID,
				SnapshotID:  report.CurrentID,
				CreatedAt:   report.Timestamp,
				ChangeCount: len(report.Changes),
				FilePath:    filepath.Join(fs.dataDir, "history", "drift-reports", entry.Name()),
				FileSize:    stat.Size(),
			}
			reports = append(reports, info)
		}
	}
	
	return reports, nil
}

// DeleteDriftReport removes a drift report from disk
func (fs *FileStorage) DeleteDriftReport(id string) error {
	filename := filepath.Join(fs.dataDir, "history", "drift-reports", id+".json")
	return os.Remove(filename)
}

func (fs *FileStorage) saveJSON(filename string, data interface{}) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func (fs *FileStorage) loadJSON(filename string, data interface{}) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	return json.NewDecoder(file).Decode(data)
}