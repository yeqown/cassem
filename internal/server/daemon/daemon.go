package daemon

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
	"github.com/yeqown/cassem/internal/conf"
	coord "github.com/yeqown/cassem/internal/coordinator"
	"github.com/yeqown/cassem/internal/persistence"
	"github.com/yeqown/cassem/internal/persistence/mysql"
	apihtp "github.com/yeqown/cassem/internal/server/api/http"
	"github.com/yeqown/log"
)

type Config struct {
	Base     string `toml:"dir"`
	BindAddr string `toml:"bind_addr"`
	Join     string `toml:"tryJoinCluster"`
}

// Daemon is the cassemd server that would guards api server running and alas controls other components. Especially,
// raft protocol which supports the architecture of cassemd (master-slave). All writes must be operated on master node,
// salve nodes could execute read operations.
type Daemon struct {
	coord.ICoordinator

	// cfg
	cfg *conf.Config

	// components
	// repo
	repo persistence.Repository
	// restapi
	restapi *apihtp.Server
	// cache TODO(@yeqown):
	// watcher TODO(@yeqown):

	// raft related properties.
	serverId      string
	joinedCluster bool
	raft          *raft.Raft
	fsm           raft.FSM
}

func New(cfg *conf.Config) (*Daemon, error) {
	d := new(Daemon)
	if err := d.initialize(cfg); err != nil {
		return nil, err
	}

	go d.loop()

	return d, nil
}

func (d *Daemon) initialize(cfg *conf.Config) (err error) {
	d.cfg = cfg

	d.repo, err = mysql.New(cfg.Persistence.Mysql)
	if err != nil {
		return errors.Wrapf(err, "Daemon.initialize failed to load persistence: %v", err)
	}
	log.Info("Daemon: persistence component loaded")

	d.restapi = apihtp.New(cfg.Server.HTTP, d)
	log.Info("Daemon: HTTP server loaded")

	// start raft
	// DONE(@yeqown) serverId shoule be persistence so that we can recover it from panic.
	d.serverId = cfg.Server.Raft.ServerID
	d.fsm = newFSM()
	if err = d.bootstrapRaft(); err != nil {
		return errors.Wrapf(err, "Daemon.initialize failed to load raft")
	}

	return nil
}

func (d *Daemon) Heartbeat() {
	tick := time.NewTicker(10 * time.Second)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Kill, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	for {
		select {
		case <-tick.C:
			log.Info("Daemon is running")
			if !d.joinedCluster {
				if err := d.tryJoinCluster(); err != nil {
					log.Errorf("could not tryJoinCluster cluster: %v", err)
				}
			}
		case <-quit:
			log.Info("Daemon quit, start release resources...")
			//retryLeave:
			//	if err := d.tryLeaveCluster(); err != nil {
			//		log.Errorf("could not tryLeaveCluster cluster: %v", err)
			//		goto retryLeave
			//	}
			// TODO(@yeqown): graceful shutdown components
			return
		}
	}
}

func (d Daemon) loop() {
	// start restapi
	go startWithRecover("restapi", d.startHTTP)

	// cluster-daemon
	//go startWithRecover("cluster-daemon", d.serveClusterNode)
}

func (d Daemon) startHTTP() (err error) {
	if err = d.restapi.ListenAndServe(); err != nil {
		log.Errorf("Daemon.failed to start: %v", err)
	}

	return
}
