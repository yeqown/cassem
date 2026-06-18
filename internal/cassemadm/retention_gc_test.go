package cassemadm

import (
	"context"
	"encoding/json"
	"errors"
	apikv "github.com/yeqown/cassem/api/kv"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/pkg/conf"
)

func TestRetentionPolicyFromConfig(t *testing.T) {
	daySeconds := int64((24 * time.Hour) / time.Second)

	t.Run("uses defaults for nil config", func(t *testing.T) {
		policy := retentionPolicyFromConfig(nil)

		require.NotNil(t, policy)
		require.Equal(t, int32(20), policy.KeepVersionCount)
		require.Equal(t, int64(30)*daySeconds, policy.KeepVersionSeconds)
		require.Equal(t, int64(180)*daySeconds, policy.KeepOperationSeconds)
		require.Equal(t, int32(100), policy.PageSize)
	})

	t.Run("uses helper fallbacks for partial config", func(t *testing.T) {
		pageSize := 50
		config := &conf.RetentionConfig{ElementPageSize: &pageSize}

		policy := retentionPolicyFromConfig(config)

		require.NotNil(t, policy)
		require.Equal(t, int32(20), policy.KeepVersionCount)
		require.Equal(t, int64(30)*daySeconds, policy.KeepVersionSeconds)
		require.Equal(t, int64(180)*daySeconds, policy.KeepOperationSeconds)
		require.Equal(t, int32(50), policy.PageSize)
	})
}

func TestRetentionFailureKeyIsStable(t *testing.T) {
	key := retentionFailureStorageKey(time.Unix(1_700_000_000, 0), "demo", "prod", "db.url")

	require.Equal(t, "cassem/gc/retention/failures/1700000000-8e6403c93e02", key)
	require.Equal(t, "-8e6403c93e02", key[len(key)-13:])
}

func TestRetentionFailureTTLSecondsRoundsUp(t *testing.T) {
	require.Equal(t, int32(1), ttlSeconds(500*time.Millisecond))
	require.Equal(t, int32(2), ttlSeconds(1500*time.Millisecond))
	require.Equal(t, int32(7200), ttlSeconds(2*time.Hour))
}

func TestRetentionRunProcessesBoundedElementsAndPersistsSummary(t *testing.T) {
	fake := newRetentionFakeKV()
	fake.returnFullKeys = true
	seedRetentionTree(fake, []string{"demo/prod/a", "demo/prod/b", "demo/prod/c"})
	gc := &retentionGC{client: fake, config: retentionTestConfig(2, 1, "2h")}
	now := time.Unix(1_700_000_000, 0)

	summary := gc.runOnce(now)

	require.Equal(t, int64(1_700_000_000), summary.StartedAt)
	require.GreaterOrEqual(t, summary.FinishedAt, summary.StartedAt)
	require.Equal(t, 2, summary.ScannedElements)
	require.Equal(t, 2, summary.CleanedElements)
	require.Equal(t, int32(2), summary.DeletedVersions)
	require.Equal(t, []string{concept.GenElementKey("demo", "prod", "a"), concept.GenElementKey("demo", "prod", "b")}, fake.compactCalls)
	require.NotContains(t, fake.compactCalls, concept.GenElementKey("demo", "prod", "c"))

	latest := decodeRetentionSet[retentionRunSummary](t, fake, retentionLatestKey)
	require.Equal(t, summary, latest)
	require.True(t, fake.set[retentionLatestKey].GetOverwrite())
	require.Zero(t, fake.set[retentionLatestKey].GetTtl())

	cursor := decodeRetentionSet[retentionCursor](t, fake, retentionCursorKey)
	require.Equal(t, retentionCursor{App: "demo", Env: "prod", Element: nextSeekAfter("b")}, cursor)
	require.True(t, fake.set[retentionCursorKey].GetOverwrite())
	require.Zero(t, fake.set[retentionCursorKey].GetTtl())
	require.Empty(t, fake.failureSetKeys())
}

