package core

import (
	"os"
	"os/signal"
	"syscall"
	"time"

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
	// containerCache
	containerCache cache.ICache
	// watcher TODO(@yeqown):

	// raft related properties.
	serverId      string
	joinedCluster bool
	raft          *raft.Raft
	fsm           raft.FSM
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

	// start raft
	// DONE(@yeqown) serverId shoule be persistence so that we can recover it from panic.
	c.serverId = cfg.Server.Raft.ServerId
	c.fsm = newFSM()
	if err = c.bootstrapRaft(); err != nil {
		return errors.Wrapf(err, "Core.initialize failed to load raft")
	}

	return nil
}

func (c *Core) Heartbeat() {
	tick := time.NewTicker(10 * time.Second)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Kill, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

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
			//retryLeave:
			//	if err := c.tryLeaveCluster(); err != nil {
			//		log.Errorf("could not tryLeaveCluster cluster: %v", err)
			//		goto retryLeave
			//	}
			// TODO(@yeqown): graceful shutdown components
			return
		}
	}
}

func (c Core) loop() {
	// start restapi
	go startWithRecover("restapi", c.startHTTP)

	// cluster-daemon
	//go startWithRecover("cluster-daemon", c.serveClusterNode)
}

func (c Core) startHTTP() (err error) {
	if err = c.restapi.ListenAndServe(); err != nil {
		log.Errorf("Core.failed to start: %v", err)
	}

	return
}
