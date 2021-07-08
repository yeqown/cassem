package app

import (
	"net/http"
	"time"

	"github.com/yeqown/log"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"

	"github.com/yeqown/cassem/pkg/httpx"
	"github.com/yeqown/cassem/pkg/runtime"
	"github.com/yeqown/cassem/pkg/types"
	"github.com/yeqown/cassem/pkg/watcher"
)

// httpServer provides both RESTFul API for client also provides part of API for internal cluster, all internal APIs
// stay in handler_cluster.go and register in httpServer.mountRaftClusterInternalAPI.
type httpServer struct {
	engi  *gin.Engine
	coord ICoordinator
}

func debugHTTP(coord ICoordinator) *httpServer {
	srv := &httpServer{
		coord: coord,
		engi:  gin.New(),
	}

	srv.initialize()

	return srv
}

func (srv *httpServer) initialize() {
	gin.EnableJsonDecoderUseNumber()
	if !runtime.IsDebug() {
		gin.SetMode(gin.ReleaseMode)
	}

	// mount middlewares
	// DONE(@yeqown): replace Recovery middleware so that we response error messages.
	srv.engi.Use(httpx.Recovery())
	srv.engi.Use(gin.Logger())

	if runtime.IsDebug() {
		pprof.Register(srv.engi, "/debug/pprof")
	}

	// mount API
	srv.mountAPI()
}

func (srv *httpServer) mountAPI() {
	// DONE(@yeqown) authorize middleware is needed.
	g := srv.engi.Group("/api")

	ns := g.Group("/kv")
	{
		ns.GET("", srv.GetKV)
		ns.POST("", srv.SetKV)
		ns.DELETE("", srv.DeleteKV)

		ns.GET("/watch", srv.Watch)
	}
}

func (srv *httpServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	srv.engi.ServeHTTP(w, req)
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
	TTL         uint32 `json:"ttl"`
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
		TTL:         v.TTL,
	}
}

func (srv *httpServer) GetKV(c *gin.Context) {
	req := new(getKVReq)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	out, err := srv.coord.getKV(req.Key)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, newStoreVO(out))
}

type setKVReq struct {
	Key       string `json:"key" binding:"required"`
	Value     []byte `json:"value" binding:"required"`
	IsDir     bool   `json:"isDir"`
	Overwrite bool   `json:"overwrite"`
	TTL       uint32 `json:"ttl"`
}

func (srv *httpServer) SetKV(c *gin.Context) {
	req := new(setKVReq)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	err := srv.coord.setKV(&setKVParam{
		key:       req.Key,
		val:       req.Value,
		isDir:     req.IsDir,
		overwrite: req.Overwrite,
		ttl:       req.TTL,
	})
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
	req := new(deleteKVReq)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	err := srv.coord.unsetKV(&unsetKVParam{
		key:   req.Key,
		isDir: req.IsDir,
	})
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, nil)
}

type watchKVReq struct {
	Keys []string `form:"key" binding:"required"`
}

// Watch
// TODO(@yeqown) all API implemented by grpc
func (srv *httpServer) Watch(c *gin.Context) {
	//if srv.needForwardAndExecute(c) {
	//	return
	//}

	req := new(watchKVReq)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	ob, cancel := srv.coord.watch(req.Keys...)
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
	}

	httpx.ResponseJSON(c, change)
}
