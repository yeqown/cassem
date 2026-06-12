package coordinator

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/yeqown/cassem/api/concept"
	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
	"github.com/yeqown/cassem/pkg/errorx"
)

type kvReadOnlyTestKV struct {
	entities map[string]*apicassemdb.Entity
}

func (f *kvReadOnlyTestKV) GetKV(_ context.Context, req *apicassemdb.GetKVReq, _ ...grpc.CallOption) (*apicassemdb.GetKVResp, error) {
	entity, ok := f.entities[req.GetKey()]
	if !ok {
		return nil, errorx.Err_NOT_FOUND
	}
	return &apicassemdb.GetKVResp{Entity: entity}, nil
}

func (f *kvReadOnlyTestKV) GetKVs(context.Context, *apicassemdb.GetKVsReq, ...grpc.CallOption) (*apicassemdb.GetKVsResp, error) {
	return nil, errors.New("unused")
}
func (f *kvReadOnlyTestKV) SetKV(context.Context, *apicassemdb.SetKVReq, ...grpc.CallOption) (*apicassemdb.Empty, error) {
	return nil, errors.New("unused")
}
func (f *kvReadOnlyTestKV) UnsetKV(context.Context, *apicassemdb.UnsetKVReq, ...grpc.CallOption) (*apicassemdb.Empty, error) {
	return nil, errors.New("unused")
}
func (f *kvReadOnlyTestKV) Watch(context.Context, *apicassemdb.WatchReq, ...grpc.CallOption) (apicassemdb.KV_WatchClient, error) {
	return nil, errors.New("unused")
}
func (f *kvReadOnlyTestKV) TTL(context.Context, *apicassemdb.TtlReq, ...grpc.CallOption) (*apicassemdb.TtlResp, error) {
	return nil, errors.New("unused")
}
func (f *kvReadOnlyTestKV) Expire(context.Context, *apicassemdb.ExpireReq, ...grpc.CallOption) (*apicassemdb.Empty, error) {
	return nil, errors.New("unused")
}
func (f *kvReadOnlyTestKV) Range(context.Context, *apicassemdb.RangeReq, ...grpc.CallOption) (*apicassemdb.RangeResp, error) {
	return nil, errors.New("unused")
}

func TestGetElementWithVersionReturnsVersionLookupError(t *testing.T) {
	metadata := &concept.ElementMetadata{Key: "key", App: "app", Env: "env", UsingVersion: 1}
	data, err := concept.MarshalProto(metadata)
	require.NoError(t, err)

	kv := &kvReadOnlyTestKV{entities: map[string]*apicassemdb.Entity{
		concept.WithMetadataSuffix(concept.GenElementKey("app", "env", "key")): apicassemdb.NewEntityWithCreated("md", data, 0, 1),
	}}

	_, err = (kvReadOnly{cassemdb: kv}).GetElementWithVersion(context.Background(), "app", "env", "key", 99)
	require.ErrorIs(t, err, errorx.Err_NOT_FOUND)
}
