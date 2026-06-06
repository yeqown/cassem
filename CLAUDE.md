# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Cassem is a distributed configuration management system written in Go 1.26. It uses etcd's Raft consensus for the storage layer and exposes both gRPC and HTTP APIs. The system has three core binaries:

- **cassemdb** — Raft-based distributed KV storage engine (data plane), serves gRPC
- **cassemadm** — Admin/management HTTP server (control plane), serves REST API with Gin
- **cassemagent** — Stateless caching agent layer (edge proxy), serves gRPC to clients and receives dispatched changes from adm

## Build & Run Commands

The Makefile requires Go 1.26. All binaries are built with ldflags injecting Version, BuildTime, and GitHash.

```bash
make cassemdb.build          # Build cassemdb
make cassemadm.build         # Build cassemadm
make cassemagent.build       # Build cassemagent
make build-all               # Build all three

make cassemdb.run            # Build and run 3-node cassemdb cluster (needs ./examples/cassemdb/cassemdb.toml)
make cassemadm.run           # Build and run cassemadm (needs ./examples/cassemadm/cassemadm.toml)
make cassemagent.run         # Build and run cassemagent (needs ./examples/cassemagent/cassemagent.toml)

make cassemdb.kill           # Kill running cassemdb cluster processes

go test ./...                # Run all tests
go test ./pkg/watcher/...   # Run tests for a specific package
```

## Protobuf Generation

Three separate proto modules, each with its own Makefile:

```bash
make -C internal/cassemdb/api        # cassemdb.api.proto, cassemdb.raft.proto
make -C api/concept                  # types.proto, acl.proto
make -C api/agent                    # cassemagent.api.proto
make proto-all                       # All of the above
```

All proto generation requires `protoc-gen-validate` and `protoc-gen-go` (with gRPC plugin). The `thirdparty/` directory contains vendored `.proto` includes (envoyproxy validate, google protobuf).

## Architecture

### Key Namespace Hierarchy (`api/concept/key_generator.go`)

All application data is stored as KV pairs in BoltDB with a structured key namespace:

```
cassem/elements/{app}/{env}/{key}/metadata   — element metadata (version tracking)
cassem/elements/{app}/{env}/{key}/v{N}       — element version content
cassem/apps/{appId}                          — app metadata
cassem/instances/normalized/{insId}          — client instance data
cassem/instances/reversed/{app-env-key}/{insId} — reverse index for watch lookup
cassem/agents/{agentId}                      — agent node data
cassem/acl/policy                            — casbin RBAC policies
cassem/acl/users/{account}                   — user credentials (scrypt hashed)
```

### Aggregate Interfaces (`api/concept/coordinator.go`)

The system uses **composition-over-inheritance** to define access profiles:

- `AdmAggregate` = `KVReadOnly` + `KVWriteOnly` + `InstanceHybrid` + `AgentHybrid` + `RBAC`
- `AgentAggregate` = `KVReadOnly` + `InstanceHybrid` + `AgentHybrid` (no write or ACL)

Concrete implementations (`coordinator_adm_agg.go`, `coordinator_agent_agg.go`) compose shared interface implementations over a common gRPC connection to cassemdb.

### Data Flow

1. **Write path**: cassemadm → gRPC → cassemdb leader (Raft consensus) → BoltDB
2. **Read path**: client → cassemagent (LRU-2 cache, 10s stale window) → cassemdb (reads can hit any node)
3. **Change push**: cassemadm → agent `Dispatch` (batched: up to 100 elements or 1s window) → agent → client `Watch` stream

### Leader-Aware gRPC Routing

The cassemdb client (`internal/cassemdb/api/client.go`) uses a custom gRPC resolver (`cassemdb://` scheme). Only the Raft leader marks its gRPC health services as `SERVING`. Write-mode clients (`Mode_X`) use health-checking LB to route to leader; read-mode clients (`Mode_R`) use round-robin to any node.

### Element Versioning

Elements enforce a **publish-before-update** workflow: create v1 → review → publish → update creates v2 (fails if v1 is still unpublished). This is enforced in `kvWriteOnly` (`api/concept/coordinator_kv_w.go`), not in the storage layer.

### Single-Port HTTP+gRPC

Both cassemagent and cassemdb use `httpx.Gateway` to serve HTTP and gRPC on the same TCP port via h2c (cleartext HTTP/2). Routing is by `Content-Type: application/grpc` and protocol version.

### Change Notification in Raft

Writes produce two Raft log entries: a `SetCommand` (data mutation) followed by a `ChangeCommand` (notification). `ChangeCommand` has a 10-second TTL to prevent stale notifications after recovery. The watcher system (`pkg/watcher/`) is a channel-based pub/sub with topic-scoped distribution.

### Agent Cache

The agent uses a three-level `sync.Map` hierarchy (`appPool → envPool → elemPool`) with an LRU-2 eviction algorithm (`infras/lru/`). Items must be accessed twice before promotion to the main cache. Items older than 10 seconds (`dirtyTime`) trigger a background refresh.

## Key Packages

| Package | Role |
|---|---|
| `pkg/errorx/` | Custom error type with bidirectional gRPC status conversion. Sentinel errors: `Err_NOT_FOUND`, `Err_ALREADY_EXISTS` |
| `pkg/grpcx/` | gRPC interceptor chains (Recovery, Logger, Errorx, Validation) for both server and client |
| `pkg/httpx/` | HTTP+gRPC gateway (h2c), common JSON response format, recovery middleware |
| `pkg/watcher/` | Channel-based pub/sub with topic-scoped distribution |
| `pkg/runtime/` | `GoFunc` runs goroutines with auto-restart on panic; `IsDebug()` reads `DEBUG` env var |
| `pkg/hash/` | Scrypt password hashing, MD5 fingerprinting, SHA-256 checksums |

## `api/` is a Separate Go Module

`api/` has its own `go.mod` (module `github.com/yeqown/cassem/api`) and is consumed as a dependency by the main module. It contains the protobuf-generated types, the client SDK (`api/agent/client.go`), and the shared concept interfaces.

## Testing

Tests use `testify/assert`. Unit tests exist in `pkg/`, `api/concept/`, `internal/cassemagent/domain/`, and `internal/cassemdb/infras/storage/`. Some tests use `storage.EmptyImpl` (a no-op KV implementation) as a test double.

## Configuration

Config files are TOML, loaded via `pkg/conf.Load()`. cassemdb cluster endpoints and Raft bind/cluster addresses are passed as CLI flags (not TOML). Set `DEBUG=1` to enable debug mode (extra logging, cassemdb debug HTTP endpoints).
