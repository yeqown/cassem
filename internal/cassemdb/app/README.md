## API 

### Watching and Notifying 

Notifier actually is a watch mechanism, normally implement it in following ways:
1. HTTP polling
2. HTTP long polling
3. Long link
4. Multiplexing over one TCP connection.

### TODO

- [ ] grpc error interceptors
- [ ] app error definitions
- [ ] TTL reimplementation or make up these codes.

### References

* http://liangjf.top/2019/12/31/110.etcd-watch%E6%9C%BA%E5%88%B6%E5%88%86%E6%9E%90/
* https://github.com/etcd-io/etcd