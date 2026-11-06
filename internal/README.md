# Server internals

- `protogen/`: all generated internal KV, Raft, Delivery, management and ACL protocol bindings.
- `concept/`: handwritten coordination interfaces, event adapters, errors and key rules; client message aliases reuse SDK bindings.
- `kvclient/`: KV connection handling, leader-aware routing and distributed locking used across services.
- `pkg/`: configuration, transport, hashing, retry and runtime helpers.
- `app/`: cassemkv, cassemadm and cassemagent implementations. Entity TTL handling belongs to KV storage, proposal envelopes to Raft, and HTTP JSON adapters to Admin. The admin's delivery client lives in `app/adm`.
- `coord/`: coordination between the services and storage.

These packages belong to the root Go module. Application developers use the independent language packages in [`sdk`](../sdk/README.md).

Agent servers and internal callers import the client `sdk/go/protogen` contract directly. Generated packages never import service implementations or handwritten domain packages. Generate all bindings from the canonical schemas with `make -C thirdparty/cassem` at the repository root.
