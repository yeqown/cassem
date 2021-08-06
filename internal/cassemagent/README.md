## cassemagent

The agent of cassem's clients.


### features

- [ ] Read config 
- [ ] Gray release support
- [ ] Publish release support
- [ ] Cache config while cassemdb is unavailable or other situations those don't need to request again.

### client presudo code

```go
// initialize
c = NewClient({clientId, clientIp, app, env})
c.WatchKeys({keys})
c.Register() // register instance itself to agent
c.Heartbeat() // keep heartbeat to agent

// usage
ele = c.Get("key", reciver) // query and unmarshal from raw bytes to config structure.
if reciver.Filed_X {
	// do something
} 
```