# Breeoche

Breeoche is a simple key/value storage service for study purpose, now backed by a Raft consensus log.

## Roadmap

+ A command line interface (Cobra)
+ A hash map playing the role of k/v storage
+ Server-Client architecture
+ RPC
+ Distributed design
+ Log replication (Raft)
+ ...

## Docs
- User guide: `docs/USER_GUIDE.md`
- Developer guide: `docs/DEVELOPER_GUIDE.md`

## Quick Start (Local)
1. Build the binary:
   `go build -o breeoche ./...`
2. Start a single node:
   `./breeoche server --id n1 --host 127.0.0.1 --port 15213`
3. Use the CLI:
   `./breeoche set --addr 127.0.0.1:15213 hello world`
