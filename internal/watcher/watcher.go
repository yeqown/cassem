package watcher

type ChangeNotify struct {
	CheckSum string
	Topic    string
	Data     []byte
}

type IWatcher interface {
	// tell watcher there is a client want to get notified while these topic changed.
	Subscribe(ch chan<- ChangeNotify, topics ...string)

	// how to unsubscribe safely?
	Unsubscribe(topics ...string)

	// any changes would be send to Watcher.ChangeNotifyCh.
	ChangeNotifyCh() chan<- ChangeNotify
}
