package app

import (
	"github.com/gin-gonic/gin"

	"github.com/yeqown/cassem/pkg/httpx"
)

func (d app) GetAppEnvironments(c *gin.Context) {
	req := new(getAppEnvsReq)
	_ = c.ShouldBindUri(req)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	out, err := d.aggregate.GetEnvironments(c.Request.Context(), req.App, req.Seek, req.Limit)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, out)
}

func (d app) CreateAppEnvironment(c *gin.Context) {
	req := new(createAppEnvReq)
	if err := c.ShouldBindUri(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	err := d.aggregate.CreateEnvironment(c.Request.Context(), req.AppId, req.Env)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, nil)
}

func (d app) DeleteAppEnvironment(c *gin.Context) {
	req := new(deleteAppEnvReq)
	if err := c.ShouldBindUri(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	err := d.aggregate.DeleteEnvironment(c.Request.Context(), req.AppId, req.Env)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, nil)
}
