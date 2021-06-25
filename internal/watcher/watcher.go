package watcher

import "github.com/yeqown/cassem/pkg/datatypes"

type Changes struct {
	Key       string
	Namespace string
	Format    datatypes.ContainerFormat
	CheckSum  string
	Data      []byte
}

func (c Changes) Topic() string {
	return c.Namespace + "#" + c.Key + "#" + c.Format.String()
}

// IWatcher provides Subscribe(obs ...IObserver) and Unsubscribe(obs IObserver) for observers,
// and ChangeNotify(notify Changes) for producer.
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
	// Identity must keep unique in cassemagent server.
	Identity() string

	// Topics describes all topic="NAMESPACE#CONTAINER_KEY#FORMAT" those IObserver cares about.
	Topics() []string

	// ChangeNotifyCh will receive all changes about topics.
	ChangeNotifyCh() chan<- Changes

	// release observer resources, only be called IWatcher, importantly close channel.
	Close()
}
