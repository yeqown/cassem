package http

import (
	coord "github.com/yeqown/cassem/internal/coordinator"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/yeqown/log"
)

type Server struct {
	engi *gin.Engine

	_cfg Config

	// coordinator
	coordinator coord.ICoordinator
}

type Config struct {
	Addr  string `toml:"addr"`
	Debug bool   `toml:"debug"`
}

func New(c *Config, coordinator coord.ICoordinator) *Server {
	if c.Addr == "" {
		c.Addr = ":2021"
	}

	srv := &Server{
		engi:        gin.New(),
		_cfg:        *c,
		coordinator: coordinator,
	}
	srv.boot()

	return srv
}

func (srv *Server) boot() {
	// mount middlewares
	srv.engi.Use(gin.Recovery())
	srv.engi.Use(gin.Logger())

	if srv._cfg.Debug {
		pprof.Register(srv.engi, "/debug/pprof")
	}

	// mount operate raft API
	srv.mountRaftAPI(srv.engi)
	// mount API
	srv.mountAPI(srv.engi)
}

func (srv *Server) ListenAndServe() (err error) {
	log.Debugf("server running on: %s", srv._cfg.Addr)

	if err = srv.engi.Run(srv._cfg.Addr); err != nil {
		log.Errorf("server running failed: %v", err)
	}

	return
}
