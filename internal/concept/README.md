# Shared domain definitions

Handwritten coordination interfaces, watcher event adapters, error contracts, role constants, operator context and storage-key rules shared by the services live here. Client Element and Instance wire types are aliases of `sdk/go/protogen`, keeping one descriptor registration per process. `InstanceID` calculates the registration identity independently of generated messages.

Management and ACL wire types live in `internal/protogen`, generated from `thirdparty/cassem/internal/models.proto` and `acl.proto`. This package contains no generated bindings. Run `make -C thirdparty/cassem gen-internal` at the repository root to regenerate internal protocols.
