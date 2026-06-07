package coordinator

import (
	"context"
	"fmt"

	"github.com/yeqown/log"

	"github.com/yeqown/cassem/api/concept"
	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
	"github.com/yeqown/cassem/pkg/errorx"
)

const (
	_VERSION_PREFIX = "v"
	_APP_PREFIX     = "cassem/apps"
)


// kvReadOnly manages all read operation from cassemdb, it is allowed to read only.
type kvReadOnly struct {
	cassemdb apicassemdb.KVClient
}

// NewKVReader with endpoints these endpoints of cassemdb.
func NewKVReader(endpoints []string) (concept.KVReadOnly, error) {
	cc, err := apicassemdb.DialWithMode(endpoints, apicassemdb.Mode_R)
	if err != nil {
		return nil, fmt.Errorf("NewWriter: %w", err)
	}

	return kvReadOnly{
		cassemdb: apicassemdb.NewKVClient(cc),
	}, nil
}

func (_r kvReadOnly) GetElementWithVersion(
	ctx context.Context, app, env, key string, version int) (*concept.Element, error) {
	// get metadata
	k := concept.GenElementKey(app, env, key)
	r1, err := _r.cassemdb.GetKV(ctx, &apicassemdb.GetKVReq{Key: concept.WithMetadataSuffix(k)})
	if err != nil {
		return nil, err
	}
	md := new(concept.ElementMetadata)
	if err = concept.UnmarshalProto(r1.GetEntity().GetVal(), md); err != nil {
		return nil, err
	}

	if version <= 0 {
		version = int(md.UsingVersion)
	}
	if version <= 0 {
		// if there's not using version, NOT_FOUND
		return nil, fmt.Errorf("kvReadOnly.GetElementVersions: no available using version: %w", errorx.Err_NOT_FOUND)
	}

	// get element with specified version
	r2, err2 := _r.cassemdb.GetKV(ctx, &apicassemdb.GetKVReq{Key: concept.WithVersion(k, version)})
	if err2 != nil {
		return nil, err
	}
	elt := new(concept.Element)
	if err2 = concept.UnmarshalProto(r2.GetEntity().GetVal(), elt); err2 != nil {
		return nil, err2
	}
	elt.Metadata = md

	return elt, nil
}

func (_r kvReadOnly) GetElementVersions(
	ctx context.Context, app, env, key string, seek string, limit int) (*concept.GetElementsResult, error) {
	k := concept.GenElementKey(app, env, key)
	log.
		WithFields(log.Fields{
			"app":   app,
			"env":   env,
			"seek":  seek,
			"limit": limit,
			"k":     k,
		}).
		Debug("kvReadOnly.GetElementVersions enter")

	r, err := _r.cassemdb.GetKVs(ctx, &apicassemdb.GetKVsReq{
		Keys: []string{concept.WithMetadataSuffix(k)},
	})
	if err != nil {
		return nil, fmt.Errorf("kvReadOnly.GetElementVersions: %w", err)
	}

	if len(seek) == 0 {
		// default seek to skip metadata
		seek = _VERSION_PREFIX
	}

	r2, err := _r.cassemdb.Range(ctx, &apicassemdb.RangeReq{
		Key:   k,
		Seek:  seek,
		Limit: int32(limit),
	})
	if err != nil {
		return nil, err
	}

	_, _, mdMapping := ConvertFromEntitiesToMetadata(r.GetEntities(), false)
	result := &concept.GetElementsResult{
		CommonPager: concept.CommonPager{
			HasMore:  r2.GetHasMore(),
			NextSeek: r2.GetNextSeekKey(),
		},
		Elements: ConvertFromEntitiesToElements(r2.GetEntities(), mdMapping),
	}

	return result, err
}

// GetElements paging elements under app and env bucket.
func (_r kvReadOnly) GetElements(
	ctx context.Context, app, env string, seek string, limit int) (*concept.GetElementsResult, error) {
	k := concept.GenAppElementEnvKey(app, env)

	log.
		WithFields(log.Fields{
			"app":   app,
			"env":   env,
			"seek":  seek,
			"limit": limit,
			"k":     k,
		}).
		Debug("kvReadOnly.GetElements enter")
	r, err := _r.cassemdb.Range(ctx, &apicassemdb.RangeReq{
		Key:   k,
		Seek:  seek,
		Limit: int32(limit),
	})
	if err != nil {
		return nil, err
	}

	result := &concept.GetElementsResult{
		CommonPager: concept.CommonPager{
			HasMore:  r.GetHasMore(),
			NextSeek: r.GetNextSeekKey(),
		},
		Elements: make([]*concept.Element, 0, len(r.GetEntities())),
	}
	keys := make([]string, 0, len(r.GetEntities()))
	for _, v := range r.GetEntities() {
		keys = append(keys, v.GetKey())
	}

	result.Elements, err = _r.getElementsByKeys(ctx, app, env, keys, false)
	return result, err
}

