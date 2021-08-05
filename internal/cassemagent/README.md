## cassemagent

The agent of cassem's clients.


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