# Protovalidate Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `envoyproxy/protoc-gen-validate` with `bufbuild/protovalidate` across Cassem API validation while preserving `INVALID_ARGUMENT` error contracts.

**Architecture:** Protobuf files use `buf/validate/validate.proto` annotations. Standard `protoc-gen-go` preserves validation options in descriptors; runtime validation uses `protovalidate.Validate(proto.Message)` in gRPC interceptors and tests. Generated `*.pb.validate.go` files and `--validate_out` disappear.

**Tech Stack:** Go 1.26, gRPC, protobuf, `buf.build/go/protovalidate`, `buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go`, testify.

## Global Constraints

- Keep public wire formats unchanged.
- Keep invalid request transport contract as `errorx.Code_INVALID_ARGUMENT` / gRPC `codes.InvalidArgument`.
- Do not introduce `buf generate`; keep `protoc` and `api/Makefile`.
- Keep `-I ../../thirdparty` include path for vendored validation proto.
- Use TDD: write failing tests before production code changes.
- Run gofmt on all modified Go files.
- Avoid staging unrelated existing workspace changes.
- Commit each task separately if working tree state allows clean task-only staging.

---

## File Structure

- Modify: `api/agent/cassemagent.api.proto` — replace PGV imports/rules with protovalidate rules for agent request validation.
- Modify: `api/concept/types.proto` — replace PGV imports/rules for concept identifier validation.
- Modify: `api/concept/acl.proto` — replace PGV imports/rules for user ACL validation.
- Modify: `api/kv/cassemdb.api.proto` — replace PGV imports/rules for KV API validation.
- Modify: `api/kv/cassemdb.raft.proto` — replace PGV imports/rules for raft cluster API validation.
- Create: `thirdparty/buf/validate/validate.proto` — vendored upstream protovalidate annotations.
- Delete: `thirdparty/envoyproxy-validate/validate.proto` — old PGV annotations.
- Modify: `api/Makefile` — remove `--validate_out` from all generation targets.
- Delete: `api/agent/cassemagent.api.pb.validate.go` — PGV generated code.
- Delete: `api/concept/acl.pb.validate.go` — PGV generated code.
- Delete: `api/concept/types.pb.validate.go` — PGV generated code.
- Delete: `api/kv/cassemdb.api.pb.validate.go` — PGV generated code.
- Delete: `api/kv/cassemdb.raft.pb.validate.go` — PGV generated code.
- Modify: `pkg/grpcx/interceptors.go` — validate protobuf messages with protovalidate on client and server paths.
- Modify: `api/internal/grpc.go` — validate API module client protobuf messages with protovalidate.
- Modify: `pkg/grpcx/interceptors_test.go` — replace mock `Validate()` tests with real protovalidate-backed protobuf request tests.
- Create: `api/kv/protovalidate_rule_test.go` — representative KV protovalidate rule coverage.
- Modify: `api/agent/identifier_validation_test.go` — use `protovalidate.Validate` instead of generated `Validate()`.
- Modify: `api/concept/identifier_validation_test.go` — use `protovalidate.Validate` instead of generated `Validate()`.
- Modify: `internal/coord/coordinator_kv_r_test.go` — replace direct `req.Validate()` in test fake.
- Modify: `internal/coord/coordinator_acl_test.go` — replace direct `req.Validate()` in test fake.
- Modify: `api/go.mod`, `api/go.sum`, `go.mod`, `go.sum`, `go.work.sum` — module dependency updates after `go get` / `go mod tidy`.

---

### Task 1: Add protovalidate rule tests that fail on current PGV annotations

**Files:**
- Modify: `api/agent/identifier_validation_test.go`
- Modify: `api/concept/identifier_validation_test.go`
- Create: `api/kv/protovalidate_rule_test.go`
- Modify: `api/go.mod`
- Modify: `api/go.sum`

**Interfaces:**
- Consumes: generated protobuf message types already present in `api/agent`, `api/concept`, and `api/kv`.
- Produces: tests that assert `protovalidate.Validate(proto.Message) error` enforces validation descriptors.

- [ ] **Step 1: Add protovalidate test dependency**

Run from repo root:

```bash
cd api && go get buf.build/go/protovalidate@v1.2.0
```

Expected: `api/go.mod` gains `buf.build/go/protovalidate v1.2.0` and `api/go.sum` changes.

