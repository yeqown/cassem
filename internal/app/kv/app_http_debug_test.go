package kv

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	apikv "github.com/yeqown/cassem/api/kv"
	"github.com/yeqown/cassem/pkg/httpx"
	"github.com/yeqown/cassem/pkg/watcher"
)

type debugHTTPFakeCoord struct {
	getKey string
	set    *setKVParam
	unset  *unsetKVParam
	rangeP *rangeParam
}

func (f *debugHTTPFakeCoord) getKV(key string) (*apikv.Entity, error) {
	f.getKey = key
	return &apikv.Entity{Key: key, Val: []byte("value")}, nil
}

func (f *debugHTTPFakeCoord) setKV(_ context.Context, p *setKVParam) error {
	f.set = p
	return nil
}

func (f *debugHTTPFakeCoord) unsetKV(_ context.Context, p *unsetKVParam) error {
	f.unset = p
	return nil
}

func (f *debugHTTPFakeCoord) watch(keys ...string) (watcher.IObserver, func()) {
	return debugHTTPObserver{out: make(chan watcher.IChange)}, func() {}
}

func (f *debugHTTPFakeCoord) ttl(string) (int32, error) { return 0, nil }
func (f *debugHTTPFakeCoord) expire(string) error       { return nil }

func (f *debugHTTPFakeCoord) iterate(p *rangeParam) (*apikv.RangeResp, error) {
	f.rangeP = p
	return &apikv.RangeResp{Entities: []*apikv.Entity{{Key: p.key}}}, nil
}

func (f *debugHTTPFakeCoord) compactElementHistory(*apikv.CompactElementHistoryReq) (*apikv.CompactElementHistoryResp, error) {
	return nil, nil
}

func (f *debugHTTPFakeCoord) addNode(context.Context, string, string) (uint64, []string, error) {
	return 0, nil, nil
}

func (f *debugHTTPFakeCoord) removeNode(context.Context, uint64) error { return nil }
func (f *debugHTTPFakeCoord) listMembers() ([]*apikv.ClusterMember, error) {
	return nil, nil
}

type debugHTTPObserver struct {
	out chan watcher.IChange
}

func (o debugHTTPObserver) Identity() string                 { return "debug" }
func (o debugHTTPObserver) Topics() []string                 { return nil }
func (o debugHTTPObserver) Inbound() chan<- watcher.IChange  { return nil }
func (o debugHTTPObserver) Outbound() <-chan watcher.IChange { return o.out }
func (o debugHTTPObserver) Close()                           {}

func decodeDebugHTTPResponse(t *testing.T, w *httptest.ResponseRecorder) httpx.CommonResponse {
	t.Helper()
	var out httpx.CommonResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	return out
}

func TestDebugHTTPRouterGetKV(t *testing.T) {
	coord := &debugHTTPFakeCoord{}
	r := newDebugHTTPRouter(coord)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/kv?key=config/db", nil))

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "config/db", coord.getKey)
	require.Equal(t, httpx.OK, decodeDebugHTTPResponse(t, w).ErrCode)
}

func TestDebugHTTPRouterSetKV(t *testing.T) {
	coord := &debugHTTPFakeCoord{}
	r := newDebugHTTPRouter(coord)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/kv", strings.NewReader(`{"key":"config/db","value":"dmFsdWU=","isDir":true,"overwrite":true,"ttl":30}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, coord.set)
	require.Equal(t, "config/db", coord.set.key)
	require.Equal(t, []byte("value"), coord.set.val)
	require.True(t, coord.set.isDir)
	require.True(t, coord.set.overwrite)
	require.Equal(t, int32(30), coord.set.ttl)
}

func TestDebugHTTPRouterDeleteKV(t *testing.T) {
	coord := &debugHTTPFakeCoord{}
	r := newDebugHTTPRouter(coord)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/kv?key=config/db&isDir=true", nil))

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, coord.unset)
	require.Equal(t, "config/db", coord.unset.key)
	require.True(t, coord.unset.isDir)
}

func TestDebugHTTPRouterRangeKVUsesDefaultLimit(t *testing.T) {
	coord := &debugHTTPFakeCoord{}
	r := newDebugHTTPRouter(coord)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/kv/range?key=config&seek=db", nil))

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, coord.rangeP)
	require.Equal(t, "config", coord.rangeP.key)
	require.Equal(t, "db", coord.rangeP.seek)
	require.Equal(t, 10, coord.rangeP.limit)
}
