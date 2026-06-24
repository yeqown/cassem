package adm

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

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

type topologyHealth string

const (
	topologyHealthHealthy   topologyHealth = "healthy"
	topologyHealthUnhealthy topologyHealth = "unhealthy"
	topologyHealthOffline   topologyHealth = "offline"
)

const instanceHealthyWindow = 60 * time.Second
const instanceUnhealthyWindow = 120 * time.Second
const topologyInstancePageLimit = 100

type topologyDBNode struct {
	ID     string         `json:"id"`
	Addr   string         `json:"addr"`
	IP     string         `json:"ip"`
	Health topologyHealth `json:"health"`
}

type topologyAgentNode struct {
	AgentID     string            `json:"agentId"`
	Addr        string            `json:"addr"`
	IP          string            `json:"ip"`
	Health      topologyHealth    `json:"health"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type topologyInstanceNode struct {
	ClientID           string                       `json:"clientId"`
	AgentID            string                       `json:"agentId"`
	ClientIP           string                       `json:"clientIp"`
	Health             topologyHealth               `json:"health"`
	LastRenewTimestamp int64                        `json:"lastRenewTimestamp,omitempty"`
	Watching           []*concept.Instance_Watching `json:"watching,omitempty"`
}

type clusterTopologyResp struct {
	DBs       []topologyDBNode       `json:"dbs"`
	Agents    []topologyAgentNode    `json:"agents"`
	Instances []topologyInstanceNode `json:"instances"`
}

func (d app) GetAgentsHTTP(w http.ResponseWriter, r *http.Request) {
	out := []*concept.AgentInstance(nil)
	if d.ap != nil {
		out = d.ap.all()
	}
	httpx.WriteJSON(w, out)
}

func (d app) GetInstanceHTTP(w http.ResponseWriter, r *http.Request) {
	req := new(getInstanceReq)
	if err := bindRequest(r, req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	out, err := d.aggregate.GetInstance(r.Context(), req.InsId)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, out)
}

func (d app) GetInstancesHTTP(w http.ResponseWriter, r *http.Request) {
	req := new(getInstancesReq)
	if err := bindRequest(r, req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	out, err := d.aggregate.GetInstances(r.Context(), req.Seek, req.Limit)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, newInstancesResp(out))
}

func (d app) GetInstancesByElementHTTP(w http.ResponseWriter, r *http.Request) {
	req := new(getInstancesByElementReq)
	if err := bindRequest(r, req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	out, err := d.aggregate.GetInstancesByElement(r.Context(), req.AppId, req.Env, req.ElementKey)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, newInstancesResp(out))
}

func (d app) GetClusterTopologyHTTP(w http.ResponseWriter, r *http.Request) {
	out, err := d.clusterTopology(r.Context(), time.Now())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, out)
}

func (d app) clusterTopology(ctx context.Context, now time.Time) (*clusterTopologyResp, error) {
	agents := make([]*concept.AgentInstance, 0)
	if d.ap != nil {
		agents = d.ap.all()
	}

	instances, err := d.collectTopologyInstances(ctx)
	if err != nil {
		return nil, err
	}

	out := &clusterTopologyResp{
		DBs:       d.topologyDBNodes(ctx),
		Agents:    topologyAgentNodes(agents),
		Instances: topologyInstanceNodes(instances, now),
	}
	return out, nil
}

func (d app) collectTopologyInstances(ctx context.Context) ([]*concept.Instance, error) {
	out := make([]*concept.Instance, 0)
	seek := ""

	for {
		result, err := d.aggregate.GetInstances(ctx, seek, topologyInstancePageLimit)
		if err != nil {
			return nil, err
		}
		out = append(out, result.Instances...)
		if !result.HasMore || result.NextSeek == "" {
			break
		}
		seek = result.NextSeek
	}

	return out, nil
}

func (d app) topologyDBNodes(ctx context.Context) []topologyDBNode {
	endpoints := d.conf.CassemDBEndpoints
	out := make([]topologyDBNode, 0, len(endpoints))
	for i, endpoint := range endpoints {
		addr := strings.TrimSpace(endpoint)
		if addr == "" {
			continue
		}
		out = append(out, topologyDBNode{
			ID:     "db-" + strconv.Itoa(i+1),
			Addr:   addr,
			IP:     extractHost(addr),
			Health: checkDBHealth(ctx, addr),
		})
	}
	return out
}

func topologyAgentNodes(agents []*concept.AgentInstance) []topologyAgentNode {
	out := make([]topologyAgentNode, 0, len(agents))
	for _, agent := range agents {
		if agent == nil {
			continue
		}
		out = append(out, topologyAgentNode{
			AgentID:     agent.GetAgentId(),
			Addr:        agent.GetAddr(),
			IP:          extractHost(agent.GetAddr()),
			Health:      agentHealth(agent),
			Annotations: agent.GetAnnotations(),
		})
	}
	return out
}

func topologyInstanceNodes(instances []*concept.Instance, now time.Time) []topologyInstanceNode {
	out := make([]topologyInstanceNode, 0, len(instances))
	for _, instance := range instances {
		if instance == nil {
			continue
		}
		out = append(out, topologyInstanceNode{
			ClientID:           instance.GetClientId(),
			AgentID:            instance.GetAgentId(),
			ClientIP:           instance.GetClientIp(),
			Health:             instanceHealth(instance.GetLastRenewTimestamp(), now),
			LastRenewTimestamp: instance.GetLastRenewTimestamp(),
			Watching:           instance.GetWatching(),
		})
	}
	return out
}

func agentHealth(agent *concept.AgentInstance) topologyHealth {
	if agent == nil || strings.TrimSpace(agent.GetAddr()) == "" {
		return topologyHealthOffline
	}

	annotations := agent.GetAnnotations()
	ttl, ttlErr := strconv.Atoi(annotations["ttl"])
	renewInterval, renewErr := strconv.Atoi(annotations["renewInterval"])
	if ttlErr == nil && renewErr == nil && (ttl <= 0 || renewInterval <= 0 || renewInterval >= ttl) {
		return topologyHealthUnhealthy
	}

	return topologyHealthHealthy
}

func instanceHealth(lastRenewTimestamp int64, now time.Time) topologyHealth {
	if lastRenewTimestamp <= 0 {
		return topologyHealthOffline
	}

	age := now.Sub(time.Unix(lastRenewTimestamp, 0))
	if age <= instanceHealthyWindow {
		return topologyHealthHealthy
	}
	if age <= instanceUnhealthyWindow {
		return topologyHealthUnhealthy
	}
	return topologyHealthOffline
}

func checkDBHealth(ctx context.Context, addr string) topologyHealth {
	checkCtx, cancel := context.WithTimeout(ctx, 800*time.Millisecond)
	defer cancel()

	cc, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return topologyHealthOffline
	}
	defer func() { _ = cc.Close() }()

	cc.Connect()
	resp, err := grpc_health_v1.NewHealthClient(cc).Check(
		checkCtx,
		&grpc_health_v1.HealthCheckRequest{Service: "cassemdb.RaftLeader"},
		grpc.WaitForReady(true),
	)
	if err != nil {
		return topologyHealthUnhealthy
	}
	switch resp.GetStatus() {
	case grpc_health_v1.HealthCheckResponse_SERVING, grpc_health_v1.HealthCheckResponse_NOT_SERVING:
		return topologyHealthHealthy
	default:
		return topologyHealthUnhealthy
	}
}

func extractHost(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ""
	}

	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	if parsed, err := url.Parse(addr); err == nil && parsed.Host != "" {
		if host, _, splitErr := net.SplitHostPort(parsed.Host); splitErr == nil {
			return host
		}
		return parsed.Host
	}
	if strings.Contains(addr, ":") {
		return strings.Split(addr, ":")[0]
	}
	return addr
}
