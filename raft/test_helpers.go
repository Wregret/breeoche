package raft

// TriggerElection starts an election immediately. Intended for tests or simulations.
func (r *Raft) TriggerElection() {
	r.startElection()
}

// ReplicateOnce sends a single round of AppendEntries to all peers. Intended for tests.
func (r *Raft) ReplicateOnce() {
	r.broadcastAppendEntries()
}
