# TODO

## Reliability and Correctness
- Add chunked/streamed InstallSnapshot for large snapshots.
- Add deterministic Raft integration tests that avoid wall-clock sleeps (fake clock + in-memory transport).
- Add install-time config validation for cluster membership.
- Add RPC-level integration tests for leader redirection and multi-node clusters.

## Performance and Operations
- Add metrics endpoint (latency, commit lag, leader ID).
- Add CLI helpers to list cluster status.
- Add size/time based snapshot policies.
- Add connection pooling or reuse for RPC transport.

## Features
- Add read-index or lease-based linearizable reads (optional).
- Add support for batch writes.
- Add optional authentication or TLS for RPC/HTTP.

## Dev Experience
- Add Makefile targets for build/test/lint.
- Add CI workflow for `go test ./...`.
- Add cross-protocol client tests (RPC vs HTTP parity).
