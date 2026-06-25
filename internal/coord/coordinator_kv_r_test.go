package coord

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"buf.build/go/protovalidate"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/yeqown/cassem/api/concept"
	apikv "github.com/yeqown/cassem/api/kv"
)

type kvReadOnlyTestKV struct {
	entities map[string]*apikv.Entity
	errors   map[string]*apikv.KeyError
}

func (f *kvReadOnlyTestKV) GetKV(_ context.Context, req *apikv.GetKVReq, _ ...grpc.CallOption) (*apikv.GetKVResp, error) {
	entity, ok := f.entities[req.GetKey()]
	if !ok {
		return nil, concept.Err_NOT_FOUND
	}
	return &apikv.GetKVResp{Entity: entity}, nil
}

func (f *kvReadOnlyTestKV) GetKVs(_ context.Context, req *apikv.GetKVsReq, _ ...grpc.CallOption) (*apikv.GetKVsResp, error) {
	if err := protovalidate.Validate(req); err != nil {
		return nil, err
	}
	entities := make([]*apikv.Entity, 0, len(req.GetKeys()))
	errs := make([]*apikv.KeyError, 0)
	for _, key := range req.GetKeys() {
		if item, ok := f.errors[key]; ok {
			errs = append(errs, item)
			continue
		}
		entity, ok := f.entities[key]
		if !ok {
			errs = append(errs, &apikv.KeyError{Key: key, Code: "NotFound", Message: "NOT_FOUND"})
			continue
		}
		entities = append(entities, entity)
	}
	return &apikv.GetKVsResp{Entities: entities, Errors: errs}, nil
}
func (f *kvReadOnlyTestKV) SetKV(context.Context, *apikv.SetKVReq, ...grpc.CallOption) (*apikv.Empty, error) {
	return nil, errors.New("unused")
}
func (f *kvReadOnlyTestKV) UnsetKV(context.Context, *apikv.UnsetKVReq, ...grpc.CallOption) (*apikv.Empty, error) {
	return nil, errors.New("unused")
}
func (f *kvReadOnlyTestKV) Watch(context.Context, *apikv.WatchReq, ...grpc.CallOption) (apikv.KV_WatchClient, error) {
	return nil, errors.New("unused")
}
func (f *kvReadOnlyTestKV) TTL(context.Context, *apikv.TtlReq, ...grpc.CallOption) (*apikv.TtlResp, error) {
	return nil, errors.New("unused")
}
func (f *kvReadOnlyTestKV) Expire(context.Context, *apikv.ExpireReq, ...grpc.CallOption) (*apikv.Empty, error) {
	return nil, errors.New("unused")
}
func (f *kvReadOnlyTestKV) Range(_ context.Context, req *apikv.RangeReq, _ ...grpc.CallOption) (*apikv.RangeResp, error) {
	entities := make([]*apikv.Entity, 0)
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
	slices.SortFunc(entities, func(a, b *apikv.Entity) int { return strings.Compare(a.GetKey(), b.GetKey()) })
	if len(entities) > int(req.GetLimit()) {
		return &apikv.RangeResp{
			Entities:    entities[:req.GetLimit()],
			HasMore:     true,
			NextSeekKey: concept.ExtractPureKey(entities[req.GetLimit()].GetKey()),
		}, nil
	}
	return &apikv.RangeResp{Entities: entities}, nil
}
func (f *kvReadOnlyTestKV) CompactElementHistory(context.Context, *apikv.CompactElementHistoryReq, ...grpc.CallOption) (*apikv.CompactElementHistoryResp, error) {
	return nil, errors.New("unused")
}

func addElementTestData(t *testing.T, entities map[string]*apikv.Entity, app, env, key string, version int32) {
	t.Helper()
	baseKey := concept.GenElementKey(app, env, key)
	entities[baseKey] = apikv.NewEntityWithCreated(baseKey, nil, 0, 1)
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
	entities[concept.WithMetadataSuffix(baseKey)] = apikv.NewEntityWithCreated(concept.WithMetadataSuffix(baseKey), metadataData, 0, 1)

	elementData, err := concept.MarshalProto(&concept.Element{Raw: []byte(key), Version: version, Published: true})
	require.NoError(t, err)
	entities[concept.WithVersion(baseKey, int(version))] = apikv.NewEntityWithCreated(concept.WithVersion(baseKey, int(version)), elementData, 0, 1)
}

