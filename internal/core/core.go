package core

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yeqown/cassem/internal/watcher"

	"github.com/yeqown/cassem/internal/cache"
	"github.com/yeqown/cassem/internal/conf"
	coord "github.com/yeqown/cassem/internal/coordinator"
	"github.com/yeqown/cassem/internal/persistence"
	"github.com/yeqown/cassem/internal/persistence/mysql"
	apihtp "github.com/yeqown/cassem/internal/server/api/http"

	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
	"github.com/yeqown/log"
)

// Core is the cassemd server that would guards api server running and alas controls other components. Especially,
// raft protocol which supports the architecture of cassemd (master-slave). All writes must be operated on master node,
// salve nodes could execute read operations.
type Core struct {
	coord.ICoordinator

	// cfg
	cfg *conf.Config

	// components
	// repo
	repo persistence.Repository
	// restapi
	restapi *apihtp.Server
	// _containerCache
	_containerCache cache.ICache
	// watcher
	watcher watcher.IWatcher

	// raft related properties.
	serverId      string
	joinedCluster bool
	raft          *raft.Raft
	fsm           FSMWrapper
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
	c.cfg = cfg

	c.repo, err = mysql.New(cfg.Persistence.Mysql)
	if err != nil {
		return errors.Wrapf(err, "Core.initialize failed to load persistence: %v", err)
	}
	log.Info("Core: persistence component loaded")

	c.restapi = apihtp.New(cfg.Server.HTTP, c)
	log.Info("Core: HTTP server loaded")

	c._containerCache = cache.NewNonCache()
	log.Info("Core: cache loaded")

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
// NOTE: could leader call removeNode by it self?
func (c *Core) Heartbeat() {
	tick := time.NewTicker(10 * time.Second)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	for {
		select {
		case <-tick.C:
			log.Info("Core is running")
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
				return
			}

			if c.isLeader() {
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
	// start restapi
	go startWithRecover("restapi", c.startHTTP)

	// leadership changes
	go startWithRecover("leadership-changes", c.watchLeaderChanges)
}

func (c Core) startHTTP() (err error) {
	if err = c.restapi.ListenAndServe(); err != nil {
		log.Errorf("Core.failed to start: %v", err)
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

			// FIXED(@yeqown): reset leader address when leadership transition has occured.
			c.fsm.SetLeaderAddr("")

			if !isLeader {
				continue
			}

			// broadcast leader itself address to nodes.
			// DONE(@yeqown): should broadcast to other nodes of leaders
			msg, _ := newFsmLog(logActionSetLeaderAddr, setLeaderAddr{
				LeaderAddr: c.cfg.Server.HTTP.Addr,
			})
			if f := c.raft.Apply(msg, 10*time.Second); f.Error() != nil {
				log.
					WithFields(log.Fields{
						"addr": c.cfg.Server.HTTP.Addr,
						"msg":  msg,
					}).
					Errorf("Core.watchLeaderChanges applyTo raft failed: %v", f.Error())
			}
		}
	}
}

// isLeader only return true if current node is leader.
func (c Core) isLeader() bool {
	return c.raft.State() == raft.Leader
}

func (c Core) ShouldForwardToLeader() (shouldForward bool, leadAddr string) {
	return !c.isLeader(), c.fsm.LeaderAddr()
}
