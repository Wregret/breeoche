# Breeoche Developer Guide

## Architecture
- `raft`: Core Raft implementation (election, log replication, commit tracking).
- `server`: HTTP API and Raft RPC endpoints, applies committed log entries to the KV store.
- `kv`: In-memory key/value state machine.
- `client`: CLI-friendly HTTP client with leader redirects.
- `cmd`: Cobra commands for server and client operations.

## Raft Details
- Persistent state: `currentTerm`, `votedFor`, `log`, and `commitIndex` are stored in JSON at `data/<node-id>/raft.json`.
- RPCs: `RequestVote` and `AppendEntries` over HTTP JSON.
- Leader election: randomized timeouts with self-vote; majority wins.
- Log replication: leader tracks `nextIndex` and `matchIndex` for followers.
- Apply path: committed entries are delivered on `applyCh` and applied to the KV store.

## Server Flow
1. External HTTP write arrives.
2. Leader appends entry via `raft.Start`.
3. Leader replicates to followers.
4. Entry is committed, then applied to the KV store.
5. Handler returns success (or conflict for insert/delete).

## Tests
- `raft/raft_test.go`: core Raft logic tests (vote rules, append conflict, commit rules, Start behavior).
- `raft/cluster_test.go`: in-memory transport tests for leader election and replication.
- `kv/kv_test.go`: state machine tests (set/insert/delete + codec).
- `server/server_test.go`: single-node integration tests with Raft apply.

Run tests with:
- `go test ./...`

## Extending the System
- Add new KV operations: extend `kv.Command`, update `Store.Apply`, and adjust HTTP handlers.
- Add log compaction: introduce snapshots and log truncation in the `raft` package.
- Add linearizable reads: implement a read-index or lease-based read path in `server`.

## Operational Caveats
- No snapshotting or compaction yet; logs can grow without bound.
- Reads are leader-only (followers redirect).
- Time-based behavior (elections, heartbeats) is not yet covered by a fake clock.
