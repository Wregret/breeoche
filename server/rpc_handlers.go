package server

import (
	"context"
	"errors"
	"github.com/Wregret/breeoche/kv"
	"github.com/Wregret/breeoche/raft"
	"github.com/Wregret/breeoche/rpcapi"
	"github.com/gorilla/mux"
	"net/rpc"
)

type kvRPC struct {
	server *Server
}

type raftRPC struct {
	raft *raft.Raft
}

func (s *Server) registerRPC(r *mux.Router) error {
	rpcServer := rpc.NewServer()
	if err := rpcServer.RegisterName(rpcapi.KVServiceName, &kvRPC{server: s}); err != nil {
		return err
	}
	if err := rpcServer.RegisterName(rpcapi.RaftServiceName, &raftRPC{raft: s.raft}); err != nil {
		return err
	}
	r.Handle(rpcapi.RPCPath, rpcServer)
	return nil
}

func (k *kvRPC) Ping(args rpcapi.PingArgs, reply *rpcapi.PingReply) error {
	reply.Message = "pong!"
	return nil
}

func (k *kvRPC) Get(args rpcapi.GetArgs, reply *rpcapi.GetReply) error {
	value, leaderAddr, err := k.server.getValueInternal(context.Background(), args.Key)
	if errors.Is(err, raft.ErrNotLeader) {
		k.populateLeaderError(reply, leaderAddr)
		return nil
	}
	if err != nil {
		reply.Error = err.Error()
		reply.Found = false
		return nil
	}
	reply.Value = value
	reply.Found = true
	return nil
}

func (k *kvRPC) Set(args rpcapi.SetArgs, reply *rpcapi.MutateReply) error {
	applyErr, leaderAddr, err := k.server.applyCommandInternal(context.Background(), kv.Command{
		Op:    kv.OpSet,
		Key:   args.Key,
		Value: args.Value,
	})
	if errors.Is(err, raft.ErrNotLeader) {
		k.populateLeaderError(reply, leaderAddr)
		return nil
	}
	if err != nil {
		reply.Error = err.Error()
		return nil
	}
	if applyErr != nil {
		reply.Error = applyErr.Error()
		return nil
	}
	return nil
}

func (k *kvRPC) Insert(args rpcapi.InsertArgs, reply *rpcapi.MutateReply) error {
	applyErr, leaderAddr, err := k.server.applyCommandInternal(context.Background(), kv.Command{
		Op:    kv.OpInsert,
		Key:   args.Key,
		Value: args.Value,
	})
	if errors.Is(err, raft.ErrNotLeader) {
		k.populateLeaderError(reply, leaderAddr)
		return nil
	}
	if err != nil {
		reply.Error = err.Error()
		return nil
	}
	if applyErr != nil {
		reply.Error = applyErr.Error()
		return nil
	}
	return nil
}

func (k *kvRPC) Delete(args rpcapi.DeleteArgs, reply *rpcapi.MutateReply) error {
	applyErr, leaderAddr, err := k.server.applyCommandInternal(context.Background(), kv.Command{
		Op:  kv.OpDelete,
		Key: args.Key,
	})
	if errors.Is(err, raft.ErrNotLeader) {
		k.populateLeaderError(reply, leaderAddr)
		return nil
	}
	if err != nil {
		reply.Error = err.Error()
		return nil
	}
	if applyErr != nil {
		reply.Error = applyErr.Error()
		return nil
	}
	return nil
}

func (k *kvRPC) Health(args rpcapi.HealthArgs, reply *rpcapi.HealthReply) error {
	status := k.server.raft.Status()
	reply.Status = rpcapi.HealthStatus{
		ID:           status.ID,
		Term:         status.Term,
		State:        status.State,
		LeaderID:     status.LeaderID,
		CommitIndex:  status.CommitIndex,
		LastApplied:  status.LastApplied,
		LastLogIndex: status.LastLogIndex,
	}
	return nil
}

func (k *kvRPC) populateLeaderError(reply interface{}, leaderAddr string) {
	switch r := reply.(type) {
	case *rpcapi.GetReply:
		if leaderAddr == "" {
			r.Error = rpcapi.ErrLeaderUnknown
		} else {
			r.Error = rpcapi.ErrNotLeader
			r.LeaderAddr = leaderAddr
		}
	case *rpcapi.MutateReply:
		if leaderAddr == "" {
			r.Error = rpcapi.ErrLeaderUnknown
		} else {
			r.Error = rpcapi.ErrNotLeader
			r.LeaderAddr = leaderAddr
		}
	}
}

func (r *raftRPC) RequestVote(args raft.RequestVoteArgs, reply *raft.RequestVoteReply) error {
	*reply = r.raft.HandleRequestVote(args)
	return nil
}

func (r *raftRPC) AppendEntries(args raft.AppendEntriesArgs, reply *raft.AppendEntriesReply) error {
	*reply = r.raft.HandleAppendEntries(args)
	return nil
}

func (r *raftRPC) InstallSnapshot(args raft.InstallSnapshotArgs, reply *raft.InstallSnapshotReply) error {
	*reply = r.raft.HandleInstallSnapshot(args)
	return nil
}
