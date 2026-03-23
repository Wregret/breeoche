package raft

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestDebugLogsStateChanges(t *testing.T) {
	var buf bytes.Buffer
	orig := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(orig)

	r, err := NewRaft(Config{
		ID:      "n1",
		Peers:   map[string]string{},
		Storage: NewMemoryStorage(),
		ApplyCh: make(chan ApplyMsg, 1),
		Debug:   true,
	})
	if err != nil {
		t.Fatalf("new raft: %v", err)
	}

	r.TriggerElection()

	if !strings.Contains(buf.String(), "raft[n1] became leader") {
		t.Fatalf("expected debug leader log, got: %s", buf.String())
	}
}
