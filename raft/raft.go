package raft

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"
)

var (
	ErrNotLeader = errors.New("raft: not leader")
)

// Raft is a single Raft node.
type Raft struct {
	mu sync.Mutex

	id    string
	peers map[string]string

	transport Transport
	storage   Storage
	clock     Clock

	state       State
	currentTerm int
	votedFor    string
	log         []LogEntry

	commitIndex int
	lastApplied int

	nextIndex  map[string]int
	matchIndex map[string]int

	applyCh chan ApplyMsg

	notifyCh map[int]chan struct{}

	electionTimeout   time.Duration
	heartbeatInterval time.Duration

	resetElectionCh chan struct{}
	stopCh          chan struct{}

	leaderID string
	rand     *rand.Rand
}

// NewRaft creates a Raft node with the provided configuration.
func NewRaft(cfg Config) (*Raft, error) {
	if cfg.ID == "" {
		return nil, errors.New("raft: ID is required")
	}
	storage := cfg.Storage
	if storage == nil {
		storage = NewMemoryStorage()
	}
	state, err := storage.Load()
	if err != nil {
		return nil, err
	}
	if cfg.ApplyCh == nil {
		cfg.ApplyCh = make(chan ApplyMsg, 128)
	}
	if cfg.ElectionTimeout == 0 {
		cfg.ElectionTimeout = 350
	}
	if cfg.HeartbeatInterval == 0 {
		cfg.HeartbeatInterval = 100
	}
	if len(state.Log) == 0 {
		// Index 0 is a sentinel entry to simplify boundary checks.
		state.Log = []LogEntry{{Term: 0}}
	}

	clock := cfg.Clock
	if clock == nil {
		clock = realClock{}
	}

	r := &Raft{
		id:                cfg.ID,
		peers:             cfg.Peers,
		transport:         cfg.Transport,
		storage:           storage,
		clock:             clock,
		state:             Follower,
		currentTerm:       state.CurrentTerm,
		votedFor:          state.VotedFor,
		log:               state.Log,
		commitIndex:       state.CommitIndex,
		lastApplied:       0,
		nextIndex:         map[string]int{},
		matchIndex:        map[string]int{},
		applyCh:           cfg.ApplyCh,
		notifyCh:          map[int]chan struct{}{},
		electionTimeout:   time.Duration(cfg.ElectionTimeout) * time.Millisecond,
		heartbeatInterval: time.Duration(cfg.HeartbeatInterval) * time.Millisecond,
		resetElectionCh:   make(chan struct{}, 1),
		stopCh:            make(chan struct{}),
		rand:              rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	if r.commitIndex > r.lastLogIndex() {
		r.commitIndex = r.lastLogIndex()
	}
	if r.lastApplied > r.commitIndex {
		r.lastApplied = r.commitIndex
	}

	return r, nil
}

// State returns the current term and whether this node believes it is leader.
func (r *Raft) State() (int, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.currentTerm, r.state == Leader
}

// LeaderID returns the last known leader ID.
func (r *Raft) LeaderID() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.leaderID
}

// Run starts background goroutines for elections and heartbeats.
func (r *Raft) Run() {
	go r.runElectionTimer()
	go r.runHeartbeatLoop()
	go r.applyCommitted()
}

// Stop signals all background goroutines to exit.
func (r *Raft) Stop() {
	close(r.stopCh)
}

// Start appends a new log entry to the Raft log (leader only).
func (r *Raft) Start(command []byte) (int, int, bool) {
	r.mu.Lock()
	if r.state != Leader {
		term := r.currentTerm
		r.mu.Unlock()
		return -1, term, false
	}

	entry := LogEntry{Term: r.currentTerm, Command: command}
	r.log = append(r.log, entry)
	index := r.lastLogIndex()
	term := r.currentTerm
	r.matchIndex[r.id] = index
	r.nextIndex[r.id] = index + 1
	ch := make(chan struct{})
	r.notifyCh[index] = ch
	_ = r.persistLocked()
	r.mu.Unlock()

	r.advanceCommitIndex()
	r.broadcastAppendEntries()
	return index, term, true
}

