package app

import (
	"log"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/yeqown/cassem/internal/cassemadm/infras"
	"github.com/yeqown/cassem/internal/concept"
	"github.com/yeqown/cassem/pkg/conf"
	"github.com/yeqown/cassem/pkg/httpx"
	"github.com/yeqown/cassem/pkg/runtime"
)

type app struct {
	conf *conf.CassemAdminConfig

	repo infras.Repository

	// aggregate is special methods interface customized form adm component which
	// can only be used by cassemadm.app.
	aggregate concept.AdmAggregate

	// agents type agentPool is a pool contains all agents nodes, and agentPool will update
	// agent nodes  automatically.
	agents *agentPool
}

func New(c *conf.CassemAdminConfig) (*app, error) {
	if err := c.Valid(); err != nil {
		return nil, errors.Wrap(err, "cassemadm.New failed")
	}

	agg, err := concept.NewAdmAggregate(c.CassemDBEndpoints)
	if err != nil {
		return nil, errors.Wrap(err, "cassemadm.New")
	}

	d := &app{
		aggregate: agg,
		conf:      c,
		repo:      nil, // FIXME: initialize repo
		agents:    newAgentPool(agg),
	}

	return d, nil
}

func (d app) Run() {
	engi := gin.New()

	d.initialHTTP(engi)

	if err := engi.Run(d.conf.HTTP.Addr); err != nil {
		log.Fatal(err)
	}
}

func (d app) initialHTTP(engi *gin.Engine) {
	gin.EnableJsonDecoderUseNumber()
	if !runtime.IsDebug() {
		gin.SetMode(gin.ReleaseMode)
	}

	engi.Use(httpx.Recovery())
	engi.Use(httpx.Logger())

	if runtime.IsDebug() {
		pprof.Register(engi, "/debug/pprof")
	}

	// mount APIs
	// DONE(@yeqown) authorize middleware is needed.
	g := engi.Group("/api")
	apps := g.Group("/apps")
	{
		apps.GET("", d.GetApps)
		apps.GET("/:appId", d.GetApp)
		apps.POST("/:appId", d.CreateApp)
		apps.DELETE("/:appId", d.DeleteApp)

		envs := apps.Group("/:appId/envs")
		{
			envs.GET("", d.GetAppEnvironments)

			elt := envs.Group("/:env/elements")
			{
				elt.GET("", d.GetAppEnvElements)
				elt.GET("/:key", d.GetAppEnvElement)
				elt.POST("/:key", d.CreateAppEnvElement)
				elt.PUT("/:key", d.UpdateAppEnvElement)
				elt.DELETE("/:key", d.DeleteAppEnvElement)

				elt.GET("/:key/versions", d.GetAppEnvElementAllVersions)
				elt.GET("/:key/diff", d.DiffAppEnvElement)
				elt.POST("/:key/rollback", d.RollbackAppEnvElement)
				elt.POST("/:key/publish", d.PublishAppEnvElement)
				//elt.GET("/:key/operations", d.GetAppEnvElementOperations)
			}
		}
	}

	agentIns := g.Group("/agents")
	{
		agentIns.GET("", d.GetAgents)
	}

	instances := g.Group("/instances")
	{
		instances.GET("/:insId", d.GetInstance)
		instances.GET("", d.GetElementInstance)
	}
}
