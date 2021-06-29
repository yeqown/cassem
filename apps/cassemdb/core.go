package cassemdb

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yeqown/cassem/apps/cassemdb/delivery"
	"github.com/yeqown/cassem/apps/cassemdb/myraft"
	"github.com/yeqown/cassem/apps/cassemdb/persistence/bbolt"
	"github.com/yeqown/cassem/internal/conf"
	"github.com/yeqown/cassem/pkg/runtime"

	"github.com/pkg/errors"
	"github.com/yeqown/log"
)

// cassemdb is the storage server that would guards api server running and alas controls other components.
// Especially, raft protocol which supports the architecture of cassemagent (master-slave).
//
// Notice that all writes must be operated on master node, salve nodes could execute read operations.
//
type cassemdb struct {
	config *conf.Config

	// apiGate contains HTTP and gRPC protocol server. HTTP server provides all PUBLIC managing API and
	// internal cluster API. The duty of gRPC server is serving cassem's clients for watching changes.
	//
	// Notice that HTTP server and gRPC server use backend of gateway, so there is only one TCP port to
	// listen on.
	apiGate *delivery.Gateway

	//// watcher is abstract watcher.IWatcher layer, so trigger and observers could be splitting.
	//// core.cassemdb has no need to care about how to get touch with observers, just produce signal and data.
	//watcher watcher.IWatcher

	// raft is a customized raft node for cassem.
	raft myraft.IMyRaft
}

func New(cfg *conf.Config) (*cassemdb, error) {
	d := new(cassemdb)
	if err := d.bootstrap(cfg); err != nil {
		return nil, err
	}

	go d.loop()

	return d, nil
}

func (c *cassemdb) bootstrap(cfg *conf.Config) (err error) {
	c.config = cfg

	repo, err := bbolt.New(cfg.Persistence.BBolt)
	if err != nil {
		return errors.Wrap(err, "cassemdb.bootstrap failed to load bolt")
	}
	log.Info("cassemdb: persistence component loaded")

	c.apiGate = delivery.New(cfg.Server.HTTP, c)
	log.Info("cassemdb: HTTP server loaded")

	c.raft, err = myraft.New(&myraft.Conf{
		Raft: cfg.Server.Raft,
		HTTP: cfg.Server.HTTP,
		Repo: repo,
	})
	if err != nil {
		return errors.Wrapf(err, "cassemdb.bootstrap failed to load raft")
	}
	log.Info("cassemdb: raft component loaded")

	return nil
}

// Heartbeat start a ticker to print log and check healthy of each component in core.cassemdb.
// The second purpose is to watch the QUIT / KILL signal to release resources of core.cassemdb, the most important work is to
// let current node leave raft cluster. If current node is leader just quit, otherwise current node should tell the leader
// about the fact there is a node is shutting down.
//
// Notice that, tryLeaveCluster maybe failed if cluster could not be maintained while there is only one node in cluster,
// it could not be removed, it will still be elected as leader. (Situation: count of cluster nodes is less than 2).
//
// NOTE: could leader call removeNode by it self? (leader could call removeNode only when cluster has more than 1 node)
func (c *cassemdb) Heartbeat() {
	tick := time.NewTicker(10 * time.Second)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	for {
		select {
		case <-tick.C:
			log.
				WithFields(log.Fields{
					"isLeader":      c.isLeader(),
					"joinedCluster": c.raft.JoinedCluster(),
				}).
				Info("cassemdb is running")
		case <-quit:
			log.Info("cassemdb quit, start release resources...")
			// DONE(@yeqown): graceful shutdown components, snapshot something.
			failedCount := 3

		retryLeave:
			if failedCount <= 0 {
				// limit maximum failed count
				log.
					Warn("failed to quit more than 3 times, just quit.")

				return
			}

			if err := c.raft.Shutdown(); err != nil {
				time.Sleep(5 * time.Second)
				log.
					Errorf("cassemdb.Heartbeat could not remove from cluster: %v", err)
				failedCount--
				goto retryLeave
			}

			return
		}
	}
}

func (c cassemdb) loop() {
	// start apiGate
	go runtime.GoFunc("api-gate", c.runningGateway)
	//
	//// receive watch changes from raft fsm and notify watcher
	//go runtime.GoFunc("watch-changes", c.propagateChangesSignal)
}

func (c cassemdb) runningGateway() (err error) {
	if err = c.apiGate.ListenAndServe(); err != nil {
		log.Errorf("cassemdb.failed to runningGateway: %v", err)
	}

	return
}

func (c cassemdb) Apply(data []byte) error {
	return c.raft.ApplyRaw(data)
}

//
//func (c cassemdb) propagateChangesSignal() error {
//	ch := c.raft.ChangeNotifyCh()
//
//	for {
//		select {
//		case changes := <-ch:
//			log.
//				Debug("cassemdb.propagateChangesSignal got a signal")
//
//			// container in TOML format changes and delete cache
//			c.watcher.ChangeNotify(changes)
//		}
//	}
//}