func TestRetentionRunCursorAdvancesAfterElementAttempt(t *testing.T) {
	tests := []struct {
		name            string
		response        *apikv.CompactElementHistoryResp
		err             error
		expectCleaned   int
		expectPartial   int
		expectFailed    int
		expectFailures  int
		expectDeleted   int32
		expectFailedKey []string
	}{
		{
			name:           "success",
			response:       &apikv.CompactElementHistoryResp{DeletedVersions: 1},
			expectCleaned:  1,
			expectDeleted:  1,
			expectFailures: 0,
		},
		{
			name:            "partial response",
			response:        &apikv.CompactElementHistoryResp{DeletedVersions: 2, DeletedOperations: 1, Error: "partial cleanup", FailedKeys: []string{"bad-key"}},
			expectPartial:   1,
			expectFailures:  1,
			expectDeleted:   2,
			expectFailedKey: []string{"bad-key"},
		},
		{
			name:           "rpc failure",
			err:            errors.New("rpc unavailable"),
			expectFailed:   1,
			expectFailures: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := newRetentionFakeKV()
			seedRetentionTree(fake, []string{"demo/prod/a", "demo/prod/b"})
			elementKey := concept.GenElementKey("demo", "prod", "a")
			if tt.response != nil {
				fake.compact[elementKey] = tt.response
			}
			if tt.err != nil {
				fake.compactErr[elementKey] = tt.err
			}
			gc := &retentionGC{client: fake, config: retentionTestConfig(1, 10, "2h")}

			summary := gc.runOnce(time.Unix(1_700_000_000, 0))

			require.Equal(t, 1, summary.ScannedElements)
			require.Equal(t, tt.expectCleaned, summary.CleanedElements)
			require.Equal(t, tt.expectPartial, summary.PartialElements)
			require.Equal(t, tt.expectFailed, summary.FailedElements)
			require.Equal(t, tt.expectDeleted, summary.DeletedVersions)
			cursor := decodeRetentionSet[retentionCursor](t, fake, retentionCursorKey)
			require.Equal(t, retentionCursor{App: "demo", Env: "prod", Element: nextSeekAfter("a")}, cursor)
			require.Len(t, fake.failureSetKeys(), tt.expectFailures)
			if tt.expectFailures != 0 {
				record := decodeRetentionSet[retentionFailureRecord](t, fake, fake.failureSetKeys()[0])
				require.Equal(t, "demo", record.App)
				require.Equal(t, "prod", record.Env)
				require.Equal(t, "a", record.Key)
				require.Equal(t, tt.expectFailedKey, record.FailedKeys)
				require.Equal(t, int32(7200), fake.set[fake.failureSetKeys()[0]].GetTtl())
			}
		})
	}
}

func TestRetentionRunBadElementDoesNotBlockNextElement(t *testing.T) {
	fake := newRetentionFakeKV()
	seedRetentionTree(fake, []string{"demo/prod/a", "demo/prod/b", "demo/prod/c", "demo/prod/d", "demo/prod/e"})
	fake.compact[concept.GenElementKey("demo", "prod", "b")] = &apikv.CompactElementHistoryResp{
		DeletedVersions:   2,
		DeletedOperations: 1,
		FailedKeys:        []string{"cassem/elements/demo/prod/b/v1"},
	}
	fake.compactErr[concept.GenElementKey("demo", "prod", "c")] = errors.New("rpc failed")
	gc := &retentionGC{client: fake, config: retentionTestConfig(4, 2, "3h")}
	now := time.Unix(1_700_000_000, 0)

	summary := gc.runOnce(now)

	require.Equal(t, 4, summary.ScannedElements)
	require.Equal(t, 2, summary.CleanedElements)
	require.Equal(t, 1, summary.PartialElements)
	require.Equal(t, 1, summary.FailedElements)
	require.Equal(t, int32(4), summary.DeletedVersions)
	require.Equal(t, int32(1), summary.DeletedOperations)
	require.Equal(t, []string{
		concept.GenElementKey("demo", "prod", "a"),
		concept.GenElementKey("demo", "prod", "b"),
		concept.GenElementKey("demo", "prod", "c"),
		concept.GenElementKey("demo", "prod", "d"),
	}, fake.compactCalls)

	cursor := decodeRetentionSet[retentionCursor](t, fake, retentionCursorKey)
	require.Equal(t, retentionCursor{App: "demo", Env: "prod", Element: nextSeekAfter("d")}, cursor)

	failureKeys := fake.failureSetKeys()
	require.Len(t, failureKeys, 2)
	for _, key := range failureKeys {
		require.Equal(t, int32(10800), fake.set[key].GetTtl())
		require.True(t, fake.set[key].GetOverwrite())
	}
	records := decodeRetentionFailures(t, fake, failureKeys)
	require.Contains(t, records, "b")
	require.Contains(t, records, "c")
	require.NotContains(t, records, "a")
	require.NotContains(t, records, "d")
	require.Equal(t, []string{"cassem/elements/demo/prod/b/v1"}, records["b"].FailedKeys)
	require.Equal(t, "rpc failed", records["c"].Error)
}

