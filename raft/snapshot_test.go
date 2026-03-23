package raft

import "testing"

func TestSnapshotTrimsLog(t *testing.T) {
	r := newTestRaft("n1")
	r.log = []LogEntry{{Term: 0}, {Term: 1}, {Term: 1}, {Term: 2}}
	r.commitIndex = 3
	r.lastApplied = 3

	if err := r.Snapshot(2, []byte("snap")); err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}

	if r.logOffset != 2 {
		t.Fatalf("expected logOffset 2, got %d", r.logOffset)
	}
	if r.snapshot.LastIncludedIndex != 2 {
		t.Fatalf("expected snapshot index 2, got %d", r.snapshot.LastIncludedIndex)
	}
	if r.snapshot.LastIncludedTerm != 1 {
		t.Fatalf("expected snapshot term 1, got %d", r.snapshot.LastIncludedTerm)
	}
	if r.lastLogIndex() != 3 {
		t.Fatalf("expected lastLogIndex 3, got %d", r.lastLogIndex())
	}
	if r.termAt(2) != 1 || r.termAt(3) != 2 {
		t.Fatalf("unexpected terms after snapshot")
	}
	if len(r.log) != 2 {
		t.Fatalf("expected log length 2, got %d", len(r.log))
	}
}

func TestSnapshotRejectsUncommittedIndex(t *testing.T) {
	r := newTestRaft("n1")
	r.log = []LogEntry{{Term: 0}, {Term: 1}, {Term: 1}}
	r.commitIndex = 1

	if err := r.Snapshot(2, nil); err == nil {
		t.Fatalf("expected snapshot to fail for uncommitted index")
	}
}

func TestSnapshotPersistsState(t *testing.T) {
	storage := NewMemoryStorage()
	r, err := NewRaft(Config{
		ID:      "n1",
		Peers:   map[string]string{},
		Storage: storage,
		ApplyCh: make(chan ApplyMsg, 16),
	})
	if err != nil {
		t.Fatalf("new raft: %v", err)
	}
	r.log = []LogEntry{{Term: 0}, {Term: 1}, {Term: 1}, {Term: 2}}
	r.commitIndex = 3
	r.lastApplied = 3
	if err := r.Snapshot(2, []byte("snap")); err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}

	r2, err := NewRaft(Config{
		ID:      "n1",
		Peers:   map[string]string{},
		Storage: storage,
		ApplyCh: make(chan ApplyMsg, 16),
	})
	if err != nil {
		t.Fatalf("new raft reload: %v", err)
	}
	if r2.logOffset != 2 {
		t.Fatalf("expected logOffset 2, got %d", r2.logOffset)
	}
	if r2.snapshot.LastIncludedIndex != 2 {
		t.Fatalf("expected snapshot index 2, got %d", r2.snapshot.LastIncludedIndex)
	}
	if r2.termAt(2) != 1 {
		t.Fatalf("expected term 1 at index 2, got %d", r2.termAt(2))
	}
}
