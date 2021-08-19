package concept

import (
	"context"

	"github.com/pkg/errors"

	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
	"github.com/yeqown/cassem/pkg/hash"
)

var _ KVWriteOnly = kvWriteOnly{}

// kvWriteOnly can read and write to cassemdb.
type kvWriteOnly struct {
	cassemdb apicassemdb.KVClient
}

// NewKVHybrid with endpoints these endpoints of cassemdb.
func NewKVHybrid(endpoints []string) (KVWriteOnly, error) {
	cc, err := apicassemdb.DialWithMode(endpoints, apicassemdb.Mode_X)
	if err != nil {
		return nil, errors.Wrap(err, "NewWriter")
	}

	return kvWriteOnly{
		cassemdb: apicassemdb.NewKVClient(cc),
	}, nil
}

func (_h kvWriteOnly) CreateElement(ctx context.Context,
	app, env, eltKey string, raw []byte, contentTyp ContentType) error {
	k := genElementKey(app, env, eltKey)
	mdKey := withMetadataSuffix(k)
	version := 1
	// set metadata of element
	bytes, err := MarshalProto(&ElementMetadata{
		LatestVersion:     int32(version),
		LatestFingerprint: hash.MD5(raw),
		Key:               mdKey,
		ContentType:       contentTyp,
		App:               app,
		Env:               env,
	})
	if err != nil {
		return err
	}

	if _, err = _h.cassemdb.SetKV(ctx, &apicassemdb.SetKVReq{
		Key:       mdKey,
		Val:       bytes,
		IsDir:     false,
		Overwrite: true,
		Ttl:       0,
	}); err != nil {
		return err
	}

	bytes, err = MarshalProto(&Element{Version: int32(version), Raw: raw})
	if err != nil {
		return err
	}
	// set element with specified version
	if _, err = _h.cassemdb.SetKV(ctx, &apicassemdb.SetKVReq{
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
// 3. create a Element
func (_h kvWriteOnly) UpdateElement(ctx context.Context, app, env, eltKey string, raw []byte) error {
	// get metadata
	k := genElementKey(app, env, eltKey)
	r, err := _h.cassemdb.GetKV(ctx, &apicassemdb.GetKVReq{Key: withMetadataSuffix(k)})
	if err != nil {
		return err
	}
	md := new(ElementMetadata)
	if err = UnmarshalProto(r.GetEntity().GetVal(), md); err != nil {
		return err
	}

	// version auto increased
	version := md.LatestVersion + 1
	md.LatestVersion = version

	// save new element version.
	bytes, err := MarshalProto(&Element{Version: version, Raw: raw})
	if err != nil {
		return err
	}
	// set element with specified version
	if _, err = _h.cassemdb.SetKV(ctx, &apicassemdb.SetKVReq{
		Key:       withVersion(k, int(version)),
		Val:       bytes,
		IsDir:     false,
		Overwrite: true,
		Ttl:       0,
	}); err != nil {
		return err
	}

	bytes, _ = MarshalProto(md)
	// set element with specified version
	_, err = _h.cassemdb.SetKV(ctx, &apicassemdb.SetKVReq{
		Key:       withMetadataSuffix(k),
		Val:       bytes,
		IsDir:     false,
		Overwrite: true,
		Ttl:       0,
	})

	return err
}

func (_h kvWriteOnly) DeleteElement(ctx context.Context, app, env, eltKey string) error {
	k := genElementKey(app, env, eltKey)
	_, err := _h.cassemdb.UnsetKV(ctx, &apicassemdb.UnsetKVReq{
		Key:   k,
		IsDir: true,
	})

	return err
}

func (_h kvWriteOnly) CreateApp(ctx context.Context, md *AppMetadata) error {
	k := genAppKey(md.Id)
	bytes, _ := MarshalProto(md)
	_, err := _h.cassemdb.SetKV(ctx, &apicassemdb.SetKVReq{
		Key:       k,
		IsDir:     false,
		Ttl:       0,
		Val:       bytes,
		Overwrite: false,
	})
	return err
}

func (_h kvWriteOnly) DeleteApp(ctx context.Context, appId string) error {
	k := genAppKey(appId)
	eleKey := genAppElementKey(appId)

	_, err := _h.cassemdb.UnsetKV(ctx, &apicassemdb.UnsetKVReq{
		Key:   eleKey,
		IsDir: true,
	})
	if err != nil {
		return err
	}
	_, err = _h.cassemdb.UnsetKV(ctx, &apicassemdb.UnsetKVReq{
		Key:   k,
		IsDir: false,
	})
	if err != nil {
		return err
	}

	return nil
}
