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

func (fs *FileStorage) ListSnapshots() ([]*types.Snapshot, error) {
	snapshotsDir := filepath.Join(fs.dataDir, "snapshots")
	entries, err := os.ReadDir(snapshotsDir)
	if err != nil {
		return nil, err
	}
	
	var snapshots []*types.Snapshot
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			id := entry.Name()[:len(entry.Name())-5] // Remove .json
			snapshot, err := fs.LoadSnapshot(id)
			if err != nil {
				continue // Skip invalid files
			}
			snapshots = append(snapshots, snapshot)
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

func (fs *FileStorage) ListBaselines() ([]*types.Baseline, error) {
	baselinesDir := filepath.Join(fs.dataDir, "baselines")
	entries, err := os.ReadDir(baselinesDir)
	if err != nil {
		return nil, err
	}
	
	var baselines []*types.Baseline
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			id := entry.Name()[:len(entry.Name())-5] // Remove .json
			baseline, err := fs.LoadBaseline(id)
			if err != nil {
				continue // Skip invalid files
			}
			baselines = append(baselines, baseline)
		}
	}
	
	return baselines, nil
}

func (fs *FileStorage) DeleteBaseline(id string) error {
	filename := filepath.Join(fs.dataDir, "baselines", id+".json")
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