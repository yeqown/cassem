package http

import (
	"github.com/gin-gonic/gin"
)

type getContainerReq struct {
	Key       string `uri:"key" binding:"required"`
	Namespace string `uri:"ns" binding:"required"`
}

func (srv *Server) GetContainer(c *gin.Context) {
	req := new(getContainerReq)
	if err := c.ShouldBindUri(req); err != nil {
		responseError(c, err)
		return
	}

	if err := c.ShouldBind(req); err != nil {
		responseError(c, err)
		return
	}

	container, err := srv.coordinator.GetContainer(req.Key, req.Namespace)
	if err != nil {
		responseError(c, err)
		return
	}

	responseData(c, container)
}

// TODO(@yeqown): get container to file
func (srv *Server) ContainerToFile(c *gin.Context) {

}
