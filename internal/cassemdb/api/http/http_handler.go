package http

import (
	"time"

	"github.com/yeqown/log"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/yeqown/cassem/pkg/httpx"
	"github.com/yeqown/cassem/pkg/runtime"
	"github.com/yeqown/cassem/pkg/types"
	"github.com/yeqown/cassem/pkg/watcher"
)

type operateNodeReq struct {
	ServerID string `form:"serverId" binding:"required"`
	Bind     string `form:"bind"`
	Action   string `form:"action" binding:"required,oneof=join left"`
}

func (srv *httpServer) OperateNode(c *gin.Context) {
	if srv.needForwardAndExecute(c) {
		return
	}

	req := new(operateNodeReq)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
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
		log.
			WithFields(log.Fields{
				"form":  req,
				"error": err,
			}).
			Error("httpServer.OperateNode failed")

		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, nil)
}

type applyReq struct {
	Data []byte `json:"Data" binding:"required"`
}

func (srv *httpServer) Apply(c *gin.Context) {
	req := new(applyReq)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	if err := srv.coord.Apply(req.Data); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, nil)
}

type getKVReq struct {
	Key string `form:"key" binding:"required"`
}

type storeVO struct {
	Fingerprint string `json:"fingerprint"`
	Key         string `json:"key"`
	Val         string `json:"val"`
	Size        int64  `json:"size"`
	CreatedAt   int64  `json:"createdAt"`
	UpdatedAt   int64  `json:"updatedAt"`
}

func newStoreVO(v *types.StoreValue) *storeVO {
	if v == nil {
		return nil
	}

	return &storeVO{
		Fingerprint: v.Fingerprint,
		Key:         v.Key.String(),
		Val:         runtime.ToString(v.Val),
		Size:        v.Size,
		CreatedAt:   v.CreatedAt,
		UpdatedAt:   v.UpdatedAt,
	}
}

func (srv *httpServer) GetKV(c *gin.Context) {
	// TODO(@yeqown): pool getKVReq object
	req := new(getKVReq)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	out, err := srv.coord.GetKV(req.Key)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, newStoreVO(out))
}

type setKVReq struct {
	Key   string `json:"key" binding:"required"`
	Value []byte `json:"value" binding:"required"`
}

func (srv *httpServer) SetKV(c *gin.Context) {
	if srv.needForwardAndExecute(c) {
		return
	}

	req := new(setKVReq)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	err := srv.coord.SetKV(req.Key, req.Value)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, nil)
}

type deleteKVReq struct {
	Key   string `form:"key" binding:"required"`
	IsDir bool   `form:"isDir"`
}

func (srv *httpServer) DeleteKV(c *gin.Context) {
	if srv.needForwardAndExecute(c) {
		return
	}

	req := new(deleteKVReq)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	err := srv.coord.UnsetKV(req.Key, req.IsDir)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, nil)
}

type watchKVReq struct {
	Keys []string `form:"key" binding:"required"`
}

var watchTimeout = map[string]string{
	"messages": "watch timeout",
}

// Watch
func (srv *httpServer) Watch(c *gin.Context) {
	//if srv.needForwardAndExecute(c) {
	//	return
	//}

	req := new(watchKVReq)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	ob, cancel := srv.coord.Watch(req.Keys...)
	defer cancel()

	var change watcher.IChange
	select {
	case change = <-ob.Outbound():
		log.
			WithFields(log.Fields{
				"keys":   req.Keys,
				"change": change,
			}).
			Info("httpServer.Watch got a change")
	case <-time.NewTimer(30 * time.Second).C:
		log.Debugf("httpServer.Watch timeout")
		httpx.ResponseJSON(c, watchTimeout)
		return
	}

	httpx.ResponseJSON(c, change)
}
