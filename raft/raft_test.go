package raft

import "testing"

func newTestRaft(id string) *Raft {
	cfg := Config{
		ID:    id,
		Peers: map[string]string{},
	}
	r, err := NewRaft(cfg)
	if err != nil {
		panic(err)
	}
	return r
}

func TestRequestVoteRejectsStaleTerm(t *testing.T) {
	r := newTestRaft("n1")
	r.currentTerm = 2

	reply := r.HandleRequestVote(RequestVoteArgs{
		Term:         1,
		CandidateID:  "n2",
		LastLogIndex: 0,
		LastLogTerm:  0,
	})

	if reply.VoteGranted {
		t.Fatalf("expected vote denied for stale term")
	}
	if reply.Term != 2 {
		t.Fatalf("expected term 2, got %d", reply.Term)
	}
}

func TestRequestVoteGrantsWhenUpToDate(t *testing.T) {
	r := newTestRaft("n1")
	r.currentTerm = 2
	r.votedFor = ""
	r.log = []LogEntry{{Term: 0}, {Term: 1}, {Term: 2}}

	reply := r.HandleRequestVote(RequestVoteArgs{
		Term:         3,
		CandidateID:  "n2",
		LastLogIndex: 2,
		LastLogTerm:  2,
	})

	if !reply.VoteGranted {
		t.Fatalf("expected vote granted")
	}
	if r.votedFor != "n2" {
		t.Fatalf("expected votedFor to be set")
	}
	if r.currentTerm != 3 {
		t.Fatalf("expected term updated to 3")
	}
}

func TestAppendEntriesRejectsStaleTerm(t *testing.T) {
	r := newTestRaft("n1")
	r.currentTerm = 3

	reply := r.HandleAppendEntries(AppendEntriesArgs{
		Term:         2,
		LeaderID:     "n2",
		PrevLogIndex: 0,
		PrevLogTerm:  0,
	})

	if reply.Success {
		t.Fatalf("expected append rejected for stale term")
	}
	if reply.Term != 3 {
		t.Fatalf("expected term 3, got %d", reply.Term)
	}
}

func TestAppendEntriesConflictAndAppend(t *testing.T) {
	r := newTestRaft("n1")
	r.currentTerm = 2
	r.log = []LogEntry{{Term: 0}, {Term: 1}, {Term: 2}, {Term: 2}}

	reply := r.HandleAppendEntries(AppendEntriesArgs{
		Term:         3,
		LeaderID:     "n2",
		PrevLogIndex: 2,
		PrevLogTerm:  2,
		Entries: []LogEntry{
			{Term: 3, Command: []byte("set x 1")},
		},
		LeaderCommit: 2,
	})

	if !reply.Success {
		t.Fatalf("expected append success")
	}
	if got := r.log[3].Term; got != 3 {
		t.Fatalf("expected conflict overwrite term 3, got %d", got)
	}
	if r.currentTerm != 3 {
		t.Fatalf("expected term updated to 3")
	}
}

func TestCandidateStepsDownOnHigherTermAppendEntries(t *testing.T) {
	r := newTestRaft("n1")
	r.state = Candidate
	r.currentTerm = 2

	reply := r.HandleAppendEntries(AppendEntriesArgs{
		Term:         3,
		LeaderID:     "n2",
		PrevLogIndex: 0,
		PrevLogTerm:  0,
	})

	if !reply.Success {
		t.Fatalf("expected append success")
	}
	if r.state != Follower {
		t.Fatalf("expected follower state, got %v", r.state)
	}
	if r.currentTerm != 3 {
		t.Fatalf("expected term 3, got %d", r.currentTerm)
	}
	if r.leaderID != "n2" {
		t.Fatalf("expected leader id n2, got %s", r.leaderID)
	}
}

func TestLeaderCommitIndexAdvancesWithMajority(t *testing.T) {
	r := newTestRaft("n1")
	r.state = Leader
	r.currentTerm = 2
	r.log = []LogEntry{{Term: 0}, {Term: 1}, {Term: 2}, {Term: 2}}
	r.commitIndex = 0
	r.peers = map[string]string{
		"n2": "",
		"n3": "",
	}
	r.matchIndex = map[string]int{
		"n1": 3,
		"n2": 2,
		"n3": 2,
	}

	r.advanceCommitIndex()

	if r.commitIndex != 2 {
		t.Fatalf("expected commitIndex 2, got %d", r.commitIndex)
	}
}

func TestStartAppendsOnlyOnLeader(t *testing.T) {
	r := newTestRaft("n1")
	r.state = Follower
	if _, _, ok := r.Start([]byte("noop")); ok {
		t.Fatalf("expected follower to reject Start")
	}

	r.state = Leader
	r.currentTerm = 2
	index, term, ok := r.Start([]byte("set x 1"))
	if !ok {
		t.Fatalf("expected leader to accept Start")
	}
	if term != 2 {
		t.Fatalf("expected term 2, got %d", term)
	}
	if index != r.lastLogIndex() {
		t.Fatalf("expected index %d, got %d", r.lastLogIndex(), index)
	}
}
