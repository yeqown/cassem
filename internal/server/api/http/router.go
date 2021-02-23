package http

import "github.com/gin-gonic/gin"

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
		container.GET("", nil)
		container.POST("/:key", nil)
		container.GET("/:key", srv.GetContainer)
		container.GET("/:key/file", srv.ContainerToFile)
		container.DELETE("/:key", nil)
	}

	pair := ns.Group("/:ns/pairs")
	{
		pair.GET("", srv.PagingPairs)
		pair.POST("/:key", nil)
		pair.GET("/:key", srv.GetPair)
		//pair.DELETE("/:key", nil)
	}
}
