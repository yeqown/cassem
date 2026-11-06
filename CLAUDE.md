# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Cassem is a distributed configuration management system written in Go 1.26. It uses etcd's Raft consensus for the storage layer. The system has three core binaries:

- **cassemkv** — Raft-based distributed KV storage engine (data plane), serves gRPC
- **cassemadm** — Admin/management HTTP server (control plane), serves REST API with Gin
- **cassemagent** — Stateless caching agent layer (edge proxy), serves gRPC to clients and receives dispatched changes from adm

## Build & Run Commands

The Makefile exposes `build-image`, cluster lifecycle, and quality-check targets. `build-image` writes Linux binaries to ignored `bin/`, builds local images, and `cluster.start` then runs Compose from `.deploy/example-configs/compose.cluster.yaml`.

```bash
make cluster.start           # Build images and start the local Compose cluster from .deploy/example-configs/compose.cluster.yaml
make cluster.stop            # Stop the local Compose cluster
make cluster.restart         # Restart the local Compose cluster
make cluster.status          # Show local Compose cluster status
make cluster.logs            # Show local Compose cluster logs
make cluster.clean           # Stop cluster, remove volumes, and remove generated binaries

make test                    # Run all Go tests
make vet                     # Run go vet
make lint                    # Run golangci-lint

go test ./internal/pkg/conf/...    # Run tests for a specific package
```

Single-binary local builds can still be run directly with `go build ./cmd/<binary>` when needed.

## Protobuf Generation

The `thirdparty/cassem/Makefile` centralizes proto generation:

```bash
make -C thirdparty/cassem gen-proto      # Generate all current Go protobuf files
make -C thirdparty/cassem gen-internal   # Generate internal Go protocol bindings
make -C thirdparty/cassem gen-sdk-go     # Generate client Go protocol bindings
```

The top-level Makefile intentionally does not expose proto generation targets.

Generation currently uses `github.com/golang/protobuf/protoc-gen-go` v1.5.4 (including its legacy gRPC plugin for KV) and `protoc-gen-go-grpc` v1.3.0 for Agent/Delivery. The `thirdparty/` directory contains canonical Cassem schemas plus vendored buf.validate and Google protobuf includes. `thirdparty/cassem/client` generates into `sdk/go/protogen`; all schemas in `thirdparty/cassem/internal` generate into the single `internal/protogen` package. Generated packages contain no handwritten business logic. Client schemas retain validation annotations; SDK bindings reference annotation definitions but do not run Protovalidate.

## Architecture

### Key Namespace Hierarchy (`internal/concept/key_generator.go`)

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

### Aggregate Interfaces (`internal/concept/coordinator.go`)

The system uses **composition-over-inheritance** to define access profiles:

- `AdmAggregate` = `KVReadOnly` + `KVWriteOnly` + `InstanceHybrid` + `AgentHybrid` + `RBAC`
- `AgentAggregate` = `KVReadOnly` + `InstanceHybrid` + `AgentHybrid` (no write or ACL)

Concrete implementations (`internal/coord/coordinator_adm_agg.go`, `internal/coord/coordinator_agent_agg.go`) compose shared interface implementations over a common gRPC connection to cassemkv.

### Data Flow

1. **Write path**: cassemadm → gRPC → cassemkv leader → one `MutateCommand` Raft proposal → commit/apply → BoltDB + request ack + committed change event
2. **Read path**: client → cassemagent (LRU-2 cache, 10s stale window) → cassemkv (reads can hit any node)
3. **Change push**: committed apply event → agent `Dispatch` (batched: up to 100 elements or 1s window) → agent → client `Watch` stream

### Leader-Aware gRPC Routing

The cassemkv client (`internal/kvclient/client.go`) uses a custom gRPC resolver (`cassemkv://` scheme). Only the Raft leader marks its gRPC health services as `SERVING`. Write-mode clients (`Mode_X`) use health-checking LB to route to leader; read-mode clients (`Mode_R`) use round-robin to any node.

### Element Versioning

Elements enforce a **publish-before-update** workflow: create v1 → review → publish → update creates v2 (fails if v1 is still unpublished). This is enforced in `kvWriteOnly` (`internal/coord/coordinator_kv_w.go`), not in the storage layer.

### Network Serving

`cassemkv` serves gRPC only on its listen address. `cassemagent` still uses `httpx.Gateway` to serve HTTP and gRPC on the same TCP port via h2c (cleartext HTTP/2). Routing is by `Content-Type: application/grpc` and protocol version.

### Change Notification in Raft

Writes now produce one `MutateCommand` Raft log entry. FSM apply is semantic center: it mutates BoltDB, resolves pending request acknowledgment, and emits committed change events from same apply step. The watcher system (`internal/app/kv/app_watcher.go`) is still topic-scoped pub/sub, but core path no longer silently drops events; slow observers are explicitly unsubscribed after a bounded delay. Watch delivery is online-only and does not replay missed history after disconnect.

### Agent Cache

The agent uses a three-level `sync.Map` hierarchy (`appPool → envPool → elemPool`) with an LRU-2 eviction algorithm (`internal/app/agent/lru/`). Items must be accessed twice before promotion to the main cache. Items older than 10 seconds (`dirtyTime`) trigger a background refresh.

## Key Packages

| Package | Role |
|---|---|
| `internal/concept/error.go` | Custom error type with bidirectional gRPC status conversion. Sentinel errors: `Err_NOT_FOUND`, `Err_ALREADY_EXISTS` |
| `internal/pkg/grpcx/` | gRPC interceptor chains (Recovery, Logger, Errorx, Validation) for both server and client |
| `internal/pkg/httpx/` | HTTP+gRPC gateway (h2c), common JSON response format, recovery middleware |
| `internal/app/kv/app_watcher.go` | Channel-based pub/sub with topic-scoped distribution |
| `internal/pkg/runtime/` | `GoFunc` runs goroutines with auto-restart on panic; `IsDebug()` reads `DEBUG` env var |
| `internal/pkg/hash/` | Scrypt password hashing, MD5 fingerprinting, SHA-256 checksums |

## Go Module and SDK Boundaries

The root module owns all `internal/` code. Internal protocol bindings live in `internal/protogen`; handwritten domain interfaces live in `internal/concept` and the shared KV client in `internal/kvclient`. `sdk/go` is the independent public Go module (`github.com/yeqown/cassem/sdk/go`); the workspace includes only the root and this SDK. Go is currently the only SDK implementation. SDK implementations must never import server internals.

Client message definitions live in `thirdparty/cassem/client/types.proto` and Agent RPCs in `client/agent.proto`. The internal delivery service is defined separately in `internal/delivery.proto`. Client schemas may only import client contracts and third-party dependencies; internal schemas may import client contracts. Server code imports `sdk/go/protogen` directly; `internal/concept` reuses client message types through aliases. Each protobuf descriptor is registered once per process.

## Testing

Tests use `testify/assert`. Unit tests exist in `internal/pkg/`, `internal/concept/`, `internal/app/agent/`, `internal/app/kv/storage/`, and `internal/coord/`. Some tests use `storage.EmptyImpl` (a no-op KV implementation) as a test double.

## Configuration

Config files are TOML, loaded via `internal/pkg/conf.Load()`. cassemkv cluster endpoints and Raft bind/cluster addresses are passed as CLI flags (not TOML). Set `DEBUG=1` to enable debug logging.
