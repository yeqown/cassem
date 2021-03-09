package http

func (srv *Server) mountRaftClusterInternalAPI() {
	// DONE(@yeqown): cluster need authorize too to reject request from cluster outside.
	cluster := srv.engi.Group("/cluster", clusterAuthorizeSimple())
	{
		cluster.GET("/nodes", srv.OperateNode)
		cluster.POST("/apply", srv.Apply)
	}
}

func (srv *Server) mountAPI() {
	// DONE(@yeqown) authorize middleware is needed.
	gPub := srv.engi.Group("/api")
	g := srv.engi.Group("/api", authorize(srv.auth))

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

	gPub.POST("/login", srv.Login)
	user := g.Group("/users")
	{
		user.GET("", srv.PagingUsers)
		user.POST("/new", srv.CreateUser)
		user.PUT("/reset-password", srv.ResetPassword)
	}

	userPolicy := user.Group("/:userid/policies")
	{
		userPolicy.GET("", srv.GetUserPolicies)
		userPolicy.POST("/policy", srv.UpdateUserPolicies)
	}
}
