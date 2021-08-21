package watcher

type ChangeType uint8

const (
	ChangeType_KV ChangeType = iota + 1
	ChangeType_DIR
)

type IChange interface {
	Topic() string
	Type() ChangeType
}

// IWatcher provides Subscribe(obs ...IObserver) and Unsubscribe(obs IObserver) for observers,
// and ChangeNotify(notify IChange) for producer.
type IWatcher interface {
	// Subscribe tell watcher there is a client want to get notified while these topic changed.
	Subscribe(obs ...IObserver)

	// Unsubscribe how to unsubscribe safely?
	Unsubscribe(obs IObserver)

	// ChangeNotify any changes would be send to Watcher.Inbound.
	ChangeNotify(notify IChange)
}

// IObserver describes all actions those the IWatcher's client should have.
type IObserver interface {
	// Identity must keep unique in the server.
	Identity() string

	// Topics describes all topics those IObserver want to subscribe.
	Topics() []string

	// Inbound will receive all changes about topics.
	Inbound() chan<- IChange

	// Outbound will receive all changes about topics.
	Outbound() <-chan IChange

	// Close release observer resources, only be called IWatcher, importantly close channel.
	Close()
}
