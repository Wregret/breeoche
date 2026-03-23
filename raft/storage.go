package raft

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Storage persists the Raft state.
type Storage interface {
	Load() (PersistentState, error)
	Save(state PersistentState) error
}

// MemoryStorage keeps state in memory (useful for tests).
type MemoryStorage struct {
	state PersistentState
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{state: PersistentState{Log: []LogEntry{{Term: 0}}}}
}

func (m *MemoryStorage) Load() (PersistentState, error) {
	return m.state, nil
}

func (m *MemoryStorage) Save(state PersistentState) error {
	m.state = state
	return nil
}

// FileStorage persists state to a JSON file on disk.
type FileStorage struct {
	path string
}

func NewFileStorage(path string) *FileStorage {
	return &FileStorage{path: path}
}

func (f *FileStorage) Load() (PersistentState, error) {
	data, err := os.ReadFile(f.path)
	if err != nil {
		if os.IsNotExist(err) {
			return PersistentState{Log: []LogEntry{{Term: 0}}}, nil
		}
		return PersistentState{}, err
	}
	var state PersistentState
	if err := json.Unmarshal(data, &state); err != nil {
		return PersistentState{}, err
	}
	if len(state.Log) == 0 {
		state.Log = []LogEntry{{Term: 0}}
	}
	return state, nil
}

func (f *FileStorage) Save(state PersistentState) error {
	if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.tmp", f.path)
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, f.path)
}