- [ ] **Step 2: Replace agent identifier test with protovalidate API**

Edit `api/agent/identifier_validation_test.go` to this full content:

```go
package agent

import (
	"testing"

	"buf.build/go/protovalidate"
	"github.com/stretchr/testify/require"
)

func TestGetElementReqIdentifierValidation(t *testing.T) {
	valid := &GetElementReq{App: "demo_app", Env: "prod-1", Keys: []string{"db_url", "feature_flag"}}
	require.NoError(t, protovalidate.Validate(valid))

	for _, req := range []*GetElementReq{
		{App: "demo.app", Env: "prod", Keys: []string{"db_url"}},
		{App: "demo", Env: "prod env", Keys: []string{"db_url"}},
		{App: "demo", Env: "prod", Keys: []string{"db.url"}},
		{App: "demo", Env: "prod", Keys: []string{"feature,flag"}},
	} {
		require.Error(t, protovalidate.Validate(req))
	}
}
```

- [ ] **Step 3: Replace concept identifier test with protovalidate API**

Edit `api/concept/identifier_validation_test.go` to this full content:

```go
package concept

import (
	"testing"

	"buf.build/go/protovalidate"
	"github.com/stretchr/testify/require"
)

func TestInstanceWatchingIdentifierValidation(t *testing.T) {
	valid := &Instance_Watching{App: "demo_app", Env: "prod-1", WatchKeys: []string{"db_url", "feature_flag"}}
	require.NoError(t, protovalidate.Validate(valid))

	for _, watching := range []*Instance_Watching{
		{App: "demo.app", Env: "prod", WatchKeys: []string{"db_url"}},
		{App: "demo", Env: "prod env", WatchKeys: []string{"db_url"}},
		{App: "demo", Env: "prod", WatchKeys: []string{"db.url"}},
		{App: "demo", Env: "prod", WatchKeys: []string{"feature,flag"}},
	} {
		require.Error(t, protovalidate.Validate(watching))
	}
}
```

- [ ] **Step 4: Add representative KV protovalidate tests**

Create `api/kv/protovalidate_rule_test.go` with this full content:

```go
package kv

import (
	"strings"
	"testing"

	"buf.build/go/protovalidate"
	"github.com/stretchr/testify/require"
)

func TestProtovalidateKVRules(t *testing.T) {
	tests := []struct {
		name string
		msg  any
	}{
		{
			name: "get kv key must contain slash",
			msg:  &GetKVReq{Key: "noslash"},
		},
		{
			name: "get kvs keys must be unique",
			msg:  &GetKVsReq{Keys: []string{"cassem/a", "cassem/a"}},
		},
		{
			name: "set value must not exceed 256KiB",
			msg:  &SetKVReq{Key: "cassem/a", Val: []byte(strings.Repeat("x", 262145))},
		},
		{
			name: "range limit must be positive",
			msg:  &RangeReq{Key: "cassem/a", Limit: 0},
		},
		{
			name: "compact keep version count must be positive",
			msg:  &CompactElementHistoryReq{ElementKey: "cassem/elements/app/env/key", KeepVersionCount: 0, PageSize: 10},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Error(t, protovalidate.Validate(tt.msg))
		})
	}
}

func TestProtovalidateKVRulesAcceptValidMessages(t *testing.T) {
	tests := []struct {
		name string
		msg  any
	}{
		{
			name: "valid get kv",
			msg:  &GetKVReq{Key: "cassem/a"},
		},
		{
			name: "valid get kvs",
			msg:  &GetKVsReq{Keys: []string{"cassem/a", "cassem/b"}},
		},
		{
			name: "valid set kv",
			msg:  &SetKVReq{Key: "cassem/a", Val: []byte("value")},
		},
		{
			name: "valid range",
			msg:  &RangeReq{Key: "cassem/a", Limit: 10},
		},
		{
			name: "valid compact request",
			msg: &CompactElementHistoryReq{
				ElementKey:            "cassem/elements/app/env/key",
				KeepVersionCount:      1,
				KeepVersionSeconds:    0,
				KeepOperationSeconds:  0,
				PageSize:              10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, protovalidate.Validate(tt.msg))
		})
	}
}
```

- [ ] **Step 5: Run tests to verify red**

Run from repo root:

```bash
cd api && go test ./agent ./concept ./kv
```

