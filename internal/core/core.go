package core

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yeqown/cassem/internal/authorizer"
	"github.com/yeqown/cassem/internal/conf"
	coord "github.com/yeqown/cassem/internal/coordinator"
	"github.com/yeqown/cassem/internal/persistence"
	"github.com/yeqown/cassem/internal/persistence/mysql"
	"github.com/yeqown/cassem/internal/server/api"
	"github.com/yeqown/cassem/internal/watcher"
	"github.com/yeqown/cassem/pkg/runtime"

	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
	"github.com/yeqown/log"
)

// Core is the cassemd server that would guards api server running and alas controls other components.
// Especially, raft protocol which supports the architecture of cassemd (master-slave).
//
// Notice that all writes must be operated on master node, salve nodes could execute read operations.
//
type Core struct {
	coord.ICoordinator

	config *conf.Config

	// All components in core.Core are following.
	//
	// repo is the entry to communicate with persistence component. It helps core.Core to implement
	// coord.ICoordinator interface.
	repo persistence.Repository

	// convertor helps data conversion between persistence and service logic.
	convertor persistence.Converter

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

	// The following properties are all about raft consensus algorithm.
	//
	// serverId indicates the unique identify in raft cluster,
	// also used to as the raft store directory name.
	serverId string

	// tryJoinIdx is a index to memory which address should be tried when
	// next tryJoinCluster called. This property helps to support that multiple
	// addresses of cluster nodes can be passed.
	tryJoinIdx int

	// joinedCluster indicates whether current node has joined to the cluster.
	// If current node is started as leader node, joinedCluster is true as default, otherwise
	// joinedCluster controls tryJoinCluster again and again in Core.Heartbeat.
	joinedCluster bool

	// raft is a core of core.Core to construct an distributed system.
	raft *raft.Raft

	// fsm is the state machine to be used in raft.RAFT. In cassem, it's mainly used to store
	// caches those encoded bytes to containers who are requested and should be cached.
	//
	// It also be used to store and apply leaderAddr which indicates the address of the leader.
	// While a leadership changes happened, leader node calls raft.Apply() to commit a log that
	// will update slave nodes' leaderAddr. please checkout FSMWrapper for more information.
	fsm FSMWrapper
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

	_auth, err := authorizer.New(cfg.Persistence.Mysql)
	if err != nil {
		return errors.Wrapf(err, "Core.initialize failed to load authorizer: %v", err)
	}

	c.apiGate = api.New(cfg.Server.HTTP, c, _auth)
	log.Info("Core: HTTP server loaded")

	//c._containerCache = cache.NewNonCache()
	//log.Info("Core: cache loaded")

	c.watcher = watcher.NewChannelWatcher(64)
	log.Info("Core: watcher component loaded")

	c.serverId = cfg.Server.Raft.ServerId
	if err = c.bootstrapRaft(); err != nil {
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
					"joinedCluster": c.joinedCluster,
				}).
				Info("Core is running")

			if !c.joinedCluster {
				if err := c.tryJoinCluster(); err != nil {
					log.Errorf("could not tryJoinCluster cluster: %v", err)
				}
			}
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

			if c.isLeader() {
				// FIXED(@yeqown): if there is no more nodes in cluster, just let leader quit.
				if len(c.raft.GetConfiguration().Configuration().Servers) < 2 {
					return
				}

				if err := c.RemoveNode(c.serverId); err != nil {
					log.
						Errorf("Core.Heartbeat (leader) could not remove from cluster: %v", err)
				}

				failedCount--
				time.Sleep(5 * time.Second)
				goto retryLeave
			}

			if err := c.tryLeaveCluster(); err != nil {
				log.
					Errorf("Core.Heartbeat (node) could not remove from cluster: %v", err)

				failedCount--
				time.Sleep(5 * time.Second)
				goto retryLeave
			}

			return
		}
	}
}

func (c Core) loop() {
	// start apiGate
	go runtime.GoFunc("api-gate", c.startGateway)

	// leadership changes
	go runtime.GoFunc("leadership-changes", c.watchLeaderChanges)

	// snapshot executor
	go runtime.GoFunc("snapshot-strategy", c.doSnapshot)

	// receive watch changes from raft fsm and notify watcher
	go runtime.GoFunc("watch-changes", c.propagateChangesSignal)
}

func (c Core) startGateway() (err error) {
	if err = c.apiGate.ListenAndServe(); err != nil {
		log.Errorf("Core.failed to startGateway: %v", err)
	}

	return
}

// DONE(@yeqown): let node be notified while leader changes, and also mark current node is leader or not?
func (c Core) watchLeaderChanges() error {
	isLeaderCh := c.raft.LeaderCh()
	for {
		select {
		case isLeader := <-isLeaderCh:
			log.
				WithField("toBeLeader", isLeader).
				Debug("Core.watchLeaderChanges got a signal")

			// FIXED(@yeqown): reset leader address when leadership transition has occurred.
			c.fsm.setLeaderAddr("")

			if !isLeader {
				continue
			}

			// broadcast leader itself address to nodes.
			// DONE(@yeqown): should broadcast to other nodes of leaders
			fsmLog, _ := newFsmLog(logActionSetLeaderAddr, &setLeaderAddrCommand{
				LeaderAddr: c.config.Server.HTTP.Addr,
			})
			if err := c.propagateToSlaves(fsmLog); err != nil {
				log.
					WithFields(log.Fields{
						"addr":   c.config.Server.HTTP.Addr,
						"fsmLog": fsmLog,
					}).
					Errorf("Core.watchLeaderChanges applyTo raft failed: %v", err)
			}
		}
	}
}

const (
	// _SIZE_EXECUTIONS is a value which limit the minimum count of logs
	// must be executed since last snapshot action.
	_SIZE_EXECUTIONS = 100
)

// doSnapshot to execute snapshot of state machine with specified strategy:
//
// 1. just do snapshot periodically.
// 2. if state machine has executed logs more than specified size.
//
func (c Core) doSnapshot() error {
	ticker := time.NewTicker(30 * time.Minute)
	sizeTicker := time.NewTicker(10 * time.Second)

	for {
		needSnapshot := false
		select {
		case <-ticker.C:
			needSnapshot = true

		case <-sizeTicker.C:
			if !c.isLeader() {
				continue
			}
			if c.fsm.getExecutionSinceLastSnapshot() > _SIZE_EXECUTIONS {
				// if the state machine has received log over than 10
				// after last snapshot.
				needSnapshot = true
			}

			log.
				WithField("needSnapshot", needSnapshot).
				Debug("Core.doSnapshot called")

			if needSnapshot {
				if err := c.raft.Snapshot().Error(); err != nil {
					log.Errorf("Core.doSnapshot failed to snapshot: %v", err)
				}
			}
			// case done
		}
	}
}

func (c Core) propagateChangesSignal() error {
	ch := c.fsm.changeNotifyCh()

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
