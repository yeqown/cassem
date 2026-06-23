package adm

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/yeqown/cassem/pkg/conf"
	"github.com/yeqown/cassem/pkg/httpx"
)

type retentionPolicyResp struct {
	Enabled           bool   `json:"enabled"`
	KeepVersionCount  int    `json:"keepVersionCount"`
	KeepVersionDays   int    `json:"keepVersionDays"`
	KeepOperationDays int    `json:"keepOperationDays"`
	VersionPolicy     string `json:"versionPolicy"`
	OperationPolicy   string `json:"operationPolicy"`
}

func retentionPolicyResponseFromConfig(c *conf.RetentionConfig) retentionPolicyResp {
	if c == nil {
		c = conf.DefaultRetentionConfig()
	}

	keepVersionCount := c.KeepVersionCountValue()
	keepVersionDays := c.KeepVersionDaysValue()
	keepOperationDays := c.KeepOperationDaysValue()
	return retentionPolicyResp{
		Enabled:           c.EnabledValue(),
		KeepVersionCount:  keepVersionCount,
		KeepVersionDays:   keepVersionDays,
		KeepOperationDays: keepOperationDays,
		VersionPolicy: fmt.Sprintf(
			"Versions keep current, draft, latest %d, and versions from the last %d days.",
			keepVersionCount,
			keepVersionDays,
		),
		OperationPolicy: fmt.Sprintf("Operation logs keep %d days.", keepOperationDays),
	}
}

func (d app) GetRetentionPolicy(c *gin.Context) {
	httpx.ResponseJSON(c, retentionPolicyResponseFromConfig(d.conf.Retention))
}
