package app

import (
	"context"
	"errors"
	"math"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/yeqown/cassem/api/concept"
	apikv "github.com/yeqown/cassem/api/kv"
)

type retentionCompactFakeCoord struct {
	entities map[string]*apikv.Entity
	deleted  []string
	fail     map[string]error
	rangeErr map[string]error
}

func newRetentionCompactFakeCoord() *retentionCompactFakeCoord {
	return &retentionCompactFakeCoord{
		entities: make(map[string]*apikv.Entity),
		deleted:  make([]string, 0),
		fail:     make(map[string]error),
		rangeErr: make(map[string]error),
	}
}

func (f *retentionCompactFakeCoord) getKV(key string) (*apikv.Entity, error) {
	entity, ok := f.entities[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return entity, nil
}

func (f *retentionCompactFakeCoord) iterate(param *rangeParam) (*apikv.RangeResp, error) {
	if err := f.rangeErr[param.key]; err != nil {
		return nil, err
	}

	prefix := param.key + "/"
	keys := make([]string, 0)
	for key := range f.entities {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		leaf := strings.TrimPrefix(key, prefix)
		if leaf == "" || strings.ContainsRune(leaf, '/') {
			continue
		}
		if param.seek != "" && leaf < param.seek {
			continue
		}
		keys = append(keys, key)
	}

	sort.Strings(keys)
	limit := param.limit
	if limit <= 0 || limit > len(keys) {
		limit = len(keys)
	}

	entities := make([]*apikv.Entity, 0, limit)
	for _, key := range keys[:limit] {
		entities = append(entities, f.entities[key])
	}

	resp := &apikv.RangeResp{Entities: entities}
	if len(keys) > limit {
		resp.HasMore = true
		resp.NextSeekKey = leafName(keys[limit])
	}
	return resp, nil
}

func (f *retentionCompactFakeCoord) unsetKV(_ context.Context, param *unsetKVParam) error {
	if err := f.fail[param.key]; err != nil {
		return err
	}
	f.deleted = append(f.deleted, param.key)
	delete(f.entities, param.key)
	return nil
}

func leafName(key string) string {
	for idx := len(key) - 1; idx >= 0; idx-- {
		if key[idx] == '/' {
			return key[idx+1:]
		}
	}
	return key
}

func mustProtoBytes(t *testing.T, msg proto.Message) []byte {
	t.Helper()
	data, err := concept.MarshalProto(msg)
	require.NoError(t, err)
	return data
}

func addMetadata(t *testing.T, fake *retentionCompactFakeCoord, root string, md *concept.ElementMetadata) {
	t.Helper()
	key := concept.WithMetadataSuffix(root)
	fake.entities[key] = apikv.NewEntityWithCreated(key, mustProtoBytes(t, md), 0, time.Now().Unix())
}

func addVersion(t *testing.T, fake *retentionCompactFakeCoord, root string, version int32, createdAt time.Time) string {
	t.Helper()
	key := concept.WithVersion(root, int(version))
	fake.entities[key] = apikv.NewEntityWithCreated(key, mustProtoBytes(t, &concept.Element{Version: version, Raw: []byte("v")}), 0, createdAt.Unix())
	return key
}

func addOperation(t *testing.T, fake *retentionCompactFakeCoord, appID, env, key string, operatedAt time.Time) string {
	t.Helper()
	opKey := concept.GenElementOperationKey(appID, env, key, operatedAt.UnixNano())
	op := &concept.ElementOperation{OperatedAt: operatedAt.UnixNano(), OperatedKey: key, Op: concept.ElementOperation_SET}
	fake.entities[opKey] = apikv.NewEntityWithCreated(opKey, mustProtoBytes(t, op), 0, operatedAt.Unix())
	return opKey
}

func TestCompactElementHistoryKeepsProtectedVersions(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	root := "cassem/elements/demo/prod/db.url"
	fake := newRetentionCompactFakeCoord()
	addMetadata(t, fake, root, &concept.ElementMetadata{Key: "db.url", App: "demo", Env: "prod", LatestVersion: 6, UsingVersion: 2, UnpublishedVersion: 5})
	for version := int32(1); version <= 6; version++ {
		addVersion(t, fake, root, version, now.Add(-60*24*time.Hour))
	}

	resp, err := compactElementHistory(fake, &apikv.CompactElementHistoryReq{
		ElementKey:           root,
		KeepVersionCount:     2,
		KeepVersionSeconds:   int64((30 * 24 * time.Hour).Seconds()),
		KeepOperationSeconds: int64((180 * 24 * time.Hour).Seconds()),
		PageSize:             2,
	}, now)

	require.NoError(t, err)
	require.Empty(t, resp.FailedKeys)
	require.Equal(t, int32(6), resp.ScannedVersions)
	require.ElementsMatch(t, []string{concept.WithVersion(root, 1), concept.WithVersion(root, 3), concept.WithVersion(root, 4)}, fake.deleted)
	require.Equal(t, int32(3), resp.DeletedVersions)
	require.Contains(t, fake.entities, concept.WithVersion(root, 2))
	require.Contains(t, fake.entities, concept.WithVersion(root, 5))
	require.Contains(t, fake.entities, concept.WithVersion(root, 6))
}

func TestCompactElementHistoryKeepsLatestVersionsByNumber(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	root := "cassem/elements/demo/prod/db.url"
	fake := newRetentionCompactFakeCoord()
	addMetadata(t, fake, root, &concept.ElementMetadata{Key: "db.url", App: "demo", Env: "prod", LatestVersion: 5})
	for version := int32(1); version <= 5; version++ {
		addVersion(t, fake, root, version, now.Add(-60*24*time.Hour))
	}

	resp, err := compactElementHistory(fake, &apikv.CompactElementHistoryReq{
		ElementKey:           root,
		KeepVersionCount:     2,
		KeepVersionSeconds:   int64((30 * 24 * time.Hour).Seconds()),
		KeepOperationSeconds: int64((180 * 24 * time.Hour).Seconds()),
		PageSize:             2,
	}, now)

	require.NoError(t, err)
	require.Equal(t, int32(3), resp.DeletedVersions)
	require.ElementsMatch(t, []string{concept.WithVersion(root, 1), concept.WithVersion(root, 2), concept.WithVersion(root, 3)}, fake.deleted)
	require.Contains(t, fake.entities, concept.WithVersion(root, 4))
	require.Contains(t, fake.entities, concept.WithVersion(root, 5))
}

func TestCompactElementHistoryKeepsVersionsCreatedWithinWindow(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	root := "cassem/elements/demo/prod/db.url"
	fake := newRetentionCompactFakeCoord()
	addMetadata(t, fake, root, &concept.ElementMetadata{Key: "db.url", App: "demo", Env: "prod", LatestVersion: 3})
	addVersion(t, fake, root, 1, now.Add(-10*24*time.Hour))
	addVersion(t, fake, root, 2, now.Add(-60*24*time.Hour))
	addVersion(t, fake, root, 3, now.Add(-5*24*time.Hour))

	resp, err := compactElementHistory(fake, &apikv.CompactElementHistoryReq{
		ElementKey:           root,
		KeepVersionCount:     1,
		KeepVersionSeconds:   int64((30 * 24 * time.Hour).Seconds()),
		KeepOperationSeconds: int64((180 * 24 * time.Hour).Seconds()),
		PageSize:             1,
	}, now)

	require.NoError(t, err)
	require.Equal(t, int32(1), resp.DeletedVersions)
	require.Equal(t, []string{concept.WithVersion(root, 2)}, fake.deleted)
	require.Contains(t, fake.entities, concept.WithVersion(root, 1))
	require.Contains(t, fake.entities, concept.WithVersion(root, 3))
}

func TestCompactElementHistoryDeletesOldVersionsOutsideKeepRules(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	root := "cassem/elements/demo/prod/db.url"
	fake := newRetentionCompactFakeCoord()
	addMetadata(t, fake, root, &concept.ElementMetadata{Key: "db.url", App: "demo", Env: "prod", LatestVersion: 4})
	addVersion(t, fake, root, 1, now.Add(-60*24*time.Hour))
	addVersion(t, fake, root, 2, now.Add(-60*24*time.Hour))
	addVersion(t, fake, root, 3, now.Add(-10*24*time.Hour))
	addVersion(t, fake, root, 4, now.Add(-60*24*time.Hour))

	resp, err := compactElementHistory(fake, &apikv.CompactElementHistoryReq{
		ElementKey:           root,
		KeepVersionCount:     1,
		KeepVersionSeconds:   int64((30 * 24 * time.Hour).Seconds()),
		KeepOperationSeconds: int64((180 * 24 * time.Hour).Seconds()),
		PageSize:             2,
	}, now)

	require.NoError(t, err)
	require.Equal(t, int32(2), resp.DeletedVersions)
	require.ElementsMatch(t, []string{concept.WithVersion(root, 1), concept.WithVersion(root, 2)}, fake.deleted)
	require.Contains(t, fake.entities, concept.WithVersion(root, 3))
	require.Contains(t, fake.entities, concept.WithVersion(root, 4))
}

func TestCompactElementHistoryDeletesOperationLogsByOperatedAt(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	root := "cassem/elements/demo/prod/db.url"
	fake := newRetentionCompactFakeCoord()
	addMetadata(t, fake, root, &concept.ElementMetadata{Key: "db.url", App: "demo", Env: "prod", LatestVersion: 1, UsingVersion: 1})
	addVersion(t, fake, root, 1, now)
	oldOperation := addOperation(t, fake, "demo", "prod", "db.url", now.Add(-181*24*time.Hour))
	newOperation := addOperation(t, fake, "demo", "prod", "db.url", now.Add(-10*24*time.Hour))

	resp, err := compactElementHistory(fake, &apikv.CompactElementHistoryReq{
		ElementKey:           root,
		KeepVersionCount:     1,
		KeepVersionSeconds:   int64((30 * 24 * time.Hour).Seconds()),
		KeepOperationSeconds: int64((180 * 24 * time.Hour).Seconds()),
		PageSize:             1,
	}, now)

	require.NoError(t, err)
	require.Equal(t, int32(2), resp.ScannedOperations)
	require.Equal(t, int32(1), resp.DeletedOperations)
	require.Contains(t, fake.deleted, oldOperation)
	require.Contains(t, fake.entities, newOperation)
}

func TestCompactElementHistoryReportsMalformedVersionKeys(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	root := "cassem/elements/demo/prod/db.url"
	fake := newRetentionCompactFakeCoord()
	addMetadata(t, fake, root, &concept.ElementMetadata{Key: "db.url", App: "demo", Env: "prod", LatestVersion: 2, UsingVersion: 2})
	addVersion(t, fake, root, 1, now.Add(-60*24*time.Hour))
	addVersion(t, fake, root, 2, now.Add(-60*24*time.Hour))
	malformedVersionKey := root + "/vbad"
	fake.entities[malformedVersionKey] = apikv.NewEntityWithCreated(malformedVersionKey, []byte("bad"), 0, now.Add(-60*24*time.Hour).Unix())

	resp, err := compactElementHistory(fake, &apikv.CompactElementHistoryReq{
		ElementKey:           root,
		KeepVersionCount:     1,
		KeepVersionSeconds:   int64((30 * 24 * time.Hour).Seconds()),
		KeepOperationSeconds: int64((180 * 24 * time.Hour).Seconds()),
		PageSize:             2,
	}, now)

	require.NoError(t, err)
	require.Equal(t, int32(3), resp.ScannedVersions)
	require.Equal(t, int32(1), resp.DeletedVersions)
	require.Equal(t, []string{malformedVersionKey}, resp.FailedKeys)
	require.Equal(t, "partial cleanup failure", resp.Error)
	require.Contains(t, fake.deleted, concept.WithVersion(root, 1))
	require.NotContains(t, fake.deleted, malformedVersionKey)
	require.Contains(t, fake.entities, malformedVersionKey)
	require.Contains(t, fake.entities, concept.WithVersion(root, 2))
}

func TestCompactElementHistoryReturnsPartialFailures(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	root := "cassem/elements/demo/prod/db.url"

	t.Run("delete failure", func(t *testing.T) {
		fake := newRetentionCompactFakeCoord()
		addMetadata(t, fake, root, &concept.ElementMetadata{Key: "db.url", App: "demo", Env: "prod", LatestVersion: 5, UsingVersion: 5})
		for version := int32(1); version <= 5; version++ {
			addVersion(t, fake, root, version, now.Add(-60*24*time.Hour))
		}
		failedKey := concept.WithVersion(root, 1)
		fake.fail[failedKey] = errors.New("delete failed")

		resp, err := compactElementHistory(fake, &apikv.CompactElementHistoryReq{
			ElementKey:           root,
			KeepVersionCount:     1,
			KeepVersionSeconds:   int64((30 * 24 * time.Hour).Seconds()),
			KeepOperationSeconds: int64((180 * 24 * time.Hour).Seconds()),
			PageSize:             2,
		}, now)

		require.NoError(t, err)
		require.Equal(t, []string{failedKey}, resp.FailedKeys)
		require.Equal(t, "partial cleanup failure", resp.Error)
		require.Equal(t, int32(3), resp.DeletedVersions)
		require.Contains(t, fake.entities, failedKey)
		require.NotContains(t, fake.entities, concept.WithVersion(root, 2))
		require.NotContains(t, fake.entities, concept.WithVersion(root, 3))
		require.NotContains(t, fake.entities, concept.WithVersion(root, 4))
	})

	t.Run("corrupted operation payload", func(t *testing.T) {
		fake := newRetentionCompactFakeCoord()
		addMetadata(t, fake, root, &concept.ElementMetadata{Key: "db.url", App: "demo", Env: "prod", LatestVersion: 1, UsingVersion: 1})
		addVersion(t, fake, root, 1, now)
		validOperation := addOperation(t, fake, "demo", "prod", "db.url", now.Add(-181*24*time.Hour))
		corruptedOperation := concept.GenElementOperationKey("demo", "prod", "db.url", now.Add(-200*24*time.Hour).UnixNano())
		fake.entities[corruptedOperation] = apikv.NewEntityWithCreated(corruptedOperation, []byte("bad"), 0, now.Add(-200*24*time.Hour).Unix())

		resp, err := compactElementHistory(fake, &apikv.CompactElementHistoryReq{
			ElementKey:           root,
			KeepVersionCount:     1,
			KeepVersionSeconds:   int64((30 * 24 * time.Hour).Seconds()),
			KeepOperationSeconds: int64((180 * 24 * time.Hour).Seconds()),
			PageSize:             1,
		}, now)

		require.NoError(t, err)
		require.Equal(t, int32(2), resp.ScannedOperations)
		require.Equal(t, int32(1), resp.DeletedOperations)
		require.Equal(t, []string{corruptedOperation}, resp.FailedKeys)
		require.Equal(t, "partial cleanup failure", resp.Error)
		require.Contains(t, fake.deleted, validOperation)
		require.Contains(t, fake.entities, corruptedOperation)
	})
}

func TestCompactElementHistoryKeepsEverythingWhenRetentionSecondsOverflow(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	root := "cassem/elements/demo/prod/db.url"

	t.Run("versions", func(t *testing.T) {
		fake := newRetentionCompactFakeCoord()
		addMetadata(t, fake, root, &concept.ElementMetadata{Key: "db.url", App: "demo", Env: "prod", LatestVersion: 3})
		v1 := addVersion(t, fake, root, 1, now.Add(-365*24*time.Hour))
		v2 := addVersion(t, fake, root, 2, now.Add(-364*24*time.Hour))
		v3 := addVersion(t, fake, root, 3, now.Add(-363*24*time.Hour))

		resp, err := compactElementHistory(fake, &apikv.CompactElementHistoryReq{
			ElementKey:           root,
			KeepVersionCount:     0,
			KeepVersionSeconds:   math.MaxInt64,
			KeepOperationSeconds: int64((180 * 24 * time.Hour).Seconds()),
			PageSize:             2,
		}, now)

		require.NoError(t, err)
		require.Empty(t, resp.FailedKeys)
		require.Equal(t, int32(3), resp.ScannedVersions)
		require.Zero(t, resp.DeletedVersions)
		require.Empty(t, fake.deleted)
		require.Contains(t, fake.entities, v1)
		require.Contains(t, fake.entities, v2)
		require.Contains(t, fake.entities, v3)
	})

	t.Run("operations", func(t *testing.T) {
		fake := newRetentionCompactFakeCoord()
		addMetadata(t, fake, root, &concept.ElementMetadata{Key: "db.url", App: "demo", Env: "prod", LatestVersion: 1, UsingVersion: 1})
		addVersion(t, fake, root, 1, now)
		op1 := addOperation(t, fake, "demo", "prod", "db.url", now.Add(-365*24*time.Hour))
		op2 := addOperation(t, fake, "demo", "prod", "db.url", now.Add(-364*24*time.Hour))

		resp, err := compactElementHistory(fake, &apikv.CompactElementHistoryReq{
			ElementKey:           root,
			KeepVersionCount:     1,
			KeepVersionSeconds:   int64((30 * 24 * time.Hour).Seconds()),
			KeepOperationSeconds: math.MaxInt64,
			PageSize:             1,
		}, now)

		require.NoError(t, err)
		require.Empty(t, resp.FailedKeys)
		require.Equal(t, int32(2), resp.ScannedOperations)
		require.Zero(t, resp.DeletedOperations)
		require.Empty(t, fake.deleted)
		require.Contains(t, fake.entities, op1)
		require.Contains(t, fake.entities, op2)
	})
}

func TestCompactElementHistoryReturnsErrors(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	root := "cassem/elements/demo/prod/db.url"

	t.Run("nil coordinator", func(t *testing.T) {
		_, err := compactElementHistory(nil, &apikv.CompactElementHistoryReq{ElementKey: root}, now)
		require.Error(t, err)
	})

	t.Run("empty element key", func(t *testing.T) {
		_, err := compactElementHistory(newRetentionCompactFakeCoord(), &apikv.CompactElementHistoryReq{}, now)
		require.Error(t, err)
	})

	t.Run("missing metadata", func(t *testing.T) {
		_, err := compactElementHistory(newRetentionCompactFakeCoord(), &apikv.CompactElementHistoryReq{
			ElementKey:           root,
			KeepVersionCount:     1,
			KeepVersionSeconds:   int64((30 * 24 * time.Hour).Seconds()),
			KeepOperationSeconds: int64((180 * 24 * time.Hour).Seconds()),
			PageSize:             1,
		}, now)
		require.Error(t, err)
	})

	t.Run("invalid metadata", func(t *testing.T) {
		fake := newRetentionCompactFakeCoord()
		fake.entities[concept.WithMetadataSuffix(root)] = apikv.NewEntityWithCreated(concept.WithMetadataSuffix(root), []byte("bad"), 0, now.Unix())
		_, err := compactElementHistory(fake, &apikv.CompactElementHistoryReq{
			ElementKey:           root,
			KeepVersionCount:     1,
			KeepVersionSeconds:   int64((30 * 24 * time.Hour).Seconds()),
			KeepOperationSeconds: int64((180 * 24 * time.Hour).Seconds()),
			PageSize:             1,
		}, now)
		require.Error(t, err)
	})

	t.Run("range error", func(t *testing.T) {
		fake := newRetentionCompactFakeCoord()
		addMetadata(t, fake, root, &concept.ElementMetadata{Key: "db.url", App: "demo", Env: "prod"})
		fake.rangeErr[root] = errors.New("range failed")
		_, err := compactElementHistory(fake, &apikv.CompactElementHistoryReq{
			ElementKey:           root,
			KeepVersionCount:     1,
			KeepVersionSeconds:   int64((30 * 24 * time.Hour).Seconds()),
			KeepOperationSeconds: int64((180 * 24 * time.Hour).Seconds()),
			PageSize:             1,
		}, now)
		require.Error(t, err)
	})
}

var _ retentionCompactCoordinator = (*retentionCompactFakeCoord)(nil)
