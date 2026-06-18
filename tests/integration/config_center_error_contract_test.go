//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yeqown/cassem/pkg/errorx"
	"github.com/yeqown/cassem/pkg/httpx"
	"github.com/yeqown/cassem/tests/testutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func TestConfigCenterReleaseGate_APIErrorMappingSmoke(t *testing.T) {
	cluster := testutil.RequireAdmCluster(t)
	scope := testutil.NewRunScope(t, "api error mapping smoke")
	adm := newReleaseGateAdm(t, cluster.AdmBaseURL)
	app := scope.App("billing-service")
	env := "staging"
	key := scope.Key("invoice-retry-policy")

	createAppEnv(t, adm, app, env)
	createJSONElement(t, adm, app, env, key, `{"maxRetries":3}`)
	expectHTTPErrorEnvelope(t, adm.HTTPClient, http.MethodPost, fmt.Sprintf("/api/apps/%s/envs/%s/elements/%s", app, env, key), map[string]any{
		"raw":         `{"maxRetries":3}`,
		"contentType": 1,
	}, httpx.ErrorCode(errorx.Code_ALREADY_EXISTS), "ALREADY_EXISTS")
	expectHTTPErrorEnvelope(t, adm.HTTPClient, http.MethodGet, fmt.Sprintf("/api/apps/%s/envs/%s/elements/%s", app, env, "missing-invoice-policy"), nil,
		httpx.ErrorCode(errorx.Code_NOT_FOUND), "NOT_FOUND")
	expectHTTPErrorEnvelope(t, adm.HTTPClient, http.MethodPost, "/api/apps/invalid%20space", map[string]any{"name": "invalid space", "description": "bad"},
		httpx.FAILED, "identifier")

	missingKVKey := scope.TTLKey("missing", "invoice")
	require.NotEmpty(t, cluster.DBEndpoints)

	rawCC, err := grpc.NewClient(cluster.DBEndpoints[0], grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rawCC.Close() })
	rawClient := apikv.NewKVClient(rawCC)
	rawCtx, rawCancel := context.WithTimeout(context.Background(), 3*time.Second)
	_, err = rawClient.GetKV(rawCtx, &apikv.GetKVReq{Key: missingKVKey})
	rawCancel()
	require.Error(t, err)
	rawStatus, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.NotFound, rawStatus.Code())
	require.Equal(t, "NOT_FOUND", rawStatus.Message())

	cc := testutil.DialCassemDB(t, cluster.DBEndpoints, apikv.Mode_X)
	t.Cleanup(func() { _ = cc.Close() })
	client := apikv.NewKVClient(cc)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	_, err = client.GetKV(ctx, &apikv.GetKVReq{Key: missingKVKey})
	cancel()
	require.Error(t, err)
	require.True(t, errors.Is(err, errorx.Err_NOT_FOUND))
	mapped, ok := errorx.FromError(err)
	require.True(t, ok)
	require.Equal(t, errorx.Code_NOT_FOUND, mapped.Code)
	require.Equal(t, "NOT_FOUND", mapped.Message)
}

func expectHTTPErrorEnvelope(t testing.TB, client *testutil.HTTPClient, method string, path string, body any, wantCode httpx.ErrorCode, wantMessageContains string) httpx.CommonResponse {
	t.Helper()

	var r io.Reader
	if body != nil {
		buf := new(bytes.Buffer)
		require.NoError(t, json.NewEncoder(buf).Encode(body))
		r = buf
	}

	req, err := http.NewRequest(method, client.BaseURL+path, r)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if client.Session != "" {
		req.Header.Set("x-cassem-session", client.Session)
	}

	resp, err := client.Client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var out httpx.CommonResponse
	require.NoError(t, json.Unmarshal(raw, &out), "status=%d body=%s", resp.StatusCode, string(raw))
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices && out.ErrCode == httpx.OK {
		t.Fatalf("%s %s unexpectedly succeeded: status=%d body=%s", method, path, resp.StatusCode, string(raw))
	}
	require.Equal(t, wantCode, out.ErrCode, "status=%d body=%s", resp.StatusCode, string(raw))
	require.NotEmpty(t, out.ErrMessage, "status=%d body=%s", resp.StatusCode, string(raw))
	if wantMessageContains != "" {
		require.Contains(t, out.ErrMessage, wantMessageContains)
	}
	return out
}