func (_r kvReadOnly) GetElementsByKeys(
	ctx context.Context, app, env string, keys []string) (result *concept.GetElementsResult, err error) {
	result = &concept.GetElementsResult{
		CommonPager: concept.CommonPager{},
		Elements:    nil,
	}
	result.Elements, err = _r.getElementsByKeys(ctx, app, env, keys, false)
	return
}

// getElementsByKeys get elements by keys.
// keys contain all key to element.
func (_r kvReadOnly) getElementsByKeys(
	ctx context.Context, app, env string, keys []string,
	wipeUnpublish bool,
) ([]*concept.Element, error) {
	if len(keys) == 0 {
		return []*concept.Element{}, nil
	}
	mdKeys := make([]string, 0, len(keys))
	for _, key := range keys {
		k := concept.GenElementKey(app, env, key)
		mdKeys = append(mdKeys, concept.WithMetadataSuffix(k))
	}
	r, err := _r.cassemdb.GetKVs(ctx, &apicassemdb.GetKVsReq{
		Keys: mdKeys,
	})
	if err != nil {
		return nil, fmt.Errorf("kvReadOnly.getElementsByKeys: %w", err)
	}

	// DONE(@yeqown): replace this part of code with convertFromEntitiesToMetadata
	eleVersionKeys, _, metadataMapping := ConvertFromEntitiesToMetadata(r.GetEntities(), wipeUnpublish)
	r2, err2 := _r.cassemdb.GetKVs(ctx, &apicassemdb.GetKVsReq{
		Keys: eleVersionKeys,
	})
	if err2 != nil {
		return nil, fmt.Errorf("kvReadOnly.getElementsByKeys: %w", err)
	}

	out := ConvertFromEntitiesToElements(r2.GetEntities(), metadataMapping)

	return out, nil
}

func (_r kvReadOnly) GetElementOperations(
	ctx context.Context, app, env, eltKey string, start int) (ops []*concept.ElementOperation, next int, err error) {
	// TODO(@yeqown): implement this
	panic("implement me")
}

func (_r kvReadOnly) GetApp(ctx context.Context, app string) (*concept.AppMetadata, error) {
	k := concept.GenAppKey(app)
	r, err := _r.cassemdb.GetKV(ctx, &apicassemdb.GetKVReq{
		Key: k,
	})
	if err != nil {
		return nil, err
	}

	md := new(concept.AppMetadata)
	err = concept.UnmarshalProto(r.GetEntity().GetVal(), md)
	return md, err
}

func (_r kvReadOnly) GetApps(ctx context.Context, seek string, limit int) (*concept.GetAppsResult, error) {
	r, err := _r.cassemdb.Range(ctx, &apicassemdb.RangeReq{
		Key:   _APP_PREFIX,
		Seek:  seek,
		Limit: int32(limit),
	})
	if err != nil {
		return nil, err
	}

	result := &concept.GetAppsResult{
		CommonPager: concept.CommonPager{
			HasMore:  r.GetHasMore(),
			NextSeek: r.GetNextSeekKey(),
		},
		Apps: make([]*concept.AppMetadata, 0, len(r.GetEntities())),
	}

	for _, v := range r.GetEntities() {
		md := new(concept.AppMetadata)
		_ = concept.UnmarshalProto(v.Val, md)
		result.Apps = append(result.Apps, md)
	}

	return result, nil
}

func (_r kvReadOnly) GetEnvironments(ctx context.Context, app, seek string, limit int) (*concept.GetAppEnvsResult, error) {
	k := concept.GenAppElementKey(app)
	r, err := _r.cassemdb.Range(ctx, &apicassemdb.RangeReq{
		Key:   k,
		Seek:  seek,
		Limit: int32(limit),
	})
	if err != nil {
		return nil, err
	}

	result := &concept.GetAppEnvsResult{
		CommonPager: concept.CommonPager{
			HasMore:  r.GetHasMore(),
			NextSeek: r.GetNextSeekKey(),
		},
		Environments: make([]string, 0, len(r.GetEntities())),
	}

	for _, v := range r.GetEntities() {
		result.Environments = append(result.Environments, concept.ExtractPureKey(v.Key))
	}

	return result, nil
}
