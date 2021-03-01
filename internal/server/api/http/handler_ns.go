package http

import (
	coord "github.com/yeqown/cassem/internal/coordinator"

	"github.com/gin-gonic/gin"
)

type pagingNamespaceReq struct {
	Limit            int    `form:"limit,default=100"`
	Offset           int    `form:"offset,default=0"`
	NamespacePattern string `form:"key"`
}

func (srv *Server) PagingNamespace(c *gin.Context) {
	req := new(pagingNamespaceReq)
	if err := c.ShouldBindUri(req); err != nil {
		responseError(c, err)
		return
	}

	if err := c.ShouldBind(req); err != nil {
		responseError(c, err)
		return
	}

	out, _, err := srv.coordinator.PagingNamespaces(&coord.FilterNamespacesOption{
		Limit:            req.Limit,
		Offset:           req.Offset,
		NamespacePattern: req.NamespacePattern,
	})
	if err != nil {
		responseError(c, err)
		return
	}

	responseJSON(c, out)
}

type createNamespaceReq struct {
	Namespace string `uri:"ns"`
}

func (srv *Server) CreateNamespace(c *gin.Context) {
	if srv.needForwardAndExecute(c) {
		return
	}

	req := new(createNamespaceReq)
	if err := c.ShouldBindUri(req); err != nil {
		responseError(c, err)
		return
	}

	err := srv.coordinator.SaveNamespace(req.Namespace)
	if err != nil {
		responseError(c, err)
		return
	}

	responseJSON(c, nil)
}
