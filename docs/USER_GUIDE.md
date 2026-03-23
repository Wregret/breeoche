# Breeoche User Guide

## Overview
Breeoche is a small key/value service backed by a Raft log. Writes go through the Raft leader; followers will redirect requests to the current leader.

## Build
1. `/usr/local/go/bin/go build -o breeoche ./...`
2. The binary `breeoche` will be created in the repository root.

## Start a 3-Node Cluster (Localhost)
1. Terminal 1:
   `./breeoche server --id n1 --host 127.0.0.1 --port 15213 --peers n2=127.0.0.1:15214,n3=127.0.0.1:15215`
2. Terminal 2:
   `./breeoche server --id n2 --host 127.0.0.1 --port 15214 --peers n1=127.0.0.1:15213,n3=127.0.0.1:15215`
3. Terminal 3:
   `./breeoche server --id n3 --host 127.0.0.1 --port 15215 --peers n1=127.0.0.1:15213,n2=127.0.0.1:15214`

Each node writes Raft state to `data/<node-id>/raft.json` by default.

## CLI Usage
1. Ping:
   `./breeoche ping --addr 127.0.0.1:15213`
2. Set (overwrite):
   `./breeoche set --addr 127.0.0.1:15213 mykey myvalue`
3. Get:
   `./breeoche get --addr 127.0.0.1:15213 mykey`
4. Insert (fails if key exists):
   `./breeoche insert --addr 127.0.0.1:15213 mykey myvalue`
5. Delete:
   `./breeoche delete --addr 127.0.0.1:15213 mykey`

If you point the client at a follower, it will follow HTTP redirects to the current leader automatically.

## HTTP API
- `GET /ping` -> `pong!`
- `GET /health` -> Raft status snapshot
- `GET /key/{key}`
- `POST /key/{key}` (body = value)
- `PUT /key/{key}` (body = value)
- `DELETE /key/{key}`

## Limitations
- No snapshots or log compaction; Raft logs grow without bound.
- Leader-only reads; no read-index or lease reads yet.
- Static cluster membership only; no dynamic reconfiguration.
- No authentication, TLS, or access control.
- No persistence of the KV state machine beyond Raft log replay.
- No metrics endpoint beyond `/health`.
- No fault-injection or chaos testing built in.
- No WAL or fsync guarantees beyond JSON file writes.
- No CLI for cluster status (beyond `/health`).
- No sharding or multi-key transactions.

## Operational Notes
- Reads are served only by the leader to keep the behavior simple and consistent.
- No snapshots or log compaction yet, so long-running clusters will accumulate log entries.
