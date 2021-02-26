# cassem
[![Go Report Card](https://goreportcard.com/badge/github.com/yeqown/cassem)](https://goreportcard.com/report/github.com/yeqown/cassem) [![go.de
│ v reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/yeqown/cassem)

config assembler from key-value pairs' container which include basic datatypes, such as int, string, float, bool, list, dict

<img src="./assets/intro.svg" width="100%"/>

## Features

- [x] HTTP Restful API.
- [x] Export container (config container) into different file format (JSON / TOML).
- [ ] Manage `CTL` / `UI` support.
- [x] Master / Slave architecture support based raft (only write on master).
- [ ] RESTful API permission control.
- [ ] `Changes` watching and notifying.
- [x] Distributed `Cache` middleware to speed up the API which downloads container in specified format. 

## Documentation

<img src="./assets/cassem-architecture.svg" width="100%"/>

### - [cassemctl](./cmd/cassemctl/README.md)

### - [cassemd](./cmd/cassemd/README.md)

## Benchmark

benchmark tested core RESTful API and try to optimize them, each benchmark test displays the final QPS result. 

[README](./benchmark/README.md)
