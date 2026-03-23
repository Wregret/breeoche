package raft

// State represents the role of a Raft node.
type State int

const (
	Follower State = iota
	Candidate
	Leader
)

// LogEntry is a single entry in the Raft log.
type LogEntry struct {
	Term    int
	Command []byte
}

// Snapshot holds compacted state up to LastIncludedIndex.
type Snapshot struct {
	LastIncludedIndex int
	LastIncludedTerm  int
	Data              []byte
}

// RequestVoteArgs is the request for a vote during elections.
type RequestVoteArgs struct {
	Term         int
	CandidateID  string
	LastLogIndex int
	LastLogTerm  int
}

// RequestVoteReply is the response for a RequestVote RPC.
type RequestVoteReply struct {
	Term        int
	VoteGranted bool
}

// AppendEntriesArgs is the request to replicate log entries (also heartbeat).
type AppendEntriesArgs struct {
	Term         int
	LeaderID     string
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
	LeaderCommit int
}

// AppendEntriesReply is the response for AppendEntries RPC.
type AppendEntriesReply struct {
	Term    int
	Success bool
	// Optional conflict hints for faster backtracking (not required for correctness).
	ConflictIndex int
	ConflictTerm  int
}

// InstallSnapshotArgs is sent by the leader to install a snapshot on a follower.
type InstallSnapshotArgs struct {
	Term              int
	LeaderID          string
	LastIncludedIndex int
	LastIncludedTerm  int
	Data              []byte
}

// InstallSnapshotReply is the response for InstallSnapshot RPC.
type InstallSnapshotReply struct {
	Term int
}

// ApplyMsg delivers committed log entries to the state machine.
type ApplyMsg struct {
	Index   int
	Command []byte
	// Snapshot indicates this ApplyMsg carries a snapshot instead of a log entry.
	Snapshot      bool
	SnapshotData  []byte
	SnapshotIndex int
	SnapshotTerm  int
}

// PersistentState is the part of Raft state that must survive crashes.
type PersistentState struct {
	CurrentTerm int
	VotedFor    string
	Log         []LogEntry
	CommitIndex int
	Snapshot    Snapshot
}

// Config configures a Raft node.
type Config struct {
	ID                string
	Peers             map[string]string
	Transport         Transport
	Storage           Storage
	ApplyCh           chan ApplyMsg
	ElectionTimeout   int // milliseconds
	HeartbeatInterval int // milliseconds
	Clock             Clock
}
