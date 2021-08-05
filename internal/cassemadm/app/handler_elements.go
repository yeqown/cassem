package app

import (
	"github.com/gin-gonic/gin"

	"github.com/yeqown/cassem/pkg/httpx"
)

func (d app) GetAppEnvElements(c *gin.Context) {
	req := new(getAppEnvElementsReq)
	_ = c.ShouldBindUri(req)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	var (
		elements interface{}
		err      error
	)

	if len(req.ElementKeys) != 0 {
		elements, err = d.aggregate.GetElementsByKeys(c.Request.Context(), req.AppId, req.Env, req.ElementKeys)
	} else {
		elements, err = d.aggregate.GetElements(c.Request.Context(), req.AppId, req.Env, req.Seek, req.Limit)
	}

	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, elements)
}

func (d app) GetAppEnvElement(c *gin.Context) {
	req := new(getAppEnvElementReq)

	_ = c.ShouldBindUri(req)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	element, err := d.aggregate.GetElementWithVersion(
		c.Request.Context(), req.AppId, req.Env, req.ElementKey, int(req.Version))
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, element)
}

func (d app) GetAppEnvElementAllVersions(c *gin.Context) {
	req := new(getAppEnvElementReq)

	_ = c.ShouldBindUri(req)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	// TODO(@yeqown): get all versions to element
	element, err := d.aggregate.GetElementWithVersion(
		c.Request.Context(), req.AppId, req.Env, req.ElementKey, int(req.Version))
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, element)
}

func (d app) CreateAppEnvElement(c *gin.Context) {
	req := new(createAppEnvElementReq)
	_ = c.ShouldBindUri(req)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	err := d.aggregate.CreateElement(c.Request.Context(),
		req.AppId, req.Env, req.ElementKey, req.Raw, req.ContentType)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, nil)
}

func (d app) UpdateAppEnvElement(c *gin.Context) {
	req := new(updateAppEnvElementReq)
	_ = c.ShouldBindUri(req)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	err := d.aggregate.UpdateElement(c.Request.Context(),
		req.AppId, req.Env, req.ElementKey, req.Raw)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, nil)
}

func (d app) DeleteAppEnvElement(c *gin.Context) {
	req := new(deleteAppEnvElementsReq)
	_ = c.ShouldBindUri(req)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	err := d.aggregate.DeleteElement(c.Request.Context(), req.AppId, req.Env, req.ElementKey)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, nil)

}
