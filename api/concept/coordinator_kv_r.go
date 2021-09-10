package concept

import (
	"context"

	"github.com/pkg/errors"
	"github.com/yeqown/log"

	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
	"github.com/yeqown/cassem/pkg/errorx"
)

// kvReadOnly manages all read operation from cassemdb, it is allowed to read only.
type kvReadOnly struct {
	cassemdb apicassemdb.KVClient
}

// NewKVReader with endpoints these endpoints of cassemdb.
func NewKVReader(endpoints []string) (KVReadOnly, error) {
	cc, err := apicassemdb.DialWithMode(endpoints, apicassemdb.Mode_R)
	if err != nil {
		return nil, errors.Wrap(err, "NewWriter")
	}

	return kvReadOnly{
		cassemdb: apicassemdb.NewKVClient(cc),
	}, nil
}

func (_r kvReadOnly) GetElementWithVersion(
	ctx context.Context, app, env, key string, version int) (*Element, error) {
	// get metadata
	k := genElementKey(app, env, key)
	r1, err := _r.cassemdb.GetKV(ctx, &apicassemdb.GetKVReq{Key: withMetadataSuffix(k)})
	if err != nil {
		return nil, err
	}
	md := new(ElementMetadata)
	if err = UnmarshalProto(r1.GetEntity().GetVal(), md); err != nil {
		return nil, err
	}

	if version <= 0 {
		version = int(md.UsingVersion)
	}
	if version <= 0 {
		// if there's not using version, NOT_FOUND
		return nil, errors.Wrap(errorx.Err_NOT_FOUND,
			"kvReadOnly.GetElementVersions: no available using version")
	}

	// get element with specified version
	r2, err2 := _r.cassemdb.GetKV(ctx, &apicassemdb.GetKVReq{Key: withVersion(k, version)})
	if err2 != nil {
		return nil, err
	}
	elt := new(Element)
	if err2 = UnmarshalProto(r2.GetEntity().GetVal(), elt); err2 != nil {
		return nil, err2
	}
	elt.Metadata = md

	return elt, nil
}

func (_r kvReadOnly) GetElementVersions(
	ctx context.Context, app, env, key string, seek string, limit int) (*getElementsResult, error) {
	k := genElementKey(app, env, key)
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
		Keys: []string{withMetadataSuffix(k)},
	})
	if err != nil {
		return nil, errors.Wrap(err, "kvReadOnly.GetElementVersions")
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

	_, _, mdMapping := convertFromEntitiesToMetadata(r.GetEntities(), false)
	result := &getElementsResult{
		commonPager: commonPager{
			HasMore:  r2.GetHasMore(),
			NextSeek: r2.GetNextSeekKey(),
		},
		Elements: convertFromEntitiesToElements(r2.GetEntities(), mdMapping),
	}

	return result, err
}

// GetElements paging elements under app and env bucket.
func (_r kvReadOnly) GetElements(
	ctx context.Context, app, env string, seek string, limit int) (*getElementsResult, error) {
	k := genAppElementEnvKey(app, env)

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

	result := &getElementsResult{
		commonPager: commonPager{
			HasMore:  r.GetHasMore(),
			NextSeek: r.GetNextSeekKey(),
		},
		Elements: make([]*Element, 0, len(r.GetEntities())),
	}
	keys := make([]string, 0, len(r.GetEntities()))
	for _, v := range r.GetEntities() {
		keys = append(keys, v.GetKey())
	}

	result.Elements, err = _r.getElementsByKeys(ctx, app, env, keys, false)
	return result, err
}

func (_r kvReadOnly) GetElementsByKeys(
	ctx context.Context, app, env string, keys []string) (result *getElementsResult, err error) {
	result = &getElementsResult{
		commonPager: commonPager{},
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
) ([]*Element, error) {
	if len(keys) == 0 {
		return []*Element{}, nil
	}
	mdKeys := make([]string, 0, len(keys))
	for _, key := range keys {
		k := genElementKey(app, env, key)
		mdKeys = append(mdKeys, withMetadataSuffix(k))
	}
	r, err := _r.cassemdb.GetKVs(ctx, &apicassemdb.GetKVsReq{
		Keys: mdKeys,
	})
	if err != nil {
		return nil, errors.Wrap(err, "kvReadOnly.getElementsByKeys")
	}

	// DONE(@yeqown): replace this part of code with convertFromEntitiesToMetadata
	eleVersionKeys, _, metadataMapping := convertFromEntitiesToMetadata(r.GetEntities(), wipeUnpublish)
	r2, err2 := _r.cassemdb.GetKVs(ctx, &apicassemdb.GetKVsReq{
		Keys: eleVersionKeys,
	})
	if err2 != nil {
		return nil, errors.Wrap(err, "kvReadOnly.getElementsByKeys")
	}

	out := convertFromEntitiesToElements(r2.GetEntities(), metadataMapping)

	return out, nil
}

func (_r kvReadOnly) GetElementOperations(
	ctx context.Context, app, env, eltKey string, start int) (ops []*ElementOperation, next int, err error) {
	// TODO(@yeqown): implement this
	panic("implement me")
}

func (_r kvReadOnly) GetApp(ctx context.Context, app string) (*AppMetadata, error) {
	k := genAppKey(app)
	r, err := _r.cassemdb.GetKV(ctx, &apicassemdb.GetKVReq{
		Key: k,
	})
	if err != nil {
		return nil, err
	}

	md := new(AppMetadata)
	err = UnmarshalProto(r.GetEntity().GetVal(), md)
	return md, err
}

func (_r kvReadOnly) GetApps(ctx context.Context, seek string, limit int) (*getAppsResult, error) {
	r, err := _r.cassemdb.Range(ctx, &apicassemdb.RangeReq{
		Key:   _APP_PREFIX,
		Seek:  seek,
		Limit: int32(limit),
	})
	if err != nil {
		return nil, err
	}

	result := &getAppsResult{
		commonPager: commonPager{
			HasMore:  r.GetHasMore(),
			NextSeek: r.GetNextSeekKey(),
		},
		Apps: make([]*AppMetadata, 0, len(r.GetEntities())),
	}

	for _, v := range r.GetEntities() {
		md := new(AppMetadata)
		_ = UnmarshalProto(v.Val, md)
		result.Apps = append(result.Apps, md)
	}

	return result, nil
}

func (_r kvReadOnly) GetEnvironments(ctx context.Context, app, seek string, limit int) (*getAppEnvsResult, error) {
	k := genAppElementKey(app)
	r, err := _r.cassemdb.Range(ctx, &apicassemdb.RangeReq{
		Key:   k,
		Seek:  seek,
		Limit: int32(limit),
	})
	if err != nil {
		return nil, err
	}

	result := &getAppEnvsResult{
		commonPager: commonPager{
			HasMore:  r.GetHasMore(),
			NextSeek: r.GetNextSeekKey(),
		},
		Environments: make([]string, 0, len(r.GetEntities())),
	}

	for _, v := range r.GetEntities() {
		result.Environments = append(result.Environments, extractPureKey(v.Key))
	}

	return result, nil
}
