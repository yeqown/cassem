package app

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/yeqown/log"

	"github.com/yeqown/cassem/internal/cassemdb/infras"
	"github.com/yeqown/cassem/pkg/runtime"
	"github.com/yeqown/cassem/pkg/watcher"

	"github.com/yeqown/cassem/pkg/conf"
	"github.com/yeqown/cassem/pkg/types"
)

// app is the storage server that would guards api server running and alas controls other components.
// Especially, raft protocol which supports the architecture of cassemdb (master-slave).
//
// Notice that all writes must be operated on master node, salve nodes could execute read operations.
//
type app struct {
	config *conf.CassemdbConfig

	// watcher is abstract watcher.IWatcher layer, so trigger and observers could be splitting.
	// core.app has no need to care about how to get touch with observers, just produce signal and data.
	watcher watcher.IWatcher

	// raft is a customized raft node for cassem.
	raft infras.IMyRaft
}

func New(cfg *conf.CassemdbConfig) (*app, error) {
	d := new(app)
	if err := d.bootstrap(cfg); err != nil {
		return nil, err
	}

	return d, nil
}

func (d *app) bootstrap(cfg *conf.CassemdbConfig) (err error) {
	d.config = cfg
	d.watcher = watcher.NewChannelWatcher(100)
	d.raft, err = infras.NewMyRaft(&infras.Conf{
		Raft:        cfg.Server.Raft,
		HTTP:        cfg.Server.HTTP,
		Persistence: cfg.Persistence.BBolt,
	})
	if err != nil {
		return errors.Wrapf(err, "app.bootstrap failed to load raft")
	}
	log.Info("app: raft component loaded")

	d.startRoutines()

	return nil
}

// Heartbeat start a ticker to print log and check healthy of each component in core.app.
// The second purpose is to watch the QUIT / KILL signal to release resources of core.app, the most important work is to
// let current node leave raft cluster. If current node is leader just quit, otherwise current node should tell the leader
// about the fact there is a node is shutting down.
//
// Notice that, tryLeaveCluster maybe failed if cluster could not be maintained while there is only one node in cluster,
// it could not be removed, it will still be elected as leader. (Situation: count of cluster nodes is less than 2).
//
// NOTE: could leader call removeNode by it self? (leader could call removeNode only when cluster has more than 1 node)
func (d *app) Heartbeat() {
	tick := time.NewTicker(10 * time.Second)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	for {
		select {
		case <-tick.C:
			log.
				WithFields(log.Fields{
					"isLeader":      d.isLeader(),
					"joinedCluster": d.raft.JoinedCluster(),
				}).
				Info("app heartbeat")
		case <-quit:
			log.Info("app quit, start release resources...")
			// DONE(@yeqown): graceful shutdown components, snapshot something.
			failedCount := 3

		retryLeave:
			if failedCount <= 0 {
				// limit maximum failed count
				log.
					Warn("failed to quit more than 3 times, just quit.")

				return
			}

			if err := d.raft.Shutdown(); err != nil {
				time.Sleep(5 * time.Second)
				log.
					Errorf("app.Heartbeat could not remove from cluster: %v", err)
				failedCount--
				goto retryLeave
			}

			return
		}
	}
}

// startRoutines Nonblocking API
func (d *app) startRoutines() {
	// receive watch changes from raft fsm and notify watcher
	runtime.GoFunc("watch-changes", d.propagateChangesSignal)
}

func (d app) propagateChangesSignal() error {
	ch := d.raft.ChangeNotifyCh()

	for change := range ch {
		log.
			Debug("app.propagateChangesSignal got a signal")

		// container in TOML format change and delete cache
		d.watcher.ChangeNotify(change)
	}

	return nil
}

//var (
//	ErrNotLeader = errors.New("current node is not allow to write, should not be triggered normally")
//)

func (d app) Apply(data []byte) error {
	return d.raft.ApplyRaw(data)
}

// AddNode only leader node would receive such request. MAYBE?
func (d app) AddNode(serverId, addr string) error {
	log.Infof("received AddNode request for remote node %s, addr %s", serverId, addr)
	return d.raft.AddNode(serverId, addr)
}

// RemoveNode only leader node would receive such request.
func (d app) RemoveNode(nodeID string) error {
	return d.raft.RemoveNode(nodeID)
}

//func (c app) Apply(msg []byte) (err error) {
//	return c.raft.ApplyFromMessage(msg)
//}

// isLeader only return true if current node is leader.
func (d app) isLeader() bool {
	return d.raft.IsLeader()
}

func (d app) ShouldForwardToLeader() (shouldForward bool, leadAddr string) {
	return !d.isLeader(), d.raft.GetLeaderAddr()
}

func (d *app) GetKV(key string) (*types.StoreValue, error) {
	val, err := d.raft.GetKV(key)
	if err != nil {
		return nil, err
	}

	return val, nil
}

func (d *app) SetKV(key string, val []byte) (err error) {
	log.
		WithFields(log.Fields{
			"key": key,
			"val": runtime.ToString(val),
		}).
		Debug("app.SetKV called")

	return d.raft.SetKV(key, val)
}

func (d *app) UnsetKV(key string) error {
	return d.raft.UnsetKV(key)
}

func (d *app) Watch(keys ...string) (ob watcher.IObserver, cancelFn func()) {
	ch := make(chan watcher.IChange, 2)
	closeFn := func() {
		log.Debug("observer closeFn called")
		close(ch)
	}
	ob = NewTopicObserver(ch, closeFn, keys)
	d.watcher.Subscribe(ob)

	return ob, func() {
		d.watcher.Unsubscribe(ob)
	}

	//timeout := time.NewTimer(30 * time.Second)
	//select {
	//case change := <-ch:
	//	log.
	//		WithFields(log.Fields{
	//			"keys":      keys,
	//			"changeKey": change.Topic(),
	//			"change":    change,
	//		}).
	//		Info("app.Watch received one change")
	//case <-timeout.C:
	//	log.
	//		WithFields(log.Fields{
	//			"keys": keys,
	//		}).
	//		Debug("app.Watch timeout")
	//}
	//
	//return
}
