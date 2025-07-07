package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yairfalse/wgo/pkg/types"
)

type Storage interface {
	SaveSnapshot(snapshot *types.Snapshot) error
	LoadSnapshot(id string) (*types.Snapshot, error)
	ListSnapshots() ([]*types.Snapshot, error)
	DeleteSnapshot(id string) error
}

type LocalStorage struct {
	basePath string
}

func NewLocal(basePath string) Storage {
	return &LocalStorage{
		basePath: basePath,
	}
}

func (s *LocalStorage) SaveSnapshot(snapshot *types.Snapshot) error {
	if err := os.MkdirAll(s.basePath, 0755); err != nil {
		return err
	}

	filename := filepath.Join(s.basePath, fmt.Sprintf("%s.json", snapshot.ID))
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

func (s *LocalStorage) LoadSnapshot(id string) (*types.Snapshot, error) {
	filename := filepath.Join(s.basePath, fmt.Sprintf("%s.json", id))
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var snapshot types.Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, err
	}

	return &snapshot, nil
}

func (s *LocalStorage) ListSnapshots() ([]*types.Snapshot, error) {
	files, err := filepath.Glob(filepath.Join(s.basePath, "*.json"))
	if err != nil {
		return nil, err
	}

	var snapshots []*types.Snapshot
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var snapshot types.Snapshot
		if err := json.Unmarshal(data, &snapshot); err != nil {
			continue
		}

		snapshots = append(snapshots, &snapshot)
	}

	return snapshots, nil
}

func (s *LocalStorage) DeleteSnapshot(id string) error {
	filename := filepath.Join(s.basePath, fmt.Sprintf("%s.json", id))
	return os.Remove(filename)
}