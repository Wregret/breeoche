package raft

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// HTTPTransport sends Raft RPCs over HTTP+JSON.
type HTTPTransport struct {
	client *http.Client
	peers  map[string]string
}

func NewHTTPTransport(peers map[string]string) *HTTPTransport {
	copyPeers := make(map[string]string, len(peers))
	for id, addr := range peers {
		copyPeers[id] = addr
	}
	return &HTTPTransport{
		client: &http.Client{Timeout: 2 * time.Second},
		peers:  copyPeers,
	}
}

func (t *HTTPTransport) RequestVote(ctx context.Context, peerID string, args RequestVoteArgs) (RequestVoteReply, error) {
	addr, ok := t.peers[peerID]
	if !ok {
		return RequestVoteReply{}, fmt.Errorf("unknown peer %s", peerID)
	}
	url := fmt.Sprintf("http://%s/raft/request-vote", addr)
	var reply RequestVoteReply
	if err := t.postJSON(ctx, url, args, &reply); err != nil {
		return RequestVoteReply{}, err
	}
	return reply, nil
}

func (t *HTTPTransport) AppendEntries(ctx context.Context, peerID string, args AppendEntriesArgs) (AppendEntriesReply, error) {
	addr, ok := t.peers[peerID]
	if !ok {
		return AppendEntriesReply{}, fmt.Errorf("unknown peer %s", peerID)
	}
	url := fmt.Sprintf("http://%s/raft/append-entries", addr)
	var reply AppendEntriesReply
	if err := t.postJSON(ctx, url, args, &reply); err != nil {
		return AppendEntriesReply{}, err
	}
	return reply, nil
}

func (t *HTTPTransport) postJSON(ctx context.Context, url string, reqBody interface{}, respBody interface{}) error {
	data, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		msg, _ := ioutil.ReadAll(res.Body)
		return errors.New(string(msg))
	}
	decoder := json.NewDecoder(res.Body)
	return decoder.Decode(respBody)
}
