package app

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"

	"github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/internal/cassemadm/infras"
	"github.com/yeqown/cassem/internal/coordinator"
	"github.com/yeqown/cassem/pkg/conf"
	"github.com/yeqown/cassem/pkg/httpx"
)

func isDebug() bool {
	v := os.Getenv("DEBUG")
	return v == "1" || v == "TRUE" || v == "true"
}

type app struct {
	conf *conf.CassemAdminConfig

	// aggregate is special methods interface customized form adm component which
	// can only be used by cassemadm.app.
	aggregate concept.AdmAggregate

	// ap type agentPool is a pool contains all ap nodes, and agentPool will update
	// agent nodes  automatically.
	ap *agentPool
}

func New(c *conf.CassemAdminConfig) (*app, error) {
	if err := c.Valid(); err != nil {
		return nil, fmt.Errorf("cassemadm.New failed: %w", err)
	}

	agg, err := coordinator.NewAdmAggregate(c.CassemDBEndpoints)
	if err != nil {
		return nil, fmt.Errorf("cassemadm.New: %w", err)
	}
	if err = agg.AutoMigrate(); err != nil {
		return nil, fmt.Errorf("cassemadm.New.AutoMigrate: %w", err)
	}
	if c.Auth.HasBootstrapAdmin() {
		if err = agg.BootstrapAdmin(c.Auth.BootstrapAccount, c.Auth.BootstrapNickname, c.Auth.BootstrapPassword); err != nil {
			return nil, fmt.Errorf("cassemadm.New.BootstrapAdmin: %w", err)
		}
	}

	d := &app{
		aggregate: agg,
		conf:      c,
		ap:        newAgentPool(agg),
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
	if !isDebug() {
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

	if isDebug() {
		pprof.Register(engi, "/debug/pprof")
	}

	if err := mountUI(engi); err != nil {
		log.Fatalf("mount UI: %v", err)
	}

	// mount APIs
	// DONE(@yeqown) authorize middleware is needed.
	public := engi.Group("/api")
	auth := engi.Group("/api", infras.Authorization(d.aggregate, d.conf.Auth.SessionSecret), infras.Authentication(d.aggregate))
	accountp := public.Group("/account")
	{
		accountp.POST("/login", d.UserLogin)
	}

	accounta := auth.Group("/account")
	{
		accounta.GET("/users", d.GetUsers)
		accounta.GET("/users/:account/acl", d.GetUserACL)
		accounta.POST("/add", d.AddUser)
		accounta.GET("/disable", d.DisableUser)
		accounta.GET("/reset", d.ResetUser)
		accounta.POST("/reset", d.ResetUser)
		accounta.GET("/acl/domains", d.GetACLDomainOptions)
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
				elt.GET("/:key/operations", d.GetAppEnvElementOperations)
			}
		}
	}

	cluster := auth.Group("/cluster")
	{
		cluster.GET("/topology", d.GetClusterTopology)

		agentIns := cluster.Group("/agents")
		{
			agentIns.GET("", d.GetAgents)
		}

		instances := cluster.Group("/instances")
		{
			instances.GET("", d.GetInstances)
			instances.GET("/detail/:insId", d.GetInstance)
			instances.GET("/filter", d.GetInstancesByElement)
		}
	}
}
