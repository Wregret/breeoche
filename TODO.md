# TODO

## Reliability and Correctness
- Add automatic snapshotting and InstallSnapshot RPC to keep followers in sync.
- Add deterministic Raft integration tests (fake clock + in-memory transport).
- Add install-time config validation for cluster membership.

## Performance and Operations
- Add metrics and health endpoints (latency, commit lag, leader ID).
- Add CLI helpers to list cluster status.

## Features
- Add read-index or lease-based linearizable reads (optional).
- Add support for batch writes.

## Dev Experience
- Add Makefile targets for build/test/lint.
- Add CI workflow for `go test ./...`.
