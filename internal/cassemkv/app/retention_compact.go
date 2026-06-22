package app

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/yeqown/cassem/api/concept"
	apikv "github.com/yeqown/cassem/api/kv"
)

type retentionCompactCoordinator interface {
	getKV(key string) (*apikv.Entity, error)
	iterate(param *rangeParam) (*apikv.RangeResp, error)
	unsetKV(ctx context.Context, param *unsetKVParam) error
}

type compactVersion struct {
	key       string
	version   int32
	createdAt int64
}

type compactOperation struct {
	key        string
	operatedAt int64
}

func compactElementHistory(coord retentionCompactCoordinator, req *apikv.CompactElementHistoryReq, now time.Time) (*apikv.CompactElementHistoryResp, error) {
	if coord == nil {
		return nil, fmt.Errorf("compact element history: nil coordinator")
	}
	if req == nil || req.GetElementKey() == "" {
		return nil, fmt.Errorf("compact element history: empty element key")
	}

	pageSize := int(req.GetPageSize())
	if pageSize <= 0 {
		pageSize = 100
	}

	metadata, err := loadElementMetadataForCompact(coord, req.GetElementKey())
	if err != nil {
		return nil, err
	}

	versions, failedVersionKeys, err := loadCompactVersions(coord, req.GetElementKey(), pageSize)
	if err != nil {
		return nil, err
	}

	operations, failedOperationKeys, err := loadCompactOperations(coord, metadata.GetApp(), metadata.GetEnv(), metadata.GetKey(), pageSize)
	if err != nil {
		return nil, err
	}

	versionCutoff := safeRetentionCutoff(now.Unix(), req.GetKeepVersionSeconds(), 1)
	keep := retainedVersionSet(metadata, versions, req.GetKeepVersionCount(), versionCutoff)
	resp := &apikv.CompactElementHistoryResp{
		ScannedVersions:   int32(len(versions) + len(failedVersionKeys)),
		ScannedOperations: int32(len(operations) + len(failedOperationKeys)),
		FailedKeys:        append(append([]string(nil), failedVersionKeys...), failedOperationKeys...),
	}

	for _, version := range versions {
		if keep[version.version] {
			continue
		}
		if err := coord.unsetKV(context.Background(), &unsetKVParam{key: version.key}); err != nil {
			resp.FailedKeys = append(resp.FailedKeys, version.key)
			continue
		}
		resp.DeletedVersions++
	}

	operationCutoff := safeRetentionCutoff(now.UnixNano(), req.GetKeepOperationSeconds(), int64(time.Second))
	for _, operation := range operations {
		if operation.operatedAt >= operationCutoff {
			continue
		}
		if err := coord.unsetKV(context.Background(), &unsetKVParam{key: operation.key}); err != nil {
			resp.FailedKeys = append(resp.FailedKeys, operation.key)
			continue
		}
		resp.DeletedOperations++
	}

	if len(resp.FailedKeys) > 0 {
		resp.Error = "partial cleanup failure"
	}

	return resp, nil
}

func loadElementMetadataForCompact(coord retentionCompactCoordinator, root string) (*concept.ElementMetadata, error) {
	entity, err := coord.getKV(concept.WithMetadataSuffix(root))
	if err != nil {
		return nil, fmt.Errorf("load compact metadata %q: %w", root, err)
	}

	metadata := new(concept.ElementMetadata)
	if err := concept.UnmarshalProto(entity.GetVal(), metadata); err != nil {
		return nil, fmt.Errorf("unmarshal compact metadata %q: %w", root, err)
	}

	return metadata, nil
}

func loadCompactVersions(coord retentionCompactCoordinator, root string, pageSize int) ([]compactVersion, []string, error) {
	versions := make([]compactVersion, 0)
	failedKeys := make([]string, 0)
	seek := "v"

	for {
		resp, err := coord.iterate(&rangeParam{key: root, seek: seek, limit: pageSize})
		if err != nil {
			return nil, nil, fmt.Errorf("range compact versions %q: %w", root, err)
		}

		for _, entity := range resp.GetEntities() {
			version, ok := versionFromStorageKey(root, entity.GetKey())
			if !ok {
				failedKeys = append(failedKeys, entity.GetKey())
				continue
			}
			versions = append(versions, compactVersion{
				key:       entity.GetKey(),
				version:   version,
				createdAt: entity.GetCreatedAt(),
			})
		}

		if !resp.GetHasMore() || resp.GetNextSeekKey() == "" {
			break
		}
		seek = resp.GetNextSeekKey()
	}

	return versions, failedKeys, nil
}

func loadCompactOperations(coord retentionCompactCoordinator, appID, env, key string, pageSize int) ([]compactOperation, []string, error) {
	operations := make([]compactOperation, 0)
	failedKeys := make([]string, 0)
	seek := ""
	dir := concept.GenElementOperationDirKey(appID, env, key)

	for {
		resp, err := coord.iterate(&rangeParam{key: dir, seek: seek, limit: pageSize})
		if err != nil {
			return nil, nil, fmt.Errorf("range compact operations %q: %w", dir, err)
		}

		for _, entity := range resp.GetEntities() {
			op := new(concept.ElementOperation)
			if err := concept.UnmarshalProto(entity.GetVal(), op); err != nil {
				failedKeys = append(failedKeys, entity.GetKey())
				continue
			}
			operations = append(operations, compactOperation{
				key:        entity.GetKey(),
				operatedAt: op.GetOperatedAt(),
			})
		}

		if !resp.GetHasMore() || resp.GetNextSeekKey() == "" {
			break
		}
		seek = resp.GetNextSeekKey()
	}

	return operations, failedKeys, nil
}

func safeRetentionCutoff(nowValue, keepSeconds, unit int64) int64 {
	if keepSeconds <= 0 {
		return nowValue
	}
	if keepSeconds > math.MaxInt64/unit {
		return 0
	}

	retention := keepSeconds * unit
	if retention > nowValue {
		return 0
	}
	return nowValue - retention
}

func retainedVersionSet(metadata *concept.ElementMetadata, versions []compactVersion, keepCount int32, keepCreatedAfter int64) map[int32]bool {
	if keepCount < 0 {
		keepCount = 0
	}

	keep := make(map[int32]bool, len(versions))
	if metadata.GetUsingVersion() > 0 {
		keep[metadata.GetUsingVersion()] = true
	}
	if metadata.GetUnpublishedVersion() > 0 {
		keep[metadata.GetUnpublishedVersion()] = true
	}

	for _, version := range versions {
		if version.createdAt >= keepCreatedAfter {
			keep[version.version] = true
		}
	}

	sortedVersions := append([]compactVersion(nil), versions...)
	sort.Slice(sortedVersions, func(i, j int) bool {
		return sortedVersions[i].version > sortedVersions[j].version
	})

	for idx, version := range sortedVersions {
		if idx >= int(keepCount) {
			break
		}
		keep[version.version] = true
	}

	return keep
}

func versionFromStorageKey(root, key string) (int32, bool) {
	leaf := strings.TrimPrefix(key, root+"/")
	if !strings.HasPrefix(leaf, "v") || strings.ContainsRune(leaf, '/') {
		return 0, false
	}

	version, err := strconv.ParseInt(strings.TrimPrefix(leaf, "v"), 10, 32)
	if err != nil || version <= 0 {
		return 0, false
	}

	return int32(version), true
}
