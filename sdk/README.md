# Cassem SDK

Use the Go SDK to connect your application to `cassemagent` (default `127.0.0.1:20219`). It includes installation instructions, examples and tests that run against a local gRPC fixture. SDKs are organized by language; Go is currently the supported implementation.

| Language | Package | Guide |
| --- | --- | --- |
| Go | `github.com/yeqown/cassem/sdk/go` | [Go SDK](go/README.md) |

The Go SDK implements registration, configuration reads, change subscriptions, renewal and cleanup. Configuration content is returned as bytes. Give each application instance a unique client ID and its actual IP address. Close clients on shutdown so their registrations and connections are released.

The current Agent supports one active Watch stream per client identity. Subscribe to multiple keys in that stream; use distinct clients with unique IDs for independent subscriptions. Changes are delivered while connected, with no history replay. After a stream ends, close and create a new client, and read the latest configuration before watching again. The Go SDK delivers changes through asynchronous callbacks.

The SDK contains only the client protocol and its language implementation. KV, Raft, delivery, admin domain interfaces and server utility packages are internal. The canonical schemas and regeneration commands are documented in [`thirdparty/cassem`](../thirdparty/cassem/README.md). The canonical client contracts live in `thirdparty/cassem/client`. The Go SDK ships generated messages and gRPC bindings in `sdk/go/protogen`. Consumers do not need `protoc` or a server checkout at runtime.

Package publishing is separate from this source change. Use the local installation instructions until versions are released.
