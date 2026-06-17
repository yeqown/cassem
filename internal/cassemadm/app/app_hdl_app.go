package app

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/pkg/httpx"
)

func (d app) GetApps(c *gin.Context) {
	req := new(pagingAppsReq)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	out, err := d.aggregate.GetApps(c.Request.Context(), req.Seek, req.Limit, req.Query)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, out)
}

func (d app) GetApp(c *gin.Context) {
	req := new(getAppReq)
	if err := c.ShouldBindUri(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	out, err := d.aggregate.GetApp(c.Request.Context(), req.App)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, out)
}

func (d app) CreateApp(c *gin.Context) {
	req := new(createAppReq)
	if err := c.ShouldBindUri(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	md := &concept.AppMetadata{
		Id:          req.App,
		Description: req.Description,
		CreatedAt:   time.Now().Unix(),
		Creator:     "todo(@yeqown)",
		Owner:       "todo(@yeqown)",
	}
	err := d.aggregate.CreateApp(c.Request.Context(), md)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, nil)
}

func (d app) DeleteApp(c *gin.Context) {
	req := new(deleteAppReq)
	if err := c.ShouldBindUri(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	err := d.aggregate.DeleteApp(c.Request.Context(), req.App)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, nil)
}
