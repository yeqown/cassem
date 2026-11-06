package coord

import (
	"context"
	"errors"
	"testing"

	pb "github.com/yeqown/cassem/internal/protogen"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/yeqown/cassem/internal/concept"
)

type kvWriteOnlyTestKV struct {
	unset []*pb.UnsetKVReq
}

func (f *kvWriteOnlyTestKV) GetKV(context.Context, *pb.GetKVReq, ...grpc.CallOption) (*pb.GetKVResp, error) {
	return nil, errors.New("unused")
}

func (f *kvWriteOnlyTestKV) GetKVs(context.Context, *pb.GetKVsReq, ...grpc.CallOption) (*pb.GetKVsResp, error) {
	return nil, errors.New("unused")
}

func (f *kvWriteOnlyTestKV) SetKV(context.Context, *pb.SetKVReq, ...grpc.CallOption) (*pb.Empty, error) {
	return nil, errors.New("unused")
}

func (f *kvWriteOnlyTestKV) UnsetKV(_ context.Context, req *pb.UnsetKVReq, _ ...grpc.CallOption) (*pb.Empty, error) {
	f.unset = append(f.unset, req)
	return &pb.Empty{}, nil
}

func (f *kvWriteOnlyTestKV) Watch(context.Context, *pb.WatchReq, ...grpc.CallOption) (pb.KV_WatchClient, error) {
	return nil, errors.New("unused")
}

func (f *kvWriteOnlyTestKV) TTL(context.Context, *pb.TtlReq, ...grpc.CallOption) (*pb.TtlResp, error) {
	return nil, errors.New("unused")
}

func (f *kvWriteOnlyTestKV) Expire(context.Context, *pb.ExpireReq, ...grpc.CallOption) (*pb.Empty, error) {
	return nil, errors.New("unused")
}

func (f *kvWriteOnlyTestKV) Range(context.Context, *pb.RangeReq, ...grpc.CallOption) (*pb.RangeResp, error) {
	return nil, errors.New("unused")
}

func (f *kvWriteOnlyTestKV) CompactElementHistory(context.Context, *pb.CompactElementHistoryReq, ...grpc.CallOption) (*pb.CompactElementHistoryResp, error) {
	return nil, errors.New("unused")
}

func TestKVWriteOnlySaveRawPreservesMarshalFailure(t *testing.T) {
	err := (kvWriteOnly{cassemkv: &kvWriteOnlyTestKV{}}).saveRaw(
		context.Background(),
		"bad",
		&pb.AppMetadata{Id: string([]byte{0xff})},
		false,
	)
	require.ErrorIs(t, err, concept.Err_INTERNAL)
	require.Contains(t, err.Error(), "invalid UTF-8")
}

func TestKVWriteOnlyDeleteElementDeletesOperationPrefix(t *testing.T) {
	kv := &kvWriteOnlyTestKV{}
	err := (kvWriteOnly{cassemkv: kv}).DeleteElement(context.Background(), "app", "env", "key")
	require.NoError(t, err)
	require.Equal(t, []*pb.UnsetKVReq{
		{Key: concept.GenElementKey("app", "env", "key"), IsDir: true},
		{Key: concept.GenElementOperationKeyPrefix("app", "env", "key"), IsDir: true},
	}, kv.unset)
}

func TestKVWriteOnlyDeleteEnvironmentDeletesOperationPrefix(t *testing.T) {
	kv := &kvWriteOnlyTestKV{}
	err := (kvWriteOnly{cassemkv: kv}).DeleteEnvironment(context.Background(), "app", "env")
	require.NoError(t, err)
	require.Equal(t, []*pb.UnsetKVReq{
		{Key: concept.GenAppElementEnvKey("app", "env"), IsDir: true},
		{Key: concept.GenAppEnvOperationKey("app", "env"), IsDir: true},
	}, kv.unset)
}

func TestKVWriteOnlyDeleteAppDeletesOperationPrefix(t *testing.T) {
	kv := &kvWriteOnlyTestKV{}
	err := (kvWriteOnly{cassemkv: kv}).DeleteApp(context.Background(), "app")
	require.NoError(t, err)
	require.Equal(t, []*pb.UnsetKVReq{
		{Key: concept.GenAppElementKey("app"), IsDir: true},
		{Key: concept.GenAppKey("app"), IsDir: false},
		{Key: concept.GenAppOperationKey("app"), IsDir: true},
	}, kv.unset)
}
