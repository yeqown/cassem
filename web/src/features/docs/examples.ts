export const configurationExample = `{
  "logLevel": "info",
  "requestTimeoutMs": 3000,
  "featureEnabled": true
}`

export const httpFetchExample = `export CASSEM_TOKEN='TOKEN_FROM_CREATE_RESPONSE'

curl --fail-with-body -sS \\
  'http://127.0.0.1:20219/api/apps/order-service/envs/dev/elements?key=app-config' \\
  -H "Authorization: Bearer $CASSEM_TOKEN"`

export const httpResponseExample = `{
  "errcode": 0,
  "errmsg": "success",
  "data": {
    "elems": [
      {
        "metadata": {"app":"order-service","env":"dev","key":"app-config"},
        "raw": "eyJsb2dMZXZlbCI6ImluZm8iLCJyZXF1ZXN0VGltZW91dE1zIjozMDAwLCJmZWF0dXJlRW5hYmxlZCI6dHJ1ZX0=",
        "version": 1,
        "published": true
      }
    ]
  }
}`

export const tokenLoginExample = `curl --fail-with-body -sS -X POST \\
  'http://127.0.0.1:20218/api/account/login' \\
  -H 'Content-Type: application/json' \\
  -d '{"account":"admin@example.com","password":"YOUR_PASSWORD"}'`

export const tokenCreateExample = `export CASSEM_SESSION='SESSION_FROM_LOGIN_RESPONSE'

curl --fail-with-body -sS -X POST \\
  'http://127.0.0.1:20218/api/account/users/reader@example.com/tokens' \\
  -H "X-CASSEM-SESSION: $CASSEM_SESSION" \\
  -H 'Content-Type: application/json' \\
  -d '{"name":"order-service-dev","expiresInSeconds":86400}'`

export const tokenListExample = `curl --fail-with-body -sS \\
  'http://127.0.0.1:20218/api/account/users/reader@example.com/tokens?limit=100' \\
  -H "X-CASSEM-SESSION: $CASSEM_SESSION"`

export const tokenRevokeExample = `export CASSEM_TOKEN_ID='ID_FROM_CREATE_OR_LIST_RESPONSE'

curl --fail-with-body -sS -X DELETE \\
  "http://127.0.0.1:20218/api/account/users/reader@example.com/tokens/$CASSEM_TOKEN_ID" \\
  -H "X-CASSEM-SESSION: $CASSEM_SESSION"`

export const installExample = 'go get github.com/yeqown/cassem/sdk/go@SDK_VERSION'

export const localModuleExample = `go mod edit -replace=github.com/yeqown/cassem/sdk/go=/absolute/path/to/cassem/sdk/go
go get github.com/yeqown/cassem/sdk/go`

export const goExample = `package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"
    "os/signal"

    cassem "github.com/yeqown/cassem/sdk/go"
)

type Config struct {
    LogLevel         string \`json:"logLevel"\`
    RequestTimeoutMs int    \`json:"requestTimeoutMs"\`
    FeatureEnabled   bool   \`json:"featureEnabled"\`
}

func showConfig(element *cassem.Element) error {
    var config Config
    if err := json.Unmarshal(element.GetRaw(), &config); err != nil {
        return err
    }
    fmt.Printf("version=%d logLevel=%s timeout=%dms enabled=%t\\n",
        element.GetVersion(), config.LogLevel,
        config.RequestTimeoutMs, config.FeatureEnabled)
    return nil
}

func main() {
    if err := run(); err != nil {
        log.Fatal(err)
    }
}

func run() error {
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
    defer stop()

    token := os.Getenv("CASSEM_TOKEN")
    if token == "" {
        return fmt.Errorf("set CASSEM_TOKEN to the issued access token")
    }

    client, err := cassem.New("127.0.0.1:20219",
        cassem.WithClientID("order-service-local-01"),
        cassem.WithClientIP("127.0.0.1"),
        cassem.WithToken(token))
    if err != nil {
        return err
    }
    defer client.Close()

    elements, err := client.GetElement(ctx, "order-service", "dev", "app-config")
    if err != nil {
        return err
    }
    if len(elements) == 0 {
        return fmt.Errorf("app-config not found; check app, env, key and publishing")
    }
    if err := showConfig(elements[0]); err != nil {
        return err
    }

    err = client.Watch(ctx, "order-service", "dev", func(next *cassem.Element) {
        if err := showConfig(next); err != nil {
            log.Printf("invalid configuration: %v", err)
        }
    }, "app-config")
    if err != nil {
        return err
    }

    <-ctx.Done()
    return nil
}`
