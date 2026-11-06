# Shared KV client

This package owns connection establishment, the `cassemkv` resolver, leader-aware/read routing and distributed locking used by coordinators, admin maintenance and internal tools. Its requests and service interfaces come from `internal/protogen`.

Use `DialWithModeContext` when a caller already has a deadline; `Mode_R` routes reads across nodes and `Mode_X` uses leader health checks. This is an internal service client. Application integrations use the language SDKs under `sdk/`.
