# Agent configuration access

Cassemagent serves HTTP fetch and the public gRPC Agent service on the same
configured `server.addr` (the example deployment uses port `20219`). Both fetch
transports and gRPC watch require a user-bound access token. The HTTP contract is
defined in [agent-access.openapi.yaml](agent-access.openapi.yaml).

## Issue and revoke tokens

Use a cassemadm session with cluster ACL write permission to issue a token for an
enabled user. The user's current roles determine which app/environment
configuration the token can read. A dedicated user with a `visitor` binding to
`checkout/production`, for example, can read that namespace; `checkout/*` grants
all environments in that app. A token grants no management API access.

```sh
# CASSEM_SESSION is the session returned by POST /api/account/login.
curl -sS -X POST \
  'http://127.0.0.1:20218/api/account/users/reader@example.com/tokens' \
  -H "X-CASSEM-SESSION: $CASSEM_SESSION" \
  -H 'Content-Type: application/json' \
  -d '{"name":"checkout-worker","expiresInSeconds":86400}'
```

The response's `data` contains `token`, `id`, `account`, `name`, `createdAt`, and
`expiresAt`. Save the bearer value securely when it is returned: subsequent
requests cannot retrieve it. Names contain 1–100 characters and must include
non-whitespace; stored names are trimmed.
`expiresInSeconds` defaults to 2,592,000 (30 days); when supplied, it must be an
integer from 1 to 31,536,000 (365 days). Times are Unix seconds.

List token metadata with cluster ACL read permission:

```sh
curl -sS \
  'http://127.0.0.1:20218/api/account/users/reader@example.com/tokens?limit=100' \
  -H "X-CASSEM-SESSION: $CASSEM_SESSION"
```

The result has `tokens`, `hasMore`, and `nextSeek`. Pass `nextSeek` as the next
request's `seek`; `limit` is 1–100, default 100. Expired credentials remain listed
with their expiry metadata until revoked. Lists never contain bearer values.

Revoke a token with cluster ACL write permission and its 64-character `id`:

```sh
curl -sS -X DELETE \
  "http://127.0.0.1:20218/api/account/users/reader@example.com/tokens/$TOKEN_ID" \
  -H "X-CASSEM-SESSION: $CASSEM_SESSION"
```

Revocation is idempotent and scoped to the account in the URL. A successful revoke
removes the stored credential; all agents reject it on subsequent authorization.
Management also checks the operator's current stored permissions. Only a global
superadmin can issue or revoke tokens bound to a superadmin account.
The management endpoints retain cassemadm's existing error envelope and status
conventions, including HTTP 400 with `errcode: 7` for an ACL denial.

## HTTP fetch

Send the token using the standard `Authorization` header. Query parameters and
admin session headers are not accepted as fetch credentials.

```sh
curl -sS \
  'http://127.0.0.1:20219/api/apps/checkout/envs/production/elements?key=config&key=limits' \
  -H "Authorization: Bearer $CASSEM_TOKEN"
```

Request 1–100 distinct keys with repeated `key` parameters. App/environment
identifiers contain 3–30 characters; keys are nonempty. All identifiers start
with an ASCII letter or digit and contain only letters, digits, `_`, and `-`.
Validation and cache behavior match gRPC `GetElement`; missing or unpublished
keys are omitted according to the existing fetch semantics. Results are not
guaranteed to follow request order.

```json
{
  "errcode": 0,
  "errmsg": "success",
  "data": {
    "elems": [
      {
        "metadata": {"app":"checkout","env":"production","key":"config"},
        "raw": "aGVsbG8=",
        "version": 1,
        "published": true
      }
    ]
  }
}
```

`raw` is base64-encoded bytes (`aGVsbG8=` decodes to `hello`). Response fields use
the generated Go JSON representation. Fetch and token management responses use
`Cache-Control: no-store`. HTTP watch is not provided.

| Condition | Agent HTTP status | `errcode` / gRPC status |
| --- | --- | --- |
| Missing, malformed, unknown, revoked, or expired token; disabled/deleted user | 401 | 16 / `Unauthenticated` |
| Valid token without element READ permission for the requested app/env | 403 | 7 / `PermissionDenied` |
| Invalid request identifiers or keys | 400 | 3 / `InvalidArgument` |
| Unavailable authorization storage | 503 | 14 / `Unavailable` |

## gRPC and the Go SDK

Low-level gRPC callers send a single `authorization: Bearer <token>` metadata
entry. Protobuf message shapes and RPC names are unchanged. Go SDK callers use
`WithToken`:

```go
client, err := cassem.New("127.0.0.1:20219",
    cassem.WithClientID("checkout-01"),
    cassem.WithClientIP("127.0.0.1"),
    cassem.WithToken(os.Getenv("CASSEM_TOKEN")),
)
```

See the [Go SDK guide](../sdk/go/README.md) for fetch/watch, lifecycle, TLS, and
error handling. Existing callers must supply a token for `GetElement` and `Watch`.
Registration and the internal Delivery RPC retain their existing contracts.

## Authorization and storage behavior

The agent checks token existence and expiry, the bound user's enabled state, and
the user's current Casbin element READ permission for every requested namespace.
Authorization happens before reading cached configuration. A warm configuration
cache cannot bypass a revoked token or a changed role binding; authorization
storage must remain available even for cached values.
Each authorization bounds storage calls to three seconds, respecting earlier
caller deadlines and cancellation.

Watch checks authorization before subscribing and before every outgoing update.
Revocation, expiry, disabling the user, or withdrawing a role stops the stream
when the next delivery is attempted. An idle stream may remain open until then.
Each update must also belong to the stream's requested app, environment, and keys.

Tokens contain 256 bits of cryptographic randomness and are opaque to callers.
Storage under `cassem/acl/tokens/` contains a SHA-256 digest identifier and public
metadata, never the bearer value. Permissions are read from current ACL policies,
not copied into the token. Password resets leave access tokens valid; revoke
credentials explicitly or disable their user to invalidate them.