Expected: FAIL. At least one `require.Error(t, protovalidate.Validate(...))` fails because current descriptors contain PGV options, not `buf.validate` options.

- [ ] **Step 6: Commit failing tests if repository policy allows red commits**

If red commits are allowed:

```bash
git add api/agent/identifier_validation_test.go api/concept/identifier_validation_test.go api/kv/protovalidate_rule_test.go api/go.mod api/go.sum
git commit -m "test(api): add protovalidate rule coverage"
```

If red commits are not allowed, skip commit and keep files unstaged until Task 2 makes them pass.

---

### Task 2: Migrate proto annotations and generation to protovalidate

**Files:**
- Create: `thirdparty/buf/validate/validate.proto`
- Delete: `thirdparty/envoyproxy-validate/validate.proto`
- Modify: `api/agent/cassemagent.api.proto`
- Modify: `api/concept/types.proto`
- Modify: `api/concept/acl.proto`
- Modify: `api/kv/cassemdb.api.proto`
- Modify: `api/kv/cassemdb.raft.proto`
- Modify: `api/Makefile`
- Delete generated: `api/agent/cassemagent.api.pb.validate.go`, `api/concept/acl.pb.validate.go`, `api/concept/types.pb.validate.go`, `api/kv/cassemdb.api.pb.validate.go`, `api/kv/cassemdb.raft.pb.validate.go`
- Modify generated: matching `*.pb.go` files after `make -C api`
- Modify: `api/go.mod`, `api/go.sum`

**Interfaces:**
- Consumes: tests from Task 1.
- Produces: generated protobuf descriptors containing `buf.validate` options; no generated `Validate()` / `ValidateAll()` methods.

- [ ] **Step 1: Vendor upstream protovalidate proto**

Run from repo root:

```bash
mkdir -p thirdparty/buf/validate
curl -fsSL https://raw.githubusercontent.com/bufbuild/protovalidate/main/proto/protovalidate/buf/validate/validate.proto \
  -o thirdparty/buf/validate/validate.proto
```

Expected first lines in `thirdparty/buf/validate/validate.proto` include:

```proto
syntax = "proto2";
package buf.validate;
option go_package = "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate";
```

- [ ] **Step 2: Ensure vendored Google protos cover protovalidate imports**

Run:

```bash
ls thirdparty/google/protobuf/descriptor.proto \
   thirdparty/google/protobuf/duration.proto \
   thirdparty/google/protobuf/timestamp.proto
```

If `thirdparty/google/protobuf/field_mask.proto` is missing, add it:

```bash
curl -fsSL https://raw.githubusercontent.com/protocolbuffers/protobuf/main/src/google/protobuf/field_mask.proto \
  -o thirdparty/google/protobuf/field_mask.proto
```

Expected: all four files exist.

- [ ] **Step 3: Update `api/agent/cassemagent.api.proto` annotations**

Replace import and validation annotations with these exact field forms:

```proto
import "buf/validate/validate.proto";

message getElementReq {
  string          app = 1 [(buf.validate.field).string = {min_len: 3, max_len: 30, pattern: "^[A-Za-z0-9][A-Za-z0-9_-]*$"}];
  string          env = 2 [(buf.validate.field).string = {min_len: 3, max_len: 30, pattern: "^[A-Za-z0-9][A-Za-z0-9_-]*$"}];
  repeated string keys = 3 [(buf.validate.field).repeated = {unique: true, min_items: 1, max_items: 100, items: {string: {pattern: "^[A-Za-z0-9][A-Za-z0-9_-]*$"}}}];
}

message unregisterReq {
  string clientId = 1 [(buf.validate.field).string = {min_len: 5, max_len: 64}];
  string clientIp = 2 [(buf.validate.field).string.ip = true];
}

message registerReq {
  string clientId = 1 [(buf.validate.field).string = {min_len: 5, max_len: 64}];
  string clientIp = 2 [(buf.validate.field).string.ip = true];
  repeated concept.Instance.Watching watching = 3 [(buf.validate.field).ignore = IGNORE_IF_ZERO_VALUE];
}

message watchReq {
  repeated concept.Instance.Watching watching = 1 [(buf.validate.field).ignore = IGNORE_IF_ZERO_VALUE];
  string clientId = 4 [(buf.validate.field).string = {min_len: 5, max_len: 64}];
  string clientIp = 5 [(buf.validate.field).string.ip = true];
}
```

