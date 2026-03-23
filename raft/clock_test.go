package raft

import (
	"testing"
	"time"
)

func TestElectionTimerUsesFakeClock(t *testing.T) {
	clock := NewFakeClock(time.Unix(0, 0))
	r, err := NewRaft(Config{
		ID:                "n1",
		Peers:             map[string]string{},
		Transport:         nil,
		Storage:           NewMemoryStorage(),
		ApplyCh:           make(chan ApplyMsg, 16),
		ElectionTimeout:   100,
		HeartbeatInterval: 50,
		Clock:             clock,
	})
	if err != nil {
		t.Fatalf("new raft: %v", err)
	}
	r.rand.Seed(1)
	r.Run()
	defer r.Stop()

	time.Sleep(5 * time.Millisecond)
	clock.Advance(300 * time.Millisecond)

	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		_, isLeader := r.State()
		if isLeader {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("expected leader after election timeout")
}
