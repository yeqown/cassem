package http

import "github.com/gin-gonic/gin"

func (srv *Server) mountAPI(engi *gin.Engine) {
	g := engi.Group("/api")

	ns := g.Group("/namespaces")
	{
		ns.GET("/", nil)
		ns.POST("/:ns", nil)
		// ns.DELETE("/:ns", nil)
	}

	container := ns.Group("/:ns/containers")
	{
		container.GET("/", nil)
		container.POST("/:key", nil)
		container.GET("/:key", nil)
		container.DELETE("/:key", nil)
	}

	pair := ns.Group("/:ns/pairs")
	{
		pair.GET("/", nil)
		pair.POST("/:key", nil)
		pair.GET("/:key", nil)
		pair.DELETE("/:key", nil)
	}
}
