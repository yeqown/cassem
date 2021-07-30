package app

import (
	"github.com/gin-gonic/gin"

	"github.com/yeqown/cassem/pkg/httpx"
)

func (d app) GetInstance(c *gin.Context) {
	req := new(getInstanceReq)
	if err := c.ShouldBindUri(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	out, err := d.aggregate.GetInstance(c.Request.Context(), req.InsId)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, out)
}

func (d app) GetElementInstance(c *gin.Context) {
	req := new(getEleInstancesReq)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	out, err := d.aggregate.GetElementInstances(c.Request.Context(), req.AppId, req.Env, req.ElementKey)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, out)
}
