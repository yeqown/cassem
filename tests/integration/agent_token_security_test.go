//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yeqown/cassem/internal/concept"
	sdk "github.com/yeqown/cassem/sdk/go"
	"github.com/yeqown/cassem/tests/testutil"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAgentAccessTokenSecurityAcrossHTTPAndGRPC(t *testing.T) {
	cluster := testutil.UseFullCluster(t)
	scope := testutil.NewRunScope(t, "agent token security")
	adm := newReleaseGateAdm(t, cluster.AdmBaseURL)
	app, otherApp := scope.App("allowed"), scope.App("forbidden")
	env, otherEnv, key := "production", "staging", "config"
	const raw = `{"enabled":true}`
	for _, namespace := range []struct{ app, env string }{{app, env}, {otherApp, env}} {
		createAppEnv(t, adm, namespace.app, namespace.env)
		createJSONElement(t, adm, namespace.app, namespace.env, key, raw)
		publishReleaseGateElement(t, adm, namespace.app, namespace.env, key, 1, concept.PublishMode_FULL, nil)
	}
	adm.DoJSON(t, http.MethodPost, fmt.Sprintf("/api/apps/%s/envs/%s", app, otherEnv), nil, nil)
	createJSONElement(t, adm, app, otherEnv, key, raw)
	publishReleaseGateElement(t, adm, app, otherEnv, key, 1, concept.PublishMode_FULL, nil)

	account := scope.Account("token-viewer")
	addAccount(t, adm, account, "token security visitor")
	assignRole(t, adm, account, concept.Role_VISITOR, app+"/"+env)
	token, revoke := issueAgentTokenForAccount(t, adm.HTTPClient, account)
	newClient := func(label, credential string) *sdk.Client {
		t.Helper()
		client, err := sdk.New(cluster.AgentEndpoint,
			sdk.WithClientID(scope.ClientID(label)), sdk.WithClientIP("127.0.0.1"), sdk.WithToken(credential))
		require.NoError(t, err, "SDK registration must retain its existing contract")
		t.Cleanup(client.Quit)
		return client
	}
	reader := newClient("authorized-reader", token)
	anonymous := newClient("anonymous-reader", "")
	tampered := token + "x"
	invalid := newClient("invalid-reader", tampered)
	httpClient := &http.Client{Timeout: 5 * time.Second}
	t.Cleanup(httpClient.CloseIdleConnections)

	assertAccess := func(t *testing.T, label, credential string, client *sdk.Client, targetApp, targetEnv string, wantHTTP int, wantGRPC codes.Code) {
		t.Helper()
		endpoint := url.URL{Scheme: "http", Host: cluster.AgentEndpoint,
			Path:     fmt.Sprintf("/api/apps/%s/envs/%s/elements", targetApp, targetEnv),
			RawQuery: url.Values{"key": {key}}.Encode()}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
		require.NoError(t, err)
		if credential != "" {
			req.Header.Set("Authorization", "Bearer "+credential)
		}
		resp, err := httpClient.Do(req)
		require.NoError(t, err, "%s HTTP request failed", label)
		defer func() { require.NoError(t, resp.Body.Close()) }()
		require.Equal(t, wantHTTP, resp.StatusCode, "%s HTTP status", label)
		if wantHTTP == http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err, "%s HTTP response reading", label)
			require.Equal(t, raw, string(body), "%s HTTP published bytes", label)
			require.Equal(t, "application/json", resp.Header.Get("Content-Type"), "%s HTTP content type", label)
			require.Equal(t, "inline", resp.Header.Get("Content-Disposition"), "%s HTTP disposition", label)
			require.Equal(t, key, resp.Header.Get("X-CASSEM-Key"), "%s HTTP key", label)
			require.Equal(t, "1", resp.Header.Get("X-CASSEM-Version"), "%s HTTP version", label)
			require.Equal(t, "true", resp.Header.Get("X-CASSEM-Published"), "%s HTTP published state", label)
			require.Equal(t, "no-store", resp.Header.Get("Cache-Control"), "%s HTTP cache policy", label)
		}
		elems, err := client.GetElement(ctx, targetApp, targetEnv, key)
		require.Equal(t, wantGRPC, status.Code(err), "%s gRPC status", label)
		if wantGRPC == codes.OK {
			require.Len(t, elems, 1)
			require.Equal(t, raw, string(elems[0].GetRaw()), "%s gRPC published bytes", label)
		}
	}

	assertAccess(t, "authorized and cache warmed", token, reader, app, env, http.StatusOK, codes.OK)
	assertAccess(t, "missing credential", "", anonymous, app, env, http.StatusUnauthorized, codes.Unauthenticated)
	assertAccess(t, "tampered credential", tampered, invalid, app, env, http.StatusUnauthorized, codes.Unauthenticated)
	assertAccess(t, "other environment", token, reader, app, otherEnv, http.StatusForbidden, codes.PermissionDenied)
	assertAccess(t, "other application", token, reader, otherApp, env, http.StatusForbidden, codes.PermissionDenied)

	// Keep the same credential and connection to verify authorization is evaluated
	// again even when the requested configuration is already cached by the agent.
	q := url.Values{"account": {account}, "role": {concept.Role_VISITOR}, "domain": {app + "/" + env}}
	adm.DoJSON(t, http.MethodGet, "/api/account/acl/revoke?"+q.Encode(), nil, nil)
	assertAccess(t, "role revoked with warm cache", token, reader, app, env, http.StatusForbidden, codes.PermissionDenied)
	assignRole(t, adm, account, concept.Role_VISITOR, app+"/"+env)
	assertAccess(t, "role restored on same token", token, reader, app, env, http.StatusOK, codes.OK)
	revoke()
	assertAccess(t, "token revoked with warm cache", token, reader, app, env, http.StatusUnauthorized, codes.Unauthenticated)

	freshToken, _ := issueAgentTokenForAccount(t, adm.HTTPClient, account)
	freshReader := newClient("fresh-reader", freshToken)
	assertAccess(t, "fresh credential before disable", freshToken, freshReader, app, env, http.StatusOK, codes.OK)
	adm.DoJSON(t, http.MethodGet, "/api/account/disable?account="+url.QueryEscape(account), nil, nil)
	assertAccess(t, "user disabled with warm cache", freshToken, freshReader, app, env, http.StatusUnauthorized, codes.Unauthenticated)
}
