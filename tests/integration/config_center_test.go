//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yeqown/cassem/api/agent"
	"github.com/yeqown/cassem/api/concept"
	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
	"github.com/yeqown/cassem/tests/testutil"
	"google.golang.org/grpc"
)

func TestElementLifecycleThroughAdmAndAgent(t *testing.T) {
	cluster := testutil.UseFullCluster(t)
	adm := newAdmClient(t, cluster.AdmBaseURL)
	app, env, key := uniqueNames(t)

	createAppEnvElement(t, adm, app, env, key, "value-v1")
	publishElement(t, adm, app, env, key, 1, concept.PublishMode_FULL, nil)

	agentClient, err := agent.New(cluster.AgentEndpoint,
		agent.WithClientId("client-lifecycle"),
		agent.WithClientIp("127.0.0.1"))
	require.NoError(t, err)
	t.Cleanup(agentClient.Quit)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	elems, err := agentClient.GetElement(ctx, app, env, key)
	require.NoError(t, err)
	require.Len(t, elems, 1)
	require.Equal(t, "value-v1", string(elems[0].GetRaw()))

	received := make(chan *concept.Element, 1)
	watchCtx, stopWatch := context.WithCancel(context.Background())
	defer stopWatch()
	require.NoError(t, agentClient.Watch(watchCtx, app, env, func(next *concept.Element) {
		select {
		case received <- next:
		default:
		}
	}, key))
	waitDeliveryToInstance(t, cluster.AgentEndpoint, app, env, key, "client-lifecycle@127.0.0.1", received)

	adm.put(t, fmt.Sprintf("/api/apps/%s/envs/%s/elements/%s", app, env, key), map[string]any{"raw": "value-v2"}, nil)
	publishElement(t, adm, app, env, key, 2, concept.PublishMode_FULL, nil)

	waitForElementRaw(t, received, "value-v2", 10*time.Second)
}

func TestOperationAuditAndResetUser(t *testing.T) {
	cluster := testutil.UseAdmCluster(t)
	adm := newAdmClient(t, cluster.AdmBaseURL)
	app, env, key := uniqueNames(t)

	createAppEnvElement(t, adm, app, env, key, "value-v1")
	publishElement(t, adm, app, env, key, 1, concept.PublishMode_FULL, nil)
	adm.put(t, fmt.Sprintf("/api/apps/%s/envs/%s/elements/%s", app, env, key), map[string]any{"raw": "value-v2"}, nil)

	var ops concept.GetElementOperationsResult
	adm.get(t, fmt.Sprintf("/api/apps/%s/envs/%s/elements/%s/operations", app, env, key), &ops)
	require.GreaterOrEqual(t, len(ops.Operations), 3)
	require.Equal(t, "superadmin@example.com", ops.Operations[0].GetOperator())
	require.Equal(t, key, ops.Operations[0].GetOperatedKey())
	require.Equal(t, concept.ElementOperation_SET, ops.Operations[0].GetOp())

	account := fmt.Sprintf("user-%d@example.com", time.Now().UnixNano())
	adm.post(t, "/api/account/add", map[string]any{
		"account":  account,
		"password": "old-password",
		"nickname": "integration user",
	}, nil)

	loginClient := testutil.NewHTTPClient(cluster.AdmBaseURL)
	var loginResp struct {
		User    *concept.User `json:"user"`
		Session string        `json:"session"`
	}
	loginClient.DoJSON(t, http.MethodPost, "/api/account/login", map[string]any{
		"account":  account,
		"password": "old-password",
	}, &loginResp)
	require.NotEmpty(t, loginResp.Session)

	adm.post(t, "/api/account/reset", map[string]any{
		"account":  account,
		"password": "new-password",
	}, nil)
	loginClient.DoExpectError(t, http.MethodPost, "/api/account/login", map[string]any{
		"account":  account,
		"password": "old-password",
	})
	loginClient.DoJSON(t, http.MethodPost, "/api/account/login", map[string]any{
		"account":  account,
		"password": "new-password",
	}, &loginResp)
	require.NotEmpty(t, loginResp.Session)

	adm.expectError(t, http.MethodPost, "/api/account/reset", map[string]any{
		"account":  "superadmin@example.com",
		"password": "nope",
	})
}

func TestGrayPublishToInstance(t *testing.T) {
	cluster := testutil.UseFullCluster(t)
	adm := newAdmClient(t, cluster.AdmBaseURL)
	app, env, key := uniqueNames(t)

	createAppEnvElement(t, adm, app, env, key, "value-v1")
	publishElement(t, adm, app, env, key, 1, concept.PublishMode_FULL, nil)

	targetID := "gray-target"
	otherID := "gray-other"
	target, err := agent.New(cluster.AgentEndpoint, agent.WithClientId(targetID), agent.WithClientIp("127.0.0.1"))
	require.NoError(t, err)
	t.Cleanup(target.Quit)
	other, err := agent.New(cluster.AgentEndpoint, agent.WithClientId(otherID), agent.WithClientIp("127.0.0.1"))
	require.NoError(t, err)
	t.Cleanup(other.Quit)

	targetC := make(chan *concept.Element, 1)
	otherC := make(chan *concept.Element, 1)
	watchCtx, stopWatch := context.WithCancel(context.Background())
	defer stopWatch()
	require.NoError(t, target.Watch(watchCtx, app, env, func(next *concept.Element) { targetC <- next }, key))
	require.NoError(t, other.Watch(watchCtx, app, env, func(next *concept.Element) { otherC <- next }, key))
	waitDeliveryToInstance(t, cluster.AgentEndpoint, app, env, key, targetID+"@127.0.0.1", targetC)
	waitDeliveryToInstance(t, cluster.AgentEndpoint, app, env, key, otherID+"@127.0.0.1", otherC)

	adm.put(t, fmt.Sprintf("/api/apps/%s/envs/%s/elements/%s", app, env, key), map[string]any{"raw": "gray-value"}, nil)
	publishElement(t, adm, app, env, key, 2, concept.PublishMode_GRAY, []string{targetID + "@127.0.0.1"})

	waitForElementRaw(t, targetC, "gray-value", 10*time.Second)
	assertNoElementRaw(t, otherC, "gray-value", 500*time.Millisecond)
}

