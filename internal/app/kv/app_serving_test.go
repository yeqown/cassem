package kv

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	errorx "github.com/yeqown/cassem/api/concept"
	apikv "github.com/yeqown/cassem/api/kv"
	"github.com/yeqown/cassem/internal/app/kv/raftimpl"
	"github.com/yeqown/cassem/pkg/conf"
)

var _ raftimpl.RaftNode = (*servingAPIFakeRaft)(nil)

type servingAPIFakeRaft struct {
	changeCh chan errorx.Change
}

func newServingAPIFakeRaft() *servingAPIFakeRaft {
	return &servingAPIFakeRaft{changeCh: make(chan errorx.Change)}
}

func (f *servingAPIFakeRaft) GetKV(getReq *apikv.GetKVReq) (*apikv.Entity, error) {
	return &apikv.Entity{Key: getReq.GetKey(), Val: []byte("value")}, nil
}

func (f *servingAPIFakeRaft) SetKV(context.Context, *apikv.SetKVReq) error {
	return errors.New("unused")
}
func (f *servingAPIFakeRaft) UnsetKV(context.Context, *apikv.UnsetKVReq) error {
	return errors.New("unused")
}
func (f *servingAPIFakeRaft) Range(*apikv.RangeReq) (*apikv.RangeResp, error) {
	return nil, errors.New("unused")
}
func (f *servingAPIFakeRaft) Expire(*apikv.ExpireReq) error { return errors.New("unused") }
func (f *servingAPIFakeRaft) IsLeader() bool                { return true }
func (f *servingAPIFakeRaft) NodeID() uint64                { return 1 }
func (f *servingAPIFakeRaft) RaftAddr() string              { return "127.0.0.1:17001" }
func (f *servingAPIFakeRaft) Peers() []string               { return []string{"127.0.0.1:17001"} }
func (f *servingAPIFakeRaft) LeaderID() uint64              { return 1 }
func (f *servingAPIFakeRaft) LeaderChangeCh(chan<- bool)    {}
func (f *servingAPIFakeRaft) ChangeNotifyCh() <-chan errorx.Change {
	return f.changeCh
}
func (f *servingAPIFakeRaft) AddNode(context.Context, string) (uint64, []string, error) {
	return 0, nil, errors.New("unused")
}
func (f *servingAPIFakeRaft) RemoveNode(context.Context, uint64) error { return errors.New("unused") }
func (f *servingAPIFakeRaft) Shutdown() error {
	close(f.changeCh)
	return nil
}

func TestServingAPIInDebugModeDoesNotExposeKVHTTPRoutes(t *testing.T) {
	t.Setenv("DEBUG", "1")
	addr := testServingAPIListenAddr(t)
	d := &app{
		config:  &conf.CassemdbConfig{ListenAddr: addr},
		watcher: newChannelWatcher(1),
		raft:    newServingAPIFakeRaft(),
	}

	go func() {
		_ = d.servingAPI()
	}()

	waitForServingAPIPort(t, addr)

	resp, err := http.Get(fmt.Sprintf("http://%s/api/kv?key=config/db", addr))
	if err == nil {
		defer resp.Body.Close()
		require.NotEqual(t, http.StatusOK, resp.StatusCode)
	}
}

func TestServingAPIInDebugModeDoesNotExposePprof(t *testing.T) {
	t.Setenv("DEBUG", "1")
	addr := testServingAPIListenAddr(t)
	d := &app{
		config:  &conf.CassemdbConfig{ListenAddr: addr},
		watcher: newChannelWatcher(1),
		raft:    newServingAPIFakeRaft(),
	}

	go func() {
		_ = d.servingAPI()
	}()

	waitForServingAPIPort(t, addr)

	resp, err := http.Get(fmt.Sprintf("http://%s/debug/pprof/", addr))
	if err == nil {
		defer resp.Body.Close()
		require.NotEqual(t, http.StatusOK, resp.StatusCode)
	}
}

func testServingAPIListenAddr(t *testing.T) string {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := lis.Addr().String()
	require.NoError(t, lis.Close())
	return addr
}

func waitForServingAPIPort(t *testing.T, addr string) {
	t.Helper()
	require.Eventually(t, func() bool {
		conn, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, time.Second, 10*time.Millisecond)
}
