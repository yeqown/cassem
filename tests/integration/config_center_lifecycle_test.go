//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yeqown/cassem/api/agent"
	"github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/tests/testutil"
)

func TestConfigCenterReleaseGate_OnlinePaymentFeatureFlagLifecycle(t *testing.T) {
	cluster := testutil.RequireFullCluster(t)
	scope := testutil.NewRunScope(t, "online payment feature flag lifecycle")
	adm := newReleaseGateAdm(t, cluster.AdmBaseURL)
	app := scope.App("payment-service")
	env := "production"
	key := scope.Key("checkout-feature-dynamic-risk-control")
	v1 := `{"enabled":false,"threshold":0.75,"regions":["CN","SG"]}`
	v2 := `{"enabled":true,"threshold":0.82,"regions":["CN","SG","US"]}`

	createAppEnv(t, adm, app, env)
	createJSONElement(t, adm, app, env, key, v1)
	publishReleaseGateElement(t, adm, app, env, key, 1, concept.PublishMode_FULL, nil)

	agentClient, err := agent.New(cluster.AgentEndpoint, agent.WithClientId(scope.ClientID("checkout-worker")), agent.WithClientIp("127.0.0.1"))
	require.NoError(t, err)
	t.Cleanup(agentClient.Quit)
	waitAgentRaw(t, agentClient, app, env, key, v1)

	received := make(chan *concept.Element, 2)
	watchCtx, stopWatch := context.WithCancel(context.Background())
	defer stopWatch()
	require.NoError(t, agentClient.Watch(watchCtx, app, env, func(next *concept.Element) {
		select {
		case received <- next:
		default:
		}
	}, key))
	waitWatchRegistered(t, cluster.AgentEndpoint, app, env, key, scope.ClientID("checkout-worker")+"@127.0.0.1", received)

	updateElementRaw(t, adm, app, env, key, v2)
	latest := getElement(t, adm, app, env, key, 2)
	require.Equal(t, int32(2), latest.GetVersion())
	require.False(t, latest.GetPublished())
	waitAgentRaw(t, agentClient, app, env, key, v1)

	publishReleaseGateElement(t, adm, app, env, key, 2, concept.PublishMode_FULL, nil)
	waitForRaw(t, received, v2, 10*time.Second)
	waitAgentRaw(t, agentClient, app, env, key, v2)

	current := getElement(t, adm, app, env, key, 0)
	require.Equal(t, int32(2), current.GetVersion())
	require.Equal(t, v2, string(current.GetRaw()))
	var diff map[string]any
	adm.DoJSON(t, http.MethodGet, fmt.Sprintf("/api/apps/%s/envs/%s/elements/%s/diff?base=1&compare=2", app, env, key), nil, &diff)
	require.NotEmpty(t, diff)
	var ops concept.GetElementOperationsResult
	adm.DoJSON(t, http.MethodGet, fmt.Sprintf("/api/apps/%s/envs/%s/elements/%s/operations", app, env, key), nil, &ops)
	require.NotEmpty(t, ops.Operations)
}

func TestConfigCenterReleaseGate_PreventUnpublishedDraftOverwrite(t *testing.T) {
	cluster := testutil.RequireAdmCluster(t)
	scope := testutil.NewRunScope(t, "prevent unpublished draft overwrite")
	adm := newReleaseGateAdm(t, cluster.AdmBaseURL)
	app := scope.App("logistics-service")
	env := "staging"
	key := scope.Key("dispatch-route-assignment-policy")

	createAppEnv(t, adm, app, env)
	createJSONElement(t, adm, app, env, key, `{"strategy":"nearest","maxBatchSize":20}`)
	adm.DoExpectError(t, http.MethodPut, fmt.Sprintf("/api/apps/%s/envs/%s/elements/%s", app, env, key), map[string]any{"raw": `{"strategy":"balanced","maxBatchSize":30}`})
	publishReleaseGateElement(t, adm, app, env, key, 1, concept.PublishMode_FULL, nil)
	updateElementRaw(t, adm, app, env, key, `{"strategy":"balanced","maxBatchSize":30}`)
	adm.DoExpectError(t, http.MethodPut, fmt.Sprintf("/api/apps/%s/envs/%s/elements/%s", app, env, key), map[string]any{"raw": `{"strategy":"cost-aware","maxBatchSize":25}`})
}

func TestConfigCenterReleaseGate_RollbackPaymentLimitPolicyToStableVersion(t *testing.T) {
	cluster := testutil.RequireFullCluster(t)
	scope := testutil.NewRunScope(t, "rollback payment limit policy")
	adm := newReleaseGateAdm(t, cluster.AdmBaseURL)
	app := scope.App("payment-service")
	env := "production"
	key := scope.Key("checkout-limit-policy")
	v1 := `{"maxAmount":5000,"currency":"USD","riskLevel":"standard"}`
	v2 := `{"maxAmount":8000,"currency":"USD","riskLevel":"relaxed"}`

	createAppEnv(t, adm, app, env)
	createJSONElement(t, adm, app, env, key, v1)
	publishReleaseGateElement(t, adm, app, env, key, 1, concept.PublishMode_FULL, nil)
	updateElementRaw(t, adm, app, env, key, v2)
	publishReleaseGateElement(t, adm, app, env, key, 2, concept.PublishMode_FULL, nil)
	adm.DoJSON(t, http.MethodPost, fmt.Sprintf("/api/apps/%s/envs/%s/elements/%s/rollback?version=1", app, env, key), nil, nil)

	current := getElement(t, adm, app, env, key, 0)
	require.Equal(t, v1, string(current.GetRaw()))

	agentClient, err := agent.New(cluster.AgentEndpoint, agent.WithClientId(scope.ClientID("payment-reader")), agent.WithClientIp("127.0.0.1"))
	require.NoError(t, err)
	t.Cleanup(agentClient.Quit)
	waitAgentRaw(t, agentClient, app, env, key, v1)
}
