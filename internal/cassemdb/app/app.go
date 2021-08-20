package app

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/yeqown/log"

	"github.com/yeqown/cassem/internal/cassemdb/domain"
	raftleader "github.com/yeqown/cassem/internal/cassemdb/infras/raft-leader-grpc"
	"github.com/yeqown/cassem/internal/cassemdb/infras/repository"
	"github.com/yeqown/cassem/pkg/conf"
	"github.com/yeqown/cassem/pkg/httpx"
	"github.com/yeqown/cassem/pkg/runtime"
	"github.com/yeqown/cassem/pkg/watcher"
)

// app is the storage server that would guard api server running and alas controls other components.
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
	raft domain.IMyRaft
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
	d.raft, err = domain.NewMyRaft(cfg)
	if err != nil {
		return errors.Wrapf(err, "app.bootstrap failed to load raft")
	}
	log.Info("app: raft component loaded")

	d.startRoutines()

	return nil
}

// Run start a ticker to print log and check healthy of each component in core.app.
// The second purpose is to watch the QUIT / KILL signal to release resources of core.app, the most important work is to
// let current node leave raft cluster. If current node is leader just quit, otherwise current node should tell the leader
// about the fact there is a node is shutting down.
//
// Notice that, tryLeaveCluster maybe failed if cluster could not be maintained while there is only one node in cluster,
// it could not be removed, it will still be elected as leader. (Situation: count of cluster nodes is less than 2).
//
// NOTE: could leader call removeNode by itself? (leader could call removeNode only when cluster has more than 1 node)
func (d *app) Run() {
	tick := time.NewTicker(10 * time.Second)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	for {
		select {
		case <-tick.C:
			log.
				WithFields(log.Fields{
					"isLeader": d.raft.IsLeader(),
				}).
				Info("app heartbeat")
		case <-quit:
			log.Info("app quit, start release resources...")
			// DONE(@yeqown): graceful shutdown components, snapshot something.
			failedCount := 3

		retryLeave:
			if failedCount <= 0 {
				// maximum failed count limit overflow
				log.
					Warn("failed to quit more than 3 times, just quit.")

				return
			}
			if err := d.raft.Shutdown(); err != nil {
				time.Sleep(3 * time.Second)
				log.
					Errorf("app.Run could not remove from cluster: %v", err)
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

	// gRPC serving
	runtime.GoFunc("serving-grpc-api", d.servingAPI)
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

func (d *app) servingAPI() error {
	s := gRPC(d)
	raftleader.Setup(d.raft.RAFT(), s, d.config.Raft.ClusterAddresses)

	if runtime.IsDebug() {
		g := httpx.NewGateway(d.config.Addr, debugHTTP(d), s)
		return g.ListenAndServe()
	}

	return serve(s, d.config.Addr)
}

// addNode only leader node would receive such request. MAYBE?
func (d app) addNode(serverId, addr string) error {
	log.Infof("received addNode request for remote node %s, addr %s", serverId, addr)
	// return d.raft.addNode(serverId, addr)
	panic("TODO")
}

// removeNode only leader node would receive such request.
func (d app) removeNode(nodeID string) error {
	// return d.raft.removeNode(nodeID)
	panic("TODO")
}

func (d *app) getKV(key string) (*repository.StoreValue, error) {
	val, err := d.raft.GetKV(key)
	if err != nil {
		return nil, err
	}

	return val, nil
}

const (
	// MAX_TTL 2d
	MAX_TTL = 2 * 24 * 3600
)

func (d *app) setKV(param *setKVParam) (err error) {
	if param.ttl > MAX_TTL {
		return errors.New("ttl overflow: maximum is 172800(2*24*3600)")
	}

	log.
		WithFields(log.Fields{
			"param": param,
		}).
		Debug("app.setKV called")

	return d.raft.SetKV(param.key, param.val, param.isDir, param.overwrite, param.ttl)
}

func (d *app) unsetKV(param *unsetKVParam) error {
	log.
		WithFields(log.Fields{
			"param": param,
		}).
		Debug("app.unsetKV called")

	return d.raft.UnsetKV(param.key, param.isDir)
}

func (d *app) watch(keys ...string) (ob watcher.IObserver, cancelFn func()) {
	ch := make(chan watcher.IChange, 2)
	closeFn := func() {
		log.Debug("observer closeFn called")
		close(ch)
	}
	ob = newTopicObserver(ch, closeFn, keys)
	d.watcher.Subscribe(ob)

	return ob, func() {
		d.watcher.Unsubscribe(ob)
	}
}

func (d *app) iterate(param *rangeParam) (*repository.RangeResult, error) {
	return d.raft.Range(param.key, param.seek, param.limit)
}

// expire one key in cassemdb, but notice that the never expired key
// will skip expire operation.
//
// FIXED(@yeqown): expire the key instead of clear it directly.
func (d *app) expire(key string) error {
	return d.raft.Expire(key)
}

func (d *app) ttl(key string) (int32, error) {
	v, err := d.getKV(key)
	if err != nil {
		return 0, err
	}

	return v.TTL, nil
}
