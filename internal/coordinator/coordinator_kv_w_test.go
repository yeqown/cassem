package coordinator

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/yeqown/cassem/api/concept"
	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
)

type kvWriteOnlyTestKV struct {
	unset []*apicassemdb.UnsetKVReq
}

func (f *kvWriteOnlyTestKV) GetKV(context.Context, *apicassemdb.GetKVReq, ...grpc.CallOption) (*apicassemdb.GetKVResp, error) {
	return nil, errors.New("unused")
}

func (f *kvWriteOnlyTestKV) GetKVs(context.Context, *apicassemdb.GetKVsReq, ...grpc.CallOption) (*apicassemdb.GetKVsResp, error) {
	return nil, errors.New("unused")
}

func (f *kvWriteOnlyTestKV) SetKV(context.Context, *apicassemdb.SetKVReq, ...grpc.CallOption) (*apicassemdb.Empty, error) {
	return nil, errors.New("unused")
}

func (f *kvWriteOnlyTestKV) UnsetKV(_ context.Context, req *apicassemdb.UnsetKVReq, _ ...grpc.CallOption) (*apicassemdb.Empty, error) {
	f.unset = append(f.unset, req)
	return &apicassemdb.Empty{}, nil
}

func (f *kvWriteOnlyTestKV) Watch(context.Context, *apicassemdb.WatchReq, ...grpc.CallOption) (apicassemdb.KV_WatchClient, error) {
	return nil, errors.New("unused")
}

func (f *kvWriteOnlyTestKV) TTL(context.Context, *apicassemdb.TtlReq, ...grpc.CallOption) (*apicassemdb.TtlResp, error) {
	return nil, errors.New("unused")
}

func (f *kvWriteOnlyTestKV) Expire(context.Context, *apicassemdb.ExpireReq, ...grpc.CallOption) (*apicassemdb.Empty, error) {
	return nil, errors.New("unused")
}

func (f *kvWriteOnlyTestKV) Range(context.Context, *apicassemdb.RangeReq, ...grpc.CallOption) (*apicassemdb.RangeResp, error) {
	return nil, errors.New("unused")
}

func (f *kvWriteOnlyTestKV) CompactElementHistory(context.Context, *apicassemdb.CompactElementHistoryReq, ...grpc.CallOption) (*apicassemdb.CompactElementHistoryResp, error) {
	return nil, errors.New("unused")
}

func TestKVWriteOnlyDeleteElementDeletesOperationPrefix(t *testing.T) {
	kv := &kvWriteOnlyTestKV{}
	err := (kvWriteOnly{cassemdb: kv}).DeleteElement(context.Background(), "app", "env", "key")
	require.NoError(t, err)
	require.Equal(t, []*apicassemdb.UnsetKVReq{
		{Key: concept.GenElementKey("app", "env", "key"), IsDir: true},
		{Key: concept.GenElementOperationKeyPrefix("app", "env", "key"), IsDir: true},
	}, kv.unset)
}

func TestKVWriteOnlyDeleteEnvironmentDeletesOperationPrefix(t *testing.T) {
	kv := &kvWriteOnlyTestKV{}
	err := (kvWriteOnly{cassemdb: kv}).DeleteEnvironment(context.Background(), "app", "env")
	require.NoError(t, err)
	require.Equal(t, []*apicassemdb.UnsetKVReq{
		{Key: concept.GenAppElementEnvKey("app", "env"), IsDir: true},
		{Key: concept.GenAppEnvOperationKey("app", "env"), IsDir: true},
	}, kv.unset)
}

func TestKVWriteOnlyDeleteAppDeletesOperationPrefix(t *testing.T) {
	kv := &kvWriteOnlyTestKV{}
	err := (kvWriteOnly{cassemdb: kv}).DeleteApp(context.Background(), "app")
	require.NoError(t, err)
	require.Equal(t, []*apicassemdb.UnsetKVReq{
		{Key: concept.GenAppElementKey("app"), IsDir: true},
		{Key: concept.GenAppKey("app"), IsDir: false},
		{Key: concept.GenAppOperationKey("app"), IsDir: true},
	}, kv.unset)
}