// WaitForCommit blocks until the given index is committed or context expires.
func (r *Raft) WaitForCommit(ctx context.Context, index int, term int) error {
	r.mu.Lock()
	if r.currentTerm != term || r.state != Leader {
		r.mu.Unlock()
		return ErrNotLeader
	}
	if r.commitIndex >= index {
		r.mu.Unlock()
		return nil
	}
	ch := r.notifyCh[index]
	if ch == nil {
		ch = make(chan struct{})
		r.notifyCh[index] = ch
	}
	r.mu.Unlock()

	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		r.mu.Lock()
		if r.notifyCh[index] == ch {
			delete(r.notifyCh, index)
		}
		r.mu.Unlock()
		return ctx.Err()
	}
}

// HandleRequestVote processes a RequestVote RPC.
func (r *Raft) HandleRequestVote(args RequestVoteArgs) RequestVoteReply {
	r.mu.Lock()
	defer r.mu.Unlock()

	reply := RequestVoteReply{Term: r.currentTerm, VoteGranted: false}
	if args.Term < r.currentTerm {
		return reply
	}
	if args.Term > r.currentTerm {
		r.becomeFollowerLocked(args.Term)
	}

	if (r.votedFor == "" || r.votedFor == args.CandidateID) && r.isUpToDate(args.LastLogIndex, args.LastLogTerm) {
		r.votedFor = args.CandidateID
		r.persistLocked()
		reply.VoteGranted = true
		reply.Term = r.currentTerm
		r.resetElectionTimerLocked()
		return reply
	}

	reply.Term = r.currentTerm
	return reply
}

func (r *Raft) runElectionTimer() {
	for {
		timeout := r.randomizedElectionTimeout()
		timer := r.clock.NewTimer(timeout)
		select {
		case <-timer.C():
			r.startElection()
		case <-r.resetElectionCh:
			if !timer.Stop() {
				select {
				case <-timer.C():
				default:
				}
			}
		case <-r.stopCh:
			if !timer.Stop() {
				select {
				case <-timer.C():
				default:
				}
			}
			return
		}
	}
}

func (r *Raft) runHeartbeatLoop() {
	ticker := r.clock.NewTicker(r.heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C():
			r.broadcastAppendEntries()
		case <-r.stopCh:
			return
		}
	}
}

func (r *Raft) startElection() {
	r.mu.Lock()
	if r.state == Leader {
		r.mu.Unlock()
		return
	}
	r.state = Candidate
	r.currentTerm++
	r.votedFor = r.id
	term := r.currentTerm
	lastIndex := r.lastLogIndex()
	lastTerm := r.lastLogTerm()
	_ = r.persistLocked()
	r.mu.Unlock()

	if r.quorumSize() == 1 {
		r.mu.Lock()
		if r.currentTerm == term {
			r.becomeLeaderLocked()
		}
		r.mu.Unlock()
		r.broadcastAppendEntries()
		return
	}

	var voteMu sync.Mutex
	votes := 1

	for peerID := range r.peers {
		peerID := peerID
		go func() {
			if r.transport == nil {
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), r.electionTimeout)
			defer cancel()
			reply, err := r.transport.RequestVote(ctx, peerID, RequestVoteArgs{
				Term:         term,
				CandidateID:  r.id,
				LastLogIndex: lastIndex,
				LastLogTerm:  lastTerm,
			})
			if err != nil {
				return
			}

			r.mu.Lock()
			defer r.mu.Unlock()
			if r.state != Candidate || r.currentTerm != term {
				return
			}
			if reply.Term > r.currentTerm {
				r.becomeFollowerLocked(reply.Term)
				return
			}
			if reply.VoteGranted {
				voteMu.Lock()
				votes++
				shouldLead := votes >= r.quorumSize()
				voteMu.Unlock()
				if shouldLead {
					r.becomeLeaderLocked()
					go r.broadcastAppendEntries()
				}
			}
		}()
	}
}

