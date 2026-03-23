package server

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func TestServerVerboseLogsHTTP(t *testing.T) {
	var buf bytes.Buffer
	orig := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(orig)

	s, err := NewServer(Config{
		ID:      "n1",
		Addr:    "127.0.0.1:0",
		Peers:   map[string]string{},
		DataDir: t.TempDir(),
		Verbose: true,
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	go s.applyLoop()
	s.raft.Run()
	defer s.raft.Stop()
	waitForLeader(t, s, 2*time.Second)

	setReq := httptest.NewRequest(http.MethodPost, "/key/a", strings.NewReader("1"))
	setReq = mux.SetURLVars(setReq, map[string]string{"key": "a"})
	setRec := httptest.NewRecorder()
	s.postHandler(setRec, setReq)
	if setRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", setRec.Code)
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if strings.Contains(buf.String(), "http POST /key/a") {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected verbose HTTP log, got: %s", buf.String())
}
