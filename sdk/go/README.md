# Cassem Go SDK

Read published configuration and receive changes from a Cassem agent. This is a standalone Go module; applications import `github.com/yeqown/cassem/sdk/go` and do not depend on the server's internal packages.

Requires Go 1.26 or newer. After an SDK release is published, install it with:

```sh
go get github.com/yeqown/cassem/sdk/go@latest
```

SDK release tags use the `sdk/go/vX.Y.Z` prefix. To use a local checkout before publishing, add a replacement in your application's `go.mod`:

```go
require github.com/yeqown/cassem/sdk/go v0.0.0

replace github.com/yeqown/cassem/sdk/go => /absolute/path/to/cassem/sdk/go
```

## Read and watch

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"

    cassem "github.com/yeqown/cassem/sdk/go"
)

func main() {
    client, err := cassem.New("127.0.0.1:20219",
        cassem.WithClientID("checkout-01"), // Unique to this client instance.
        cassem.WithClientIP("127.0.0.1"),
        cassem.WithToken(os.Getenv("CASSEM_TOKEN")),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer func() {
        if err := client.Close(); err != nil {
            log.Printf("close: %v", err)
        }
    }()

    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
    defer stop()

    elements, err := client.GetElement(ctx, "checkout", "production", "config")
    if err != nil {
        log.Printf("read: %v", err)
        return
    }
    for _, element := range elements {
        log.Printf("initial %s: %s", element.GetMetadata().GetKey(), element.GetRaw())
    }

    err = client.Watch(ctx, "checkout", "production", func(next *cassem.Element) {
        log.Printf("changed %s, version %d: %s",
            next.GetMetadata().GetKey(), next.GetVersion(), next.GetRaw())
    }, "config")
    if err != nil {
        log.Printf("watch: %v", err)
        return
    }

    select {
    case <-ctx.Done():
    case err := <-client.Errors():
        log.Printf("background operation failed: %v", err)
    }
}
```

`Watch` returns after opening the stream and delivers future updates asynchronously. Use `GetElement` to obtain current values. The initial read and stream subscription are separate operations, so they do not provide an atomic snapshot and watch boundary; periodically reread values if that gap matters to your application. Each handler receives a separate `Element` message and runs serially; keep handlers short.

The server routes a stream by its client identity. A client therefore permits one active watch, which can include multiple keys in one app/environment. A second watch returns `ErrWatchActive`. Use separate clients with distinct IDs for additional namespaces. After a watch ends, close that client and create a new client with a distinct ID before subscribing again.

Canceling the watch context stops delivery and removes the subscription from subsequent renewal requests. An unexpected stream failure stops delivery, reports a `*BackgroundError`, and keeps the watch intent in renewals until cancellation or `Close`. The SDK does not reconnect a failed watch automatically. Registration renewal continues independently of the stream.

## Access tokens and migration

Agents require a user-bound access token for reads and watches. With an authenticated admin session, issue one through `POST /api/account/users/{account}/tokens` with a body such as `{"name":"checkout-production","expiresInSeconds":3600}`. The response's `token` is returned only once; store it securely and supply it with `WithToken`, for example through `CASSEM_TOKEN` in the application above. Omitted expiration defaults to 30 days; supported lifetimes are positive and at most 365 days. The token inherits its user's current app/environment READ permissions, so role changes take effect without reissuing it. Revoke it through `DELETE /api/account/users/{account}/tokens/{tokenId}`.

Existing constructor signatures and the `WithClientId` / `WithClientIp` aliases remain compatible. Add `WithToken` to existing clients before using fetch or watch against an agent that requires tokens. Registration, renewal, and unregister keep their existing server contracts; a successful constructor alone does not establish permission to read. An omitted or empty token leaves authorization metadata unchanged. Nonempty token values must contain visible ASCII characters without whitespace; malformed values fail construction without including the token in the error.

The SDK attaches `authorization: Bearer <token>` to every unary and streaming RPC, including registration, renewal, and unregister. It preserves caller metadata and cancellation; when configured, its token replaces any caller-supplied authorization metadata. Use TLS transport credentials to protect credentials in transit.

For reads and watches, `codes.Unauthenticated` means the token is missing, invalid, expired, or revoked, or its user is disabled or deleted. `codes.PermissionDenied` means the identity is valid but lacks current READ permission for the requested app/environment. `status.Code(err)` preserves these statuses on read errors and on `*BackgroundError` from `Errors()`. Watch authorization failures can arrive asynchronously through `Errors()` after `Watch` returns. Rotate or issue credentials as needed and recreate the client; failed watches do not reconnect automatically.

## Options and errors

| Option | Default | Purpose |
| --- | --- | --- |
| `WithClientID` | Required | Distinct instance ID |
| `WithClientIP` | Required | Advertised client IP |
| `WithToken` | Empty | User-bound access token sent as Bearer metadata on every RPC |
| `WithConnectTimeout` | 10 seconds | Maximum connection establishment time |
| `WithRequestTimeout` | 3 seconds | Maximum unary RPC duration, including register, read, renew, and unregister |
| `WithRenewInterval` | 20 seconds | Time between registration renewals |
| `WithDialOptions` | Plaintext gRPC | TLS credentials, custom dialers, interceptors, or other gRPC client options |

`WithClientId` and `WithClientIp` remain available as compatibility spellings. Timeout and renewal durations must be positive. Earlier caller deadlines take precedence over the request timeout. The request timeout does not shorten a watch's lifetime; use its context to control that.

The default transport is plaintext. Configure TLS using standard gRPC credentials:

```go
import (
    "crypto/tls"

    cassem "github.com/yeqown/cassem/sdk/go"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"
)

// Pass this option to cassem.New along with the client identity options.
option := cassem.WithDialOptions(grpc.WithTransportCredentials(credentials.NewTLS(
    &tls.Config{MinVersion: tls.VersionTLS12, ServerName: "agent.example.com"},
)))
```

Synchronous calls return their errors directly, preserving wrapped gRPC status codes. `Errors()` reports background failures as `*BackgroundError` with `Operation` (`"watch"` or `"renew"`), `App`, `Env`, and the underlying `Err`. `errors.As`, `errors.Is`, and `status.Code` work on these errors. The channel buffers 16 errors and drops later errors while full, so consumers should keep reading it. Explicit cancellation and normal shutdown do not produce background errors.

`Close` stops renewal, cancels active requests and streams, unregisters the instance, and releases its connection. It is concurrency safe and idempotent, returns any unregister error, and still releases local resources when unregister fails. It also closes the errors channel. A handler already executing may finish after `Close` returns. `Quit` is the compatibility equivalent that discards the close error. Calls to read or watch after closure return `ErrClosed`.

`Element`, `ElementMetadata`, `ContentType`, and the `ContentTypeJSON`, `ContentTypeTOML`, `ContentTypeINI`, `ContentTypePlaintext`, and `ContentTypeUnknown` constants are available from the root package. These types are aliases of generated protobuf messages. Low-level messages and gRPC interfaces live in `github.com/yeqown/cassem/sdk/go/protogen`; their canonical contracts are `thirdparty/cassem/client/{agent,types}.proto`. The generated package depends on protobuf, gRPC and validation annotation definitions; the SDK does not run protovalidate.

Use `google.golang.org/protobuf/encoding/protojson` when you need protobuf JSON (for example, enum names and base64-encoded bytes). Use `protojson.MarshalOptions{EmitDefaultValues: true}` to include default scalar fields. Standard `encoding/json` uses the generated Go struct tags instead.

## Example and verification

Run the application above against a running agent with an issued token:

```sh
CASSEM_TOKEN='<issued-token>' go run .
```

The application fetches initial values, watches changes, and unregisters on Ctrl+C. It requires the selected app, environment, and keys to have published values in Cassem, and the token's user to have READ permission for that app/environment.

Tests run a real gRPC server over an in-memory transport and need no running Cassem services:

```sh
GOWORK=off go test -race ./...
GOWORK=off go vet ./...
```