// HandleAppendEntries processes an AppendEntries RPC.
func (r *Raft) HandleAppendEntries(args AppendEntriesArgs) AppendEntriesReply {
	r.mu.Lock()

	reply := AppendEntriesReply{Term: r.currentTerm, Success: false}
	if args.Term < r.currentTerm {
		r.mu.Unlock()
		return reply
	}
	if args.Term > r.currentTerm {
		r.becomeFollowerLocked(args.Term)
	} else if r.state != Follower {
		r.state = Follower
		r.persistLocked()
	}

	r.leaderID = args.LeaderID
	r.resetElectionTimerLocked()

	if args.PrevLogIndex > r.lastLogIndex() {
		reply.ConflictIndex = r.lastLogIndex() + 1
		r.mu.Unlock()
		return reply
	}
	if args.PrevLogIndex >= 0 {
		if r.log[args.PrevLogIndex].Term != args.PrevLogTerm {
			conflictTerm := r.log[args.PrevLogIndex].Term
			idx := args.PrevLogIndex
			for idx > 0 && r.log[idx-1].Term == conflictTerm {
				idx--
			}
			reply.ConflictTerm = conflictTerm
			reply.ConflictIndex = idx
			r.mu.Unlock()
			return reply
		}
	}

	// Append entries, deleting conflicts.
	for i, entry := range args.Entries {
		idx := args.PrevLogIndex + 1 + i
		if idx <= r.lastLogIndex() {
			if r.log[idx].Term != entry.Term {
				r.log = append(r.log[:idx], args.Entries[i:]...)
				r.persistLocked()
				break
			}
			continue
		}
		r.log = append(r.log, args.Entries[i:]...)
		r.persistLocked()
		break
	}

	applyNeeded := false
	if args.LeaderCommit > r.commitIndex {
		r.commitIndex = min(args.LeaderCommit, r.lastLogIndex())
		r.persistLocked()
		applyNeeded = true
	}

	reply.Success = true
	reply.Term = r.currentTerm
	r.mu.Unlock()
	if applyNeeded {
		r.applyCommitted()
	}
	return reply
}

func (r *Raft) broadcastAppendEntries() {
	r.mu.Lock()
	if r.state != Leader {
		r.mu.Unlock()
		return
	}
	peers := make([]string, 0, len(r.peers))
	for peerID := range r.peers {
		peers = append(peers, peerID)
	}
	r.mu.Unlock()

	for _, peerID := range peers {
		peerID := peerID
		go r.sendAppendEntries(peerID)
	}
}

func (r *Raft) sendAppendEntries(peerID string) {
	if r.transport == nil {
		return
	}

	r.mu.Lock()
	if r.state != Leader {
		r.mu.Unlock()
		return
	}
	nextIdx := r.nextIndex[peerID]
	if nextIdx == 0 {
		nextIdx = r.lastLogIndex() + 1
		r.nextIndex[peerID] = nextIdx
	}
	prevIdx := nextIdx - 1
	prevTerm := r.log[prevIdx].Term
	entries := append([]LogEntry(nil), r.log[nextIdx:]...)
	args := AppendEntriesArgs{
		Term:         r.currentTerm,
		LeaderID:     r.id,
		PrevLogIndex: prevIdx,
		PrevLogTerm:  prevTerm,
		Entries:      entries,
		LeaderCommit: r.commitIndex,
	}
	r.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), r.heartbeatInterval)
	defer cancel()
	reply, err := r.transport.AppendEntries(ctx, peerID, args)
	if err != nil {
		return
	}
	r.handleAppendEntriesReply(peerID, args, reply)
}

func (r *Raft) handleAppendEntriesReply(peerID string, args AppendEntriesArgs, reply AppendEntriesReply) {
	r.mu.Lock()

	if r.state != Leader || args.Term != r.currentTerm {
		r.mu.Unlock()
		return
	}
	if reply.Term > r.currentTerm {
		r.becomeFollowerLocked(reply.Term)
		r.mu.Unlock()
		return
	}
	if reply.Success {
		matchIdx := args.PrevLogIndex + len(args.Entries)
		r.matchIndex[peerID] = matchIdx
		r.nextIndex[peerID] = matchIdx + 1
		r.mu.Unlock()
		r.advanceCommitIndex()
		return
	} else {
		if reply.ConflictTerm != 0 {
			lastIdx := r.lastIndexOfTerm(reply.ConflictTerm)
			if lastIdx > 0 {
				r.nextIndex[peerID] = lastIdx + 1
			} else {
				r.nextIndex[peerID] = reply.ConflictIndex
			}
		} else if reply.ConflictIndex > 0 {
			r.nextIndex[peerID] = reply.ConflictIndex
		} else {
			r.nextIndex[peerID] = max(1, r.nextIndex[peerID]-1)
		}
	}
	r.mu.Unlock()
}