Keep service definitions and comments unchanged.

- [ ] **Step 4: Update `api/concept/types.proto` annotations**

Replace import and validation annotations with these exact field forms:

```proto
import "buf/validate/validate.proto";

message ElementMetadata {
  string      key                = 1 [(buf.validate.field).string = {pattern: "^[A-Za-z0-9][A-Za-z0-9_-]*$"}];
  string      app                = 2 [(buf.validate.field).string = {pattern: "^[A-Za-z0-9][A-Za-z0-9_-]*$"}];
  string      env                = 3 [(buf.validate.field).string = {pattern: "^[A-Za-z0-9][A-Za-z0-9_-]*$"}];
  int32       latestVersion      = 4;
  int32       unpublishedVersion = 5;
  int32       usingVersion       = 6;
  string      usingFingerprint   = 7;
  ContentType contentType        = 8;
}

message Instance {
  message Watching {
    string          app       = 1 [(buf.validate.field).string = {pattern: "^[A-Za-z0-9][A-Za-z0-9_-]*$"}];
    string          env       = 2 [(buf.validate.field).string = {pattern: "^[A-Za-z0-9][A-Za-z0-9_-]*$"}];
    repeated string watchKeys = 3 [(buf.validate.field).repeated = {items: {string: {pattern: "^[A-Za-z0-9][A-Za-z0-9_-]*$"}}}];
  }

  string            clientId           = 1;
  string            agentId            = 2;
  string            clientIp           = 3;
  repeated Watching watching           = 4;
  int64             lastRenewTimestamp = 5;
}
```

Also update `AppMetadata.id`:

```proto
string id = 1 [(buf.validate.field).string = {pattern: "^[A-Za-z0-9][A-Za-z0-9_-]*$"}];
```

- [ ] **Step 5: Update `api/concept/acl.proto` annotations**

Replace import and `User` fields with:

```proto
import "buf/validate/validate.proto";

message User {
    enum Status {
        NORMAL    = 0;
        // FORBIDDEN indicates the user is forbidden to access the system.
        FORBIDDEN = 1;
    }

    // account
    string account        = 1 [(buf.validate.field).string.email = true];
    string nickname       = 2 [(buf.validate.field).string = {min_len: 1, max_len: 64}];
    string hashedPassword = 3 [(buf.validate.field).string = {min_len: 6, max_len: 12}];
    string salt           = 4 [(buf.validate.field).string = {len: 8}];
    Status status         = 5 [(buf.validate.field).enum = {defined_only: true}];
}
```

- [ ] **Step 6: Update `api/kv/cassemdb.api.proto` annotations**

Replace import and annotated fields with these exact forms:

```proto
import "buf/validate/validate.proto";

message Change {
  enum Op {
    Invalid = 0;
    Set     = 1;
    Unset   = 2;
  }

  Op       op      = 1 [(buf.validate.field).enum = {defined_only: true}];
  string   key     = 2 [(buf.validate.field).string = {min_len: 2, contains: "/"}];
  Entity   last    = 3;
  Entity   current = 4;
};

message getKVReq {
  string key = 1 [(buf.validate.field).string = {min_len: 2, contains: "/"}];
};

message getKVsReq {
  repeated string keys = 1 [(buf.validate.field).repeated = {unique: true, min_items: 1, max_items: 100}];
};

message setKVReq {
  string key      = 1 [(buf.validate.field).string = {min_len: 2, contains: "/"}];
  bool   isDir    = 2;
  int32  ttl      = 3;
  bytes  val      = 4 [(buf.validate.field).bytes = {min_len: 0, max_len: 262144}];
  bool   overwrite = 5;
};

message unsetKVReq {
  string key   = 1 [(buf.validate.field).string = {min_len: 2}];
  bool   isDir = 2;
};

message watchReq {
  repeated string keys = 2 [(buf.validate.field).repeated = {unique: true, min_items: 1, max_items: 20}];
};

message ttlReq {
  string key = 1 [(buf.validate.field).string = {min_len: 2, contains: "/"}];
};

message expireReq {
  string key = 1 [(buf.validate.field).string = {min_len: 2, contains: "/"}];
};

message rangeReq {
  string key   = 1 [(buf.validate.field).string = {min_len: 2}];
  string seek  = 2 [(buf.validate.field).ignore = IGNORE_IF_ZERO_VALUE, (buf.validate.field).string = {min_len: 1}];
  int32  limit = 3 [(buf.validate.field).int32 = {gte: 1, lte: 100}];
};

message compactElementHistoryReq {
  string elementKey           = 1 [(buf.validate.field).string = {min_len: 2, prefix: "cassem/elements/"}];
  int32  keepVersionCount     = 2 [(buf.validate.field).int32 = {gte: 1, lte: 10000}];
  int64  keepVersionSeconds   = 3 [(buf.validate.field).int64 = {gte: 0}];
  int64  keepOperationSeconds = 4 [(buf.validate.field).int64 = {gte: 0}];
  int32  pageSize             = 5 [(buf.validate.field).int32 = {gte: 1, lte: 100}];
};
```

