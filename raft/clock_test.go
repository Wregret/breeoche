package raft

import (
	"context"
	"fmt"
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

func TestHeartbeatPreventsElectionWithFakeClock(t *testing.T) {
	clock := NewFakeClock(time.Unix(0, 0))
	nodes, _ := newFakeCluster(t, 2, clock, 200, 20)
	defer stopCluster(nodes)

	nodes[0].TriggerElection()
	leader := waitForSingleLeader(t, nodes, 200*time.Millisecond)
	if leader.id != "n1" {
		t.Fatalf("expected leader n1, got %s", leader.id)
	}

	for i := 0; i < 10; i++ {
		clock.Advance(50 * time.Millisecond)
		time.Sleep(5 * time.Millisecond)
		if state := nodeState(nodes[1]); state != Follower {
			t.Fatalf("expected follower to remain follower, got %s", state)
		}
	}
}

func TestFollowerStartsElectionAfterHeartbeatsStop(t *testing.T) {
	clock := NewFakeClock(time.Unix(0, 0))
	nodes, _ := newFakeCluster(t, 2, clock, 200, 20)

	nodes[0].TriggerElection()
	leader := waitForSingleLeader(t, nodes, 200*time.Millisecond)
	if leader.id != "n1" {
		t.Fatalf("expected leader n1, got %s", leader.id)
	}

	nodes[0].Stop()
	defer nodes[1].Stop()
	time.Sleep(5 * time.Millisecond)

	clock.Advance(500 * time.Millisecond)

	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if state := nodeState(nodes[1]); state != Follower {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("expected follower to start election after heartbeats stop")
}

func TestHeartbeatReplicatesEntryWithFakeClock(t *testing.T) {
	clock := NewFakeClock(time.Unix(0, 0))
	nodes, _ := newFakeCluster(t, 2, clock, 200, 20)
	defer stopCluster(nodes)

	nodes[0].TriggerElection()
	leader := waitForSingleLeader(t, nodes, 200*time.Millisecond)

	index, term, ok := leader.Start([]byte("set x 1"))
	if !ok {
		t.Fatalf("expected leader start")
	}

	clock.Advance(50 * time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	_, isLeader := leader.State()
	if !isLeader {
		t.Fatalf("expected leader to remain leader")
	}

	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		nodes[1].mu.Lock()
		hasEntry := nodes[1].lastLogIndex() >= index
		nodes[1].mu.Unlock()
		if hasEntry {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if err := leader.WaitForCommit(ctx, index, term); err != nil {
		t.Fatalf("expected commit, got %v", err)
	}
}

func newFakeCluster(t *testing.T, size int, clock *FakeClock, electionTimeout, heartbeatInterval int) ([]*Raft, *LocalTransport) {
	t.Helper()
	transport := NewLocalTransport()
	nodes := make([]*Raft, size)
	ids := make([]string, size)
	for i := 0; i < size; i++ {
		ids[i] = fmt.Sprintf("n%d", i+1)
	}
	for i := 0; i < size; i++ {
		peers := map[string]string{}
		for j := 0; j < size; j++ {
			if i == j {
				continue
			}
			peers[ids[j]] = ""
		}
		r, err := NewRaft(Config{
			ID:                ids[i],
			Peers:             peers,
			Transport:         transport,
			Storage:           NewMemoryStorage(),
			ApplyCh:           make(chan ApplyMsg, 16),
			ElectionTimeout:   electionTimeout,
			HeartbeatInterval: heartbeatInterval,
			Clock:             clock,
		})
		if err != nil {
			t.Fatalf("new raft: %v", err)
		}
		r.rand.Seed(int64(i + 1))
		transport.Register(r)
		nodes[i] = r
	}
	for _, node := range nodes {
		node.Run()
	}
	return nodes, transport
}

func stopCluster(nodes []*Raft) {
	for _, node := range nodes {
		node.Stop()
	}
}

func nodeState(r *Raft) State {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.state
}
