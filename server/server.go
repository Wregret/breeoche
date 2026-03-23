package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Wregret/breeoche/kv"
	"github.com/Wregret/breeoche/raft"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"sync"
	"time"
)

// Config configures a Breeoche server.
type Config struct {
	ID                string
	Addr              string
	Peers             map[string]string
	DataDir           string
	SnapshotThreshold int
	Debug             bool
	Verbose           bool
}

// Server hosts the KV API and Raft RPC endpoints.
type Server struct {
	id    string
	addr  string
	peers map[string]string

	store   *kv.Store
	raft    *raft.Raft
	applyCh chan raft.ApplyMsg

	snapshotThreshold int
	debug             bool
	verbose           bool

	applyMu      sync.Mutex
	applyWaiters map[int]chan error
	applyResults map[int]error
}

// NewServer initializes a server with Raft-backed storage.
func NewServer(cfg Config) (*Server, error) {
	if cfg.ID == "" {
		return nil, errors.New("server: id is required")
	}
	if cfg.Addr == "" {
		return nil, errors.New("server: addr is required")
	}
	if cfg.DataDir == "" {
		cfg.DataDir = "data"
	}
	if cfg.SnapshotThreshold == 0 {
		cfg.SnapshotThreshold = 100
	}

	allPeers := make(map[string]string, len(cfg.Peers)+1)
	for id, addr := range cfg.Peers {
		allPeers[id] = addr
	}
	allPeers[cfg.ID] = cfg.Addr

	raftPeers := make(map[string]string)
	for id, addr := range allPeers {
		if id == cfg.ID {
			continue
		}
		raftPeers[id] = addr
	}

	applyCh := make(chan raft.ApplyMsg, 256)
	storage := raft.NewFileStorage(filepath.Join(cfg.DataDir, cfg.ID, "raft.json"))
	transport := raft.NewHTTPTransport(raftPeers)
	r, err := raft.NewRaft(raft.Config{
		ID:        cfg.ID,
		Peers:     raftPeers,
		Transport: transport,
		Storage:   storage,
		ApplyCh:   applyCh,
		Debug:     cfg.Debug || cfg.Verbose,
		Verbose:   cfg.Verbose,
	})
	if err != nil {
		return nil, err
	}

	s := &Server{
		id:                cfg.ID,
		addr:              cfg.Addr,
		peers:             allPeers,
		store:             kv.NewStore(),
		raft:              r,
		applyCh:           applyCh,
		applyWaiters:      map[int]chan error{},
		applyResults:      map[int]error{},
		snapshotThreshold: cfg.SnapshotThreshold,
		debug:             cfg.Debug || cfg.Verbose,
		verbose:           cfg.Verbose,
	}
	if err := s.restoreSnapshot(); err != nil {
		return nil, err
	}
	return s, nil
}

// Start begins serving HTTP requests and Raft background work.
func (s *Server) Start() error {
	go s.applyLoop()
	s.raft.Run()

	r := mux.NewRouter()
	r.HandleFunc("/ping", s.pingHandler)
	r.HandleFunc("/key/{key}", s.getHandler).Methods(http.MethodGet)
	r.HandleFunc("/key/{key}", s.postHandler).Methods(http.MethodPost)
	r.HandleFunc("/key/{key}", s.putHandler).Methods(http.MethodPut)
	r.HandleFunc("/key/{key}", s.deleteHandler).Methods(http.MethodDelete)
	r.HandleFunc("/health", s.healthHandler).Methods(http.MethodGet)
	r.HandleFunc("/raft/request-vote", s.requestVoteHandler).Methods(http.MethodPost)
	r.HandleFunc("/raft/append-entries", s.appendEntriesHandler).Methods(http.MethodPost)
	r.HandleFunc("/raft/install-snapshot", s.installSnapshotHandler).Methods(http.MethodPost)

	log.Println("start server on: " + s.addr)
	s.debugf("debug logging enabled")
	s.verbosef("verbose logging enabled")
	return http.ListenAndServe(s.addr, r)
}

