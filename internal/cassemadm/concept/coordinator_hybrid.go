package concept

import (
	"context"

	"github.com/pkg/errors"

	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
	pbcassemdb "github.com/yeqown/cassem/internal/cassemdb/api/gen"
	"github.com/yeqown/cassem/pkg/hash"
)

var _ Hybrid = hybrid{}

// hybrid can read and write to cassemdb.
type hybrid struct {
	readOnly

	cassemdb pbcassemdb.KVClient
}

// NewHybrid with endpoints these endpoints of cassemdb.
func NewHybrid(endpoints []string) (Hybrid, error) {
	cc, err := apicassemdb.DialWithMode(endpoints, apicassemdb.Mode_X)
	if err != nil {
		return nil, errors.Wrap(err, "NewWriter")
	}

	return hybrid{
		readOnly: readOnly{
			cassemdb: pbcassemdb.NewKVClient(cc),
		},
		cassemdb: pbcassemdb.NewKVClient(cc),
	}, nil
}

func (_h hybrid) CreateElement(ctx context.Context,
	app, env, eltKey string, raw []byte, contentTyp RawContentType) error {
	k := genElementKey(app, env, eltKey)
	mdKey := withMetadataSuffix(k)
	version := 1
	md := EltMetadataDO{
		LatestVersion:     version,
		LatestFingerprint: hash.MD5(raw),
		Key:               mdKey,
		ContentType:       contentTyp,
		App:               app,
		Env:               env,
	}
	// set metadata of element
	bytes, err := md.Marshal()
	if err != nil {
		return err
	}

	if _, err = _h.cassemdb.SetKV(ctx, &pbcassemdb.SetKVReq{
		Key:       mdKey,
		Val:       bytes,
		IsDir:     false,
		Overwrite: true,
		Ttl:       0,
	}); err != nil {
		return err
	}

	versionedEltDo := VersionedEltDO{
		Version: version,
		Raw:     raw,
	}
	bytes, err = versionedEltDo.Marshal()
	if err != nil {
		return err
	}
	// set element with specified version
	if _, err = _h.cassemdb.SetKV(ctx, &pbcassemdb.SetKVReq{
		Key:       withVersion(k, version),
		Val:       bytes,
		IsDir:     false,
		Overwrite: true,
		Ttl:       0,
	}); err != nil {
		return err
	}

	return nil
}

// UpdateElement add a new version to element, and update element's metadata info.
// 1. get metadata
// 2. lock element W operations to prevent concurrent writing operation.
// 3. create a VersionedEltDO
func (_h hybrid) UpdateElement(ctx context.Context, app, env, eltKey string, raw []byte) error {
	// get metadata
	k := genElementKey(app, env, eltKey)
	r, err := _h.cassemdb.GetKV(ctx, &pbcassemdb.GetKVReq{Key: withMetadataSuffix(k)})
	if err != nil {
		return err
	}
	md := new(EltMetadataDO)
	if err = md.Unmarshal(r.GetEntity().GetVal()); err != nil {
		return err
	}

	// version auto increased
	version := md.LatestVersion + 1
	md.LatestVersion = version
	elt := VersionedEltDO{
		Version: version,
		Raw:     raw,
	}

	// save new element version.
	bytes, err := elt.Marshal()
	if err != nil {
		return err
	}
	// set element with specified version
	if _, err = _h.cassemdb.SetKV(ctx, &pbcassemdb.SetKVReq{
		Key:       withVersion(k, version),
		Val:       bytes,
		IsDir:     false,
		Overwrite: true,
		Ttl:       0,
	}); err != nil {
		return err
	}

	bytes, _ = md.Marshal()
	// set element with specified version
	_, err = _h.cassemdb.SetKV(ctx, &pbcassemdb.SetKVReq{
		Key:       withMetadataSuffix(k),
		Val:       bytes,
		IsDir:     false,
		Overwrite: true,
		Ttl:       0,
	})

	return err
}

func (_h hybrid) DeleteElement(ctx context.Context, app, env, eltKey string) error {
	k := genElementKey(app, env, eltKey)
	_, err := _h.cassemdb.UnsetKV(ctx, &pbcassemdb.UnsetKVReq{
		Key:   k,
		IsDir: true,
	})

	return err
}

func (_h hybrid) CreateApp(ctx context.Context, md *AppMetadataDO) error {
	k := genAppKey(md.Id)
	bytes, _ := md.Marshal()
	_, err := _h.cassemdb.SetKV(ctx, &pbcassemdb.SetKVReq{
		Key:       k,
		IsDir:     false,
		Ttl:       0,
		Val:       bytes,
		Overwrite: false,
	})
	return err
}

func (_h hybrid) DeleteApp(ctx context.Context, appId string) error {
	k := genAppKey(appId)
	eleKey := genAppElementKey(appId)

	_, err := _h.cassemdb.UnsetKV(ctx, &pbcassemdb.UnsetKVReq{
		Key:   eleKey,
		IsDir: true,
	})
	if err != nil {
		return err
	}
	_, err = _h.cassemdb.UnsetKV(ctx, &pbcassemdb.UnsetKVReq{
		Key:   k,
		IsDir: false,
	})
	if err != nil {
		return err
	}

	return nil
}

func (_h hybrid) CreateEnvironment(ctx context.Context, md *AppMetadataDO) error {
	panic("implement me")
}

func (_h hybrid) DeleteEnvironment(ctx context.Context, envId string) error {
	panic("implement me")
}
