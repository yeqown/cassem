# cassem
config assembler from key-value pairs' container which include basic datatypes, such as int, string, float, bool, list, dict

<img src="./assets/intro.svg" width="100%"/>

## Features

- [x] HTTP Restful API.
- [x] Export container (config container) into different file format (JSON / TOML).
- [ ] Manage `CTL` / `UI` support.
- [ ] Master / Slave architecture support based raft (only write on master).
- [ ] RESTful API permission control.
- [ ] `Changes` watching and notifying.
- [ ] Distributed `Cache` middleware to speed up server query performance. 

## Benchmark

benchmark tested core RESTful API and try to optimize them, each benchmark test displays the final QPS result. 

[README](./benchmark/README.md)
