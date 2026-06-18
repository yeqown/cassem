//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yeqown/cassem/api/agent"
	"github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/tests/testutil"
)

func TestConfigCenterReleaseGate_FullPublishNotifiesWatchingClients(t *testing.T) {
	cluster := testutil.RequireFullCluster(t)
	scope := testutil.NewRunScope(t, "full publish notifies watching clients")
	adm := newReleaseGateAdm(t, cluster.AdmBaseURL)
	app := scope.App("inventory-service")
	env := "production"
	key := scope.Key("stock-reservation-ttl")
	unrelatedKey := scope.Key("stock-replenishment-window")

	createAppEnv(t, adm, app, env)
	createJSONElement(t, adm, app, env, key, `{"seconds":30}`)
	publishReleaseGateElement(t, adm, app, env, key, 1, concept.PublishMode_FULL, nil)
	createJSONElement(t, adm, app, env, unrelatedKey, `{"minutes":5}`)
	publishReleaseGateElement(t, adm, app, env, unrelatedKey, 1, concept.PublishMode_FULL, nil)

	target, err := agent.New(cluster.AgentEndpoint, agent.WithClientId(scope.ClientID("inventory-target")), agent.WithClientIp("127.0.0.1"))
	require.NoError(t, err)
	t.Cleanup(target.Quit)
	other, err := agent.New(cluster.AgentEndpoint, agent.WithClientId(scope.ClientID("inventory-other")), agent.WithClientIp("127.0.0.1"))
	require.NoError(t, err)
	t.Cleanup(other.Quit)

	targetC := make(chan *concept.Element, 2)
	otherC := make(chan *concept.Element, 2)
	watchCtx, stopWatch := context.WithCancel(context.Background())
	defer stopWatch()
	require.NoError(t, target.Watch(watchCtx, app, env, func(next *concept.Element) { targetC <- next }, key))
	require.NoError(t, other.Watch(watchCtx, app, env, func(next *concept.Element) { otherC <- next }, unrelatedKey))
	waitWatchRegistered(t, cluster.AgentEndpoint, app, env, key, scope.ClientID("inventory-target")+"@127.0.0.1", targetC)
	waitWatchRegistered(t, cluster.AgentEndpoint, app, env, unrelatedKey, scope.ClientID("inventory-other")+"@127.0.0.1", otherC)

	updateElementRaw(t, adm, app, env, key, `{"seconds":45}`)
	publishReleaseGateElement(t, adm, app, env, key, 2, concept.PublishMode_FULL, nil)
	waitForRaw(t, targetC, `{"seconds":45}`, 10*time.Second)
	assertNoRaw(t, otherC, `{"seconds":45}`, 500*time.Millisecond)
}

func TestConfigCenterReleaseGate_GrayPublishTargetsSelectedCheckoutInstance(t *testing.T) {
	cluster := testutil.RequireFullCluster(t)
	scope := testutil.NewRunScope(t, "gray publish targets selected checkout instance")
	adm := newReleaseGateAdm(t, cluster.AdmBaseURL)
	app := scope.App("checkout-service")
	env := "production"
	key := scope.Key("pricing-discount-strategy")
	stable := `{"strategy":"stable","maxDiscount":15}`
	gray := `{"strategy":"risk-aware","maxDiscount":20}`

	createAppEnv(t, adm, app, env)
	createJSONElement(t, adm, app, env, key, stable)
	publishReleaseGateElement(t, adm, app, env, key, 1, concept.PublishMode_FULL, nil)
	targetID := scope.ClientID("checkout-worker-a")
	otherID := scope.ClientID("checkout-worker-b")
	target, err := agent.New(cluster.AgentEndpoint, agent.WithClientId(targetID), agent.WithClientIp("127.0.0.1"))
	require.NoError(t, err)
	t.Cleanup(target.Quit)
	other, err := agent.New(cluster.AgentEndpoint, agent.WithClientId(otherID), agent.WithClientIp("127.0.0.1"))
	require.NoError(t, err)
	t.Cleanup(other.Quit)

	targetC := make(chan *concept.Element, 2)
	otherC := make(chan *concept.Element, 2)
	watchCtx, stopWatch := context.WithCancel(context.Background())
	defer stopWatch()
	require.NoError(t, target.Watch(watchCtx, app, env, func(next *concept.Element) { targetC <- next }, key))
	require.NoError(t, other.Watch(watchCtx, app, env, func(next *concept.Element) { otherC <- next }, key))
	waitWatchRegistered(t, cluster.AgentEndpoint, app, env, key, targetID+"@127.0.0.1", targetC)
	waitWatchRegistered(t, cluster.AgentEndpoint, app, env, key, otherID+"@127.0.0.1", otherC)

	updateElementRaw(t, adm, app, env, key, gray)
	publishReleaseGateElement(t, adm, app, env, key, 2, concept.PublishMode_GRAY, []string{targetID + "@127.0.0.1"})
	waitForRaw(t, targetC, gray, 10*time.Second)
	assertNoRaw(t, otherC, gray, 500*time.Millisecond)
}

func TestConfigCenterReleaseGate_AgentReadThroughCacheRefreshesAfterPublish(t *testing.T) {
	cluster := testutil.RequireFullCluster(t)
	scope := testutil.NewRunScope(t, "agent read through cache refreshes after publish")
	adm := newReleaseGateAdm(t, cluster.AdmBaseURL)
	app := scope.App("catalog-service")
	env := "production"
	key := scope.Key("product-detail-cache-policy")
	v1 := `{"ttlSeconds":60,"staleWhileRevalidate":true}`
	v2 := `{"ttlSeconds":30,"staleWhileRevalidate":false}`

	createAppEnv(t, adm, app, env)
	createJSONElement(t, adm, app, env, key, v1)
	publishReleaseGateElement(t, adm, app, env, key, 1, concept.PublishMode_FULL, nil)
	reader, err := agent.New(cluster.AgentEndpoint, agent.WithClientId(scope.ClientID("catalog-reader")), agent.WithClientIp("127.0.0.1"))
	require.NoError(t, err)
	t.Cleanup(reader.Quit)
	waitAgentRaw(t, reader, app, env, key, v1)
	waitAgentRaw(t, reader, app, env, key, v1)

	updateElementRaw(t, adm, app, env, key, v2)
	publishReleaseGateElement(t, adm, app, env, key, 2, concept.PublishMode_FULL, nil)
	waitAgentRawWithin(t, reader, app, env, key, v2, 20*time.Second)
}