func TestKVReadOnlyGetAppsSearchesByIdAndDescription(t *testing.T) {
	apps := []*concept.AppMetadata{
		{Id: "alpha", Description: "General app"},
		{Id: "billing", Description: "Payment Demo"},
		{Id: "demo-api", Description: "Backend service"},
		{Id: "zeta", Description: "Other app"},
	}
	entities := make(map[string]*apikv.Entity, len(apps))
	for _, app := range apps {
		data, err := concept.MarshalProto(app)
		require.NoError(t, err)
		entities[concept.GenAppKey(app.Id)] = apikv.NewEntityWithCreated(concept.GenAppKey(app.Id), data, 0, 1)
	}

	out, err := (kvReadOnly{cassemkv: &kvReadOnlyTestKV{entities: entities}}).GetApps(context.Background(), "", 1, "demo")
	require.NoError(t, err)
	require.Len(t, out.Apps, 1)
	require.Equal(t, "billing", out.Apps[0].Id)
	require.True(t, out.HasMore)
	require.NotEmpty(t, out.NextSeek)

	out, err = (kvReadOnly{cassemkv: &kvReadOnlyTestKV{entities: entities}}).GetApps(context.Background(), out.NextSeek, 1, "demo")
	require.NoError(t, err)
	require.Len(t, out.Apps, 1)
	require.Equal(t, "demo-api", out.Apps[0].Id)
	require.False(t, out.HasMore)
}

func TestKVReadOnlyGetElementsSearchesByKey(t *testing.T) {
	entities := make(map[string]*apikv.Entity)
	addElementTestData(t, entities, "demo", "prod", "api.host", 1)
	addElementTestData(t, entities, "demo", "prod", "db.url", 1)
	addElementTestData(t, entities, "demo", "prod", "feature.flag", 1)
	addElementTestData(t, entities, "demo", "prod", "service.API.timeout", 1)

	out, err := (kvReadOnly{cassemkv: &kvReadOnlyTestKV{entities: entities}}).GetElements(context.Background(), "demo", "prod", "", 1, "api")
	require.NoError(t, err)
	require.Len(t, out.Elements, 1)
	require.Equal(t, "api.host", out.Elements[0].Metadata.Key)
	require.True(t, out.HasMore)
	require.NotEmpty(t, out.NextSeek)

	out, err = (kvReadOnly{cassemkv: &kvReadOnlyTestKV{entities: entities}}).GetElements(context.Background(), "demo", "prod", out.NextSeek, 1, "api")
	require.NoError(t, err)
	require.Len(t, out.Elements, 1)
	require.Equal(t, "service.API.timeout", out.Elements[0].Metadata.Key)
	require.False(t, out.HasMore)
}

func TestKVReadOnlyGetElementsNormalizesRangeKeys(t *testing.T) {
	entities := make(map[string]*apikv.Entity)
	baseKey := concept.GenElementKey("demo", "prod", "api.host")
	entities[baseKey] = apikv.NewEntityWithCreated(baseKey, nil, 0, 1)
	metadata := &concept.ElementMetadata{
		Key:          "api.host",
		App:          "demo",
		Env:          "prod",
		UsingVersion: 1,
		ContentType:  concept.ContentType_JSON,
	}
	metadataData, err := concept.MarshalProto(metadata)
	require.NoError(t, err)
	entities[concept.WithMetadataSuffix(baseKey)] = apikv.NewEntityWithCreated(concept.WithMetadataSuffix(baseKey), metadataData, 0, 1)
	elementData, err := concept.MarshalProto(&concept.Element{Raw: []byte("value"), Version: 1, Published: true})
	require.NoError(t, err)
	entities[concept.WithVersion(baseKey, 1)] = apikv.NewEntityWithCreated(concept.WithVersion(baseKey, 1), elementData, 0, 1)

	out, err := (kvReadOnly{cassemkv: &kvReadOnlyTestKV{entities: entities}}).GetElements(context.Background(), "demo", "prod", "", 15, "")
	require.NoError(t, err)
	require.Len(t, out.Elements, 1)
	require.Equal(t, "api.host", out.Elements[0].Metadata.Key)
}

