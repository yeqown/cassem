package core

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yeqown/cassem/internal/authorizer"
	"github.com/yeqown/cassem/internal/conf"
	"github.com/yeqown/cassem/internal/core/api"
	"github.com/yeqown/cassem/internal/myraft"
	"github.com/yeqown/cassem/internal/persistence"
	"github.com/yeqown/cassem/internal/persistence/mysql"
	"github.com/yeqown/cassem/internal/watcher"
	"github.com/yeqown/cassem/pkg/runtime"

	"github.com/pkg/errors"
	"github.com/yeqown/log"
)

// Core is the cassemd server that would guards api server running and alas controls other components.
// Especially, raft protocol which supports the architecture of cassemd (master-slave).
//
// Notice that all writes must be operated on master node, salve nodes could execute read operations.
//
type Core struct {
	// coord.ICoordinator

	config *conf.Config

	// All components in core.Core are following.
	//
	// repo is the entry to communicate with persistence component. It helps core.Core to implement
	// coord.ICoordinator interface.
	repo persistence.Repository

	// convertor helps data conversion between persistence and service logic.
	convertor persistence.Converter

	// auth
	auth authorizer.IAuthorizer

	// apiGate contains HTTP and gRPC protocol server. HTTP server provides all PUBLIC managing API and
	// internal cluster API. The duty of gRPC server is serving cassem's clients for watching changes.
	//
	// Notice that HTTP server and gRPC server use backend of gateway, so there is only one TCP port to
	// listen on.
	apiGate *api.Gateway

	// _containerCache is a cache component which provides set, delete, get, persist and restore abilities.
	// DONE(@yeqown): remove this component from core.Core but hold in fsm.
	// checkout https://github.com/yeqown/cassem/issues/7 for more information.
	//_containerCache cache.ICache

	// watcher is abstract watcher.IWatcher layer, so trigger and observers could be splitting.
	// core.Core has no need to care about how to get touch with observers, just produce signal and data.
	watcher watcher.IWatcher

	// raft is a customized raft node for cassem.
	raft myraft.IMyRaft
}

func New(cfg *conf.Config) (*Core, error) {
	d := new(Core)
	if err := d.initialize(cfg); err != nil {
		return nil, err
	}

	go d.loop()

	return d, nil
}

func (c *Core) initialize(cfg *conf.Config) (err error) {
	c.config = cfg

	c.repo, err = mysql.New(cfg.Persistence.Mysql)
	if err != nil {
		return errors.Wrapf(err, "Core.initialize failed to load persistence: %v", err)
	}
	log.Info("Core: persistence component loaded")

	c.convertor = mysql.NewConverter()

	c.auth, err = authorizer.New(cfg.Persistence.Mysql)
	if err != nil {
		return errors.Wrapf(err, "Core.initialize failed to load auth: %v", err)
	}

	c.apiGate = api.New(cfg.Server.HTTP, c)
	log.Info("Core: HTTP server loaded")

	c.watcher = watcher.NewChannelWatcher(64)
	log.Info("Core: watcher component loaded")

	c.raft, err = myraft.New(&myraft.Conf{Raft: cfg.Server.Raft, HTTP: cfg.Server.HTTP})
	if err != nil {
		return errors.Wrapf(err, "Core.initialize failed to load raft")
	}
	log.Info("Core: raft component loaded")

	return nil
}

// Heartbeat start a ticker to print log and check healthy of each component in core.Core.
// The second purpose is to watch the QUIT / KILL signal to release resources of core.Core, the most important work is to
// let current node leave raft cluster. If current node is leader just quit, otherwise current node should tell the leader
// about the fact there is a node is shutting down.
//
// Notice that, tryLeaveCluster maybe failed if cluster could not be maintained while there is only one node in cluster,
// it could not be removed, it will still be elected as leader. (Situation: count of cluster nodes is less than 2).
//
// NOTE: could leader call removeNode by it self? (leader could call removeNode only when cluster has more than 1 node)
func (c *Core) Heartbeat() {
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
				Info("Core is running")
		case <-quit:
			log.Info("Core quit, start release resources...")
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
					Errorf("Core.Heartbeat could not remove from cluster: %v", err)
				failedCount--
				goto retryLeave
			}

			return
		}
	}
}

func (c Core) loop() {
	// start apiGate
	go runtime.GoFunc("api-gate", c.runningGateway)

	// receive watch changes from raft fsm and notify watcher
	go runtime.GoFunc("watch-changes", c.propagateChangesSignal)
}

func (c Core) runningGateway() (err error) {
	if err = c.apiGate.ListenAndServe(); err != nil {
		log.Errorf("Core.failed to runningGateway: %v", err)
	}

	return
}

func (c Core) propagateChangesSignal() error {
	ch := c.raft.ChangeNotifyCh()

	for {
		select {
		case changes := <-ch:
			log.
				Debug("Core.propagateChangesSignal got a signal")

			// container in TOML format changes and delete cache
			c.watcher.ChangeNotify(changes)
		}
	}
}
