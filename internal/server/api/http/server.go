package http

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	coord "github.com/yeqown/cassem/internal/coordinator"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/yeqown/log"
)

// Server provides both RESTFul API for client also provides part of API for internal cluster, all internal APIs
// stay in handler_cluster.go and register in Server.mountRaftClusterInternalAPI.
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
	srv.mountRaftClusterInternalAPI()
	// mount API
	srv.mountAPI()
}

func (srv *Server) ListenAndServe() (err error) {
	log.Debugf("server running on: %s", srv._cfg.Addr)

	if err = srv.engi.Run(srv._cfg.Addr); err != nil {
		log.Errorf("server running failed: %v", err)
	}

	return
}

// needForwardAndExecute checks current request should be forwarded to leader, if needed
// forwarding calling would be executed and handle response by needForwardAndExecute itself.
func (srv *Server) needForwardAndExecute(c *gin.Context) (forwarded bool) {
	var leaderAddr string
	if forwarded, leaderAddr = srv.coordinator.ShouldForwardToLeader(); !forwarded {
		return
	}

	// execute forward calling
	if err := forwardToLeader(c, leaderAddr); err != nil {
		responseError(c, err)
		return
	}

	return
}

// TODO(@yeqown): maybe cache the reverse proxy client?
func forwardToLeader(c *gin.Context, leaderAddr string) error {
	log.
		WithFields(log.Fields{
			"leaderAddr": leaderAddr,
		}).
		Debug("forwardToLeader called caused by current node is not leader")

	// fix leaderAddr schema
	if !strings.HasPrefix(leaderAddr, "http://") && !strings.HasPrefix(leaderAddr, "https://") {
		leaderAddr = "http://" + leaderAddr
	}

	remote, err := url.Parse(leaderAddr)
	if err != nil {
		return errors.Wrap(err, "forwardToLeader failed to parse leaderAddr")
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)

	// define the director func
	proxy.Director = func(req *http.Request) {
		req.Header = c.Request.Header
		req.Host = remote.Host
		req.URL.Scheme = remote.Scheme
		req.URL.Host = remote.Host
		req.URL.Path = c.Request.URL.Path
	}

	proxy.ServeHTTP(c.Writer, c.Request)

	return nil
}
