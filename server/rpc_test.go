package server

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wregret/breeoche/rpcapi"
)

func TestRPCSetGet(t *testing.T) {
	s, err := NewServer(Config{
		ID:      "n1",
		Addr:    "127.0.0.1:0",
		Peers:   map[string]string{},
		DataDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	go s.applyLoop()
	s.raft.Run()
	defer s.raft.Stop()
	waitForLeader(t, s, 2*time.Second)

	router, err := s.buildRouter()
	if err != nil {
		t.Fatalf("build router: %v", err)
	}
	ts := httptest.NewServer(router)
	defer ts.Close()
	addr := strings.TrimPrefix(ts.URL, "http://")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	client, err := rpcapi.Dial(ctx, addr)
	if err != nil {
		t.Fatalf("rpc dial: %v", err)
	}
	defer client.Close()

	setArgs := rpcapi.SetArgs{Key: "a", Value: "1"}
	var setReply rpcapi.MutateReply
	if err := client.Call(rpcapi.KVServiceName+".Set", setArgs, &setReply); err != nil {
		t.Fatalf("rpc set: %v", err)
	}
	if setReply.Error != "" {
		t.Fatalf("unexpected set error: %s", setReply.Error)
	}

	getArgs := rpcapi.GetArgs{Key: "a"}
	var getReply rpcapi.GetReply
	if err := client.Call(rpcapi.KVServiceName+".Get", getArgs, &getReply); err != nil {
		t.Fatalf("rpc get: %v", err)
	}
	if getReply.Error != "" {
		t.Fatalf("unexpected get error: %s", getReply.Error)
	}
	if !getReply.Found || getReply.Value != "1" {
		t.Fatalf("expected value 1, got %q", getReply.Value)
	}
}
