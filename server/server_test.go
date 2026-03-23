package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
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
	waitForLeader(t, s, 2*time.Second)
	return s
}

func waitForLeader(t *testing.T, s *Server, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		_, isLeader := s.raft.State()
		if isLeader {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("leader not elected within %s", timeout)
}

func TestServerSetAndGet(t *testing.T) {
	s := newTestServer(t)
	defer s.raft.Stop()

	setReq := httptest.NewRequest(http.MethodPost, "/key/a", strings.NewReader("1"))
	setReq = mux.SetURLVars(setReq, map[string]string{"key": "a"})
	setRec := httptest.NewRecorder()
	s.postHandler(setRec, setReq)
	if setRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", setRec.Code)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/key/a", nil)
	getReq = mux.SetURLVars(getReq, map[string]string{"key": "a"})
	getRec := httptest.NewRecorder()
	s.getHandler(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getRec.Code)
	}
	if body := strings.TrimSpace(getRec.Body.String()); body != "1" {
		t.Fatalf("expected value 1, got %q", body)
	}
}

func TestServerInsertConflict(t *testing.T) {
	s := newTestServer(t)
	defer s.raft.Stop()

	setReq := httptest.NewRequest(http.MethodPost, "/key/a", strings.NewReader("1"))
	setReq = mux.SetURLVars(setReq, map[string]string{"key": "a"})
	setRec := httptest.NewRecorder()
	s.postHandler(setRec, setReq)
	if setRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", setRec.Code)
	}

	insertReq := httptest.NewRequest(http.MethodPut, "/key/a", strings.NewReader("2"))
	insertReq = mux.SetURLVars(insertReq, map[string]string{"key": "a"})
	insertRec := httptest.NewRecorder()
	s.putHandler(insertRec, insertReq)
	if insertRec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", insertRec.Code)
	}
}
