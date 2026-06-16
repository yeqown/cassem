package coordinator

import (
	"context"
	"errors"
	"slices"
	"strings"
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

func (f *kvReadOnlyTestKV) GetKVs(_ context.Context, req *apicassemdb.GetKVsReq, _ ...grpc.CallOption) (*apicassemdb.GetKVsResp, error) {
	entities := make([]*apicassemdb.Entity, 0, len(req.GetKeys()))
	for _, key := range req.GetKeys() {
		entity, ok := f.entities[key]
		if !ok {
			return nil, errorx.Err_NOT_FOUND
		}
		entities = append(entities, entity)
	}
	return &apicassemdb.GetKVsResp{Entities: entities}, nil
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
func (f *kvReadOnlyTestKV) Range(_ context.Context, req *apicassemdb.RangeReq, _ ...grpc.CallOption) (*apicassemdb.RangeResp, error) {
	entities := make([]*apicassemdb.Entity, 0)
	for key, entity := range f.entities {
		if req.GetKey() != "" {
			prefix := req.GetKey() + "/"
			if !strings.HasPrefix(key, prefix) {
				continue
			}
			if strings.Contains(strings.TrimPrefix(key, prefix), "/") {
				continue
			}
		}
		if req.GetSeek() != "" && concept.ExtractPureKey(key) < req.GetSeek() {
			continue
		}
		entities = append(entities, entity)
	}
	slices.SortFunc(entities, func(a, b *apicassemdb.Entity) int { return strings.Compare(a.GetKey(), b.GetKey()) })
	if len(entities) > int(req.GetLimit()) {
		return &apicassemdb.RangeResp{
			Entities:    entities[:req.GetLimit()],
			HasMore:     true,
			NextSeekKey: concept.ExtractPureKey(entities[req.GetLimit()].GetKey()),
		}, nil
	}
	return &apicassemdb.RangeResp{Entities: entities}, nil
}

func addElementTestData(t *testing.T, entities map[string]*apicassemdb.Entity, app, env, key string, version int32) {
	t.Helper()
	baseKey := concept.GenElementKey(app, env, key)
	entities[baseKey] = apicassemdb.NewEntityWithCreated(key, nil, 0, 1)
	metadata := &concept.ElementMetadata{
		Key:           key,
		App:           app,
		Env:           env,
		LatestVersion: version,
		UsingVersion:  version,
		ContentType:   concept.ContentType_JSON,
	}
	metadataData, err := concept.MarshalProto(metadata)
	require.NoError(t, err)
	entities[concept.WithMetadataSuffix(baseKey)] = apicassemdb.NewEntityWithCreated(concept.WithMetadataSuffix(baseKey), metadataData, 0, 1)

	elementData, err := concept.MarshalProto(&concept.Element{Raw: []byte(key), Version: version, Published: true})
	require.NoError(t, err)
	entities[concept.WithVersion(baseKey, int(version))] = apicassemdb.NewEntityWithCreated(concept.WithVersion(baseKey, int(version)), elementData, 0, 1)
}

func TestKVReadOnlyGetAppsSearchesByIdAndDescription(t *testing.T) {
	apps := []*concept.AppMetadata{
		{Id: "alpha", Description: "General app"},
		{Id: "billing", Description: "Payment Demo"},
		{Id: "demo-api", Description: "Backend service"},
		{Id: "zeta", Description: "Other app"},
	}
	entities := make(map[string]*apicassemdb.Entity, len(apps))
	for _, app := range apps {
		data, err := concept.MarshalProto(app)
		require.NoError(t, err)
		entities[concept.GenAppKey(app.Id)] = apicassemdb.NewEntityWithCreated(concept.GenAppKey(app.Id), data, 0, 1)
	}

	out, err := (kvReadOnly{cassemdb: &kvReadOnlyTestKV{entities: entities}}).GetApps(context.Background(), "", 1, "demo")
	require.NoError(t, err)
	require.Len(t, out.Apps, 1)
	require.Equal(t, "billing", out.Apps[0].Id)
	require.True(t, out.HasMore)
	require.NotEmpty(t, out.NextSeek)

	out, err = (kvReadOnly{cassemdb: &kvReadOnlyTestKV{entities: entities}}).GetApps(context.Background(), out.NextSeek, 1, "demo")
	require.NoError(t, err)
	require.Len(t, out.Apps, 1)
	require.Equal(t, "demo-api", out.Apps[0].Id)
	require.False(t, out.HasMore)
}

func TestKVReadOnlyGetElementsSearchesByKey(t *testing.T) {
	entities := make(map[string]*apicassemdb.Entity)
	addElementTestData(t, entities, "demo", "prod", "api.host", 1)
	addElementTestData(t, entities, "demo", "prod", "db.url", 1)
	addElementTestData(t, entities, "demo", "prod", "feature.flag", 1)
	addElementTestData(t, entities, "demo", "prod", "service.API.timeout", 1)

	out, err := (kvReadOnly{cassemdb: &kvReadOnlyTestKV{entities: entities}}).GetElements(context.Background(), "demo", "prod", "", 1, "api")
	require.NoError(t, err)
	require.Len(t, out.Elements, 1)
	require.Equal(t, "api.host", out.Elements[0].Metadata.Key)
	require.True(t, out.HasMore)
	require.NotEmpty(t, out.NextSeek)

	out, err = (kvReadOnly{cassemdb: &kvReadOnlyTestKV{entities: entities}}).GetElements(context.Background(), "demo", "prod", out.NextSeek, 1, "api")
	require.NoError(t, err)
	require.Len(t, out.Elements, 1)
	require.Equal(t, "service.API.timeout", out.Elements[0].Metadata.Key)
	require.False(t, out.HasMore)
}

func TestGetElementWithVersionReturnsVersionZeroWhenNoUsingVersion(t *testing.T) {
	baseKey := concept.GenElementKey("app", "env", "key")
	metadata := &concept.ElementMetadata{
		Key:                "key",
		App:                "app",
		Env:                "env",
		LatestVersion:      1,
		UnpublishedVersion: 1,
		ContentType:        concept.ContentType_JSON,
	}
	metadataData, err := concept.MarshalProto(metadata)
	require.NoError(t, err)
	elementData, err := concept.MarshalProto(&concept.Element{Raw: []byte("draft"), Version: 1})
	require.NoError(t, err)

	kv := &kvReadOnlyTestKV{entities: map[string]*apicassemdb.Entity{
		concept.WithMetadataSuffix(baseKey): apicassemdb.NewEntityWithCreated("md", metadataData, 0, 1),
		concept.WithVersion(baseKey, 1):     apicassemdb.NewEntityWithCreated("v1", elementData, 0, 1),
	}}

	out, err := (kvReadOnly{cassemdb: kv}).GetElementWithVersion(context.Background(), "app", "env", "key", 0)
	require.NoError(t, err)
	require.Equal(t, int32(0), out.GetVersion())
	require.Empty(t, out.GetRaw())
	require.Equal(t, "key", out.GetMetadata().GetKey())
	require.Equal(t, int32(0), out.GetMetadata().GetUsingVersion())
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