Keep unannotated messages and services unchanged.

- [ ] **Step 7: Update `api/kv/cassemdb.raft.proto` annotations**

Replace import and annotated fields with:

```proto
import "buf/validate/validate.proto";

message addNodeRequest {
  string raft_addr     = 1 [(buf.validate.field).string = {prefix: "http"}];
  string grpc_endpoint = 2 [(buf.validate.field).string = {min_len: 1}];
}

message removeNodeRequest {
  uint64 node_id = 1 [(buf.validate.field).uint64 = {gt: 0}];
}
```

- [ ] **Step 8: Remove PGV generator from `api/Makefile`**

Edit `api/Makefile` so it has this full content:

```make
.DEFAULT_GOAL := gen-proto

.PHONY: gen-proto concept.gen-proto kv.gen-proto agent.gen-proto

gen-proto: concept.gen-proto kv.gen-proto agent.gen-proto

concept.gen-proto:
	cd concept && protoc \
		-I . \
		-I ../../thirdparty \
		--go_out=paths=source_relative,plugins=grpc:. \
		types.proto acl.proto

kv.gen-proto:
	cd kv && protoc \
		-I . \
		-I ../../thirdparty \
		--go_out=paths=source_relative,plugins=grpc:. \
		cassemdb.api.proto cassemdb.raft.proto

agent.gen-proto:
	cd agent && protoc \
		-I . \
		-I ../ \
		-I ../../thirdparty \
		--go_opt=Mconcept/types.proto=github.com/yeqown/cassem/api/concept \
		--go_out=paths=source_relative:. \
		--go-grpc_out=paths=source_relative:. \
		cassemagent.api.proto
```

- [ ] **Step 9: Remove generated PGV files and old vendored proto**

Run:

```bash
rm api/agent/cassemagent.api.pb.validate.go \
   api/concept/acl.pb.validate.go \
   api/concept/types.pb.validate.go \
   api/kv/cassemdb.api.pb.validate.go \
   api/kv/cassemdb.raft.pb.validate.go \
   thirdparty/envoyproxy-validate/validate.proto
```

- [ ] **Step 10: Regenerate protobuf code**

Run:

```bash
make -C api
```

Expected: generated `*.pb.go` files import Buf validation descriptor package with blank imports. No `*.pb.validate.go` files are recreated.

- [ ] **Step 11: Tidy API module**

Run:

```bash
cd api && go mod tidy
```

Expected: `github.com/envoyproxy/protoc-gen-validate` removed from `api/go.mod`; Buf protovalidate modules present.

- [ ] **Step 12: Run Task 1 tests to verify green**

Run:

```bash
cd api && go test ./agent ./concept ./kv
```

Expected: PASS for protovalidate rule tests.

- [ ] **Step 13: Commit proto migration**

```bash
git add api/Makefile api/agent api/concept api/kv api/go.mod api/go.sum thirdparty/buf thirdparty/envoyproxy-validate thirdparty/google/protobuf/field_mask.proto
git commit -m "refactor(api): migrate proto rules to protovalidate"
```

---

### Task 3: Add failing interceptor tests for runtime protobuf validation

**Files:**
- Modify: `pkg/grpcx/interceptors_test.go`

**Interfaces:**
- Consumes: Task 2 generated API messages without `Validate()` methods.
- Produces: failing tests proving old interface-based interceptor no longer validates generated protobuf requests.

- [ ] **Step 1: Remove `mockValidator` helper**

Delete this block from `pkg/grpcx/interceptors_test.go`:

