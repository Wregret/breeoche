package raft

import (
	"testing"
	"time"
)

func TestInstallSnapshotAppliesState(t *testing.T) {
	applyCh := make(chan ApplyMsg, 1)
	r, err := NewRaft(Config{
		ID:      "n1",
		Peers:   map[string]string{},
		Storage: NewMemoryStorage(),
		ApplyCh: applyCh,
		Clock:   NewFakeClock(time.Unix(0, 0)),
	})
	if err != nil {
		t.Fatalf("new raft: %v", err)
	}
	r.currentTerm = 1

	reply := r.HandleInstallSnapshot(InstallSnapshotArgs{
		Term:              2,
		LeaderID:          "n2",
		LastIncludedIndex: 5,
		LastIncludedTerm:  2,
		Data:              []byte("snapshot"),
	})

	if reply.Term != 2 {
		t.Fatalf("expected reply term 2, got %d", reply.Term)
	}
	if r.logOffset != 5 {
		t.Fatalf("expected logOffset 5, got %d", r.logOffset)
	}
	if r.commitIndex != 5 {
		t.Fatalf("expected commitIndex 5, got %d", r.commitIndex)
	}
	if r.lastApplied != 5 {
		t.Fatalf("expected lastApplied 5, got %d", r.lastApplied)
	}

	select {
	case msg := <-applyCh:
		if !msg.Snapshot {
			t.Fatalf("expected snapshot ApplyMsg")
		}
		if msg.SnapshotIndex != 5 {
			t.Fatalf("expected snapshot index 5, got %d", msg.SnapshotIndex)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("expected snapshot apply message")
	}
}

func TestSendInstallSnapshotWhenFollowerBehind(t *testing.T) {
	transport := NewLocalTransport()
	leader, err := NewRaft(Config{
		ID:        "n1",
		Peers:     map[string]string{"n2": ""},
		Transport: transport,
		Storage:   NewMemoryStorage(),
		ApplyCh:   make(chan ApplyMsg, 16),
	})
	if err != nil {
		t.Fatalf("new leader: %v", err)
	}
	followerApplyCh := make(chan ApplyMsg, 1)
	follower, err := NewRaft(Config{
		ID:        "n2",
		Peers:     map[string]string{"n1": ""},
		Transport: transport,
		Storage:   NewMemoryStorage(),
		ApplyCh:   followerApplyCh,
	})
	if err != nil {
		t.Fatalf("new follower: %v", err)
	}
	transport.Register(leader)
	transport.Register(follower)

	leader.mu.Lock()
	leader.currentTerm = 2
	leader.state = Leader
	leader.leaderID = "n1"
	leader.log = []LogEntry{{Term: 0}, {Term: 1}, {Term: 1}, {Term: 2}}
	leader.commitIndex = 3
	leader.lastApplied = 3
	leader.nextIndex = map[string]int{"n2": 1}
	leader.matchIndex = map[string]int{"n1": 3, "n2": 0}
	leader.mu.Unlock()

	if err := leader.Snapshot(2, []byte("snap")); err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}

	leader.sendAppendEntries("n2")

	if follower.SnapshotState().LastIncludedIndex != 2 {
		t.Fatalf("expected follower snapshot index 2, got %d", follower.SnapshotState().LastIncludedIndex)
	}

	select {
	case msg := <-followerApplyCh:
		if !msg.Snapshot {
			t.Fatalf("expected snapshot ApplyMsg")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("expected snapshot ApplyMsg")
	}
}
