# Protovalidate Migration Design

Date: 2026-06-23

## Goal

Replace `envoyproxy/protoc-gen-validate` with `bufbuild/protovalidate` for all API protobuf validation. Keep current API behavior: invalid gRPC requests return `errorx.Code_INVALID_ARGUMENT`, generated protobuf APIs remain source-compatible except generated `Validate()` / `ValidateAll()` methods disappear.

## Current State

- Proto files import `envoyproxy-validate/validate.proto` and use `(validate.rules)` annotations.
- `api/Makefile` runs `--validate_out=paths=source_relative,lang=go`.
- Generated `*.pb.validate.go` files provide `Validate()` and `ValidateAll()` methods.
- gRPC validation interceptors in `pkg/grpcx` and `api/internal/grpcx` check a local interface with `Validate()` / `ValidateAll()`.
- `api/go.mod` depends on `github.com/envoyproxy/protoc-gen-validate`.
- `thirdparty/envoyproxy-validate/validate.proto` is vendored.

## Chosen Approach

Use protovalidate as a runtime validator.

Generated Go code only needs standard protobuf generation. Protovalidate reads validation options from protobuf descriptors at runtime, so no `--validate_out` plugin is needed.

## Proto Changes

For each API proto file:

- Replace:
  ```proto
  import "envoyproxy-validate/validate.proto";
  ```
  with:
  ```proto
  import "buf/validate/validate.proto";
  ```
- Replace PGV annotations with protovalidate annotations:
  - `(validate.rules).string` -> `(buf.validate.field).string`
  - `(validate.rules).bytes` -> `(buf.validate.field).bytes`
  - `(validate.rules).repeated` -> `(buf.validate.field).repeated`
  - `(validate.rules).int32` / `int64` -> `(buf.validate.field).int32` / `int64`
  - `(validate.rules).enum` -> `(buf.validate.field).enum`

Rules to preserve:

- string length, exact length, regex pattern, contains, prefix, email, IP
- bytes min/max length
- repeated unique, min/max items, item string patterns, ignore-empty behavior
- integer gte/lte bounds
- enum defined-only behavior

## Thirdparty Dependencies

- Add upstream `buf/validate/validate.proto` under `thirdparty/buf/validate/validate.proto`.
- Keep vendored proto content unmodified so its `go_package` resolves to Buf's runtime package.
- Remove `thirdparty/envoyproxy-validate/validate.proto` after all imports are replaced.

## Generation Changes

Remove all `--validate_out` usage from `api/Makefile`.

Keep existing `protoc` commands and include paths. `-I ../../thirdparty` remains required so `buf/validate/validate.proto` resolves.

Expected output changes:

- Generated `*.pb.go` files retain validation options in descriptors.
- Generated `*.pb.validate.go` files are deleted.
- No `protoc-gen-validate` binary is required.

## Runtime Validation Changes

Replace `Validate()` interface checks with protovalidate runtime calls.

Target files:

- `pkg/grpcx/interceptors.go`
- `api/internal/grpcx/interceptors.go`

New behavior:

- If request implements `proto.Message`, run `protovalidate.Validate(req)`.
- On validation error, wrap with `errorx.New(errorx.Code_INVALID_ARGUMENT, err.Error())`.
- If request is not a protobuf message, skip validation and continue.
- Server and client validation behavior remain aligned with current contract.

## Dependency Changes

In `api/go.mod`:

- Remove `github.com/envoyproxy/protoc-gen-validate`.
- Add `buf.build/go/protovalidate`.

Main module dependency updates may be needed because `pkg/grpcx` imports protovalidate directly.

## Testing Strategy

Use TDD for behavior changes.

1. Add failing tests for interceptor validation without generated `Validate()` methods:
   - invalid protobuf request returns `Code_INVALID_ARGUMENT`
   - valid protobuf request reaches handler/invoker
   - non-protobuf request bypasses validation
2. Add or update API validation tests for representative rules:
   - invalid app/env/key pattern
   - repeated uniqueness and min/max items
   - invalid IP
   - integer bounds for range/compact requests
3. Regenerate protobuf files with `make -C api`.
4. Run targeted tests:
   - `go test ./pkg/grpcx/...`
   - `go test ./api/...` from API module or equivalent package tests
5. Run broader validation if time permits:
   - `make test`
   - `go vet ./...`

## Compatibility Notes

- Code calling generated `Validate()` or `ValidateAll()` directly must be updated. Current known production validation entry points are gRPC interceptors.
- Error text will change because protovalidate formats violations differently from PGV. Error code contract remains `INVALID_ARGUMENT`.
- No API wire format changes expected.
- Generated code may import Buf validation package via descriptor extension blank import.

## Out of Scope

- Switching generation from `protoc` to `buf generate`.
- Changing HTTP validation behavior beyond shared interceptor effects.
- Reworking domain validation rules unrelated to PGV/protovalidate migration.
- Refactoring existing errorx migration work already present in workspace.

## Rollout Plan

1. Write failing tests around validation interceptors and representative rules.
2. Vendor Buf validation proto and update proto imports/rules.
3. Remove PGV generator flags and generated validation files.
4. Update runtime interceptors to use protovalidate.
5. Update Go module dependencies.
6. Regenerate protobuf output.
7. Run tests and fix only migration-related regressions.

## Self-Review

- No placeholders remain.
- Scope fits one implementation plan.
- Error contract is explicit.
- Generation behavior states no validation plugin flag is needed.
- Known compatibility risk, changed error text, is documented.
