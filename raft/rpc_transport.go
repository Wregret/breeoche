package raft

import (
	"context"
	"fmt"
	"github.com/Wregret/breeoche/rpcapi"
	"net/rpc"
)

// RPCTransport sends Raft RPCs using net/rpc over HTTP.
type RPCTransport struct {
	peers map[string]string
}

func NewRPCTransport(peers map[string]string) *RPCTransport {
	copyPeers := make(map[string]string, len(peers))
	for id, addr := range peers {
		copyPeers[id] = addr
	}
	return &RPCTransport{peers: copyPeers}
}

func (t *RPCTransport) RequestVote(ctx context.Context, peerID string, args RequestVoteArgs) (RequestVoteReply, error) {
	var reply RequestVoteReply
	if err := t.call(ctx, peerID, "RequestVote", args, &reply); err != nil {
		return RequestVoteReply{}, err
	}
	return reply, nil
}

func (t *RPCTransport) AppendEntries(ctx context.Context, peerID string, args AppendEntriesArgs) (AppendEntriesReply, error) {
	var reply AppendEntriesReply
	if err := t.call(ctx, peerID, "AppendEntries", args, &reply); err != nil {
		return AppendEntriesReply{}, err
	}
	return reply, nil
}

func (t *RPCTransport) InstallSnapshot(ctx context.Context, peerID string, args InstallSnapshotArgs) (InstallSnapshotReply, error) {
	var reply InstallSnapshotReply
	if err := t.call(ctx, peerID, "InstallSnapshot", args, &reply); err != nil {
		return InstallSnapshotReply{}, err
	}
	return reply, nil
}

func (t *RPCTransport) call(ctx context.Context, peerID string, method string, args interface{}, reply interface{}) error {
	addr, ok := t.peers[peerID]
	if !ok {
		return fmt.Errorf("unknown peer %s", peerID)
	}
	client, err := rpcapi.Dial(ctx, addr)
	if err != nil {
		return err
	}
	defer client.Close()

	fullMethod := rpcapi.RaftServiceName + "." + method
	done := make(chan error, 1)
	go func(c *rpc.Client) {
		done <- c.Call(fullMethod, args, reply)
	}(client)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}
