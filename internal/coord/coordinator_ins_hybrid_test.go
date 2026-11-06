package coord

import (
	"context"
	"errors"
	"strings"
	"testing"

	pb "github.com/yeqown/cassem/internal/protogen"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/yeqown/cassem/internal/concept"
)

type instanceHybridTestKV struct {
	entities map[string]*pb.Entity
	setErr   map[string]error
	unsetErr map[string]error
	set      []*pb.SetKVReq
	unset    []*pb.UnsetKVReq
}

func newInstanceHybridTestKV() *instanceHybridTestKV {
	return &instanceHybridTestKV{
		entities: make(map[string]*pb.Entity),
		setErr:   make(map[string]error),
		unsetErr: make(map[string]error),
	}
}

func (f *instanceHybridTestKV) GetKV(_ context.Context, req *pb.GetKVReq, _ ...grpc.CallOption) (*pb.GetKVResp, error) {
	entity, ok := f.entities[req.GetKey()]
	if !ok {
		return nil, concept.Err_NOT_FOUND
	}
	return &pb.GetKVResp{Entity: entity}, nil
}

func (f *instanceHybridTestKV) GetKVs(context.Context, *pb.GetKVsReq, ...grpc.CallOption) (*pb.GetKVsResp, error) {
	return nil, errors.New("unused")
}

func (f *instanceHybridTestKV) SetKV(_ context.Context, req *pb.SetKVReq, _ ...grpc.CallOption) (*pb.Empty, error) {
	f.set = append(f.set, req)
	if err := f.setErr[req.GetKey()]; err != nil {
		return nil, err
	}
	f.entities[req.GetKey()] = &pb.Entity{Key: req.GetKey(), Val: req.GetVal(), Ttl: req.GetTtl()}
	return &pb.Empty{}, nil
}

func (f *instanceHybridTestKV) UnsetKV(_ context.Context, req *pb.UnsetKVReq, _ ...grpc.CallOption) (*pb.Empty, error) {
	f.unset = append(f.unset, req)
	if err := f.unsetErr[req.GetKey()]; err != nil {
		return nil, err
	}
	delete(f.entities, req.GetKey())
	return &pb.Empty{}, nil
}

func (f *instanceHybridTestKV) Watch(context.Context, *pb.WatchReq, ...grpc.CallOption) (pb.KV_WatchClient, error) {
	return nil, errors.New("unused")
}

func (f *instanceHybridTestKV) TTL(context.Context, *pb.TtlReq, ...grpc.CallOption) (*pb.TtlResp, error) {
	return nil, errors.New("unused")
}

func (f *instanceHybridTestKV) Expire(context.Context, *pb.ExpireReq, ...grpc.CallOption) (*pb.Empty, error) {
	return nil, errors.New("unused")
}

func (f *instanceHybridTestKV) Range(context.Context, *pb.RangeReq, ...grpc.CallOption) (*pb.RangeResp, error) {
	return nil, errors.New("unused")
}

func (f *instanceHybridTestKV) CompactElementHistory(context.Context, *pb.CompactElementHistoryReq, ...grpc.CallOption) (*pb.CompactElementHistoryResp, error) {
	return nil, errors.New("unused")
}

func testInstance() *concept.Instance {
	return &concept.Instance{
		ClientId: "client",
		ClientIp: "127.0.0.1",
		Watching: []*concept.Instance_Watching{
			{App: "app", Env: "env", WatchKeys: []string{"key1", "key2"}},
		},
	}
}

func TestInstanceHybridSetInstanceInfoReturnsReversedWriteError(t *testing.T) {
	ins := testInstance()
	kv := newInstanceHybridTestKV()
	failedKey := concept.GenInstanceReversedKeyWithInsId("app", "env", "key2", concept.InstanceID(ins))
	kv.setErr[failedKey] = errors.New("boom")

	err := (instanceHybrid{cassemkv: kv}).setInstanceInfo(context.Background(), ins)
	require.Error(t, err)
	require.Contains(t, err.Error(), failedKey)
	require.Contains(t, err.Error(), "boom")
	require.Len(t, kv.set, 3)
}

func TestInstanceHybridUnregisterInstanceReturnsAllReversedDeleteErrors(t *testing.T) {
	ins := testInstance()
	data, err := concept.MarshalProto(ins)
	require.NoError(t, err)

	kv := newInstanceHybridTestKV()
	kv.entities[concept.GenInstanceNormalKey(concept.InstanceID(ins))] = &pb.Entity{Key: concept.GenInstanceNormalKey(concept.InstanceID(ins)), Val: data}
	firstFailedKey := concept.GenInstanceReversedKeyWithInsId("app", "env", "key1", concept.InstanceID(ins))
	secondFailedKey := concept.GenInstanceReversedKeyWithInsId("app", "env", "key2", concept.InstanceID(ins))
	kv.unsetErr[firstFailedKey] = errors.New("first")
	kv.unsetErr[secondFailedKey] = errors.New("second")

	err = (instanceHybrid{cassemkv: kv}).UnregisterInstance(context.Background(), concept.InstanceID(ins))
	require.Error(t, err)
	require.Contains(t, err.Error(), firstFailedKey)
	require.Contains(t, err.Error(), secondFailedKey)
	require.True(t, strings.Contains(err.Error(), "first"))
	require.True(t, strings.Contains(err.Error(), "second"))
	require.Len(t, kv.unset, 3)
}