func TestRetentionRunResetsCursorAfterReachingEnd(t *testing.T) {
	fake := newRetentionFakeKV()
	seedRetentionTree(fake, []string{"demo/prod/a", "demo/prod/b"})
	fake.mustSetJSON(t, retentionCursorKey, retentionCursor{App: "demo", Env: "prod", Element: nextSeekAfter("a")}, 0)
	gc := &retentionGC{client: fake, config: retentionTestConfig(5, 1, "2h")}

	summary := gc.runOnce(time.Unix(1_700_000_000, 0))

	require.Equal(t, 1, summary.ScannedElements)
	require.Equal(t, []string{concept.GenElementKey("demo", "prod", "b")}, fake.compactCalls)
	cursor := decodeRetentionSet[retentionCursor](t, fake, retentionCursorKey)
	require.Equal(t, retentionCursor{}, cursor)
}

func TestRetentionRunResetsCursorWhenEndFallsOnRunLimit(t *testing.T) {
	fake := newRetentionFakeKV()
	seedRetentionTree(fake, []string{"demo/prod/a", "demo/prod/b"})
	gc := &retentionGC{client: fake, config: retentionTestConfig(2, 1, "2h")}

	summary := gc.runOnce(time.Unix(1_700_000_000, 0))

	require.Equal(t, 2, summary.ScannedElements)
	require.Equal(t, []string{concept.GenElementKey("demo", "prod", "a"), concept.GenElementKey("demo", "prod", "b")}, fake.compactCalls)
	cursor := decodeRetentionSet[retentionCursor](t, fake, retentionCursorKey)
	require.Equal(t, retentionCursor{}, cursor)
}

