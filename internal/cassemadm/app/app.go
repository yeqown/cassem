package app

import (
	"context"

	"github.com/yeqown/cassem/internal/cassemadm/infras"
	"github.com/yeqown/cassem/internal/cassemdb/api"
	cassemdb_pb "github.com/yeqown/cassem/internal/cassemdb/api/gen"
	"github.com/yeqown/cassem/pkg/conf"
	"github.com/yeqown/cassem/pkg/hash"
)

// ICoordinator describes all methods should storage component should support.
type ICoordinator interface {
	GetElement(ctx context.Context, app, env, eltKey string, version int) (*infras.VersionedEltDO, error)
	CreateElement(ctx context.Context, app, env, eltKey string, raw []byte) error
	UpdateElement(ctx context.Context, app, env, eltKey string, raw []byte) error
	DeleteElement(ctx context.Context, app, env, eltKey string) error

	GetElementOperations(
		ctx context.Context, app, env, eltKey string, start int) (ops []*infras.EltOperateLog, next int, err error)

	CreateApp(ctx context.Context, md *infras.AppMetadataDO) error
	GetApps(ctx context.Context) ([]*infras.AppMetadataDO, error)
	DeleteApp(ctx context.Context, appId string) error

	CreateEnvironment(ctx context.Context, md *infras.AppMetadataDO) error
	GetEnvironments(ctx context.Context) ([]*infras.EnvMetadataDO, error)
	DeleteEnvironment(ctx context.Context, envId string) error
}

var (
	_ ICoordinator = app{}
)

type app struct {
	cassemdb cassemdb_pb.KVClient
}

func New(config *conf.CassemAdminConfig) (*app, error) {
	cc, err := api.Dial(config.CassemDBCluster)
	if err != nil {
		return nil, err
	}

	return &app{
		cassemdb: cassemdb_pb.NewKVClient(cc),
	}, nil
}

func (d app) GetElement(
	ctx context.Context, app, env, eltKey string, version int) (*infras.VersionedEltDO, error) {
	// get metadata
	k := genEltKey(app, env, eltKey)
	r, err := d.cassemdb.GetKV(ctx, &cassemdb_pb.GetKVReq{Key: withMetadataSuffix(k)})
	if err != nil {
		return nil, err
	}
	md := new(infras.EltMetadataDO)
	if err = md.Unmarshal(r.GetEntity().GetVal()); err != nil {
		return nil, err
	}

	if version <= 0 {
		version = md.LatestVersion
	}
	// get element with specified version
	r2, err2 := d.cassemdb.GetKV(ctx, &cassemdb_pb.GetKVReq{Key: withVersion(k, version)})
	if err2 != nil {
		return nil, err
	}
	elt := new(infras.VersionedEltDO)
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
	md := infras.EltMetadataDO{
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

	if _, err = d.cassemdb.SetKV(ctx, &cassemdb_pb.SetKVReq{
		Key:       mdKey,
		Val:       bytes,
		IsDir:     false,
		Overwrite: true,
		Ttl:       0,
	}); err != nil {
		return err
	}

	versionedEltDo := infras.VersionedEltDO{
		Version: version,
		Raw:     raw,
	}
	bytes, err = versionedEltDo.Marshal()
	if err != nil {
		return err
	}
	// set element with specified version
	if _, err = d.cassemdb.SetKV(ctx, &cassemdb_pb.SetKVReq{
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
func (d app) UpdateElement(ctx context.Context, app, env, eltKey string, raw []byte) error {
	// get metadata
	k := genEltKey(app, env, eltKey)
	r, err := d.cassemdb.GetKV(ctx, &cassemdb_pb.GetKVReq{Key: withMetadataSuffix(k)})
	if err != nil {
		return err
	}
	md := new(infras.EltMetadataDO)
	if err = md.Unmarshal(r.GetEntity().GetVal()); err != nil {
		return err
	}

	// version auto increased
	version := md.LatestVersion + 1
	md.LatestVersion = version
	elt := infras.VersionedEltDO{
		Version: version,
		Raw:     raw,
	}

	// save new element version.
	bytes, err := elt.Marshal()
	if err != nil {
		return err
	}
	// set element with specified version
	if _, err = d.cassemdb.SetKV(ctx, &cassemdb_pb.SetKVReq{
		Key:       withVersion(k, version),
		Val:       bytes,
		IsDir:     false,
		Overwrite: true,
		Ttl:       0,
	}); err != nil {
		return err
	}

	// update metadata
	bytes, _ = md.Marshal()
	// set element with specified version
	_, err = d.cassemdb.SetKV(ctx, &cassemdb_pb.SetKVReq{
		Key:       withMetadataSuffix(k),
		Val:       bytes,
		IsDir:     false,
		Overwrite: true,
		Ttl:       0,
	})

	return err
}

func (d app) DeleteElement(ctx context.Context, app, env, eltKey string) error {
	k := genEltKey(app, env, eltKey)
	_, err := d.cassemdb.UnsetKV(ctx, &cassemdb_pb.UnsetKVReq{
		Key:   k,
		IsDir: true,
	})

	return err
}

func (d app) GetElementOperations(
	ctx context.Context, app, env, eltKey string, start int) (ops []*infras.EltOperateLog, next int, err error) {
	panic("implement me")
}

func (d app) CreateApp(ctx context.Context, md *infras.AppMetadataDO) error {
	panic("implement me")
}

func (d app) GetApps(ctx context.Context) ([]*infras.AppMetadataDO, error) {
	panic("implement me")
}

func (d app) DeleteApp(ctx context.Context, appId string) error {
	panic("implement me")
}

func (d app) CreateEnvironment(ctx context.Context, md *infras.AppMetadataDO) error {
	panic("implement me")
}

func (d app) GetEnvironments(ctx context.Context) ([]*infras.EnvMetadataDO, error) {
	panic("implement me")
}

func (d app) DeleteEnvironment(ctx context.Context, envId string) error {
	panic("implement me")
}
