# cassem

<p align="center">
  <img src="./assets/logo.svg" width="376" height="376"/>
</p>

[![Go Report Card](https://goreportcard.com/badge/github.com/yeqown/cassem)](https://goreportcard.com/report/github.com/yeqown/cassem) [![go.de
│ v reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/yeqown/cassem)

`CASSEM` is a distributed config management center, it is totally independent, so it's easy to deploy and maintain in your environment. At the
same time, it's deployed by `Go` which gives it platform-cross ability and fast-compile.

<img src="./assets/intro.svg" width="100%"/>

## Features

- [x] HTTP Restful API to manage all configs `cassemadm`.
  - [x] Stateless so that it can be easily scaled.
  - [x] Gray released.
  - [x] Multi-version management.
  - [ ] Operation log, each operation to config elements will be recorded.
  - [x] Permission control.
  - [x] Client instance management.
  - [ ] `CTL` / `UI` tool support these are constructing on `cassemadm` RESTful API.
    - [ ] [Web UI](https://github.com/yeqown/cassem-ui) is developing.
    - [ ] [CTL](#) tool to debug and manage config from terminal. 
- [x] Distributed storage component `cassemdb`, based on raft consensus algorithm.
  - [x] Master can read and write.
  - [x] Slave node can only respond to read request.
  - [x] Use `gRPC` protocol to communicate.
  - [x] `Watch` `TTL` features support.
  - [ ] `Lazy Deletion` the expired KV. There is a deleting working thread to delete KV from queue, the queue's data is from
  two part, one is `operation check`, another is `timer check`.
- [x] Stateless agent component `cassemagent` to improve client's usability.
  - [x] Cache config elements, and using `LRU-K` replacing algorithm.
  - [ ] Language independent support `HTTP` and `gRPC` protocol.
  - [x] Client SDK, easy to use.
  - [x] `Change Push` ability, gray released also built on this.

## [Documentation](./docs/README.md)

<img src="./assets/cassem-concept.svg" width="100%"/>

Explanation: 
- **_cassemdb_** provide KV storage capacity. 
- **_cassemadm_** is the manager to whole cassem application. 
- **_cassemagent_**‘s major duty is helping clients to access config easier,
   makes cassemdb work transparently to clients.  Importantly, cassemagent
   is stateless so that it could easily scale up and load balance.

<img src="./assets/cassem-architecture.svg" width="100%" align="center"/>

### - [cassemdb](./cmd/cassemdb/README.md)

The KV storage component in cassem, provide gRPC API.

<img src="./assets/cassemdb-architecture.svg" width="100%" />

### - [cassemadm](cmd/cassemadm/README.md)

The manager in cassem, provide RESTful API to communicate. It is serving for CTL and Dashboard UI.

### - [cassemagent](cmd/cassemagent/README.md)

Agent is serving for user's client, agent SDK, actually. Of course, agent is stateless server.

## [Benchmark](./benchmark)

benchmark tested core RESTful API and try to optimize them, each benchmark test displays the final QPS result. 


## References

* https://github.com/yongman/leto
* https://github.com/laohanlinux/riot