func TestRetentionRunLoadsEmptyCursorForMissingOrInvalidCursor(t *testing.T) {
	tests := []struct {
		name string
		seed func(*testing.T, *retentionFakeKV)
	}{
		{
			name: "missing cursor",
		},
		{
			name: "invalid json",
			seed: func(t *testing.T, fake *retentionFakeKV) {
				fake.entities[retentionCursorKey] = &apikv.Entity{Key: retentionCursorKey, Val: []byte("{")}
			},
		},
		{
			name: "invalid shape",
			seed: func(t *testing.T, fake *retentionFakeKV) {
				fake.mustSetJSON(t, retentionCursorKey, retentionCursor{Env: "prod", Element: "a"}, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := newRetentionFakeKV()
			seedRetentionTree(fake, []string{"demo/prod/a"})
			if tt.seed != nil {
				tt.seed(t, fake)
			}
			gc := &retentionGC{client: fake, config: retentionTestConfig(1, 10, "2h")}

			summary := gc.runOnce(time.Unix(1_700_000_000, 0))

			require.Equal(t, 1, summary.ScannedElements)
			require.Equal(t, []string{concept.GenElementKey("demo", "prod", "a")}, fake.compactCalls)
		})
	}
}

type retentionFakeKV struct {
	entities       map[string]*apikv.Entity
	compact        map[string]*apikv.CompactElementHistoryResp
	compactErr     map[string]error
	set            map[string]*apikv.SetKVReq
	compactCalls   []string
	returnFullKeys bool
}

func newRetentionFakeKV() *retentionFakeKV {
	return &retentionFakeKV{
		entities:   make(map[string]*apikv.Entity),
		compact:    make(map[string]*apikv.CompactElementHistoryResp),
		compactErr: make(map[string]error),
		set:        make(map[string]*apikv.SetKVReq),
	}
}

func (f *retentionFakeKV) GetKV(_ context.Context, req *apikv.GetKVReq, _ ...grpc.CallOption) (*apikv.GetKVResp, error) {
	entity, ok := f.entities[req.GetKey()]
	if !ok {
		return nil, errors.New("not found")
	}
	return &apikv.GetKVResp{Entity: entity}, nil
}

func (f *retentionFakeKV) GetKVs(context.Context, *apikv.GetKVsReq, ...grpc.CallOption) (*apikv.GetKVsResp, error) {
	return nil, errors.New("unused")
}

func (f *retentionFakeKV) SetKV(_ context.Context, req *apikv.SetKVReq, _ ...grpc.CallOption) (*apikv.Empty, error) {
	val := append([]byte(nil), req.GetVal()...)
	stored := &apikv.SetKVReq{
		Key:       req.GetKey(),
		IsDir:     req.GetIsDir(),
		Ttl:       req.GetTtl(),
		Val:       val,
		Overwrite: req.GetOverwrite(),
	}
	f.set[req.GetKey()] = stored
	f.entities[req.GetKey()] = &apikv.Entity{Key: req.GetKey(), Val: val, Ttl: req.GetTtl(), CreatedAt: time.Now().Unix()}
	return &apikv.Empty{}, nil
}

func (f *retentionFakeKV) UnsetKV(context.Context, *apikv.UnsetKVReq, ...grpc.CallOption) (*apikv.Empty, error) {
	return nil, errors.New("unused")
}

func (f *retentionFakeKV) Watch(context.Context, *apikv.WatchReq, ...grpc.CallOption) (apikv.KV_WatchClient, error) {
	return nil, errors.New("unused")
}

func (f *retentionFakeKV) TTL(context.Context, *apikv.TtlReq, ...grpc.CallOption) (*apikv.TtlResp, error) {
	return nil, errors.New("unused")
}

func (f *retentionFakeKV) Expire(context.Context, *apikv.ExpireReq, ...grpc.CallOption) (*apikv.Empty, error) {
	return nil, errors.New("unused")
}

func (f *retentionFakeKV) Range(_ context.Context, req *apikv.RangeReq, _ ...grpc.CallOption) (*apikv.RangeResp, error) {
	prefix := req.GetKey() + "/"
	seen := make(map[string]struct{})
	leaves := make([]string, 0)
	for key := range f.entities {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		rest := strings.TrimPrefix(key, prefix)
		leaf := strings.Split(rest, "/")[0]
		if leaf == "" || leaf < req.GetSeek() {
			continue
		}
		if _, ok := seen[leaf]; ok {
			continue
		}
		seen[leaf] = struct{}{}
		leaves = append(leaves, leaf)
	}
	sort.Strings(leaves)

	limit := int(req.GetLimit())
	if limit <= 0 || limit > len(leaves) {
		limit = len(leaves)
	}

	entities := make([]*apikv.Entity, 0, limit)
	for _, leaf := range leaves[:limit] {
		entityKey := leaf
		if f.returnFullKeys {
			entityKey = prefix + leaf
		}
		entities = append(entities, &apikv.Entity{Key: entityKey})
	}

	resp := &apikv.RangeResp{Entities: entities}
	if len(leaves) > limit {
		resp.HasMore = true
		resp.NextSeekKey = leaves[limit]
		if f.returnFullKeys {
			resp.NextSeekKey = prefix + leaves[limit]
		}
	}
	return resp, nil
}

func (f *retentionFakeKV) CompactElementHistory(_ context.Context, req *apikv.CompactElementHistoryReq, _ ...grpc.CallOption) (*apikv.CompactElementHistoryResp, error) {
	f.compactCalls = append(f.compactCalls, req.GetElementKey())
	if err := f.compactErr[req.GetElementKey()]; err != nil {
		return nil, err
	}
	resp, ok := f.compact[req.GetElementKey()]
	if !ok {
		return nil, errors.New("compact failed")
	}
	return resp, nil
}

func (f *retentionFakeKV) mustSetJSON(t *testing.T, key string, value any, ttl int32) {
	t.Helper()
	data, err := json.Marshal(value)
	require.NoError(t, err)
	f.entities[key] = &apikv.Entity{Key: key, Val: data, Ttl: ttl}
}

func (f *retentionFakeKV) failureSetKeys() []string {
	keys := make([]string, 0)
	for key := range f.set {
		if strings.HasPrefix(key, "cassem/gc/retention/failures/") {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func seedRetentionTree(fake *retentionFakeKV, refs []string) {
	for _, ref := range refs {
		parts := strings.Split(ref, "/")
		appID, env, key := parts[0], parts[1], parts[2]
		appKey := concept.GenAppKey(appID)
		envKey := concept.GenAppElementEnvKey(appID, env)
		elementKey := concept.GenElementKey(appID, env, key)
		fake.entities[appKey] = &apikv.Entity{Key: appKey}
		fake.entities[envKey] = &apikv.Entity{Key: envKey}
		fake.entities[elementKey] = &apikv.Entity{Key: elementKey}
		if _, ok := fake.compact[elementKey]; !ok {
			fake.compact[elementKey] = &apikv.CompactElementHistoryResp{DeletedVersions: 1}
		}
	}
}

func retentionTestConfig(maxElements, pageSize int, failureTTL string) *conf.RetentionConfig {
	config := conf.DefaultRetentionConfig()
	config.MaxElementsPerRun = &maxElements
	config.ElementPageSize = &pageSize
	config.FailureResultTTL = failureTTL
	return config
}

func decodeRetentionSet[T any](t *testing.T, fake *retentionFakeKV, key string) T {
	t.Helper()
	req, ok := fake.set[key]
	require.True(t, ok, "missing set key %s", key)
	var out T
	require.NoError(t, json.Unmarshal(req.GetVal(), &out))
	return out
}

func decodeRetentionFailures(t *testing.T, fake *retentionFakeKV, keys []string) map[string]retentionFailureRecord {
	t.Helper()
	records := make(map[string]retentionFailureRecord, len(keys))
	for _, key := range keys {
		record := decodeRetentionSet[retentionFailureRecord](t, fake, key)
		records[record.Key] = record
	}
	return records
}