func (s *Server) pingHandler(w http.ResponseWriter, r *http.Request) {
	s.debugf("%s!", r.URL.Path[1:])
	w.Write([]byte("pong!"))
}

func getKey(r *http.Request) string {
	vars := mux.Vars(r)
	return vars["key"]
}

func (s *Server) getHandler(w http.ResponseWriter, r *http.Request) {
	s.verbosef("http %s %s", r.Method, r.URL.Path)
	if s.redirectIfNotLeader(w, r) {
		return
	}
	key := getKey(r)
	s.debugf("get key %s", key)

	value, ok := s.store.Get(key)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf("key %s not found", key)))
		return
	}

	w.Write([]byte(value))
}

func (s *Server) postHandler(w http.ResponseWriter, r *http.Request) {
	key := getKey(r)
	value, err := readBody(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
		return
	}
	cmd := kv.Command{Op: kv.OpSet, Key: key, Value: value}
	s.applyCommand(w, r, cmd)
}

func (s *Server) putHandler(w http.ResponseWriter, r *http.Request) {
	key := getKey(r)
	value, err := readBody(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
		return
	}
	cmd := kv.Command{Op: kv.OpInsert, Key: key, Value: value}
	s.applyCommand(w, r, cmd)
}

func (s *Server) deleteHandler(w http.ResponseWriter, r *http.Request) {
	key := getKey(r)
	cmd := kv.Command{Op: kv.OpDelete, Key: key}
	s.applyCommand(w, r, cmd)
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	status := s.raft.Status()
	writeJSON(w, status)
}

func (s *Server) applyCommand(w http.ResponseWriter, r *http.Request, cmd kv.Command) {
	if s.redirectIfNotLeader(w, r) {
		return
	}

	s.verbosef("http %s %s op=%s key=%s", r.Method, r.URL.Path, cmd.Op, cmd.Key)
	s.debugf("command op=%s key=%s", cmd.Op, cmd.Key)
	data, err := kv.EncodeCommand(cmd)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("encode error"))
		return
	}

	index, term, ok := s.raft.Start(data)
	if !ok {
		s.redirectIfNotLeader(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if err := s.raft.WaitForCommit(ctx, index, term); err != nil {
		if errors.Is(err, raft.ErrNotLeader) {
			s.redirectIfNotLeader(w, r)
			return
		}
		w.WriteHeader(http.StatusGatewayTimeout)
		w.Write([]byte("commit timeout"))
		return
	}

	applyErr := s.waitForApply(ctx, index)
	if applyErr != nil {
		s.writeApplyError(w, applyErr)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) writeApplyError(w http.ResponseWriter, err error) {
	switch err {
	case kv.ErrKeyExists:
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(err.Error()))
	case kv.ErrKeyNotFound:
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(err.Error()))
	default:
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("apply error"))
	}
}

func (s *Server) redirectIfNotLeader(w http.ResponseWriter, r *http.Request) bool {
	_, isLeader := s.raft.State()
	if isLeader {
		return false
	}
	leaderID := s.raft.LeaderID()
	if leaderID == "" || leaderID == s.id {
		s.debugf("leader unknown for %s", r.URL.Path)
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("leader unknown"))
		return true
	}
	addr := s.peers[leaderID]
	if addr == "" {
		s.debugf("leader address missing for %s", leaderID)
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("leader unknown"))
		return true
	}
	target := fmt.Sprintf("http://%s%s", addr, r.URL.RequestURI())
	s.debugf("redirecting to leader %s", leaderID)
	http.Redirect(w, r, target, http.StatusTemporaryRedirect)
	return true
}

func readBody(r *http.Request) (string, error) {
	defer r.Body.Close()
	value, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return "", err
	}
	return string(value), nil
}

func (s *Server) waitForApply(ctx context.Context, index int) error {
	s.applyMu.Lock()
	if result, ok := s.applyResults[index]; ok {
		delete(s.applyResults, index)
		s.applyMu.Unlock()
		return result
	}
	ch := make(chan error, 1)
	s.applyWaiters[index] = ch
	s.applyMu.Unlock()

	select {
	case err := <-ch:
		return err
	case <-ctx.Done():
		s.applyMu.Lock()
		if s.applyWaiters[index] == ch {
			delete(s.applyWaiters, index)
		}
		s.applyMu.Unlock()
		return ctx.Err()
	}
}

