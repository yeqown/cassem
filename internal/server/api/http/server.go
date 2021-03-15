package http

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/yeqown/cassem/internal/authorizer"
	"github.com/yeqown/cassem/internal/conf"
	coord "github.com/yeqown/cassem/internal/coordinator"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/yeqown/log"
)

// Server provides both RESTFul API for client also provides part of API for internal cluster, all internal APIs
// stay in handler_cluster.go and register in Server.mountRaftClusterInternalAPI.
type Server struct {
	engi *gin.Engine

	_cfg conf.HTTP

	auth authorizer.IAuthorizer

	// coordinator
	coordinator coord.ICoordinator
}

func New(c *conf.HTTP, coordinator coord.ICoordinator, auth authorizer.IAuthorizer) *Server {
	if c.Addr == "" {
		c.Addr = ":2021"
	}

	srv := &Server{
		engi:        gin.New(),
		_cfg:        *c,
		auth:        auth,
		coordinator: coordinator,
	}

	srv.initialize()

	return srv
}

func (srv *Server) initialize() {
	gin.EnableJsonDecoderUseNumber()
	if !srv._cfg.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	// mount middlewares
	// DONE(@yeqown): replace Recovery middleware so that we response error messages.
	srv.engi.Use(recovery())
	srv.engi.Use(gin.Logger())

	if srv._cfg.Debug {
		pprof.Register(srv.engi, "/debug/pprof")
	}

	// mount operate raft API
	srv.mountRaftClusterInternalAPI()
	// mount API
	srv.mountAPI()
}

func (srv *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	srv.engi.ServeHTTP(w, req)
}

// needForwardAndExecute checks current request should be forwarded to leader or not, if it's needed to
// forward, HTTP invocation would be executed, and handle HTTP response by needForwardAndExecute itself.
func (srv *Server) needForwardAndExecute(c *gin.Context) (shouldForward bool) {
	var leaderAddr string
	if shouldForward, leaderAddr = srv.coordinator.ShouldForwardToLeader(); !shouldForward {
		return
	}

	// execute forward calling
	if err := forwardToLeader(c, leaderAddr); err != nil {
		responseError(c, err)
		return
	}

	return
}

var (
	_isProxySetting int32 = 0
	_lastLeaderAddr       = ""
	_lastProxy      *httputil.ReverseProxy
)

func getProxy(leaderAddr string) (*httputil.ReverseProxy, error) {
	if leaderAddr == _lastLeaderAddr && _lastProxy != nil {
		// hit cache and proxy has been initialized.
		return _lastProxy, nil
	}

	// initialize proxy
	for !atomic.CompareAndSwapInt32(&_isProxySetting, 0, 1) {
		// blocking lock
	}

	// fix leaderAddr schema
	if !strings.HasPrefix(leaderAddr, "http://") && !strings.HasPrefix(leaderAddr, "https://") {
		leaderAddr = "http://" + leaderAddr
	}

	remote, err := url.Parse(leaderAddr)
	if err != nil {
		return nil, errors.Wrap(err, "getProxy failed to parse leaderAddr")
	}
	_lastLeaderAddr = leaderAddr
	_lastProxy = httputil.NewSingleHostReverseProxy(remote)
	atomic.CompareAndSwapInt32(&_isProxySetting, 1, 0)

	return _lastProxy, nil
}

// DONE(@yeqown): maybe cache the reverse proxy client
func forwardToLeader(c *gin.Context, leaderAddr string) error {
	log.
		WithFields(log.Fields{
			"leaderAddr": leaderAddr,
		}).
		Debug("forwardToLeader called caused by current node is not leader")

	if leaderAddr == "" {
		return errors.New("forwardToLeader could not executed: empty leaderAddr")
	}

	proxy, err := getProxy(leaderAddr)
	if err != nil {
		return err
	}

	proxy.ServeHTTP(c.Writer, c.Request)

	return nil
}
