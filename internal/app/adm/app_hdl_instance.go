package adm

import (
	"github.com/gin-gonic/gin"

	"github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/pkg/httpx"
)

type instanceTargetView struct {
	App string `json:"app"`
	Env string `json:"env"`
	Key string `json:"key"`
}

type instanceView struct {
	ID                 string               `json:"id,omitempty"`
	ClientID           string               `json:"clientId,omitempty"`
	AgentID            string               `json:"agentId,omitempty"`
	ClientIP           string               `json:"clientIp,omitempty"`
	Targets            []instanceTargetView `json:"targets"`
	LastRenewTimestamp int64                `json:"lastRenewTimestamp,omitempty"`
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
		Targets:            instanceTargets(instance.GetWatching()),
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

func instanceTargets(watchings []*concept.Instance_Watching) []instanceTargetView {
	seen := make(map[instanceTargetView]struct{})
	targets := make([]instanceTargetView, 0)
	for _, watching := range watchings {
		for _, key := range watching.GetWatchKeys() {
			target := instanceTargetView{App: watching.GetApp(), Env: watching.GetEnv(), Key: key}
			if target.App == "" || target.Env == "" || target.Key == "" {
				continue
			}
			if _, ok := seen[target]; ok {
				continue
			}
			seen[target] = struct{}{}
			targets = append(targets, target)
		}
	}
	return targets
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
