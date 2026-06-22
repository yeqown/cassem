package coordinator

import (
	"context"
	"errors"
	"fmt"
	apikv "github.com/yeqown/cassem/api/kv"
	"strings"

	"github.com/yeqown/log"
	"google.golang.org/grpc/codes"

	"github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/pkg/errorx"
)

const (
	_VERSION_PREFIX = "v"
	_APP_PREFIX     = "cassem/apps"
)

// kvReadOnly manages all read operation from cassemdb, it is allowed to read only.
type kvReadOnly struct {
	cassemdb apikv.KVClient
}

// NewKVReader with endpoints these endpoints of cassemdb.
func NewKVReader(endpoints []string) (concept.KVReadOnly, error) {
	cc, err := apikv.DialWithMode(endpoints, apikv.Mode_R)
	if err != nil {
		return nil, fmt.Errorf("NewWriter: %w", err)
	}

	return kvReadOnly{
		cassemdb: apikv.NewKVClient(cc),
	}, nil
}

func (_r kvReadOnly) GetElementWithVersion(
	ctx context.Context, app, env, key string, version int) (*concept.Element, error) {
	// get metadata
	k := concept.GenElementKey(app, env, key)
	r1, err := _r.cassemdb.GetKV(ctx, &apikv.GetKVReq{Key: concept.WithMetadataSuffix(k)})
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
		return &concept.Element{Metadata: md}, nil
	}

	// get element with specified version
	r2, err2 := _r.cassemdb.GetKV(ctx, &apikv.GetKVReq{Key: concept.WithVersion(k, version)})
	if err2 != nil {
		return nil, err2
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

	r, err := _r.cassemdb.GetKVs(ctx, &apikv.GetKVsReq{
		Keys: []string{concept.WithMetadataSuffix(k)},
	})
	if err != nil {
		return nil, fmt.Errorf("kvReadOnly.GetElementVersions: %w", err)
	}
	if err = getKVsNonNotFoundErrors(r); err != nil {
		return nil, fmt.Errorf("kvReadOnly.GetElementVersions: %w", err)
	}

	if len(seek) == 0 {
		// default seek to skip metadata
		seek = _VERSION_PREFIX
	}

	r2, err := _r.cassemdb.Range(ctx, &apikv.RangeReq{
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
	ctx context.Context, app, env string, seek string, limit int, query string) (*concept.GetElementsResult, error) {
	k := concept.GenAppElementEnvKey(app, env)

	log.
		WithFields(log.Fields{
			"app":   app,
			"env":   env,
			"seek":  seek,
			"limit": limit,
			"query": query,
			"k":     k,
		}).
		Debug("kvReadOnly.GetElements enter")
	if strings.TrimSpace(query) == "" {
		r, err := _r.cassemdb.Range(ctx, &apikv.RangeReq{
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
			keys = append(keys, concept.ExtractPureKey(v.GetKey()))
		}

		result.Elements, err = _r.getElementsByKeys(ctx, app, env, keys, false)
		return result, err
	}

	needle := strings.ToLower(strings.TrimSpace(query))
	matched := make([]string, 0, limit+1)
	nextSeek := seek
	for len(matched) <= limit {
		r, err := _r.cassemdb.Range(ctx, &apikv.RangeReq{
			Key:   k,
			Seek:  nextSeek,
			Limit: int32(limit),
		})
		if err != nil {
			return nil, err
		}

		for _, v := range r.GetEntities() {
			key := concept.ExtractPureKey(v.GetKey())
			if strings.Contains(strings.ToLower(key), needle) {
				matched = append(matched, key)
				if len(matched) > limit {
					break
				}
			}
		}

		if len(matched) > limit {
			break
		}
		if !r.GetHasMore() {
			elements, err := _r.getElementsByKeys(ctx, app, env, matched, false)
			if err != nil {
				return nil, err
			}
			return &concept.GetElementsResult{Elements: elements}, nil
		}
		nextSeek = r.GetNextSeekKey()
	}

	elements, err := _r.getElementsByKeys(ctx, app, env, matched[:limit], false)
	if err != nil {
		return nil, err
	}
	return &concept.GetElementsResult{
		CommonPager: concept.CommonPager{
			HasMore:  true,
			NextSeek: matched[limit],
		},
		Elements: elements,
	}, nil
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

func getKVsErrors(resp *apikv.GetKVsResp, ignoreNotFound bool) error {
	var errs []error
	for _, item := range resp.GetErrors() {
		if ignoreNotFound && item.GetCode() == codes.NotFound.String() {
			continue
		}
		errs = append(errs, keyErrorToError(item))
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func keyErrorToError(item *apikv.KeyError) error {
	msg := fmt.Errorf("key %s: %s: %s", item.GetKey(), item.GetCode(), item.GetMessage())
	sentinel, ok := keyErrorCodeSentinel(item.GetCode())
	if !ok {
		return msg
	}
	return errors.Join(msg, sentinel)
}

func keyErrorCodeSentinel(code string) (error, bool) {
	switch code {
	case codes.Canceled.String():
		return errorx.Err_CANCELLED, true
	case codes.Unknown.String():
		return errorx.Err_UNKNOWN, true
	case codes.InvalidArgument.String():
		return errorx.Err_INVALID_ARGUMENT, true
	case codes.DeadlineExceeded.String():
		return errorx.Err_DEADLINE_EXCEEDED, true
	case codes.NotFound.String():
		return errorx.Err_NOT_FOUND, true
	case codes.AlreadyExists.String():
		return errorx.Err_ALREADY_EXISTS, true
	case codes.PermissionDenied.String():
		return errorx.Err_PERMISSION_DENIED, true
	case codes.ResourceExhausted.String():
		return errorx.Err_RESOURCE_EXHAUSTED, true
	case codes.FailedPrecondition.String():
		return errorx.Err_FAILED_PRECONDITION, true
	case codes.Aborted.String():
		return errorx.Err_ABORTED, true
	case codes.OutOfRange.String():
		return errorx.Err_OUT_OF_RANGE, true
	case codes.Unimplemented.String():
		return errorx.Err_UNIMPLEMENTED, true
	case codes.Internal.String():
		return errorx.Err_INTERNAL, true
	case codes.Unavailable.String():
		return errorx.Err_UNAVAILABLE, true
	case codes.DataLoss.String():
		return errorx.Err_DATA_LOSS, true
	case codes.Unauthenticated.String():
		return errorx.Err_UNAUTHENTICATED, true
	default:
		return nil, false
	}
}

func getKVsNonNotFoundErrors(resp *apikv.GetKVsResp) error {
	return getKVsErrors(resp, true)
}

func getKVsAnyErrors(resp *apikv.GetKVsResp) error {
	return getKVsErrors(resp, false)
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
	r, err := _r.cassemdb.GetKVs(ctx, &apikv.GetKVsReq{
		Keys: mdKeys,
	})
	if err != nil {
		return nil, fmt.Errorf("kvReadOnly.getElementsByKeys: %v", err)
	}
	if err = getKVsNonNotFoundErrors(r); err != nil {
		return nil, fmt.Errorf("kvReadOnly.getElementsByKeys.metadata: %w", err)
	}

	// DONE(@yeqown): replace this part of code with convertFromEntitiesToMetadata
	eleVersionKeys, _, metadataMapping := ConvertFromEntitiesToMetadata(r.GetEntities(), wipeUnpublish)
	if len(eleVersionKeys) == 0 {
		return []*concept.Element{}, nil
	}
	r2, err2 := _r.cassemdb.GetKVs(ctx, &apikv.GetKVsReq{
		Keys: eleVersionKeys,
	})
	if err2 != nil {
		return nil, fmt.Errorf("kvReadOnly.getElementsByKeys: %v", err2)
	}
	if err2 = getKVsAnyErrors(r2); err2 != nil {
		return nil, fmt.Errorf("kvReadOnly.getElementsByKeys.version: %w", err2)
	}

	out := ConvertFromEntitiesToElements(r2.GetEntities(), metadataMapping)

	return out, nil
}

func (_r kvReadOnly) GetElementOperations(
	ctx context.Context, app, env, eltKey string, seek string, limit int) (*concept.GetElementOperationsResult, error) {
	r, err := _r.cassemdb.Range(ctx, &apikv.RangeReq{
		Key:   concept.GenElementOperationDirKey(app, env, eltKey),
		Seek:  seek,
		Limit: int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("kvReadOnly.GetElementOperations: %w", err)
	}

	result := &concept.GetElementOperationsResult{
		CommonPager: concept.CommonPager{
			HasMore:  r.GetHasMore(),
			NextSeek: r.GetNextSeekKey(),
		},
		Operations: make([]*concept.ElementOperation, 0, len(r.GetEntities())),
	}
	for _, entity := range r.GetEntities() {
		op := new(concept.ElementOperation)
		if err = concept.UnmarshalProto(entity.GetVal(), op); err != nil {
			continue
		}
		result.Operations = append(result.Operations, op)
	}

	return result, nil
}

func (_r kvReadOnly) GetApp(ctx context.Context, app string) (*concept.AppMetadata, error) {
	k := concept.GenAppKey(app)
	r, err := _r.cassemdb.GetKV(ctx, &apikv.GetKVReq{
		Key: k,
	})
	if err != nil {
		return nil, err
	}

	md := new(concept.AppMetadata)
	err = concept.UnmarshalProto(r.GetEntity().GetVal(), md)
	return md, err
}

func (_r kvReadOnly) GetApps(ctx context.Context, seek string, limit int, query string) (*concept.GetAppsResult, error) {
	if strings.TrimSpace(query) == "" {
		r, err := _r.cassemdb.Range(ctx, &apikv.RangeReq{
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

	needle := strings.ToLower(strings.TrimSpace(query))
	matched := make([]*concept.AppMetadata, 0, limit+1)
	nextSeek := seek
	for len(matched) <= limit {
		r, err := _r.cassemdb.Range(ctx, &apikv.RangeReq{
			Key:   _APP_PREFIX,
			Seek:  nextSeek,
			Limit: int32(limit),
		})
		if err != nil {
			return nil, err
		}

		for _, v := range r.GetEntities() {
			md := new(concept.AppMetadata)
			_ = concept.UnmarshalProto(v.Val, md)
			if strings.Contains(strings.ToLower(md.GetId()), needle) || strings.Contains(strings.ToLower(md.GetDescription()), needle) {
				matched = append(matched, md)
				if len(matched) > limit {
					break
				}
			}
		}

		if len(matched) > limit {
			break
		}
		if !r.GetHasMore() {
			return &concept.GetAppsResult{Apps: matched}, nil
		}
		nextSeek = r.GetNextSeekKey()
	}

	return &concept.GetAppsResult{
		CommonPager: concept.CommonPager{
			HasMore:  true,
			NextSeek: matched[limit].GetId(),
		},
		Apps: matched[:limit],
	}, nil
}

func (_r kvReadOnly) GetEnvironments(ctx context.Context, app, seek string, limit int) (*concept.GetAppEnvsResult, error) {
	k := concept.GenAppElementKey(app)
	r, err := _r.cassemdb.Range(ctx, &apikv.RangeReq{
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
