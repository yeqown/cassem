package http

import (
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

type operateNodeReq struct {
	ServerID string `form:"serverId" binding:"required"`
	Bind     string `form:"bind"`
	Action   string `form:"action" binding:"required,oneof=join left"`
}

func (srv *Server) OperateNode(c *gin.Context) {
	if srv.needForwardAndExecute(c) {
		return
	}

	req := new(operateNodeReq)
	if err := c.ShouldBind(req); err != nil {
		responseError(c, err)
		return
	}

	var err error
	switch req.Action {
	case "join":
		if req.Bind != "" {
			err = srv.coord.AddNode(req.ServerID, req.Bind)
		} else {
			err = errors.New("bind could not be empty")
		}
	case "left":
		err = srv.coord.RemoveNode(req.ServerID)
	default:
		err = errors.New("unknown action")
	}

	if err != nil {
		responseError(c, err)
		return
	}

	responseJSON(c, nil)
}

type applyReq struct {
	Data []byte `json:"Data" binding:"required"`
}

func (srv *Server) Apply(c *gin.Context) {
	req := new(applyReq)
	if err := c.ShouldBind(req); err != nil {
		responseError(c, err)
		return
	}

	if err := srv.coord.Apply(req.Data); err != nil {
		responseError(c, err)
		return
	}

	responseJSON(c, nil)
}
