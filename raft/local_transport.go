package raft

import (
	"context"
	"errors"
	"sync"
)

// LocalTransport is an in-memory transport for tests and simulations.
type LocalTransport struct {
	mu    sync.RWMutex
	nodes map[string]*Raft
}

func NewLocalTransport() *LocalTransport {
	return &LocalTransport{nodes: map[string]*Raft{}}
}

func (t *LocalTransport) Register(node *Raft) {
	if node == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.nodes[node.id] = node
}

func (t *LocalTransport) RequestVote(ctx context.Context, peerID string, args RequestVoteArgs) (RequestVoteReply, error) {
	peer, err := t.getPeer(peerID)
	if err != nil {
		return RequestVoteReply{}, err
	}
	select {
	case <-ctx.Done():
		return RequestVoteReply{}, ctx.Err()
	default:
	}
	return peer.HandleRequestVote(args), nil
}

func (t *LocalTransport) AppendEntries(ctx context.Context, peerID string, args AppendEntriesArgs) (AppendEntriesReply, error) {
	peer, err := t.getPeer(peerID)
	if err != nil {
		return AppendEntriesReply{}, err
	}
	select {
	case <-ctx.Done():
		return AppendEntriesReply{}, ctx.Err()
	default:
	}
	return peer.HandleAppendEntries(args), nil
}

func (t *LocalTransport) InstallSnapshot(ctx context.Context, peerID string, args InstallSnapshotArgs) (InstallSnapshotReply, error) {
	peer, err := t.getPeer(peerID)
	if err != nil {
		return InstallSnapshotReply{}, err
	}
	select {
	case <-ctx.Done():
		return InstallSnapshotReply{}, ctx.Err()
	default:
	}
	return peer.HandleInstallSnapshot(args), nil
}

func (t *LocalTransport) getPeer(peerID string) (*Raft, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	peer, ok := t.nodes[peerID]
	if !ok {
		return nil, errors.New("raft: peer not registered")
	}
	return peer, nil
}