func (s *Server) applyLoop() {
	for msg := range s.applyCh {
		if msg.Snapshot {
			s.applySnapshot(msg)
			continue
		}
		var applyErr error
		cmd, err := kv.DecodeCommand(msg.Command)
		if err != nil {
			applyErr = err
		} else {
			applyErr = s.store.Apply(cmd)
		}
		s.applyMu.Lock()
		if ch, ok := s.applyWaiters[msg.Index]; ok {
			delete(s.applyWaiters, msg.Index)
			s.applyMu.Unlock()
			ch <- applyErr
			close(ch)
		} else {
			_, isLeader := s.raft.State()
			if isLeader {
				s.applyResults[msg.Index] = applyErr
			}
			s.applyMu.Unlock()
		}

		if err != nil {
			s.debugf("apply decode error index=%d err=%v", msg.Index, err)
		} else {
			s.debugf("applied index=%d op=%s key=%s err=%v", msg.Index, cmd.Op, cmd.Key, applyErr)
		}
		s.maybeSnapshot(msg.Index)
	}
}

func (s *Server) applySnapshot(msg raft.ApplyMsg) {
	err := s.store.RestoreSnapshot(msg.SnapshotData)
	s.debugf("apply snapshot index=%d term=%d err=%v", msg.SnapshotIndex, msg.SnapshotTerm, err)
	s.applyMu.Lock()
	for idx, ch := range s.applyWaiters {
		if idx <= msg.SnapshotIndex {
			delete(s.applyWaiters, idx)
			ch <- err
			close(ch)
		}
	}
	s.applyMu.Unlock()
}

func (s *Server) maybeSnapshot(index int) {
	if s.snapshotThreshold <= 0 {
		return
	}
	logOffset := s.raft.LogOffset()
	if index-logOffset < s.snapshotThreshold {
		return
	}
	data, err := s.store.Snapshot()
	if err != nil {
		s.debugf("snapshot skipped index=%d err=%v", index, err)
		return
	}
	if err := s.raft.Snapshot(index, data); err == nil {
		s.debugf("snapshot created index=%d", index)
	}
}

func (s *Server) restoreSnapshot() error {
	snap := s.raft.SnapshotState()
	if snap.LastIncludedIndex == 0 {
		return nil
	}
	s.debugf("restore snapshot index=%d term=%d", snap.LastIncludedIndex, snap.LastIncludedTerm)
	return s.store.RestoreSnapshot(snap.Data)
}

func (s *Server) debugf(format string, args ...interface{}) {
	if !s.debug {
		return
	}
	allArgs := append([]interface{}{s.id}, args...)
	log.Printf("server[%s] "+format, allArgs...)
}

func (s *Server) verbosef(format string, args ...interface{}) {
	if !s.verbose {
		return
	}
	allArgs := append([]interface{}{s.id}, args...)
	log.Printf("server[%s] "+format, allArgs...)
}

func (s *Server) requestVoteHandler(w http.ResponseWriter, r *http.Request) {
	s.verbosef("http %s %s", r.Method, r.URL.Path)
	var args raft.RequestVoteArgs
	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
		return
	}
	reply := s.raft.HandleRequestVote(args)
	writeJSON(w, reply)
}

func (s *Server) appendEntriesHandler(w http.ResponseWriter, r *http.Request) {
	s.verbosef("http %s %s", r.Method, r.URL.Path)
	var args raft.AppendEntriesArgs
	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
		return
	}
	reply := s.raft.HandleAppendEntries(args)
	writeJSON(w, reply)
}

func (s *Server) installSnapshotHandler(w http.ResponseWriter, r *http.Request) {
	s.verbosef("http %s %s", r.Method, r.URL.Path)
	var args raft.InstallSnapshotArgs
	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
		return
	}
	reply := s.raft.HandleInstallSnapshot(args)
	writeJSON(w, reply)
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("encode error"))
		return
	}
}
