package http

import (
	"github.com/gin-gonic/gin"
)

func (srv *Server) mountAPI(engi *gin.Engine) {
	g := engi.Group("/api")

	ns := g.Group("/namespaces")
	{
		ns.GET("", srv.PagingNamespace)
		ns.POST("/:ns", srv.CreateNamespace)
		// ns.DELETE("/:ns", nil)
	}

	container := ns.Group("/:ns/containers")
	{
		container.GET("", srv.PagingContainers)
		container.POST("/:key", srv.UpsertContainer)
		container.GET("/:key", srv.GetContainer)
		container.GET("/:key/dl", srv.ContainerDownload)
		container.DELETE("/:key", srv.RemoveContainer)
	}

	pair := ns.Group("/:ns/pairs")
	{
		pair.GET("", srv.PagingPairs)
		pair.POST("/:key", srv.UpsertPair)
		pair.GET("/:key", srv.GetPair)
		//pair.DELETE("/:key", nil)
	}
}
