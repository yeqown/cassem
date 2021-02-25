package daemon

import (
	"time"

	"github.com/pkg/errors"
	"github.com/yeqown/cassem/internal/conf"
	coord "github.com/yeqown/cassem/internal/coordinator"
	"github.com/yeqown/cassem/internal/persistence"
	"github.com/yeqown/cassem/internal/persistence/mysql"
	apihtp "github.com/yeqown/cassem/internal/server/api/http"
	"github.com/yeqown/log"
)

// Daemon is the cassemd server that would guards api server running and alas controls other components. Especially,
// raft protocol which supports the architecture of cassemd (master-slave). All writes must be operated on master node,
// salve nodes could execute read operations.
type Daemon struct {
	cfg *conf.Config

	isMaster bool

	// cache TODO(@yeqown):
	// watcher TODO(@yeqown):

	containerPairs persistence.Repository

	coordinator coord.ICoordinator

	httpd *apihtp.Server
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

	d.containerPairs, err = mysql.New(cfg.Persistence.Mysql)
	if err != nil {
		return errors.Wrapf(err, "Daemon.initialize failed: %v", err)
	}
	log.Info("Daemon: persistence component loaded")

	d.coordinator = coord.New(d.containerPairs)
	log.Info("Daemon: coordinator component loaded")

	d.httpd = apihtp.New(cfg.Server.HTTP, d.coordinator)
	log.Info("Daemon: HTTP server loaded")

	return nil
}

func (d *Daemon) Heartbeat() {
	tick := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-tick.C:
			log.Info("Daemon is running")
		}
	}
}

func (d Daemon) loop() {
	// start httpd
	go d.startWithRecover("httpd", d.startHTTP)
}

func (d Daemon) startHTTP() (err error) {
	if err = d.httpd.ListenAndServe(); err != nil {
		log.Errorf("Daemon.failed to start: %v", err)
	}

	return
}
