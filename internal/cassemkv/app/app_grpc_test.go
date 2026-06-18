package app

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yeqown/cassem/pkg/errorx"
	"github.com/yeqown/cassem/pkg/watcher"
)

type grpcGetKVsFakeCoord struct {
	entities map[string]*apikv.Entity
	errors   map[string]error
}

func (f *grpcGetKVsFakeCoord) getKV(key string) (*apikv.Entity, error) {
	if err := f.errors[key]; err != nil {
		return nil, err
	}
	entity, ok := f.entities[key]
	if !ok {
		return nil, errorx.Err_NOT_FOUND
	}
	return entity, nil
}

func (f *grpcGetKVsFakeCoord) setKV(*setKVParam) error                     { return errors.New("unused") }
func (f *grpcGetKVsFakeCoord) unsetKV(*unsetKVParam) error                 { return errors.New("unused") }
func (f *grpcGetKVsFakeCoord) watch(...string) (watcher.IObserver, func()) { return nil, func() {} }
func (f *grpcGetKVsFakeCoord) ttl(string) (int32, error)                   { return 0, errors.New("unused") }
func (f *grpcGetKVsFakeCoord) expire(string) error                         { return errors.New("unused") }
func (f *grpcGetKVsFakeCoord) iterate(*rangeParam) (*apikv.RangeResp, error) {
	return nil, errors.New("unused")
}
func (f *grpcGetKVsFakeCoord) compactElementHistory(*apikv.CompactElementHistoryReq) (*apikv.CompactElementHistoryResp, error) {
	return nil, errors.New("unused")
}
func (f *grpcGetKVsFakeCoord) addNode(string, string) (uint64, []string, error) {
	return 0, nil, errors.New("unused")
}
func (f *grpcGetKVsFakeCoord) removeNode(uint64) error { return errors.New("unused") }
func (f *grpcGetKVsFakeCoord) listMembers() ([]*apikv.ClusterMember, error) {
	return nil, errors.New("unused")
}

func TestGrpcServerGetKVsReturnsPerKeyErrors(t *testing.T) {
	coord := &grpcGetKVsFakeCoord{
		entities: map[string]*apikv.Entity{
			"cassem/key1": {Key: "cassem/key1", Val: []byte("value")},
		},
		errors: map[string]error{
			"cassem/key3": errorx.Err_INTERNAL,
		},
	}

	resp, err := (grpcServer{coord: coord}).GetKVs(context.Background(), &apikv.GetKVsReq{Keys: []string{"cassem/key1", "cassem/key2", "cassem/key3"}})
	require.NoError(t, err)
	require.Len(t, resp.GetEntities(), 1)
	require.Equal(t, "cassem/key1", resp.GetEntities()[0].GetKey())
	require.Len(t, resp.GetErrors(), 2)
	require.Equal(t, "cassem/key2", resp.GetErrors()[0].GetKey())
	require.Equal(t, "NotFound", resp.GetErrors()[0].GetCode())
	require.Equal(t, "NOT_FOUND", resp.GetErrors()[0].GetMessage())
	require.Equal(t, "cassem/key3", resp.GetErrors()[1].GetKey())
	require.Equal(t, "Internal", resp.GetErrors()[1].GetCode())
	require.Equal(t, "INTERNAL", resp.GetErrors()[1].GetMessage())
}
