package concept

import (
	"context"
	"strconv"

	proto "github.com/golang/protobuf/proto"
	"github.com/pkg/errors"

	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
	"github.com/yeqown/cassem/pkg/errorx"
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
	app, env, key string, raw []byte, contentTyp ContentType) error {
	k := genElementKey(app, env, key)
	mdKey := withMetadataSuffix(k)
	version := int32(1)

	// set metadata of element
	md := &ElementMetadata{
		LatestVersion:      version,
		UnpublishedVersion: version,
		UsingVersion:       0,
		UsingFingerprint:   "", // hash.MD5(raw)
		Key:                mdKey,
		ContentType:        contentTyp,
		App:                app,
		Env:                env,
	}
	if err := _h.saveRaw(ctx, mdKey, md, 0, false); err != nil {
		return err
	}

	// set element with specified version
	ele := &Element{
		Version:   version,
		Raw:       raw,
		Published: false,
	}
	if err := _h.saveRaw(ctx, withVersion(k, int(version)), ele, 0, false); err != nil {
		return err
	}

	return nil
}

// UpdateElement add a new version to element, and update element's metadata info.
// 1. get metadata
// 2. lock element W operations to prevent concurrent writing operation.
// 3. create a Element
func (_h kvWriteOnly) UpdateElement(ctx context.Context, app, env, key string, raw []byte) error {
	k := genElementKey(app, env, key)
	md, err := _h.getElementMetadata(ctx, k)
	if err != nil {
		return err
	}
	// if there is an unpublished version, update is not allowed.
	if unpublished := md.GetUnpublishedVersion(); unpublished != 0 {
		return errors.Wrap(errorx.Err_ALREADY_EXISTS,
			"unpublished version: "+strconv.Itoa(int(unpublished)))
	}

	// marking version and update
	version := md.LatestVersion + 1
	md.LatestVersion = version
	md.UnpublishedVersion = version

	// save new element version.
	ele := &Element{
		Version:   version,
		Raw:       raw,
		Published: false,
	}
	if err = _h.saveRaw(ctx, withVersion(k, int(version)), ele, 0, false); err != nil {
		return err
	}

	// save metadata of element.
	return _h.saveRaw(ctx, withMetadataSuffix(k), md, 0, true)
}

func (_h kvWriteOnly) DeleteElement(ctx context.Context, app, env, eltKey string) error {
	k := genElementKey(app, env, eltKey)
	_, err := _h.cassemdb.UnsetKV(ctx, &apicassemdb.UnsetKVReq{
		Key:   k,
		IsDir: true,
	})

	return err
}

func (_h kvWriteOnly) CreateEnvironment(ctx context.Context, app, env string) error {
	k := genAppElementEnvKey(app, env)
	_, err := _h.cassemdb.SetKV(ctx, &apicassemdb.SetKVReq{
		Key:   k,
		IsDir: true,
		//Ttl:                  0,
		//Val:                  nil,
		//Overwrite:            false,
	})

	return err
}

// RollbackElementVersion reset element's latest published version as rollbackVersion
// elementMetadata.usingVersion => rollbackVersion
// elementMetadata.usingFingerprint = md5(rollbackVersion.raw)
//
func (_h kvWriteOnly) RollbackElementVersion(ctx context.Context, app string, env string, key string,
	rollbackVersion uint32) error {
	k := genElementKey(app, env, key)
	md, err := _h.getElementMetadata(ctx, k)
	if err != nil {
		return err
	}

	// check rollback version is available
	rollback, err := _h.getElementWithoutMetadata(ctx, k, rollbackVersion)
	if err != nil {
		return err
	}

	// could not roll back to bigger version than now using version.
	if md.GetUsingVersion() <= int32(rollbackVersion) {
		return errors.Wrap(errorx.Err_INVALID_ARGUMENT, "rollback version lte using version")
	}

	md.UsingVersion = rollback.GetVersion()
	md.UsingFingerprint = hash.MD5(rollback.GetRaw())
	return _h.saveRaw(ctx, withMetadataSuffix(k), md, 0, true)
}

// PublishElementVersion publish element version.
func (_h kvWriteOnly) PublishElementVersion(ctx context.Context, app string, env string, key string,
	publishVersion uint32) (*Element, error) {
	k := genElementKey(app, env, key)
	md, err := _h.getElementMetadata(ctx, k)
	if err != nil {
		return nil, err
	}

	if publishVersion == 0 && md.UnpublishedVersion != 0 {
		publishVersion = uint32(md.GetUnpublishedVersion())
	}

	// There is no available version
	if publishVersion == 0 {
		return nil, nil
	}

	// Check the element has  version or not.
	publish, err := _h.getElementWithoutMetadata(ctx, k, publishVersion)
	if err != nil {
		return nil, err
	}

	// update metadata UsingVersion, UsingFingerprint, reset UnpublishedVersion.
	md.UsingVersion = publish.Version
	md.UsingFingerprint = hash.MD5(publish.GetRaw())
	md.UnpublishedVersion = 0
	if err = _h.saveRaw(ctx, withMetadataSuffix(k), md, 0, true); err != nil {
		return nil, err
	}

	// Update  version's published be TRUE.
	publish.Published = true
	err = _h.saveRaw(ctx, withVersion(k, int(publishVersion)), publish, 0, true)
	publish.Metadata = md
	return publish, err
}

func (_h kvWriteOnly) CreateApp(ctx context.Context, md *AppMetadata) error {
	k := genAppKey(md.Id)
	return _h.saveRaw(ctx, k, md, 0, false)
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

// getElementMetadata returns element by specified version without metadata.
func (_h kvWriteOnly) getElementWithoutMetadata(ctx context.Context, key string, version uint32) (*Element, error) {
	if version == 0 {
		return nil, errors.Wrap(errorx.Err_INVALID_ARGUMENT, "version could not be 0")
	}

	r, err := _h.cassemdb.GetKV(ctx, &apicassemdb.GetKVReq{Key: withVersion(key, int(version))})
	if err != nil {
		return nil, err
	}
	ele := new(Element)
	if err = UnmarshalProto(r.GetEntity().GetVal(), ele); err != nil {
		return nil, err
	}

	return ele, nil
}

// getElementMetadata returns metadata of specified element.
func (_h kvWriteOnly) getElementMetadata(ctx context.Context, key string) (*ElementMetadata, error) {
	r, err := _h.cassemdb.GetKV(ctx, &apicassemdb.GetKVReq{Key: withMetadataSuffix(key)})
	if err != nil {
		return nil, err
	}
	md := new(ElementMetadata)
	if err = UnmarshalProto(r.GetEntity().GetVal(), md); err != nil {
		return nil, err
	}

	return md, nil
}

// saveRaw calls cassemdb.SetKV to save val.
// Notice that this method could not create directory which means SetKVReq{IsDir: false}.
func (_h kvWriteOnly) saveRaw(ctx context.Context, key string, val proto.Message, ttl int32, overwrite bool) error {
	bytes, err := MarshalProto(val)
	if err != nil {
		return errors.Wrap(errorx.Err_INTERNAL, err.Error())
	}

	if _, err = _h.cassemdb.SetKV(ctx, &apicassemdb.SetKVReq{
		Key:       key,
		Ttl:       ttl,
		Val:       bytes,
		Overwrite: overwrite,
		//IsDir:     false,
	}); err != nil {
		return errors.Wrap(err, "kvWrite.saveRaw")
	}

	return err
}
