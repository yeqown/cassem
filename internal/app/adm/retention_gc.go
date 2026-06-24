package adm

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/yeqown/cassem/api/concept"
	apikv "github.com/yeqown/cassem/api/kv"
	"github.com/yeqown/cassem/pkg/conf"
	"github.com/yeqown/cassem/pkg/runtime"
	"github.com/yeqown/log"
	"path"
	"strconv"
	"strings"
	"time"
)

const retentionRunTimeout = 30 * time.Second

type retentionGC struct {
	client apikv.KVClient
	config *conf.RetentionConfig
}

type retentionElementRef struct {
	App string
	Env string
	Key string
}

func newRetentionGC(endpoints []string, config *conf.RetentionConfig) (*retentionGC, error) {
	if config == nil || !config.EnabledValue() {
		return &retentionGC{config: config}, nil
	}

	cc, err := apikv.DialWithMode(endpoints, apikv.Mode_X)
	if err != nil {
		return nil, fmt.Errorf("retention gc dial cassemdb: %w", err)
	}

	return &retentionGC{client: apikv.NewKVClient(cc), config: config}, nil
}

func (g *retentionGC) run() {
	if g.inert() {
		return
	}

	interval, err := g.config.IntervalDuration()
	if err != nil || interval <= 0 {
		log.WithFields(log.Fields{"error": err, "interval": interval}).Error("retention gc invalid interval")
		return
	}

	runtime.GoFunc("retention-gc", func() error {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			g.runOnce(time.Now())
			<-ticker.C
		}
	})
}

func (g *retentionGC) runOnce(now time.Time) retentionRunSummary {
	summary := retentionRunSummary{StartedAt: now.Unix()}
	if g.inert() {
		summary.FinishedAt = time.Now().Unix()
		return summary
	}

	ctx, cancel := context.WithTimeout(context.Background(), retentionRunTimeout)
	defer cancel()
	maxElements := g.config.MaxElementsPerRunValue()
	if maxElements <= 0 {
		summary.FinishedAt = time.Now().Unix()
		g.logPersistJSON(ctx, retentionLatestKey, summary, 0)
		return summary
	}

	cursor := g.loadCursor(ctx)
	nextCursor := cursor
	reachedEnd := g.scanElements(ctx, cursor, func(ref retentionElementRef) bool {
		summary.ScannedElements++
		nextCursor = retentionCursor{App: ref.App, Env: ref.Env, Element: nextSeekAfter(ref.Key)}

		resp, err := g.compactElement(ctx, ref)
		if err != nil {
			summary.FailedElements++
			g.logPersistFailure(ctx, now, ref, err.Error(), nil)
			return summary.ScannedElements < maxElements
		}

		summary.DeletedVersions += resp.GetDeletedVersions()
		summary.DeletedOperations += resp.GetDeletedOperations()
		if resp.GetError() != "" || len(resp.GetFailedKeys()) != 0 {
			summary.PartialElements++
			g.logPersistFailure(ctx, now, ref, resp.GetError(), resp.GetFailedKeys())
			return summary.ScannedElements < maxElements
		}

		summary.CleanedElements++
		return summary.ScannedElements < maxElements
	})
	if !reachedEnd && summary.ScannedElements == maxElements {
		hasMoreElements, ok := g.hasElementAtOrAfterCursor(ctx, nextCursor)
		reachedEnd = ok && !hasMoreElements
	}
	if reachedEnd {
		nextCursor = retentionCursor{}
	}

	summary.FinishedAt = time.Now().Unix()
	g.logPersistJSON(ctx, retentionCursorKey, nextCursor, 0)
	g.logPersistJSON(ctx, retentionLatestKey, summary, 0)
	return summary
}

func (g *retentionGC) inert() bool {
	return g == nil || g.config == nil || !g.config.EnabledValue() || g.client == nil
}

func (g *retentionGC) compactElement(ctx context.Context, ref retentionElementRef) (*apikv.CompactElementHistoryResp, error) {
	req := retentionPolicyFromConfig(g.config)
	req.ElementKey = concept.GenElementKey(ref.App, ref.Env, ref.Key)
	return g.client.CompactElementHistory(ctx, req)
}

