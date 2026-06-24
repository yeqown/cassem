package coord

import (
	"context"
	"errors"
	apikv "github.com/yeqown/cassem/api/kv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/yeqown/cassem/api/concept"
)

type instanceHybridTestKV struct {
	entities map[string]*apikv.Entity
	setErr   map[string]error
	unsetErr map[string]error
	set      []*apikv.SetKVReq
	unset    []*apikv.UnsetKVReq
}

func newInstanceHybridTestKV() *instanceHybridTestKV {
	return &instanceHybridTestKV{
		entities: make(map[string]*apikv.Entity),
		setErr:   make(map[string]error),
		unsetErr: make(map[string]error),
	}
}

func (f *instanceHybridTestKV) GetKV(_ context.Context, req *apikv.GetKVReq, _ ...grpc.CallOption) (*apikv.GetKVResp, error) {
	entity, ok := f.entities[req.GetKey()]
	if !ok {
		return nil, concept.Err_NOT_FOUND
	}
	return &apikv.GetKVResp{Entity: entity}, nil
}

func (f *instanceHybridTestKV) GetKVs(context.Context, *apikv.GetKVsReq, ...grpc.CallOption) (*apikv.GetKVsResp, error) {
	return nil, errors.New("unused")
}

func (f *instanceHybridTestKV) SetKV(_ context.Context, req *apikv.SetKVReq, _ ...grpc.CallOption) (*apikv.Empty, error) {
	f.set = append(f.set, req)
	if err := f.setErr[req.GetKey()]; err != nil {
		return nil, err
	}
	f.entities[req.GetKey()] = &apikv.Entity{Key: req.GetKey(), Val: req.GetVal(), Ttl: req.GetTtl()}
	return &apikv.Empty{}, nil
}

func (f *instanceHybridTestKV) UnsetKV(_ context.Context, req *apikv.UnsetKVReq, _ ...grpc.CallOption) (*apikv.Empty, error) {
	f.unset = append(f.unset, req)
	if err := f.unsetErr[req.GetKey()]; err != nil {
		return nil, err
	}
	delete(f.entities, req.GetKey())
	return &apikv.Empty{}, nil
}

func (f *instanceHybridTestKV) Watch(context.Context, *apikv.WatchReq, ...grpc.CallOption) (apikv.KV_WatchClient, error) {
	return nil, errors.New("unused")
}

func (f *instanceHybridTestKV) TTL(context.Context, *apikv.TtlReq, ...grpc.CallOption) (*apikv.TtlResp, error) {
	return nil, errors.New("unused")
}

func (f *instanceHybridTestKV) Expire(context.Context, *apikv.ExpireReq, ...grpc.CallOption) (*apikv.Empty, error) {
	return nil, errors.New("unused")
}

func (f *instanceHybridTestKV) Range(context.Context, *apikv.RangeReq, ...grpc.CallOption) (*apikv.RangeResp, error) {
	return nil, errors.New("unused")
}

func (f *instanceHybridTestKV) CompactElementHistory(context.Context, *apikv.CompactElementHistoryReq, ...grpc.CallOption) (*apikv.CompactElementHistoryResp, error) {
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
	failedKey := concept.GenInstanceReversedKeyWithInsId("app", "env", "key2", ins.Id())
	kv.setErr[failedKey] = errors.New("boom")

	err := (instanceHybrid{cassemdb: kv}).setInstanceInfo(context.Background(), ins)
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
	kv.entities[concept.GenInstanceNormalKey(ins.Id())] = &apikv.Entity{Key: concept.GenInstanceNormalKey(ins.Id()), Val: data}
	firstFailedKey := concept.GenInstanceReversedKeyWithInsId("app", "env", "key1", ins.Id())
	secondFailedKey := concept.GenInstanceReversedKeyWithInsId("app", "env", "key2", ins.Id())
	kv.unsetErr[firstFailedKey] = errors.New("first")
	kv.unsetErr[secondFailedKey] = errors.New("second")

	err = (instanceHybrid{cassemdb: kv}).UnregisterInstance(context.Background(), ins.Id())
	require.Error(t, err)
	require.Contains(t, err.Error(), firstFailedKey)
	require.Contains(t, err.Error(), secondFailedKey)
	require.True(t, strings.Contains(err.Error(), "first"))
	require.True(t, strings.Contains(err.Error(), "second"))
	require.Len(t, kv.unset, 3)
}
