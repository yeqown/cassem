# Cassem protobuf contracts

This directory is the canonical source of Cassem protobuf definitions. Existing protobuf package names, RPC paths and field numbers are retained.

| Source | Boundary | Generated Go destination |
| --- | --- | --- |
| `client/agent.proto` | Client configuration reads, registration, renewal and Watch | `sdk/go/protogen` |
| `client/types.proto` | Client Element, metadata and registration/watch messages | `sdk/go/protogen` |
| `internal/delivery.proto` | Internal admin-to-agent Dispatch | `internal/protogen` |
| `internal/models.proto`, `internal/acl.proto` | Internal management, topology and ACL data | `internal/protogen` |
| `internal/kv.proto`, `internal/raft.proto` | Internal KV and Raft contracts | `internal/protogen` |

Client schemas may import other client schemas and vendored dependencies, never internal schemas. Internal schemas may reuse client messages. Imports use paths rooted at `thirdparty/`, such as `cassem/client/types.proto`, so external generators need one include path.

The server imports client bindings directly from `sdk/go/protogen` and reuses client message types through aliases in `internal/concept`; it does not generate a second copy of the same descriptors. All internal generated bindings share `internal/protogen`. Both generated packages are free of business logic. The Go SDK is an independent module with no dependency on server internals.

Run from the repository root:

```sh
make -C thirdparty/cassem gen-proto       # All current Go bindings (default)
make -C thirdparty/cassem gen-internal    # Internal Go bindings only
make -C thirdparty/cassem gen-sdk-go      # Client Go bindings only
```

Go generation currently uses `protoc` 25.0, `github.com/golang/protobuf/protoc-gen-go` v1.5.4 and `protoc-gen-go-grpc` v1.3.0. KV generation retains the legacy `plugins=grpc` output because its server implementations use those interfaces. Install these generators with `go install <package>@<version>` and place their binaries on `PATH`.

Generated protocol packages must be refreshed from these files, never edited independently. Vendored `buf/validate` and `google/protobuf` dependencies remain in the parent `thirdparty` directory.

For another language SDK, generate only `cassem/client/agent.proto` and `cassem/client/types.proto` with `thirdparty/` as the include root. Supply the language bindings for their annotation dependencies as needed by that generator. The schema retains `buf.validate` annotations, so generated messages still reference the annotation definitions. SDKs need protobuf/gRPC bindings and annotation definitions; they do not run Protovalidate. The server continues to enforce those rules.
