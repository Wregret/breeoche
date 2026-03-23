package raft

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestVerboseLogsAppendEntries(t *testing.T) {
	var buf bytes.Buffer
	orig := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(orig)

	r, err := NewRaft(Config{
		ID:      "n1",
		Peers:   map[string]string{},
		Storage: NewMemoryStorage(),
		ApplyCh: make(chan ApplyMsg, 1),
		Verbose: true,
	})
	if err != nil {
		t.Fatalf("new raft: %v", err)
	}

	r.HandleAppendEntries(AppendEntriesArgs{
		Term:         1,
		LeaderID:     "n2",
		PrevLogIndex: 0,
		PrevLogTerm:  0,
		Entries:      nil,
		LeaderCommit: 0,
	})

	if !strings.Contains(buf.String(), "append entries recv") {
		t.Fatalf("expected verbose append entries log, got: %s", buf.String())
	}
}
