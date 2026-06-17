package app

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/pkg/httpx"
)

type instanceView struct {
	ID                 string                       `json:"id,omitempty"`
	ClientID           string                       `json:"clientId,omitempty"`
	AgentID            string                       `json:"agentId,omitempty"`
	ClientIP           string                       `json:"clientIp,omitempty"`
	App                string                       `json:"app"`
	Env                string                       `json:"env"`
	Key                string                       `json:"key"`
	Watching           []*concept.Instance_Watching `json:"watching,omitempty"`
	LastRenewTimestamp int64                        `json:"lastRenewTimestamp,omitempty"`
}

type instancesResp struct {
	concept.CommonPager

	Instances []instanceView `json:"instances"`
}

func newInstanceView(instance *concept.Instance) instanceView {
	if instance == nil {
		return instanceView{}
	}

	return instanceView{
		ID:                 instance.Id(),
		ClientID:           instance.GetClientId(),
		AgentID:            instance.GetAgentId(),
		ClientIP:           instance.GetClientIp(),
		App:                joinUnique(instanceWatchingApps(instance.GetWatching())),
		Env:                joinUnique(instanceWatchingEnvs(instance.GetWatching())),
		Key:                joinUnique(instanceWatchingKeys(instance.GetWatching())),
		Watching:           instance.GetWatching(),
		LastRenewTimestamp: instance.GetLastRenewTimestamp(),
	}
}

func newInstancesResp(result *concept.GetInstancesResult) instancesResp {
	if result == nil {
		return instancesResp{}
	}

	out := instancesResp{
		CommonPager: result.CommonPager,
		Instances:   make([]instanceView, 0, len(result.Instances)),
	}
	for _, instance := range result.Instances {
		out.Instances = append(out.Instances, newInstanceView(instance))
	}
	return out
}

func instanceWatchingApps(watchings []*concept.Instance_Watching) []string {
	values := make([]string, 0, len(watchings))
	for _, watching := range watchings {
		values = append(values, watching.GetApp())
	}
	return values
}

func instanceWatchingEnvs(watchings []*concept.Instance_Watching) []string {
	values := make([]string, 0, len(watchings))
	for _, watching := range watchings {
		values = append(values, watching.GetEnv())
	}
	return values
}

func instanceWatchingKeys(watchings []*concept.Instance_Watching) []string {
	values := make([]string, 0, len(watchings))
	for _, watching := range watchings {
		values = append(values, watching.GetWatchKeys()...)
	}
	return values
}

func joinUnique(values []string) string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return strings.Join(out, ", ")
}

func (d app) GetInstance(c *gin.Context) {
	req := new(getInstanceReq)
	if err := c.ShouldBindUri(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	out, err := d.aggregate.GetInstance(c.Request.Context(), req.InsId)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, out)
}

func (d app) GetInstances(c *gin.Context) {
	req := new(getInstancesReq)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	out, err := d.aggregate.GetInstances(c.Request.Context(), req.Seek, req.Limit)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, newInstancesResp(out))
}

func (d app) GetInstancesByElement(c *gin.Context) {
	req := new(getInstancesByElementReq)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	out, err := d.aggregate.GetInstancesByElement(c.Request.Context(), req.AppId, req.Env, req.ElementKey)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, newInstancesResp(out))
}
