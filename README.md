# cassem
config assembler from key-value pairs' container which include basic datatypes, such as int, string, float, bool, list, dict


## Features

- [x] HTTP Restful API.
- [ ] Export container (config container) into different file format (JSON / TOML).
- [ ] Manage UI support.
- [ ] Master / Slave architecture support based raft (only write on master).

## Benchmark

benchmark tested core RESTful API and try to optimize them, each benchmark test displays the final QPS result. 

[README](./benchmark/README.md)