func TestKVReadOnlyGetElementsByKeysReturnsEmptyWhenMetadataHasNoAvailableVersion(t *testing.T) {
	baseKey := concept.GenElementKey("app", "env", "key")
	metadata := &concept.ElementMetadata{
		Key:           "key",
		App:           "app",
		Env:           "env",
		LatestVersion: 1,
		ContentType:   concept.ContentType_JSON,
	}
	metadataData, err := concept.MarshalProto(metadata)
	require.NoError(t, err)

	kv := &kvReadOnlyTestKV{entities: map[string]*apikv.Entity{
		concept.WithMetadataSuffix(baseKey): apikv.NewEntityWithCreated(concept.WithMetadataSuffix(baseKey), metadataData, 0, 1),
	}}

	out, err := (kvReadOnly{cassemkv: kv}).GetElementsByKeys(context.Background(), "app", "env", []string{"key"})
	require.NoError(t, err)
	require.Empty(t, out.Elements)
}

func TestKVReadOnlyGetElementsByKeysIgnoresMetadataNotFoundErrors(t *testing.T) {
	entities := make(map[string]*apikv.Entity)
	addElementTestData(t, entities, "app", "env", "exists", 1)
	kv := &kvReadOnlyTestKV{entities: entities}

	out, err := (kvReadOnly{cassemkv: kv}).GetElementsByKeys(context.Background(), "app", "env", []string{"exists", "missing"})
	require.NoError(t, err)
	require.Len(t, out.Elements, 1)
	require.Equal(t, "exists", out.Elements[0].Metadata.Key)
}

func TestKVReadOnlyGetElementsByKeysReturnsVersionNotFoundErrors(t *testing.T) {
	baseKey := concept.GenElementKey("app", "env", "key")
	metadata := &concept.ElementMetadata{Key: "key", App: "app", Env: "env", UsingVersion: 1, ContentType: concept.ContentType_JSON}
	metadataData, err := concept.MarshalProto(metadata)
	require.NoError(t, err)
	kv := &kvReadOnlyTestKV{entities: map[string]*apikv.Entity{
		concept.WithMetadataSuffix(baseKey): apikv.NewEntityWithCreated(concept.WithMetadataSuffix(baseKey), metadataData, 0, 1),
	}}

	_, err = (kvReadOnly{cassemkv: kv}).GetElementsByKeys(context.Background(), "app", "env", []string{"key"})
	require.ErrorIs(t, err, concept.Err_NOT_FOUND)
	require.Contains(t, err.Error(), concept.WithVersion(baseKey, 1))
	require.Contains(t, err.Error(), "NotFound")
}

func TestKVReadOnlyGetElementsByKeysReturnsNonNotFoundMetadataErrors(t *testing.T) {
	baseKey := concept.GenElementKey("app", "env", "key")
	kv := &kvReadOnlyTestKV{
		entities: map[string]*apikv.Entity{},
		errors: map[string]*apikv.KeyError{
			concept.WithMetadataSuffix(baseKey): {Key: concept.WithMetadataSuffix(baseKey), Code: "Internal", Message: "boom"},
		},
	}

	_, err := (kvReadOnly{cassemkv: kv}).GetElementsByKeys(context.Background(), "app", "env", []string{"key"})
	require.ErrorIs(t, err, concept.Err_INTERNAL)
	require.Contains(t, err.Error(), "Internal")
	require.Contains(t, err.Error(), "boom")
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

	kv := &kvReadOnlyTestKV{entities: map[string]*apikv.Entity{
		concept.WithMetadataSuffix(baseKey): apikv.NewEntityWithCreated("md", metadataData, 0, 1),
		concept.WithVersion(baseKey, 1):     apikv.NewEntityWithCreated("v1", elementData, 0, 1),
	}}

	out, err := (kvReadOnly{cassemkv: kv}).GetElementWithVersion(context.Background(), "app", "env", "key", 0)
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

	kv := &kvReadOnlyTestKV{entities: map[string]*apikv.Entity{
		concept.WithMetadataSuffix(concept.GenElementKey("app", "env", "key")): apikv.NewEntityWithCreated("md", data, 0, 1),
	}}

	_, err = (kvReadOnly{cassemkv: kv}).GetElementWithVersion(context.Background(), "app", "env", "key", 99)
	require.ErrorIs(t, err, concept.Err_NOT_FOUND)
}
