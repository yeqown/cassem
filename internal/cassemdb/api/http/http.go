package http

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/yeqown/cassem/internal/cassemdb/app"
	"github.com/yeqown/cassem/pkg/httpx"
	"github.com/yeqown/cassem/pkg/runtime"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/yeqown/log"
)

// httpServer provides both RESTFul API for client also provides part of API for internal cluster, all internal APIs
// stay in handler_cluster.go and register in httpServer.mountRaftClusterInternalAPI.
type httpServer struct {
	engi  *gin.Engine
	coord app.ICoordinator
}

func New(coord app.ICoordinator) *httpServer {
	srv := &httpServer{
		coord: coord,
		engi:  gin.New(),
	}

	srv.initialize()

	return srv
}

func (srv *httpServer) initialize() {
	gin.EnableJsonDecoderUseNumber()
	if runtime.IsDebug() {
		gin.SetMode(gin.ReleaseMode)
	}

	// mount middlewares
	// DONE(@yeqown): replace Recovery middleware so that we response error messages.
	srv.engi.Use(httpx.Recovery())
	srv.engi.Use(gin.Logger())

	if runtime.IsDebug() {
		pprof.Register(srv.engi, "/debug/pprof")
	}

	// mount operate raft API
	srv.mountRaftClusterInternalAPI()
	// mount API
	srv.mountAPI()
}

func (srv *httpServer) mountRaftClusterInternalAPI() {
	// DONE(@yeqown): cluster need authorize too to reject request from cluster outside.
	cluster := srv.engi.Group("/cluster", httpx.ClusterAuthorizeSimple())
	{
		cluster.GET("/nodes", srv.OperateNode)
		cluster.POST("/apply", srv.Apply)
	}
}

func (srv *httpServer) mountAPI() {
	// DONE(@yeqown) authorize middleware is needed.
	g := srv.engi.Group("/api")

	ns := g.Group("/kv")
	{
		ns.GET("", srv.GetKV)
		ns.POST("", srv.SetKV)
		ns.DELETE("", srv.DeleteKV)

		ns.GET("/watch", srv.Watch)
	}
}

func (srv *httpServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	srv.engi.ServeHTTP(w, req)
}

// needForwardAndExecute checks current request should be forwarded to leader or not, if it's needed to
// forward, HTTP invocation would be executed, and handle HTTP response by needForwardAndExecute itself.
func (srv *httpServer) needForwardAndExecute(c *gin.Context) (shouldForward bool) {
	var leaderAddr string
	if shouldForward, leaderAddr = srv.coord.ShouldForwardToLeader(); !shouldForward {
		return
	}

	// execute forward calling
	if err := forwardToLeader(c, leaderAddr); err != nil {
		httpx.ResponseError(c, err)
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
