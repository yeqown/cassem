package app

import (
	"fmt"

	"github.com/gin-gonic/gin"
	dmp "github.com/sergi/go-diff/diffmatchpatch"

	"github.com/yeqown/cassem/pkg/httpx"
	"github.com/yeqown/cassem/pkg/runtime"
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
	req := new(getAppEnvElementVersionsReq)

	_ = c.ShouldBindUri(req)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	// TODO(@yeqown): get specified versions of element, if there's not version specified
	// get all version.
	element, err := d.aggregate.GetElementVersions(
		c.Request.Context(), req.AppId, req.Env, req.ElementKey, req.Seek, req.Limit)
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
		req.AppId, req.Env, req.ElementKey, runtime.ToBytes(req.Raw), req.ContentType)
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
		req.AppId, req.Env, req.ElementKey, runtime.ToBytes(req.Raw))
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

// DiffAppEnvElement diff between element's versions
func (d app) DiffAppEnvElement(c *gin.Context) {
	req := new(diffAppEnvElementsReq)
	_ = c.ShouldBindUri(req)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	base, err := d.aggregate.
		GetElementWithVersion(c.Request.Context(), req.AppId, req.Env, req.ElementKey, int(req.Base))
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}
	compare, err := d.aggregate.
		GetElementWithVersion(c.Request.Context(), req.AppId, req.Env, req.ElementKey, int(req.Compare))
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	pretty := diff(runtime.ToString(base.GetRaw()), runtime.ToString(compare.GetRaw()))
	fmt.Println(pretty)
	httpx.ResponseJSON(c, diffAppEnvElementsResp{
		Base:    base,
		Compare: compare,
		Diff:    pretty,
	})
}

func diff(src1, src2 string) string {
	// TODO(@yeqown): object pool for dmp if needed.
	_dmp := dmp.New()
	diffs := _dmp.DiffMain(src1, src2, false)

	// TODO(@yeqown): may customize pretty text string, render in HTML or others format.
	//_dmp.DiffPrettyText()
	return _dmp.DiffText1(diffs)
}