func TestKVTTLExpireThroughDB(t *testing.T) {
	cluster := testutil.UseDBCluster(t)
	cc := testutil.DialCassemDB(t, cluster.DBEndpoints, apicassemdb.Mode_X)
	t.Cleanup(func() { _ = cc.Close() })
	client := apicassemdb.NewKVClient(cc)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	dir := fmt.Sprintf("tests/ttl/%d", time.Now().UnixNano())
	key := dir + "/item"
	_, err := client.SetKV(ctx, &apicassemdb.SetKVReq{Key: key, Val: []byte("ttl"), Ttl: 1, Overwrite: true})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, err := client.GetKV(ctx, &apicassemdb.GetKVReq{Key: key})
		return err != nil
	}, 5*time.Second, 200*time.Millisecond)

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	r, err := client.Range(ctx, &apicassemdb.RangeReq{Key: dir, Limit: 100})
	require.NoError(t, err)
	for _, entity := range r.GetEntities() {
		require.NotEqual(t, key, entity.GetKey())
	}
}

type admClient struct {
	*testutil.HTTPClient
}

func newAdmClient(t testing.TB, baseURL string) admClient {
	t.Helper()
	c := testutil.NewHTTPClient(baseURL)
	c.Session = testutil.LoginSuperadmin(t, baseURL)
	return admClient{HTTPClient: c}
}

func (c admClient) get(t testing.TB, path string, data any) {
	t.Helper()
	c.DoJSON(t, http.MethodGet, path, nil, data)
}

func (c admClient) post(t testing.TB, path string, body any, data any) {
	t.Helper()
	c.DoJSON(t, http.MethodPost, path, body, data)
}

func (c admClient) put(t testing.TB, path string, body any, data any) {
	t.Helper()
	c.DoJSON(t, http.MethodPut, path, body, data)
}

func (c admClient) expectError(t testing.TB, method string, path string, body any) {
	t.Helper()
	c.DoExpectError(t, method, path, body)
}

func uniqueNames(t testing.TB) (string, string, string) {
	t.Helper()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	return "app" + suffix, "env" + suffix, "key" + suffix
}

func createAppEnvElement(t testing.TB, adm admClient, app string, env string, key string, raw string) {
	t.Helper()
	adm.post(t, "/api/apps/"+app, map[string]any{"name": app, "description": "integration test app"}, nil)
	adm.post(t, fmt.Sprintf("/api/apps/%s/envs/%s", app, env), nil, nil)
	adm.post(t, fmt.Sprintf("/api/apps/%s/envs/%s/elements/%s", app, env, key), map[string]any{
		"raw":         raw,
		"contentType": concept.ContentType_PLAINTEXT,
	}, nil)
}

func publishElement(t testing.TB, adm admClient, app string, env string, key string, version uint32, mode concept.PublishingMode, instanceIDs []string) {
	t.Helper()
	v := url.Values{}
	v.Set("version", fmt.Sprintf("%d", version))
	v.Set("publishMode", fmt.Sprintf("%d", mode))
	for _, id := range instanceIDs {
		v.Add("instanceId", id)
	}
	adm.post(t, fmt.Sprintf("/api/apps/%s/envs/%s/elements/%s/publish?%s", app, env, key, v.Encode()), nil, nil)
}

func waitDeliveryToInstance(t testing.TB, endpoint string, app string, env string, key string, instanceID string, received <-chan *concept.Element) {
	t.Helper()

	cc, err := grpc.Dial(endpoint, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(5*time.Second))
	require.NoError(t, err)
	defer cc.Close()
	client := agent.NewDeliveryClient(cc)

	probe := &concept.Element{Metadata: &concept.ElementMetadata{App: app, Env: env, Key: key}, Raw: []byte("probe")}
	require.Eventually(t, func() bool {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_, err := client.Dispatch(ctx, &agent.DispatchReq{Elems: []*concept.Element{probe}, InstanceIds: []string{instanceID}})
		cancel()
		if err != nil {
			return false
		}
		select {
		case got := <-received:
			return string(got.GetRaw()) == "probe"
		case <-time.After(100 * time.Millisecond):
			return false
		}
	}, 10*time.Second, 200*time.Millisecond)

	for {
		select {
		case <-received:
		default:
			return
		}
	}
}

func waitForElementRaw(t testing.TB, received <-chan *concept.Element, expected string, timeout time.Duration) {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case got := <-received:
			if got != nil && string(got.GetRaw()) == expected {
				return
			}
		case <-deadline:
			t.Fatalf("did not receive expected element raw=%q", expected)
		}
	}
}

func assertNoElementRaw(t testing.TB, received <-chan *concept.Element, unexpected string, timeout time.Duration) {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case got := <-received:
			if got != nil && string(got.GetRaw()) == unexpected {
				t.Fatalf("received unexpected element raw=%q", unexpected)
			}
		case <-deadline:
			return
		}
	}
}
