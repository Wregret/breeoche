package raft

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func newCluster(t *testing.T, size int) ([]*Raft, *LocalTransport) {
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
			ApplyCh:           make(chan ApplyMsg, 128),
			ElectionTimeout:   500,
			HeartbeatInterval: 100,
		})
		if err != nil {
			t.Fatalf("new raft: %v", err)
		}
		transport.Register(r)
		nodes[i] = r
	}
	return nodes, transport
}

func waitForSingleLeader(t *testing.T, nodes []*Raft, timeout time.Duration) *Raft {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var leader *Raft
		leaderCount := 0
		for _, node := range nodes {
			_, isLeader := node.State()
			if isLeader {
				leaderCount++
				leader = node
			}
		}
		if leaderCount == 1 {
			return leader
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected single leader within %s", timeout)
	return nil
}

func waitForLogIndex(t *testing.T, nodes []*Raft, index int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		allHave := true
		for _, node := range nodes {
			node.mu.Lock()
			if node.lastLogIndex() < index {
				allHave = false
			}
			node.mu.Unlock()
			if !allHave {
				break
			}
		}
		if allHave {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("log index %d not replicated within %s", index, timeout)
}

func TestLocalClusterElectsLeader(t *testing.T) {
	nodes, _ := newCluster(t, 3)
	nodes[0].TriggerElection()

	leader := waitForSingleLeader(t, nodes, 2*time.Second)
	if leader.id != "n1" {
		t.Fatalf("expected n1 leader, got %s", leader.id)
	}
}

func TestLocalClusterReplicatesEntry(t *testing.T) {
	nodes, _ := newCluster(t, 3)
	nodes[0].TriggerElection()
	leader := waitForSingleLeader(t, nodes, 2*time.Second)

	index, term, ok := leader.Start([]byte("set x 1"))
	if !ok {
		t.Fatalf("expected leader Start to succeed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := leader.WaitForCommit(ctx, index, term); err != nil {
		t.Fatalf("wait for commit: %v", err)
	}
	leader.ReplicateOnce()

	waitForLogIndex(t, nodes, index, 2*time.Second)
	for _, node := range nodes {
		node.mu.Lock()
		entry := node.log[index]
		node.mu.Unlock()
		if string(entry.Command) != "set x 1" {
			t.Fatalf("expected command replicated, got %q", string(entry.Command))
		}
	}
}
