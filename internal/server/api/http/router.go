package http

func (srv *Server) mountRaftClusterInternalAPI() {
	// DONE(@yeqown): cluster need authorize too to reject request from cluster outside.
	cluster := srv.engi.Group("/cluster", clusterAuthorizeSimple())
	{
		cluster.GET("/node", srv.OperateNode)
		// TODO(@yeqown): apply to raft cluster
		// cluster.POST("/apply", srv.Apply)
	}
}

func (srv *Server) mountAPI() {
	// TODO(@yeqown) authorize middleware is needed.
	g := srv.engi.Group("/api", authorize())

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
