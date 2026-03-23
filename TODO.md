# TODO

## Reliability and Correctness
- Add chunked/streamed InstallSnapshot for large snapshots.
- Add deterministic Raft integration tests (fake clock + in-memory transport).
- Add install-time config validation for cluster membership.

## Performance and Operations
- Add metrics and health endpoints (latency, commit lag, leader ID).
- Add CLI helpers to list cluster status.
- Add size/time based snapshot policies.

## Features
- Add read-index or lease-based linearizable reads (optional).
- Add support for batch writes.

## Dev Experience
- Add Makefile targets for build/test/lint.
- Add CI workflow for `go test ./...`.
