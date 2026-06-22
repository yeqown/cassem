package coord

import (
	"context"
	"errors"
	apikv "github.com/yeqown/cassem/api/kv"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/pkg/errorx"
)

type kvWriteOnlyTestKV struct {
	unset []*apikv.UnsetKVReq
}

func (f *kvWriteOnlyTestKV) GetKV(context.Context, *apikv.GetKVReq, ...grpc.CallOption) (*apikv.GetKVResp, error) {
	return nil, errors.New("unused")
}

func (f *kvWriteOnlyTestKV) GetKVs(context.Context, *apikv.GetKVsReq, ...grpc.CallOption) (*apikv.GetKVsResp, error) {
	return nil, errors.New("unused")
}

func (f *kvWriteOnlyTestKV) SetKV(context.Context, *apikv.SetKVReq, ...grpc.CallOption) (*apikv.Empty, error) {
	return nil, errors.New("unused")
}

func (f *kvWriteOnlyTestKV) UnsetKV(_ context.Context, req *apikv.UnsetKVReq, _ ...grpc.CallOption) (*apikv.Empty, error) {
	f.unset = append(f.unset, req)
	return &apikv.Empty{}, nil
}

func (f *kvWriteOnlyTestKV) Watch(context.Context, *apikv.WatchReq, ...grpc.CallOption) (apikv.KV_WatchClient, error) {
	return nil, errors.New("unused")
}

func (f *kvWriteOnlyTestKV) TTL(context.Context, *apikv.TtlReq, ...grpc.CallOption) (*apikv.TtlResp, error) {
	return nil, errors.New("unused")
}

func (f *kvWriteOnlyTestKV) Expire(context.Context, *apikv.ExpireReq, ...grpc.CallOption) (*apikv.Empty, error) {
	return nil, errors.New("unused")
}

func (f *kvWriteOnlyTestKV) Range(context.Context, *apikv.RangeReq, ...grpc.CallOption) (*apikv.RangeResp, error) {
	return nil, errors.New("unused")
}

func (f *kvWriteOnlyTestKV) CompactElementHistory(context.Context, *apikv.CompactElementHistoryReq, ...grpc.CallOption) (*apikv.CompactElementHistoryResp, error) {
	return nil, errors.New("unused")
}

func TestKVWriteOnlySaveRawPreservesMarshalFailure(t *testing.T) {
	err := (kvWriteOnly{cassemdb: &kvWriteOnlyTestKV{}}).saveRaw(
		context.Background(),
		"bad",
		&concept.AppMetadata{Id: string([]byte{0xff})},
		0,
		false,
	)
	require.ErrorIs(t, err, errorx.Err_INTERNAL)
	require.Contains(t, err.Error(), "invalid UTF-8")
}

func TestKVWriteOnlyDeleteElementDeletesOperationPrefix(t *testing.T) {
	kv := &kvWriteOnlyTestKV{}
	err := (kvWriteOnly{cassemdb: kv}).DeleteElement(context.Background(), "app", "env", "key")
	require.NoError(t, err)
	require.Equal(t, []*apikv.UnsetKVReq{
		{Key: concept.GenElementKey("app", "env", "key"), IsDir: true},
		{Key: concept.GenElementOperationKeyPrefix("app", "env", "key"), IsDir: true},
	}, kv.unset)
}

func TestKVWriteOnlyDeleteEnvironmentDeletesOperationPrefix(t *testing.T) {
	kv := &kvWriteOnlyTestKV{}
	err := (kvWriteOnly{cassemdb: kv}).DeleteEnvironment(context.Background(), "app", "env")
	require.NoError(t, err)
	require.Equal(t, []*apikv.UnsetKVReq{
		{Key: concept.GenAppElementEnvKey("app", "env"), IsDir: true},
		{Key: concept.GenAppEnvOperationKey("app", "env"), IsDir: true},
	}, kv.unset)
}

func TestKVWriteOnlyDeleteAppDeletesOperationPrefix(t *testing.T) {
	kv := &kvWriteOnlyTestKV{}
	err := (kvWriteOnly{cassemdb: kv}).DeleteApp(context.Background(), "app")
	require.NoError(t, err)
	require.Equal(t, []*apikv.UnsetKVReq{
		{Key: concept.GenAppElementKey("app"), IsDir: true},
		{Key: concept.GenAppKey("app"), IsDir: false},
		{Key: concept.GenAppOperationKey("app"), IsDir: true},
	}, kv.unset)
}
