package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// FileStateManager implements StateManager using filesystem storage
type FileStateManager struct {
	stateDir string
	mu       sync.RWMutex
}

// NewFileStateManager creates a new file-based state manager
func NewFileStateManager() StateManager {
	// Default to user's home directory
	homeDir, _ := os.UserHomeDir()
	stateDir := filepath.Join(homeDir, ".tapio", "installer")

	return &FileStateManager{
		stateDir: stateDir,
	}
}

// SaveState saves the installation state
func (sm *FileStateManager) SaveState(ctx context.Context, state State) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Ensure state directory exists
	if err := os.MkdirAll(sm.stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Marshal state to JSON
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to temporary file first
	stateFile := filepath.Join(sm.stateDir, "install-state.json")
	tempFile := stateFile + ".tmp"

	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	// Atomically rename
	if err := os.Rename(tempFile, stateFile); err != nil {
		// Fall back to direct write if rename fails
		if err := os.WriteFile(stateFile, data, 0644); err != nil {
			return fmt.Errorf("failed to save state: %w", err)
		}
	}

	return nil
}

// LoadState loads the saved installation state
func (sm *FileStateManager) LoadState(ctx context.Context) (State, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stateFile := filepath.Join(sm.stateDir, "install-state.json")

	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return State{}, nil // No saved state
		}
		return State{}, fmt.Errorf("failed to read state file: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return state, nil
}

// ClearState removes the saved state
func (sm *FileStateManager) ClearState(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	stateFile := filepath.Join(sm.stateDir, "install-state.json")

	if err := os.Remove(stateFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clear state: %w", err)
	}

	// Also remove any backup files
	backupFile := stateFile + ".backup"
	os.Remove(backupFile)

	return nil
}

// MemoryStateManager implements StateManager in memory (for testing)
type MemoryStateManager struct {
	state State
	mu    sync.RWMutex
}

// NewMemoryStateManager creates a new memory-based state manager
func NewMemoryStateManager() StateManager {
	return &MemoryStateManager{}
}

func (sm *MemoryStateManager) SaveState(ctx context.Context, state State) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.state = state
	return nil
}

func (sm *MemoryStateManager) LoadState(ctx context.Context) (State, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state, nil
}

func (sm *MemoryStateManager) ClearState(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.state = State{}
	return nil
}
