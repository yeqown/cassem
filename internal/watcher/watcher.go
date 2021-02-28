package watcher

type Changes struct {
	CheckSum string
	Topic    string
	Data     []byte
}

// IWatcher
type IWatcher interface {
	// tell watcher there is a client want to get notified while these topic changed.
	Subscribe(obs ...IObserver)

	// how to unsubscribe safely?
	Unsubscribe(obs IObserver)

	// any changes would be send to Watcher.ChangeNotifyCh.
	ChangeNotify(notify Changes)
}

// IObserver describes all actions those the IWatcher's client should have.
type IObserver interface {
	// Identity must keep unique in cassemd server.
	Identity() string

	// Topics describes all topic(containerKey) those IObserver cares about.
	Topics() []string

	// ChangeNotifyCh will receive all changes about topics.
	ChangeNotifyCh() chan<- Changes
}
