## cassem/client

[![Go Report Card](https://goreportcard.com/badge/github.com/yeqown/cassem/client)](https://goreportcard.com/report/github.com/yeqown/cassem/client) [![go.de
│ v reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/yeqown/cassem/client)

### Install

```sh
go get github.com/yeqown/cassm/client
```

### Get Started

Create a client using `client.New()`:

```go
c, err := client.New(client.Config{
	Endpoints:   []string{"localhost:2379", "localhost:22379", "localhost:32379"},
	DialTimeout: 5 * time.Second,
})
if err != nil {
	// handle error!
}
defer c.Close()
```