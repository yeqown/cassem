package http

import (
	coord "github.com/yeqown/cassem/internal/coordinator"

	"github.com/gin-gonic/gin"
	"github.com/yeqown/log"
)

type Server struct {
	engi *gin.Engine

	addr string

	// coordinator
	coordinator coord.ICoordinator
}

func New(addr string, coordinator coord.ICoordinator) *Server {
	if addr == "" {
		addr = ":2021"
	}

	srv := &Server{
		engi:        gin.New(),
		addr:        addr,
		coordinator: coordinator,
	}
	srv.boot()

	return srv
}

func (srv *Server) boot() {
	// mount middlewares
	srv.engi.Use(gin.Recovery())
	srv.engi.Use(gin.Logger())
	// TODO(@yeqown) authorize middleware is needed.

	// mount API
	srv.mountAPI(srv.engi)
}

func (srv *Server) ListenAndServe() (err error) {
	log.Debugf("server running on: %s", srv.addr)

	if err = srv.engi.Run(srv.addr); err != nil {
		log.Errorf("server running failed: %v", err)
	}

	return
}
