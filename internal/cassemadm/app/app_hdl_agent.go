package app

import (
	"github.com/gin-gonic/gin"

	"github.com/yeqown/cassem/pkg/httpx"
)

// GetAgents list all agent instances.
func (d app) GetAgents(c *gin.Context) {
	req := new(pagingAgentInstanceReq)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	out, err := d.aggregate.GetAgents(c.Request.Context(), req.Seek, req.Limit)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, out)
}