```go
// mockValidator implements the validator interface for testing
type mockValidator struct {
	validateError error
}

func (m *mockValidator) Validate() error {
	return m.validateError
}

func (m *mockValidator) ValidateAll() error {
	return m.validateError
}
```

- [ ] **Step 2: Add agent import**

Update imports in `pkg/grpcx/interceptors_test.go` to include the generated agent package:

```go
import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	errorx "github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/api/agent"
)
```

Run `gofmt` later; it will reorder imports.

- [ ] **Step 3: Replace `TestServerValidation` with protobuf message cases**

Replace the full `TestServerValidation` function with:

```go
func TestServerValidation(t *testing.T) {
	tests := []struct {
		name        string
		req         any
		expectError bool
		errMsg      string
	}{
		{
			name:        "non-protobuf request",
			req:         "regular string",
			expectError: false,
		},
		{
			name: "valid protobuf request",
			req: &agent.GetElementReq{
				App:  "demo_app",
				Env:  "prod-1",
				Keys: []string{"db_url"},
			},
			expectError: false,
		},
		{
			name: "invalid protobuf request",
			req: &agent.GetElementReq{
				App:  "demo.app",
				Env:  "prod-1",
				Keys: []string{"db_url"},
			},
			expectError: true,
			errMsg:      "demo.app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := ServerValidation()
			handlerCalled := false
			handler := func(ctx context.Context, req any) (any, error) {
				handlerCalled = true
				return "ok", nil
			}

			resp, err := interceptor(context.Background(), tt.req, &grpc.UnaryServerInfo{}, handler)

			if tt.expectError {
				assert.Error(t, err)
				assert.False(t, handlerCalled)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, "ok", resp)
			assert.True(t, handlerCalled)
		})
	}
}
```

- [ ] **Step 4: Replace `TestClientValidation` with protobuf message cases**

Replace the full `TestClientValidation` function with:

```go
func TestClientValidation(t *testing.T) {
	tests := []struct {
		name          string
		req           any
		invokerCalled bool
		expectError   bool
		errMsg        string
	}{
		{
			name:          "non-protobuf request",
			req:           "regular string",
			invokerCalled: true,
			expectError:   false,
		},
		{
			name: "valid protobuf request",
			req: &agent.GetElementReq{
				App:  "demo_app",
				Env:  "prod-1",
				Keys: []string{"db_url"},
			},
			invokerCalled: true,
			expectError:   false,
		},
		{
			name: "invalid protobuf request",
			req: &agent.GetElementReq{
				App:  "demo.app",
				Env:  "prod-1",
				Keys: []string{"db_url"},
			},
			invokerCalled: false,
			expectError:   true,
			errMsg:        "demo.app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := ClientValidation()
			invokerCalled := false
			invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
				invokerCalled = true
				return nil
			}

			err := interceptor(context.Background(), "/TestMethod", tt.req, "reply", nil, invoker)

			assert.Equal(t, tt.invokerCalled, invokerCalled, "Invoker called status")

			if tt.expectError {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			assert.NoError(t, err)
		})
	}
}
```

- [ ] **Step 5: Format test file**

Run:

```bash
gofmt -w pkg/grpcx/interceptors_test.go
```

- [ ] **Step 6: Run tests to verify red**

Run:

```bash
go test ./pkg/grpcx -run 'Test(Server|Client)Validation' -count=1
```

Expected: FAIL. Invalid protobuf request reaches handler/invoker because production code still checks removed `Validate()` interface.

---

### Task 4: Implement protovalidate runtime validation in interceptors

**Files:**
- Modify: `pkg/grpcx/interceptors.go`
- Modify: `api/internal/grpc.go`
- Modify: `go.mod`, `go.sum`
- Modify: `api/go.mod`, `api/go.sum` if tidy changes them

**Interfaces:**
- Consumes: Task 3 failing interceptor tests.
- Produces: validation behavior via helper `validateRequest(req any) error` in `pkg/grpcx` and direct protovalidate call in `api/internal`.

- [ ] **Step 1: Add root module dependency**

Run from repo root:

```bash
go get buf.build/go/protovalidate@v1.2.0
```

Expected: root `go.mod` includes `buf.build/go/protovalidate` and required indirect Buf generated module.

- [ ] **Step 2: Replace validator interface in `pkg/grpcx/interceptors.go`**

