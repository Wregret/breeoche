package raft

import "fmt"

// String returns a human-readable representation of State.
func (s State) String() string {
	switch s {
	case Follower:
		return "follower"
	case Candidate:
		return "candidate"
	case Leader:
		return "leader"
	default:
		return fmt.Sprintf("state(%d)", int(s))
	}
}