func (r *Raft) applyCommitted() {
	for {
		r.mu.Lock()
		if r.lastApplied >= r.commitIndex {
			r.mu.Unlock()
			return
		}
		r.lastApplied++
		entry := r.log[r.lastApplied]
		ch := r.notifyCh[r.lastApplied]
		if ch != nil {
			delete(r.notifyCh, r.lastApplied)
		}
		r.mu.Unlock()
		if ch != nil {
			close(ch)
		}
		r.applyCh <- ApplyMsg{Index: r.lastApplied, Command: entry.Command}
	}
}

func (r *Raft) advanceCommitIndex() {
	r.mu.Lock()

	if r.state != Leader {
		r.mu.Unlock()
		return
	}

	lastIdx := r.lastLogIndex()
	advanced := false
	for idx := r.commitIndex + 1; idx <= lastIdx; idx++ {
		// Only commit entries from current term (Raft safety rule).
		if r.log[idx].Term != r.currentTerm {
			continue
		}
		count := 1 // self
		for peerID := range r.peers {
			if r.matchIndex[peerID] >= idx {
				count++
			}
		}
		if count >= r.quorumSize() {
			r.commitIndex = idx
			advanced = true
		}
	}
	if advanced {
		_ = r.persistLocked()
	}
	r.mu.Unlock()
	if advanced {
		r.applyCommitted()
	}
}

func (r *Raft) quorumSize() int {
	return (len(r.peers)+1)/2 + 1
}

func (r *Raft) lastLogIndex() int {
	return len(r.log) - 1
}

func (r *Raft) lastLogTerm() int {
	if len(r.log) == 0 {
		return 0
	}
	return r.log[len(r.log)-1].Term
}

func (r *Raft) isUpToDate(lastIndex, lastTerm int) bool {
	myTerm := r.lastLogTerm()
	if lastTerm != myTerm {
		return lastTerm > myTerm
	}
	return lastIndex >= r.lastLogIndex()
}

func (r *Raft) becomeLeaderLocked() {
	r.state = Leader
	r.leaderID = r.id
	lastIdx := r.lastLogIndex()
	r.nextIndex = map[string]int{}
	r.matchIndex = map[string]int{}
	for peerID := range r.peers {
		r.nextIndex[peerID] = lastIdx + 1
		r.matchIndex[peerID] = 0
	}
	r.matchIndex[r.id] = lastIdx
	r.nextIndex[r.id] = lastIdx + 1
}

func (r *Raft) becomeFollowerLocked(term int) {
	r.state = Follower
	if term > r.currentTerm {
		r.currentTerm = term
		r.votedFor = ""
	}
	r.persistLocked()
}

func (r *Raft) resetElectionTimerLocked() {
	select {
	case r.resetElectionCh <- struct{}{}:
	default:
	}
}

func (r *Raft) randomizedElectionTimeout() time.Duration {
	base := r.electionTimeout
	if base <= 0 {
		return 300 * time.Millisecond
	}
	// Randomize between base and 2*base to reduce split votes.
	return base + time.Duration(r.rand.Int63n(int64(base)))
}

func (r *Raft) persistLocked() error {
	if r.storage == nil {
		return nil
	}
	state := PersistentState{
		CurrentTerm: r.currentTerm,
		VotedFor:    r.votedFor,
		Log:         r.log,
		CommitIndex: r.commitIndex,
	}
	return r.storage.Save(state)
}

func (r *Raft) lastIndexOfTerm(term int) int {
	for i := r.lastLogIndex(); i >= 1; i-- {
		if r.log[i].Term == term {
			return i
		}
	}
	return 0
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