func (g *retentionGC) loadCursor(ctx context.Context) retentionCursor {
	resp, err := g.client.GetKV(ctx, &apikv.GetKVReq{Key: retentionCursorKey})
	if err != nil || resp.GetEntity() == nil {
		return retentionCursor{}
	}

	var cursor retentionCursor
	if err = json.Unmarshal(resp.GetEntity().GetVal(), &cursor); err != nil {
		return retentionCursor{}
	}
	if !validRetentionCursor(cursor) {
		return retentionCursor{}
	}
	return cursor
}

func validRetentionCursor(cursor retentionCursor) bool {
	if cursor.App == "" {
		return cursor.Env == "" && cursor.Element == ""
	}
	if cursor.Env == "" {
		return cursor.Element == ""
	}
	return !strings.Contains(cursor.App, "/") && !strings.Contains(cursor.Env, "/") && !strings.Contains(cursor.Element, "/")
}

func (g *retentionGC) scanElements(ctx context.Context, cursor retentionCursor, visit func(retentionElementRef) bool) bool {
	appSeek := cursor.App
	activeCursor := cursor
	for {
		apps, hasMoreApps, nextAppSeek, err := g.rangeLeavesPageWithError(ctx, concept.GenAppDirKey(), appSeek)
		if err != nil {
			return false
		}
		for _, appID := range apps {
			envCursor := retentionCursor{}
			if appID == activeCursor.App {
				envCursor = activeCursor
			}
			if !g.scanEnvs(ctx, appID, envCursor.Env, envCursor, visit) {
				return false
			}
			if appID == activeCursor.App {
				activeCursor = retentionCursor{}
			}
		}
		if !hasMoreApps {
			return true
		}
		if nextAppSeek == "" {
			return false
		}
		appSeek = nextAppSeek
		activeCursor = retentionCursor{}
	}
}

func (g *retentionGC) scanEnvs(ctx context.Context, appID, envSeek string, cursor retentionCursor, visit func(retentionElementRef) bool) bool {
	activeCursor := cursor
	for {
		envs, hasMoreEnvs, nextEnvSeek, err := g.rangeLeavesPageWithError(ctx, concept.GenAppElementKey(appID), envSeek)
		if err != nil {
			return false
		}
		for _, env := range envs {
			elementSeek := ""
			if appID == activeCursor.App && env == activeCursor.Env {
				elementSeek = activeCursor.Element
			}
			if !g.scanElementKeys(ctx, appID, env, elementSeek, visit) {
				return false
			}
			if appID == activeCursor.App && env == activeCursor.Env {
				activeCursor = retentionCursor{}
			}
		}
		if !hasMoreEnvs {
			return true
		}
		if nextEnvSeek == "" {
			return false
		}
		envSeek = nextEnvSeek
		activeCursor = retentionCursor{}
	}
}

func (g *retentionGC) scanElementKeys(ctx context.Context, appID, env, elementSeek string, visit func(retentionElementRef) bool) bool {
	for {
		elements, hasMoreElements, nextElementSeek, err := g.rangeLeavesPageWithError(ctx, concept.GenAppElementEnvKey(appID, env), elementSeek)
		if err != nil {
			return false
		}
		for _, key := range elements {
			if !visit(retentionElementRef{App: appID, Env: env, Key: key}) {
				return false
			}
		}
		if !hasMoreElements {
			return true
		}
		if nextElementSeek == "" {
			return false
		}
		elementSeek = nextElementSeek
	}
}

func (g *retentionGC) hasElementAtOrAfterCursor(ctx context.Context, cursor retentionCursor) (bool, bool) {
	found := false
	reachedEnd := g.scanElements(ctx, cursor, func(retentionElementRef) bool {
		found = true
		return false
	})
	if found {
		return true, true
	}
	return false, reachedEnd
}

func (g *retentionGC) rangeLeavesPageWithError(ctx context.Context, root string, seek string) ([]string, bool, string, error) {
	resp, err := g.client.Range(ctx, &apikv.RangeReq{
		Key:   root,
		Seek:  seek,
		Limit: int32(g.config.ElementPageSizeValue()),
	})
	if err != nil {
		return nil, false, "", fmt.Errorf("range %s: %w", root, err)
	}

	leaves := make([]string, 0, len(resp.GetEntities()))
	for _, entity := range resp.GetEntities() {
		leaf := path.Base(entity.GetKey())
		if leaf == "" || leaf == "." || leaf == "/" {
			continue
		}
		leaves = append(leaves, leaf)
	}

	nextSeek := resp.GetNextSeekKey()
	if nextSeek != "" {
		nextSeek = path.Base(nextSeek)
	}
	return leaves, resp.GetHasMore(), nextSeek, nil
}

