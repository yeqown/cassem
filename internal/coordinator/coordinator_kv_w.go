package coordinator

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"

	proto "google.golang.org/protobuf/proto"

	"github.com/yeqown/cassem/api/concept"
	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
	"github.com/yeqown/cassem/pkg/errorx"
)

var _ concept.KVWriteOnly = kvWriteOnly{}

// kvWriteOnly can read and write to cassemdb.
type kvWriteOnly struct {
	cassemdb apicassemdb.KVClient
}

// NewKVHybrid with endpoints these endpoints of cassemdb.
func NewKVHybrid(endpoints []string) (concept.KVWriteOnly, error) {
	cc, err := apicassemdb.DialWithMode(endpoints, apicassemdb.Mode_X)
	if err != nil {
		return nil, fmt.Errorf("NewWriter: %w", err)
	}

	return kvWriteOnly{
		cassemdb: apicassemdb.NewKVClient(cc),
	}, nil
}

func (_h kvWriteOnly) CreateElement(ctx context.Context,
	app, env, key string, raw []byte, contentTyp concept.ContentType) error {
	k := concept.GenElementKey(app, env, key)
	mdKey := concept.WithMetadataSuffix(k)
	version := int32(1)

	// set metadata of element
	md := &concept.ElementMetadata{
		LatestVersion:      version,
		UnpublishedVersion: version,
		UsingVersion:       0,
		UsingFingerprint:   "", // hash.MD5(raw)
		Key:                key,
		ContentType:        contentTyp,
		App:                app,
		Env:                env,
	}
	if err := _h.saveRaw(ctx, mdKey, md, 0, false); err != nil {
		return err
	}

	// set element with specified version
	ele := &concept.Element{
		Version:   version,
		Raw:       raw,
		Published: false,
	}
	if err := _h.saveRaw(ctx, concept.WithVersion(k, int(version)), ele, 0, false); err != nil {
		return err
	}

	return _h.saveElementOperation(ctx, app, env, key, concept.ElementOperation_SET, 0, version, "")
}

// UpdateElement add a new version to element, and update element's metadata info.
// 1. get metadata
// 2. lock element W operations to prevent concurrent writing operation.
// 3. create a Element
func (_h kvWriteOnly) UpdateElement(ctx context.Context, app, env, key string, raw []byte) error {
	k := concept.GenElementKey(app, env, key)
	md, err := _h.getElementMetadata(ctx, k)
	if err != nil {
		return err
	}
	// if there is an unpublished version, update is not allowed.
	if unpublished := md.GetUnpublishedVersion(); unpublished != 0 {
		return fmt.Errorf("unpublished version: %d: %w", int(unpublished), errorx.Err_ALREADY_EXISTS)
	}

	// marking version and update
	lastVersion := md.GetLatestVersion()
	version := md.LatestVersion + 1
	md.LatestVersion = version
	md.UnpublishedVersion = version

	// save new element version.
	ele := &concept.Element{
		Version:   version,
		Raw:       raw,
		Published: false,
	}
	if err = _h.saveRaw(ctx, concept.WithVersion(k, int(version)), ele, 0, false); err != nil {
		return err
	}

	// save metadata of element.
	if err = _h.saveRaw(ctx, concept.WithMetadataSuffix(k), md, 0, true); err != nil {
		return err
	}

	return _h.saveElementOperation(ctx, app, env, key, concept.ElementOperation_SET, lastVersion, version, "")
}

func (_h kvWriteOnly) DeleteElement(ctx context.Context, app, env, eltKey string) error {
	k := concept.GenElementKey(app, env, eltKey)
	_, err := _h.cassemdb.UnsetKV(ctx, &apicassemdb.UnsetKVReq{
		Key:   k,
		IsDir: true,
	})
	if err != nil {
		return err
	}

	return _h.deleteOperationPrefix(ctx, concept.GenElementOperationKeyPrefix(app, env, eltKey))
}

func (_h kvWriteOnly) CreateEnvironment(ctx context.Context, app, env string) error {
	k := concept.GenAppElementEnvKey(app, env)
	_, err := _h.cassemdb.SetKV(ctx, &apicassemdb.SetKVReq{
		Key:   k,
		IsDir: true,
		//Ttl:                  0,
		//Val:                  nil,
		//Overwrite:            false,
	})

	return err
}

func (_h kvWriteOnly) DeleteEnvironment(ctx context.Context, app, env string) error {
	k := concept.GenAppElementEnvKey(app, env)
	_, err := _h.cassemdb.UnsetKV(ctx, &apicassemdb.UnsetKVReq{
		Key:   k,
		IsDir: true,
		//Ttl:                  0,
		//Val:                  nil,
		//Overwrite:            false,
	})
	if err != nil {
		return err
	}

	return _h.deleteOperationPrefix(ctx, concept.GenAppEnvOperationKey(app, env))
}

