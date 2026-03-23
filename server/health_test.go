package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wregret/breeoche/raft"
)

func TestHealthHandler(t *testing.T) {
	s := newTestServer(t)
	defer s.raft.Stop()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	s.healthHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var status raft.Status
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if status.ID != "n1" {
		t.Fatalf("expected id n1, got %s", status.ID)
	}
	if status.State == "" {
		t.Fatalf("expected state set")
	}
}
