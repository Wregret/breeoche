package kv

import (
	"encoding/json"
	"errors"
	"sync"
)

const (
	OpSet    = "set"
	OpInsert = "insert"
	OpDelete = "delete"
)

var (
	ErrKeyNotFound = errors.New("key not found")
	ErrKeyExists   = errors.New("key exists")
	ErrUnknownOp   = errors.New("unknown operation")
)

// Command represents a KV mutation to be replicated via Raft.
type Command struct {
	Op    string `json:"op"`
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

// Store is an in-memory key/value store backed by Raft log application.
type Store struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewStore() *Store {
	return &Store{data: make(map[string]string)}
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.data[key]
	return value, ok
}

// Apply applies a command to the store.
func (s *Store) Apply(cmd Command) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch cmd.Op {
	case OpSet:
		s.data[cmd.Key] = cmd.Value
		return nil
	case OpInsert:
		if _, ok := s.data[cmd.Key]; ok {
			return ErrKeyExists
		}
		s.data[cmd.Key] = cmd.Value
		return nil
	case OpDelete:
		if _, ok := s.data[cmd.Key]; !ok {
			return ErrKeyNotFound
		}
		delete(s.data, cmd.Key)
		return nil
	default:
		return ErrUnknownOp
	}
}

// EncodeCommand serializes a command to bytes for Raft replication.
func EncodeCommand(cmd Command) ([]byte, error) {
	return json.Marshal(cmd)
}

// DecodeCommand deserializes a command from bytes.
func DecodeCommand(data []byte) (Command, error) {
	var cmd Command
	if err := json.Unmarshal(data, &cmd); err != nil {
		return Command{}, err
	}
	return cmd, nil
}