// RollbackElementVersion reset element's latest published version as rollbackVersion
// elementMetadata.usingVersion => rollbackVersion
// elementMetadata.usingFingerprint = md5(rollbackVersion.raw)
func (_h kvWriteOnly) RollbackElementVersion(ctx context.Context, app string, env string, key string,
	rollbackVersion uint32) error {
	k := concept.GenElementKey(app, env, key)
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
		return fmt.Errorf("rollback version lte using version: %w", errorx.Err_INVALID_ARGUMENT)
	}

	lastUsingVersion := md.GetUsingVersion()
	md.UsingVersion = rollback.GetVersion()
	h := md5.New()
	h.Write(rollback.GetRaw())
	md.UsingFingerprint = hex.EncodeToString(h.Sum(nil))
	if err = _h.saveRaw(ctx, concept.WithMetadataSuffix(k), md, 0, true); err != nil {
		return err
	}

	return _h.saveElementOperation(ctx, app, env, key, concept.ElementOperation_PUBLISH,
		lastUsingVersion, rollback.GetVersion(), fmt.Sprintf("rollback to version %d", rollbackVersion))
}

// PublishElementVersion publish element version.
func (_h kvWriteOnly) PublishElementVersion(ctx context.Context, app string, env string, key string,
	publishVersion uint32) (*concept.Element, error) {
	k := concept.GenElementKey(app, env, key)
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

	lastUsingVersion := md.GetUsingVersion()

	// update metadata UsingVersion, UsingFingerprint, reset UnpublishedVersion.
	md.UsingVersion = publish.Version
	h := md5.New()
	h.Write(publish.GetRaw())
	md.UsingFingerprint = hex.EncodeToString(h.Sum(nil))
	md.UnpublishedVersion = 0
	if err = _h.saveRaw(ctx, concept.WithMetadataSuffix(k), md, 0, true); err != nil {
		return nil, err
	}

	// Update  version's published be TRUE.
	publish.Published = true
	if err = _h.saveRaw(ctx, concept.WithVersion(k, int(publishVersion)), publish, 0, true); err != nil {
		return nil, err
	}
	publish.Metadata = md
	if err = _h.saveElementOperation(ctx, app, env, key, concept.ElementOperation_PUBLISH,
		lastUsingVersion, publish.GetVersion(), ""); err != nil {
		return nil, err
	}
	return publish, nil
}

func (_h kvWriteOnly) CreateApp(ctx context.Context, md *concept.AppMetadata) error {
	k := concept.GenAppKey(md.Id)
	return _h.saveRaw(ctx, k, md, 0, false)
}

func (_h kvWriteOnly) DeleteApp(ctx context.Context, appId string) error {
	k := concept.GenAppKey(appId)
	eleKey := concept.GenAppElementKey(appId)

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

	return _h.deleteOperationPrefix(ctx, concept.GenAppOperationKey(appId))
}

// getElementMetadata returns element by specified version without metadata.
func (_h kvWriteOnly) getElementWithoutMetadata(ctx context.Context, key string, version uint32) (*concept.Element, error) {
	if version == 0 {
		return nil, fmt.Errorf("version could not be 0: %w", errorx.Err_INVALID_ARGUMENT)
	}

	r, err := _h.cassemdb.GetKV(ctx, &apicassemdb.GetKVReq{Key: concept.WithVersion(key, int(version))})
	if err != nil {
		return nil, err
	}
	ele := new(concept.Element)
	if err = concept.UnmarshalProto(r.GetEntity().GetVal(), ele); err != nil {
		return nil, err
	}

	return ele, nil
}

// getElementMetadata returns metadata of specified element.
func (_h kvWriteOnly) getElementMetadata(ctx context.Context, key string) (*concept.ElementMetadata, error) {
	r, err := _h.cassemdb.GetKV(ctx, &apicassemdb.GetKVReq{Key: concept.WithMetadataSuffix(key)})
	if err != nil {
		return nil, err
	}
	md := new(concept.ElementMetadata)
	if err = concept.UnmarshalProto(r.GetEntity().GetVal(), md); err != nil {
		return nil, err
	}

	return md, nil
}

func (_h kvWriteOnly) deleteOperationPrefix(ctx context.Context, key string) error {
	_, err := _h.cassemdb.UnsetKV(ctx, &apicassemdb.UnsetKVReq{
		Key:   key,
		IsDir: true,
	})
	if err != nil {
		return fmt.Errorf("kvWrite.deleteOperationPrefix: %w", err)
	}

	return nil
}

func (_h kvWriteOnly) saveElementOperation(ctx context.Context, app, env, key string,
	op concept.ElementOperation_Op, lastVersion, currentVersion int32, remark string) error {
	operatedAt := time.Now().UnixNano()
	opKey := concept.GenElementOperationKey(app, env, key, operatedAt)
	operation := &concept.ElementOperation{
		Operator:       concept.OperatorFromContext(ctx),
		OperatedAt:     operatedAt,
		OperatedKey:    key,
		Op:             op,
		LastVersion:    lastVersion,
		CurrentVersion: currentVersion,
		Remark:         remark,
	}

	return _h.saveRaw(ctx, opKey, operation, 0, false)
}

// saveRaw calls cassemdb.SetKV to save val.
// Notice that this method could not create directory which means SetKVReq{IsDir: false}.
func (_h kvWriteOnly) saveRaw(ctx context.Context, key string, val proto.Message, ttl int32, overwrite bool) error {
	bytes, err := concept.MarshalProto(val)
	if err != nil {
		return fmt.Errorf("%s: %w", err.Error(), errorx.Err_INTERNAL)
	}

	if _, err = _h.cassemdb.SetKV(ctx, &apicassemdb.SetKVReq{
		Key:       key,
		Ttl:       ttl,
		Val:       bytes,
		Overwrite: overwrite,
		//IsDir:     false,
	}); err != nil {
		return fmt.Errorf("kvWrite.saveRaw: %w", err)
	}

	return err
}
