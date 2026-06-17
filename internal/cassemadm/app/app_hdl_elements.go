package app

import (
	"fmt"

	"github.com/gin-gonic/gin"
	dmp "github.com/sergi/go-diff/diffmatchpatch"
	"github.com/yeqown/log"

	"github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/pkg/errorx"
	"github.com/yeqown/cassem/pkg/httpx"
	"github.com/yeqown/cassem/pkg/runtime"
)

func (d app) GetAppEnvElements(c *gin.Context) {
	req := new(getAppEnvElementsReq)
	if err := c.ShouldBindUri(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	var (
		elements any
		err      error
	)

	if len(req.ElementKeys) != 0 {
		elements, err = d.aggregate.GetElementsByKeys(c.Request.Context(), req.AppId, req.Env, req.ElementKeys)
	} else {
		elements, err = d.aggregate.GetElements(c.Request.Context(), req.AppId, req.Env, req.Seek, req.Limit, req.Query)
	}

	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, elements)
}

func (d app) GetAppEnvElement(c *gin.Context) {
	req := new(getAppEnvElementReq)

	if err := c.ShouldBindUri(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}
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

func (d app) CreateAppEnvElement(c *gin.Context) {
	req := new(createAppEnvElementReq)
	if err := c.ShouldBindUri(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}
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
	if err := c.ShouldBindUri(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}
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
	if err := c.ShouldBindUri(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}
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

func (d app) GetAppEnvElementAllVersions(c *gin.Context) {
	req := new(getAppEnvElementVersionsReq)

	if err := c.ShouldBindUri(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}
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

// DiffAppEnvElement diff between element's versions
func (d app) DiffAppEnvElement(c *gin.Context) {
	req := new(diffAppEnvElementsReq)
	if err := c.ShouldBindUri(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}
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

	return _dmp.DiffPrettyText(diffs)
}

func (d app) GetAppEnvElementOperations(c *gin.Context) {
	req := new(getAppEnvElementOperationsReq)
	if err := c.ShouldBindUri(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	operations, err := d.aggregate.GetElementOperations(
		c.Request.Context(), req.AppId, req.Env, req.ElementKey, req.Seek, req.Limit)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, operations)
}

func (d app) RollbackAppEnvElement(c *gin.Context) {
	req := new(rollbackAppEnvElementReq)
	if err := c.ShouldBindUri(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	err := d.aggregate.
		RollbackElementVersion(c.Request.Context(), req.AppId, req.Env, req.ElementKey, req.RollbackTo)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, nil)
}

func (d app) PublishAppEnvElement(c *gin.Context) {
	req := new(publishAppEnvElementReq)
	if err := c.ShouldBindUri(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	// DONE(@yeqown): trigger dispatch to agents.
	elem, err := d.aggregate.
		PublishElementVersion(
			c.Request.Context(), req.AppId, req.Env, req.ElementKey, req.Publish)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	if elem == nil {
		// if no element needs to notify, just return.
		httpx.ResponseJSON(c, nil)
		return
	}

	// call d.agents (agentPool) to notify agents by PublishMode and instancesIds.
	switch req.PublishMode {
	case concept.PublishMode_FULL:
		err = d.ap.notifyAll(elem)
	case concept.PublishMode_GRAY:
		if len(req.AgentIds) == 0 && len(req.InstanceIds) == 0 {
			httpx.ResponseError(c, errorx.Err_INVALID_ARGUMENT)
			return
		}
		if len(req.AgentIds) == 0 {
			err = d.ap.notifyAgentInstances(elem, req.InstanceIds, d.ap.getAgentIdKeys()...)
		} else {
			err = d.ap.notifyAgentInstances(elem, req.InstanceIds, req.AgentIds...)
		}
	}
	if err != nil {
		log.
			WithFields(log.Fields{
				"req":   req,
				"error": err,
			}).
			Error("cassemadm.app.PublishElementVersion failed to dispatch to ap")
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, nil)
}
