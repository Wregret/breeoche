# Breeoche Developer Guide

## Architecture
- `raft`: Core Raft implementation (election, log replication, commit tracking).
- `server`: RPC + HTTP API endpoints, applies committed log entries to the KV store.
- `kv`: In-memory key/value state machine.
- `client`: CLI-friendly RPC client (default) with HTTP compatibility.
- `cmd`: Cobra commands for server and client operations.
- `rpcapi`: Shared RPC request/response types and constants.

## Raft Details
- Persistent state: `currentTerm`, `votedFor`, `log`, `commitIndex`, and `snapshot` are stored in JSON at `data/<node-id>/raft.json`.
- RPCs: `RequestVote`, `AppendEntries`, and `InstallSnapshot` over net/rpc (default). HTTP JSON endpoints remain for compatibility.
- Leader election: randomized timeouts with self-vote; majority wins.
- Log replication: leader tracks `nextIndex` and `matchIndex` for followers.
- Log compaction: snapshots trim the log. The server triggers snapshots automatically based on entry count; `Snapshot(index, data)` is available for manual compaction.
- Apply path: committed entries are delivered on `applyCh` and applied to the KV store. Snapshot installs send `ApplyMsg` with `Snapshot=true`.
- Status: `GET /health` returns a Raft status snapshot (`id`, `term`, `state`, `leader_id`, `commit_index`, `last_applied`, `last_log_index`).
- Clock abstraction: Raft accepts a `Clock` implementation so tests can use a deterministic `FakeClock`.

## Server Flow
1. External RPC/HTTP write arrives.
2. Leader appends entry via `raft.Start`.
3. Leader replicates to followers.
4. Entry is committed, then applied to the KV store.
5. Handler returns success (or conflict for insert/delete).

Enable tracing with `--debug` to log operations and Raft state transitions with node IDs. Use `--verbose` for per-RPC and HTTP request logs (implies `--debug`).

## Tests
- `raft/raft_test.go`: core Raft logic tests (vote rules, append conflict, commit rules, Start behavior).
- `raft/cluster_test.go`: in-memory transport tests for leader election and replication.
- `raft/clock_test.go`: deterministic election and heartbeat tests using `FakeClock`.
- `raft/snapshot_test.go`: snapshot/log compaction tests.
- `raft/install_snapshot_test.go`: InstallSnapshot apply tests.
- `raft/verbose_test.go`: verbose logging tests for Raft RPC tracing.
- `kv/kv_test.go`: state machine tests (set/insert/delete + codec).
- `server/server_test.go`: single-node integration tests with Raft apply.
- `server/health_test.go`: health endpoint tests.
- `server/snapshot_test.go`: automatic snapshot trigger tests.
- `server/verbose_test.go`: verbose logging tests for server RPC/HTTP tracing.

Run tests with:
- `go test ./...`

## Extending the System
- Add chunked/streamed snapshot transfer for large snapshots.
- Add new KV operations: extend `kv.Command`, update `Store.Apply`, and adjust HTTP handlers.
- Add log compaction triggers based on size/age.
- Add linearizable reads: implement a read-index or lease-based read path in `server`.

## Operational Caveats
- InstallSnapshot sends the entire snapshot in one RPC (no chunking/streaming yet).
- Reads are leader-only (followers redirect).
- Time-based behavior (elections, heartbeats) is not yet fully covered by a fake clock.
