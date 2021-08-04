package concept

import (
	"context"

	"github.com/pkg/errors"
	"github.com/yeqown/log"

	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
	pbcassemdb "github.com/yeqown/cassem/internal/cassemdb/api/gen"
)

// kvReadOnly manages all read operation from cassemdb, it is allowed to read only.
type kvReadOnly struct {
	cassemdb pbcassemdb.KVClient
}

// NewKVReader with endpoints these endpoints of cassemdb.
func NewKVReader(endpoints []string) (KVReadOnly, error) {
	cc, err := apicassemdb.DialWithMode(endpoints, apicassemdb.Mode_R)
	if err != nil {
		return nil, errors.Wrap(err, "NewWriter")
	}

	return kvReadOnly{
		cassemdb: pbcassemdb.NewKVClient(cc),
	}, nil
}

func (_r kvReadOnly) GetElementWithVersion(
	ctx context.Context, app, env, key string, version int) (*VersionedEltDO, error) {
	// get metadata
	k := genElementKey(app, env, key)
	r1, err := _r.cassemdb.GetKV(ctx, &pbcassemdb.GetKVReq{Key: withMetadataSuffix(k)})
	if err != nil {
		return nil, err
	}
	md := new(EltMetadataDO)
	if err = md.Unmarshal(r1.GetEntity().GetVal()); err != nil {
		return nil, err
	}

	md.Key = key
	if version <= 0 {
		version = md.LatestVersion
	}
	// get element with specified version
	r2, err2 := _r.cassemdb.GetKV(ctx, &pbcassemdb.GetKVReq{Key: withVersion(k, version)})
	if err2 != nil {
		return nil, err
	}
	elt := new(VersionedEltDO)
	if err2 = elt.Unmarshal(r2.GetEntity().GetVal()); err2 != nil {
		return nil, err2
	}
	elt.Metadata = md

	return elt, nil
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
	r, err := _r.cassemdb.Range(ctx, &pbcassemdb.RangeReq{
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
		Elements: make([]*VersionedEltDO, 0, len(r.GetEntities())),
	}
	keys := make([]string, 0, len(r.GetEntities()))
	for _, v := range r.GetEntities() {
		keys = append(keys, v.GetKey())
	}

	result.Elements, err = _r.getElementsByKeys(ctx, app, env, keys)
	return result, err
}

func (_r kvReadOnly) GetElementsByKeys(
	ctx context.Context, app, env string, keys []string) (result *getElementsResult, err error) {
	result = &getElementsResult{
		commonPager: commonPager{},
		Elements:    nil,
	}
	result.Elements, err = _r.getElementsByKeys(ctx, app, env, keys)
	return
}

func (_r kvReadOnly) getElementsByKeys(ctx context.Context, app, env string, keys []string) ([]*VersionedEltDO, error) {
	mdKeys := make([]string, 0, len(keys))
	for _, key := range keys {
		k := genElementKey(app, env, key)
		mdKeys = append(mdKeys, withMetadataSuffix(k))
	}
	r, err := _r.cassemdb.GetKVs(ctx, &pbcassemdb.GetKVsReq{
		Keys: mdKeys,
	})
	if err != nil {
		return nil, errors.Wrap(err, "kvReadOnly.getElementsByKeys")
	}

	eleVersionKeys := make([]string, 0, len(keys))
	metadataMapping := make(map[string]*EltMetadataDO, len(keys))
	for _, entity := range r.GetEntities() {
		k := trimMetadata(entity.GetKey())
		md := new(EltMetadataDO)
		if err = md.Unmarshal(entity.GetVal()); err != nil {
			continue
		}
		md.Key = extractPureKey(k)
		metadataMapping[k] = md
		eleVersionKeys = append(eleVersionKeys, withVersion(k, md.LatestVersion))
	}

	r2, err2 := _r.cassemdb.GetKVs(ctx, &pbcassemdb.GetKVsReq{
		Keys: eleVersionKeys,
	})
	if err2 != nil {
		return nil, errors.Wrap(err, "kvReadOnly.getElementsByKeys")
	}

	out := make([]*VersionedEltDO, 0, len(keys))
	for _, entity := range r2.GetEntities() {
		elt := &VersionedEltDO{
			Metadata: new(EltMetadataDO),
			Version:  0,
			Raw:      nil,
		}
		if err = elt.Unmarshal(entity.GetVal()); err != nil {
			continue
		}
		k := trimVersion(entity.GetKey())
		elt.Metadata = metadataMapping[k]
		out = append(out, elt)
	}

	return out, nil
}

func (_r kvReadOnly) GetElementOperations(
	ctx context.Context, app, env, eltKey string, start int) (ops []*EltOperateLog, next int, err error) {
	panic("implement me")
}

func (_r kvReadOnly) GetApp(ctx context.Context, app string) (*AppMetadataDO, error) {
	k := genAppKey(app)
	r, err := _r.cassemdb.GetKV(ctx, &pbcassemdb.GetKVReq{
		Key: k,
	})
	if err != nil {
		return nil, err
	}

	md := new(AppMetadataDO)
	err = md.Unmarshal(r.GetEntity().GetVal())
	return md, err
}

func (_r kvReadOnly) GetApps(ctx context.Context, seek string, limit int) (*getAppsResult, error) {
	r, err := _r.cassemdb.Range(ctx, &pbcassemdb.RangeReq{
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
		Apps: make([]*AppMetadataDO, 0, len(r.GetEntities())),
	}

	for _, v := range r.GetEntities() {
		app := new(AppMetadataDO)
		_ = app.Unmarshal(v.Val)
		result.Apps = append(result.Apps, app)
	}

	return result, nil
}

func (_r kvReadOnly) GetEnvironments(ctx context.Context, app, seek string, limit int) (*getAppEnvsResult, error) {
	k := genAppElementKey(app)
	r, err := _r.cassemdb.Range(ctx, &pbcassemdb.RangeReq{
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
		result.Environments = append(result.Environments, v.Key)
	}

	return result, nil
}
