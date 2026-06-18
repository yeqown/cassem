package cassemadm

import (
	"crypto/sha1"
	"encoding/hex"
	apikv "github.com/yeqown/cassem/api/kv"
	"strconv"
	"time"

	"github.com/yeqown/cassem/pkg/conf"
)

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
