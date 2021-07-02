package app

import (
	"context"

	pb "github.com/yeqown/cassem/internal/cassemdb/api/grpc/gen"
	"github.com/yeqown/cassem/pkg/conf"
	"github.com/yeqown/cassem/pkg/grpcx"
	"github.com/yeqown/cassem/pkg/hash"
	"github.com/yeqown/cassem/pkg/types"
)

// ICoordinator describes all methods should storage component should support.
type ICoordinator interface {
	GetElement(ctx context.Context, app, env, eltKey string, version int) (*types.VersionedEltDO, error)
	CreateElement(ctx context.Context, app, env, eltKey string, raw []byte) error
	UpdateElement(ctx context.Context, app, env, eltKey string, raw []byte) error
	DeleteElement(ctx context.Context, app, env, eltKey string) error

	GetElementOperations(
		ctx context.Context, app, env, eltKey string, start int) (ops []*types.EltOperateLog, next int, err error)

	CreateApp(ctx context.Context, md *types.AppMetadataDO) error
	GetApps(ctx context.Context) ([]*types.AppMetadataDO, error)
	DeleteApp(ctx context.Context, appId string) error

	CreateEnvironment(ctx context.Context, md *types.AppMetadataDO) error
	GetEnvironments(ctx context.Context) ([]*types.EnvMetadataDO, error)
	DeleteEnvironment(ctx context.Context, envId string) error
}

var (
	_ ICoordinator = app{}
)

type app struct {
	cassemdb pb.ApiClient
}

func New(config *conf.CassemAdminConfig) (*app, error) {
	cc, err := grpcx.DialCassemDB(config.CassemDBCluster)
	if err != nil {
		return nil, err
	}

	return &app{
		cassemdb: pb.NewApiClient(cc),
	}, nil
}

func (d app) GetElement(
	ctx context.Context, app, env, eltKey string, version int) (*types.VersionedEltDO, error) {
	// get metadata
	k := genEltKey(app, env, eltKey)
	r, err := d.cassemdb.GetKV(ctx, &pb.GetKVReq{Key: withMetadataSuffix(k)})
	if err != nil {
		return nil, err
	}
	md := new(types.EltMetadataDO)
	if err = md.Unmarshal(r.GetEntity().GetVal()); err != nil {
		return nil, err
	}

	if version <= 0 {
		version = md.LatestVersion
	}
	// get element with specified version
	r2, err2 := d.cassemdb.GetKV(ctx, &pb.GetKVReq{Key: withVersion(k, version)})
	if err2 != nil {
		return nil, err
	}
	elt := new(types.VersionedEltDO)
	if err2 = elt.Unmarshal(r2.GetEntity().GetVal()); err2 != nil {
		return nil, err2
	}
	elt.Metadata = md

	return elt, nil
}

func (d app) CreateElement(ctx context.Context, app, env, eltKey string, raw []byte) error {
	k := genEltKey(app, env, eltKey)
	mdKey := withMetadataSuffix(k)
	version := 1
	md := types.EltMetadataDO{
		LatestVersion:     version,
		LatestFingerprint: hash.MD5(raw),
		Key:               mdKey,
		ContentType:       0,
		App:               app,
		Env:               env,
	}
	// set metadata of element
	bytes, err := md.Marshal()
	if err != nil {
		return err
	}

	if _, err = d.cassemdb.SetKV(ctx, &pb.SetKVReq{
		Key: mdKey,
		Entity: &pb.Entity{
			Val: bytes,
		},
	}); err != nil {
		return err
	}

	versionedEltDo := types.VersionedEltDO{
		Version: version,
		Raw:     raw,
	}
	bytes, err = versionedEltDo.Marshal()
	if err != nil {
		return err
	}
	// set element with specified version
	if _, err = d.cassemdb.SetKV(ctx, &pb.SetKVReq{
		Key: withVersion(k, version),
		Entity: &pb.Entity{
			Val: bytes,
		},
	}); err != nil {
		return err
	}

	return nil
}

// UpdateElement add a new version to element, and update element's metadata info.
// 1. get metadata
// 2. lock element W operations to prevent concurrent writing operation.
// 3. create a VersionedEltDO
func (d app) UpdateElement(ctx context.Context, app, env, eltKey string, raw []byte) error {
	// get metadata
	k := genEltKey(app, env, eltKey)
	r, err := d.cassemdb.GetKV(ctx, &pb.GetKVReq{Key: withMetadataSuffix(k)})
	if err != nil {
		return err
	}
	md := new(types.EltMetadataDO)
	if err = md.Unmarshal(r.GetEntity().GetVal()); err != nil {
		return err
	}

	// version auto increased
	version := md.LatestVersion + 1
	md.LatestVersion = version
	elt := types.VersionedEltDO{
		Version: version,
		Raw:     raw,
	}

	// save new element version.
	bytes, err := elt.Marshal()
	if err != nil {
		return err
	}
	// set element with specified version
	if _, err = d.cassemdb.SetKV(ctx, &pb.SetKVReq{
		Key: withVersion(k, version),
		Entity: &pb.Entity{
			Val: bytes,
		},
	}); err != nil {
		return err
	}

	// update metadata
	bytes, _ = md.Marshal()
	// set element with specified version
	_, err = d.cassemdb.SetKV(ctx, &pb.SetKVReq{
		Key: withMetadataSuffix(k),
		Entity: &pb.Entity{
			Val: bytes,
		},
	})

	return err
}

func (d app) DeleteElement(ctx context.Context, app, env, eltKey string) error {
	k := genEltKey(app, env, eltKey)
	_, err := d.cassemdb.UnsetKV(ctx, &pb.UnsetKVReq{
		Key:   k,
		IsDir: true,
	})

	return err
}

func (d app) GetElementOperations(
	ctx context.Context, app, env, eltKey string, start int) (ops []*types.EltOperateLog, next int, err error) {
	panic("implement me")
}

func (d app) CreateApp(ctx context.Context, md *types.AppMetadataDO) error {
	panic("implement me")
}

func (d app) GetApps(ctx context.Context) ([]*types.AppMetadataDO, error) {
	panic("implement me")
}

func (d app) DeleteApp(ctx context.Context, appId string) error {
	panic("implement me")
}

func (d app) CreateEnvironment(ctx context.Context, md *types.AppMetadataDO) error {
	panic("implement me")
}

func (d app) GetEnvironments(ctx context.Context) ([]*types.EnvMetadataDO, error) {
	panic("implement me")
}

func (d app) DeleteEnvironment(ctx context.Context, envId string) error {
	panic("implement me")
}