Remove the `validator` interface block and add these imports:

```go
import (
	"context"
	"fmt"
	"os"

	"buf.build/go/protovalidate"
	errorx "github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/pkg/runtime"

	"github.com/yeqown/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)
```

Add this helper where the old `validator` interface was:

```go
func validateRequest(req any) error {
	msg, ok := req.(proto.Message)
	if !ok {
		return nil
	}

	if err := protovalidate.Validate(msg); err != nil {
		return errorx.New(errorx.Code_INVALID_ARGUMENT, err.Error())
	}
	return nil
}
```

- [ ] **Step 3: Update `ServerValidation` implementation**

Replace `ServerValidation` with:

```go
// ServerValidation checks protobuf requests from clients and aborts invalid requests before handlers run.
func ServerValidation() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (resp any, err error) {

		if err = validateRequest(req); err != nil {
			return nil, err
		}

		return handler(ctx, req)
	}
}
```

- [ ] **Step 4: Update `ClientValidation` implementation**

Replace `ClientValidation` with:

```go
// ClientValidation validates protobuf requests before sending them.
func ClientValidation() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any,
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if err := validateRequest(req); err != nil {
			return err
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
```

- [ ] **Step 5: Update `api/internal/grpc.go` imports**

Replace imports with:

```go
import (
	"context"
	"fmt"
	"runtime"

	"buf.build/go/protovalidate"
	"github.com/yeqown/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	errorx "github.com/yeqown/cassem/api/concept"
)
```

Remove the `validator` interface block.

- [ ] **Step 6: Add API internal validation helper and update client interceptor**

Add helper after imports:

```go
func validateRequest(req any) error {
	msg, ok := req.(proto.Message)
	if !ok {
		return nil
	}

	if err := protovalidate.Validate(msg); err != nil {
		return errorx.New(errorx.Code_INVALID_ARGUMENT, err.Error())
	}
	return nil
}
```

Replace `ClientValidation` with:

```go
// ClientValidation validates requests before sending them.
func ClientValidation() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any,
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if err := validateRequest(req); err != nil {
			return err
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
```

- [ ] **Step 7: Format modified Go files**

Run:

```bash
gofmt -w pkg/grpcx/interceptors.go api/internal/grpc.go pkg/grpcx/interceptors_test.go
```

- [ ] **Step 8: Run interceptor tests to verify green**

Run:

```bash
go test ./pkg/grpcx -run 'Test(Server|Client)Validation' -count=1
```

Expected: PASS.

- [ ] **Step 9: Run API internal compile test**

Run:

```bash
cd api && go test ./internal -count=1
```

Expected: PASS or `[no test files]` with successful package compile.

- [ ] **Step 10: Commit interceptor runtime migration**

```bash
git add pkg/grpcx/interceptors.go pkg/grpcx/interceptors_test.go api/internal/grpc.go go.mod go.sum api/go.mod api/go.sum
git commit -m "refactor(grpcx): validate requests with protovalidate"
```

---

### Task 5: Remove direct generated `Validate()` calls from tests and fakes

**Files:**
- Modify: `internal/coord/coordinator_kv_r_test.go`
- Modify: `internal/coord/coordinator_acl_test.go`
- Verify: `api/agent/identifier_validation_test.go`
- Verify: `api/concept/identifier_validation_test.go`

**Interfaces:**
- Consumes: protovalidate runtime dependency in root module.
- Produces: no direct calls to deleted generated `Validate()` / `ValidateAll()` methods outside deleted files.

- [ ] **Step 1: Update `internal/coord/coordinator_kv_r_test.go` imports**

Add protovalidate import:

```go
import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"buf.build/go/protovalidate"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/yeqown/cassem/api/concept"
	errorx "github.com/yeqown/cassem/api/concept"
	apikv "github.com/yeqown/cassem/api/kv"
)
```

`gofmt` will reorder imports.

- [ ] **Step 2: Replace `GetKVs` fake validation**

In `internal/coord/coordinator_kv_r_test.go`, replace:

```go
if err := req.Validate(); err != nil {
	return nil, err
}
```

with:

```go
if err := protovalidate.Validate(req); err != nil {
	return nil, err
}
```

- [ ] **Step 3: Update `internal/coord/coordinator_acl_test.go` imports**

Add protovalidate import to existing import block:

