package http

import (
	"github.com/gin-gonic/gin"
)

type getKVReq struct {
	Key string `form:"key"`
}

func (srv *Server) GetKV(c *gin.Context) {
	req := new(getKVReq)
	if err := c.ShouldBindUri(req); err != nil {
		responseError(c, err)
		return
	}

	if err := c.ShouldBind(req); err != nil {
		responseError(c, err)
		return
	}

	out, err := srv.coord.GetKV(req.Key)
	if err != nil {
		responseError(c, err)
		return
	}

	responseJSON(c, out)
}

type createKVReq struct {
	Key   string `json:"key"`
	Value []byte `json:"value"`
}

func (srv *Server) SetKV(c *gin.Context) {
	if srv.needForwardAndExecute(c) {
		return
	}

	req := new(createKVReq)
	if err := c.ShouldBind(req); err != nil {
		responseError(c, err)
		return
	}

	err := srv.coord.SetKV(req.Key, req.Value)
	if err != nil {
		responseError(c, err)
		return
	}

	responseJSON(c, nil)
}

type deleteKVReq struct {
	Key string `form:"key"`
}

func (srv *Server) DeleteKV(c *gin.Context) {
	if srv.needForwardAndExecute(c) {
		return
	}

	req := new(deleteKVReq)
	if err := c.ShouldBindUri(req); err != nil {
		responseError(c, err)
		return
	}

	err := srv.coord.UnsetKV(req.Key)
	if err != nil {
		responseError(c, err)
		return
	}

	responseJSON(c, nil)
}
