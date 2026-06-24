package kv

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/yeqown/log"

	"github.com/yeqown/cassem/pkg/httpx"
	"github.com/yeqown/cassem/pkg/watcher"
)

// httpServer provides both RESTFul API for client, ONLY FOR debug.
type httpServer struct {
	handler http.Handler
	coord   ICoordinator
}

func debugHTTP(coord ICoordinator) *httpServer {
	srv := &httpServer{coord: coord}
	srv.initialize()
	return srv
}

func newDebugHTTPRouter(coord ICoordinator) chi.Router {
	srv := &httpServer{coord: coord}
	r := chi.NewRouter()
	r.Use(httpx.RecoveryHTTP)
	r.Use(httpx.LoggerHTTP)
	if isDebug() {
		r.Mount("/debug/pprof", middleware.Profiler())
	}
	srv.mountAPI(r)
	return r
}

func (srv *httpServer) initialize() {
	srv.handler = newDebugHTTPRouter(srv.coord)
}

func (srv *httpServer) mountAPI(r chi.Router) {
	r.Get("/api/kv", srv.GetKV)
	r.Post("/api/kv", srv.SetKV)
	r.Delete("/api/kv", srv.DeleteKV)
	r.Get("/api/kv/watch", srv.Watch)
	r.Get("/api/kv/range", srv.Range)
}

func (srv *httpServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	srv.handler.ServeHTTP(w, req)
}

type getKVReq struct {
	Key string `form:"key" binding:"required"`
}

func (srv *httpServer) GetKV(w http.ResponseWriter, r *http.Request) {
	req := &getKVReq{Key: r.URL.Query().Get("key")}
	if req.Key == "" {
		httpx.WriteError(w, errors.New("Key is required"))
		return
	}

	out, err := srv.coord.getKV(req.Key)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, out)
}

type setKVReq struct {
	Key       string `json:"key" binding:"required"`
	Value     []byte `json:"value" binding:"required"`
	IsDir     bool   `json:"isDir"`
	Overwrite bool   `json:"overwrite"`
	TTL       int32  `json:"ttl"`
}

func (srv *httpServer) SetKV(w http.ResponseWriter, r *http.Request) {
	req := new(setKVReq)
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if req.Key == "" || len(req.Value) == 0 {
		httpx.WriteError(w, errors.New("Key and Value are required"))
		return
	}

	err := srv.coord.setKV(r.Context(), &setKVParam{
		key:       req.Key,
		val:       req.Value,
		isDir:     req.IsDir,
		overwrite: req.Overwrite,
		ttl:       req.TTL,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, nil)
}

type deleteKVReq struct {
	Key   string `form:"key" binding:"required"`
	IsDir bool   `form:"isDir"`
}

func (srv *httpServer) DeleteKV(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	req := &deleteKVReq{Key: q.Get("key"), IsDir: parseBool(q.Get("isDir"))}
	if req.Key == "" {
		httpx.WriteError(w, errors.New("Key is required"))
		return
	}

	err := srv.coord.unsetKV(r.Context(), &unsetKVParam{
		key:   req.Key,
		isDir: req.IsDir,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, nil)
}

type watchKVReq struct {
	Keys []string `form:"key" binding:"required"`
}

// Watch waits for one change or returns nil after timeout.
func (srv *httpServer) Watch(w http.ResponseWriter, r *http.Request) {
	req := &watchKVReq{Keys: r.URL.Query()["key"]}
	if len(req.Keys) == 0 {
		httpx.WriteError(w, errors.New("Key is required"))
		return
	}

	ob, cancel := srv.coord.watch(req.Keys...)
	defer cancel()

	var change watcher.IChange
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	select {
	case change = <-ob.Outbound():
		log.
			WithFields(log.Fields{
				"keys":   req.Keys,
				"change": change,
			}).
			Info("httpServer.Watch got a change")
	case <-timer.C:
		log.Debugf("httpServer.Watch timeout")
	}

	httpx.WriteJSON(w, change)
}

type rangeReq struct {
	Key   string `form:"key" binding:"required"`
	Seek  string `form:"seek"`
	Limit int    `form:"limit,default=10" binding:"gte=1"`
}

func (srv *httpServer) Range(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 10
	if raw := q.Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			httpx.WriteError(w, errors.New("Limit must be greater than or equal to 1"))
			return
		}
		limit = parsed
	}
	req := &rangeReq{Key: q.Get("key"), Seek: q.Get("seek"), Limit: limit}
	if req.Key == "" {
		httpx.WriteError(w, errors.New("Key is required"))
		return
	}

	result, err := srv.coord.iterate(&rangeParam{
		key:   req.Key,
		seek:  req.Seek,
		limit: req.Limit,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, result)
}

func parseBool(raw string) bool {
	parsed, _ := strconv.ParseBool(raw)
	return parsed
}