```go
"buf.build/go/protovalidate"
```

- [ ] **Step 4: Replace `Range` fake validation**

In `internal/coord/coordinator_acl_test.go`, replace:

```go
if err := req.Validate(); err != nil {
	return nil, err
}
```

with:

```go
if err := protovalidate.Validate(req); err != nil {
	return nil, err
}
```

- [ ] **Step 5: Format tests**

Run:

```bash
gofmt -w internal/coord/coordinator_kv_r_test.go internal/coord/coordinator_acl_test.go
```

- [ ] **Step 6: Verify no direct generated validation calls remain**

Run:

```bash
grep -R "\.Validate()\|\.ValidateAll()" -n --include='*.go' api pkg internal tests | grep -v '\.pb\.validate\.go' || true
```

Expected: no matches. If matches exist in production or tests, replace with `protovalidate.Validate(...)` or remove obsolete helper code.

- [ ] **Step 7: Run coordinator tests touched by fake validation**

Run:

```bash
go test ./internal/coord -run 'Test.*(KV|ACL|RBAC|Range|GetKVs)' -count=1
```

Expected: PASS.

- [ ] **Step 8: Commit direct call cleanup**

```bash
git add internal/coord/coordinator_kv_r_test.go internal/coord/coordinator_acl_test.go
git commit -m "test(coord): use protovalidate in kv fakes"
```

---

### Task 6: Final dependency cleanup and full verification

**Files:**
- Modify: `api/go.mod`, `api/go.sum`, `go.mod`, `go.sum`, `go.work.sum` if tidy changes them
- Verify: all changed files

**Interfaces:**
- Consumes: all previous tasks.
- Produces: clean generated code, no PGV dependency, passing targeted validation.

- [ ] **Step 1: Tidy all workspace modules**

Run:

```bash
cd api && go mod tidy
cd .. && go mod tidy
cd examples && go mod tidy
cd ..
```

Expected: no PGV dependency remains unless pulled by unrelated existing dependency.

- [ ] **Step 2: Verify PGV references removed**

Run:

```bash
grep -R "envoyproxy\|protoc-gen-validate\|validate.rules\|--validate_out" -n api thirdparty pkg internal --exclude-dir=.git || true
```

Expected: no matches for migration targets. If matches exist only in docs/history, do not change unless they affect build.

- [ ] **Step 3: Verify generated validation files removed**

Run:

```bash
git ls-files 'api/**/*.pb.validate.go'
```

Expected: no output.

- [ ] **Step 4: Run API tests**

Run:

```bash
cd api && go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 5: Run root targeted tests**

Run:

```bash
go test ./pkg/grpcx ./internal/coord -count=1
```

Expected: PASS.

- [ ] **Step 6: Run full test suite**

Run:

```bash
make test
```

Expected: PASS. If unrelated existing failures occur, capture exact package/test names and output, then stop and report.

- [ ] **Step 7: Run vet**

Run:

```bash
make vet
```

Expected: PASS.

- [ ] **Step 8: Inspect final diff**

Run:

```bash
git diff --stat
git diff -- api/Makefile api/agent/cassemagent.api.proto api/concept/types.proto api/concept/acl.proto api/kv/cassemdb.api.proto api/kv/cassemdb.raft.proto pkg/grpcx/interceptors.go api/internal/grpc.go
```

Expected: diff only reflects protovalidate migration and no unrelated refactors.

- [ ] **Step 9: Commit final cleanup if needed**

If Task 6 produced dependency or generated cleanup changes not already committed:

```bash
git add api/go.mod api/go.sum go.mod go.sum go.work.sum
git commit -m "chore(deps): remove protoc-gen-validate"
```

---

## Self-Review

- Spec coverage: proto import/rule migration covered by Task 2; generator flag removal covered by Task 2; runtime interceptors covered by Tasks 3-4; direct `Validate()` cleanup covered by Task 5; final verification covered by Task 6.
- Placeholder scan: no `TBD`, `TODO`, or unspecified implementation steps remain.
- Type consistency: validation helper signature is consistently `func validateRequest(req any) error`; runtime API consistently uses `protovalidate.Validate(proto.Message)` or `protovalidate.Validate(req)` for generated messages.
- Scope check: plan handles one subsystem: API protobuf validation migration. Buf CLI migration, HTTP validation redesign, and unrelated errorx work remain out of scope.
