# Breeoche

Breeoche is a lightweight key/value store built for learning distributed systems. It uses a Raft consensus log to replicate writes and keep data consistent across a small cluster.

## Highlights
- Raft-based replication for writes
- Leader-only reads (followers redirect)
- Simple HTTP API + CLI
- Local, file-backed Raft state (`data/<node-id>/raft.json`)
- Health endpoint with Raft status (`GET /health`)
- Automatic log compaction via snapshots

## Docs
- User guide: `docs/USER_GUIDE.md`
- Developer guide: `docs/DEVELOPER_GUIDE.md`
- Proposed features: `TODO.md`

## Build
```
/usr/local/go/bin/go build -o breeoche ./...
```

## Quick Start (Single Node)
```
./breeoche server --id n1 --host 127.0.0.1 --port 15213
```

```
./breeoche set --addr 127.0.0.1:15213 hello world
./breeoche get --addr 127.0.0.1:15213 hello
```

## Start a 3-Node Cluster (Localhost)
Terminal 1:
```
./breeoche server --id n1 --host 127.0.0.1 --port 15213 --peers n2=127.0.0.1:15214,n3=127.0.0.1:15215
```
Terminal 2:
```
./breeoche server --id n2 --host 127.0.0.1 --port 15214 --peers n1=127.0.0.1:15213,n3=127.0.0.1:15215
```
Terminal 3:
```
./breeoche server --id n3 --host 127.0.0.1 --port 15215 --peers n1=127.0.0.1:15213,n2=127.0.0.1:15214
```

## Server Config Notes
- Automatic snapshots are entry-count based. Use `--snapshot-threshold` to change how many new log entries trigger a snapshot (default: 100).
- Enable verbose logging of operations and Raft state changes with `--debug`.

## CLI Commands
- `ping` — health check
- `get <key>` — read value
- `set <key> <value>` — upsert
- `insert <key> <value>` — insert only (409 if key exists)
- `delete <key>` — delete key

All commands accept `--addr` (defaults to `localhost:15213`).

## HTTP API
- `GET /ping` -> `pong!`
- `GET /health` -> Raft status snapshot
- `GET /key/{key}`
- `POST /key/{key}` (body = value)
- `PUT /key/{key}` (body = value)
- `DELETE /key/{key}`

## Tests
```
/usr/local/go/bin/go test ./...
```

## Limitations
- Snapshot compaction is entry-count based; no size/time policy yet.
- InstallSnapshot exists but snapshots are not chunked/streamed.
- Leader-only reads; no read-index or lease reads yet.
- Static cluster membership only; no dynamic reconfiguration.
- No authentication, TLS, or access control.
- No metrics endpoint beyond `/health`.
- No fault-injection or chaos testing built in.
- No WAL or fsync guarantees beyond JSON file writes.
- No CLI for cluster status (beyond `/health`).
- No sharding or multi-key transactions.
