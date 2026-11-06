# Local cluster configuration

`cassemkv/`, `cassemadm/` and `cassemagent/` contain example TOML configuration. `compose.cluster.yaml` mounts these directories read-only and stores KV data in named volumes.

From the repository root, use `make cluster.start`, `make cluster.status`, `make cluster.logs` and `make cluster.stop`. The Makefile sets `CASSEM_EXAMPLE_CONFIGS_DIR` to this directory's absolute path. Direct Compose use resolves the default `.` mounts relative to `compose.cluster.yaml`.

SDK usage examples live with each language under [`sdk`](../../sdk/README.md).
