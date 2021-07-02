package http

import (
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/yeqown/log"

	"github.com/yeqown/cassem/pkg/conf"
	"github.com/yeqown/cassem/pkg/httpx"
	"github.com/yeqown/cassem/pkg/runtime"
)

type httpServer struct {
	config *conf.HTTP
	engi   *gin.Engine
}

func New() *httpServer {
	s := httpServer{
		engi: gin.New(),
	}

	s.initialize()

	return &s
}

func (s httpServer) Run() {
	if err := s.engi.Run(s.config.Addr); err != nil {
		log.Fatal(err)
	}
}

func (s *httpServer) initialize() {
	gin.EnableJsonDecoderUseNumber()
	if runtime.IsDebug() {
		gin.SetMode(gin.ReleaseMode)
	}

	// mount middlewares
	// DONE(@yeqown): replace Recovery middleware so that we response error messages.
	s.engi.Use(httpx.Recovery())
	s.engi.Use(gin.Logger())

	if runtime.IsDebug() {
		pprof.Register(s.engi, "/debug/pprof")
	}

	// mount API
	s.mountAPI()
}

func (s *httpServer) mountAPI() {
	// DONE(@yeqown) authorize middleware is needed.
	g := s.engi.Group("/api")

	apps := g.Group("/apps")
	{
		apps.GET("", s.GetApps)
		apps.GET("/:appId", s.GetApp)
		apps.POST("/:appId", s.CreateApp)
		apps.DELETE("/:appId", s.DeleteApp)

		envs := apps.Group("/:appId/envs")
		_ = envs
	}
}
