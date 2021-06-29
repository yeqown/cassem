## core

`Core` is the daemon process to run the server, and coordinate all components so that `cassem`
can serve as what we expected.

All components are as below: 

* `raft` provide replication ability and cluster support.
* `coordinator` to provide API ability for the HTTP API server.
* `gateway` which provides `gRPC` and `HTTP`.
* `cache` to accelerate the download scene.
* `authorizer` to control permissions to access resources.