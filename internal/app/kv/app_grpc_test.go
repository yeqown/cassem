package kv

import (
	"context"
	"errors"
	"testing"

	pb "github.com/yeqown/cassem/internal/protogen"

	"github.com/stretchr/testify/require"
	errorx "github.com/yeqown/cassem/internal/concept"
)

type grpcGetKVsFakeCoord struct {
	entities map[string]*pb.Entity
	errors   map[string]error
}

func (f *grpcGetKVsFakeCoord) getKV(key string) (*pb.Entity, error) {
	if err := f.errors[key]; err != nil {
		return nil, err
	}
	entity, ok := f.entities[key]
	if !ok {
		return nil, errorx.Err_NOT_FOUND
	}
	return entity, nil
}

func (f *grpcGetKVsFakeCoord) setKV(context.Context, *setKVParam) error { return errors.New("unused") }
func (f *grpcGetKVsFakeCoord) unsetKV(context.Context, *unsetKVParam) error {
	return errors.New("unused")
}
func (f *grpcGetKVsFakeCoord) watch(...string) (*builtinObserver, func()) { return nil, func() {} }
func (f *grpcGetKVsFakeCoord) ttl(string) (int32, error)                  { return 0, errors.New("unused") }
func (f *grpcGetKVsFakeCoord) expire(string) error                        { return errors.New("unused") }
func (f *grpcGetKVsFakeCoord) iterate(*rangeParam) (*pb.RangeResp, error) {
	return nil, errors.New("unused")
}
func (f *grpcGetKVsFakeCoord) compactElementHistory(*pb.CompactElementHistoryReq) (*pb.CompactElementHistoryResp, error) {
	return nil, errors.New("unused")
}
func (f *grpcGetKVsFakeCoord) addNode(context.Context, string, string) (uint64, []string, error) {
	return 0, nil, errors.New("unused")
}
func (f *grpcGetKVsFakeCoord) removeNode(context.Context, uint64) error { return errors.New("unused") }
func (f *grpcGetKVsFakeCoord) listMembers() ([]*pb.ClusterMember, error) {
	return nil, errors.New("unused")
}

func TestGrpcServerGetKVsReturnsPerKeyErrors(t *testing.T) {
	coord := &grpcGetKVsFakeCoord{
		entities: map[string]*pb.Entity{
			"cassem/key1": {Key: "cassem/key1", Val: []byte("value")},
		},
		errors: map[string]error{
			"cassem/key3": errorx.Err_INTERNAL,
		},
	}

	resp, err := (grpcServer{coord: coord}).GetKVs(context.Background(), &pb.GetKVsReq{Keys: []string{"cassem/key1", "cassem/key2", "cassem/key3"}})
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
