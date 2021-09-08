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

	out, err := d.aggregate.GetApps(c.Request.Context(), "", 10)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, out)
}

func (d app) GetApp(c *gin.Context) {
	req := new(getAppReq)
	_ = c.ShouldBindUri(req)
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
	_ = c.ShouldBindUri(req)
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
	_ = c.ShouldBindUri(req)
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
