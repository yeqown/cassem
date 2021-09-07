package app

import (
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/yeqown/cassem/concept"
	"github.com/yeqown/cassem/internal/cassemadm/infras"
	"github.com/yeqown/cassem/pkg/conf"
	"github.com/yeqown/cassem/pkg/httpx"
	"github.com/yeqown/cassem/pkg/runtime"
)

type app struct {
	conf *conf.CassemAdminConfig

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
	corsConfig := cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "X-CASSEM-SESSION"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}
	engi.Use(cors.New(corsConfig))

	if runtime.IsDebug() {
		pprof.Register(engi, "/debug/pprof")
	}

	// mount APIs
	// DONE(@yeqown) authorize middleware is needed.
	public := engi.Group("/api")
	auth := engi.Group("/api", infras.Authorization(d.aggregate), infras.Authentication(d.aggregate))
	accountp := public.Group("/account")
	{
		accountp.POST("/login", d.UserLogin)
	}

	accounta := auth.Group("/account")
	{
		// accounta.GET("/users", d.GetUsers)
		accounta.POST("/add", d.AddUser)
		accounta.GET("/disable", d.DisableUser)
		accounta.GET("/reset", d.ResetUser)
		accounta.GET("/acl/assign", d.AssignRole)
		accounta.GET("/acl/revoke", d.RevokeRole)
	}

	apps := auth.Group("/apps")
	{
		apps.GET("", d.GetApps)
		apps.GET("/:appId", d.GetApp)
		apps.POST("/:appId", d.CreateApp)
		apps.DELETE("/:appId", d.DeleteApp)

		envs := apps.Group("/:appId/envs")
		{
			envs.GET("", d.GetAppEnvironments)
			{
				envs.POST("/:env", d.CreateAppEnvironment)
				envs.DELETE("/:env", d.DeleteAppEnvironment)
			}

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

	agentIns := auth.Group("/agents")
	{
		agentIns.GET("", d.GetAgents)
	}

	instances := auth.Group("/instances")
	{
		instances.GET("/:insId", d.GetInstance)
		instances.GET("", d.GetElementInstance)
	}
}
