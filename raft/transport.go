package raft

import "context"

// Transport sends Raft RPCs to peers.
type Transport interface {
	RequestVote(ctx context.Context, peerID string, args RequestVoteArgs) (RequestVoteReply, error)
	AppendEntries(ctx context.Context, peerID string, args AppendEntriesArgs) (AppendEntriesReply, error)
}
