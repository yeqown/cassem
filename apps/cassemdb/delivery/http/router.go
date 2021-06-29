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
	g := srv.engi.Group("/api")

	ns := g.Group("/kv")
	{
		ns.GET("", srv.GetKV)
		ns.POST("", srv.SetKV)
		ns.DELETE("", srv.DeleteKV)
	}
}
