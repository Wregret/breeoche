package raft

// Status provides a snapshot of a node's state.
type Status struct {
	ID           string `json:"id"`
	Term         int    `json:"term"`
	State        string `json:"state"`
	LeaderID     string `json:"leader_id"`
	CommitIndex  int    `json:"commit_index"`
	LastApplied  int    `json:"last_applied"`
	LastLogIndex int    `json:"last_log_index"`
}

// Status returns a snapshot of Raft state for diagnostics.
func (r *Raft) Status() Status {
	r.mu.Lock()
	defer r.mu.Unlock()
	return Status{
		ID:           r.id,
		Term:         r.currentTerm,
		State:        r.state.String(),
		LeaderID:     r.leaderID,
		CommitIndex:  r.commitIndex,
		LastApplied:  r.lastApplied,
		LastLogIndex: r.lastLogIndex(),
	}
}