func nextSeekAfter(value string) string {
	return value + "\x00"
}

func (g *retentionGC) persistJSON(ctx context.Context, key string, value any, ttl int32) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal retention json %s: %w", key, err)
	}
	_, err = g.client.SetKV(ctx, &apikv.SetKVReq{Key: key, Val: data, Ttl: ttl, Overwrite: true})
	if err != nil {
		return fmt.Errorf("set retention json %s: %w", key, err)
	}
	return nil
}

func (g *retentionGC) logPersistJSON(ctx context.Context, key string, value any, ttl int32) {
	if err := g.persistJSON(ctx, key, value, ttl); err != nil {
		log.WithFields(log.Fields{"key": key, "error": err}).Error("retention gc persist json failed")
	}
}

func (g *retentionGC) persistFailure(ctx context.Context, now time.Time, ref retentionElementRef, message string, failedKeys []string) error {
	ttl, err := g.config.FailureResultTTLDuration()
	if err != nil {
		return fmt.Errorf("retention failure ttl: %w", err)
	}

	record := retentionFailureRecord{
		App:        ref.App,
		Env:        ref.Env,
		Key:        ref.Key,
		Error:      message,
		FailedKeys: append([]string(nil), failedKeys...),
		OccurredAt: now.Unix(),
	}
	return g.persistJSON(ctx, retentionFailureStorageKey(now, ref.App, ref.Env, ref.Key), record, ttlSeconds(ttl))
}

func ttlSeconds(ttl time.Duration) int32 {
	seconds := ttl / time.Second
	if ttl%time.Second != 0 {
		seconds++
	}
	return int32(seconds)
}

func (g *retentionGC) logPersistFailure(ctx context.Context, now time.Time, ref retentionElementRef, message string, failedKeys []string) {
	if err := g.persistFailure(ctx, now, ref, message, failedKeys); err != nil {
		log.WithFields(log.Fields{"ref": ref, "error": err}).Error("retention gc persist failure failed")
	}
}

const (
	retentionCursorKey = "cassem/gc/retention/cursor"
	retentionLatestKey = "cassem/gc/retention/latest"
)

type retentionCursor struct {
	App     string `json:"app"`
	Env     string `json:"env"`
	Element string `json:"element"`
}

type retentionRunSummary struct {
	StartedAt         int64 `json:"startedAt"`
	FinishedAt        int64 `json:"finishedAt"`
	ScannedElements   int   `json:"scannedElements"`
	CleanedElements   int   `json:"cleanedElements"`
	DeletedVersions   int32 `json:"deletedVersions"`
	DeletedOperations int32 `json:"deletedOperations"`
	PartialElements   int   `json:"partialElements"`
	FailedElements    int   `json:"failedElements"`
}

type retentionFailureRecord struct {
	App        string   `json:"app"`
	Env        string   `json:"env"`
	Key        string   `json:"key"`
	Error      string   `json:"error"`
	FailedKeys []string `json:"failedKeys"`
	OccurredAt int64    `json:"occurredAt"`
}

func retentionPolicyFromConfig(c *conf.RetentionConfig) *apikv.CompactElementHistoryReq {
	if c == nil {
		c = conf.DefaultRetentionConfig()
	}

	daySeconds := int64((24 * time.Hour) / time.Second)

	return &apikv.CompactElementHistoryReq{
		KeepVersionCount:     int32(c.KeepVersionCountValue()),
		KeepVersionSeconds:   int64(c.KeepVersionDaysValue()) * daySeconds,
		KeepOperationSeconds: int64(c.KeepOperationDaysValue()) * daySeconds,
		PageSize:             int32(c.ElementPageSizeValue()),
	}
}

func retentionFailureStorageKey(at time.Time, appID, env, key string) string {
	sum := sha1.Sum([]byte(appID + "/" + env + "/" + key))
	return "cassem/gc/retention/failures/" + strconv.FormatInt(at.Unix(), 10) + "-" + hex.EncodeToString(sum[:])[:12]
}
