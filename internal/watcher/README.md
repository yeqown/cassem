## watcher

Watcher means to receive changing signals and delivery to observers 
those want to get changing notify of their subscribed topics. It looks like a `fan-out` pattern,
receive one signal then broadcast it to one more client.

```go
type ChangeNotify struct {
	CheckSum string
	Topic    string 
	Data     []byte
}

type Watcher interface {
	// tell watcher there is a client want to get notified while these topic changed. 
	Subscribe(ch chan<- ChangeNotify, topics ...string)

	// any changes would be send to Watcher.ChangeNotifyCh.
	ChangeNotifyCh() <-chan ChangeNotify
}
```
