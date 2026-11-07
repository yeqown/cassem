//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yeqown/cassem/internal/concept"
	"github.com/yeqown/cassem/tests/testutil"
)

func TestAgentHTTPFetchRawPublishedConfiguration(t *testing.T) {
	cluster := testutil.UseFullCluster(t)
	scope := testutil.NewRunScope(t, "agent raw HTTP fetch")
	adm := newReleaseGateAdm(t, cluster.AdmBaseURL)
	app, env := scope.App("raw"), "production"
	const raw = "{\n  \"enabled\": true\n}\n"
	createAppEnv(t, adm, app, env)
	createJSONElement(t, adm, app, env, "config", raw)
	publishReleaseGateElement(t, adm, app, env, "config", 1, concept.PublishMode_FULL, nil)
	updateElementRaw(t, adm, app, env, "config", `{"enabled":false}`)
	createJSONElement(t, adm, app, env, "draft", `{"secret":"unpublished"}`)
	token := issueAgentToken(t, adm.HTTPClient)
	client := &http.Client{Timeout: 5 * time.Second}
	t.Cleanup(client.CloseIdleConnections)

	cases := []struct {
		name    string
		keys    []string
		status  int
		errCode int
	}{
		{name: "published version with a newer draft", keys: []string{"config"}, status: http.StatusOK},
		{name: "missing", keys: []string{"missing"}, status: http.StatusNotFound, errCode: 5},
		{name: "unpublished", keys: []string{"draft"}, status: http.StatusNotFound, errCode: 5},
		{name: "multiple keys", keys: []string{"config", "draft"}, status: http.StatusBadRequest, errCode: 3},
		{name: "duplicate key", keys: []string{"config", "config"}, status: http.StatusBadRequest, errCode: 3},
		{name: "missing key parameter", status: http.StatusBadRequest, errCode: 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			endpoint := url.URL{Scheme: "http", Host: cluster.AgentEndpoint,
				Path:     fmt.Sprintf("/api/apps/%s/envs/%s/elements", app, env),
				RawQuery: url.Values{"key": tc.keys}.Encode()}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+token)
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer func() { require.NoError(t, resp.Body.Close()) }()
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.Equal(t, tc.status, resp.StatusCode, string(body))
			require.Equal(t, "no-store", resp.Header.Get("Cache-Control"))
			if tc.status == http.StatusOK {
				require.Equal(t, raw, string(body))
				require.Equal(t, "application/json", resp.Header.Get("Content-Type"))
				require.Equal(t, "inline", resp.Header.Get("Content-Disposition"))
				require.Equal(t, "config", resp.Header.Get("X-CASSEM-Key"))
				require.Equal(t, "1", resp.Header.Get("X-CASSEM-Version"))
				require.Equal(t, "true", resp.Header.Get("X-CASSEM-Published"))
				return
			}
			var failure struct {
				ErrCode int    `json:"errcode"`
				Message string `json:"errmsg"`
			}
			require.NoError(t, json.Unmarshal(body, &failure))
			require.Equal(t, tc.errCode, failure.ErrCode)
			require.NotEmpty(t, failure.Message)
			require.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
			require.NotContains(t, string(body), "secret")
			require.Empty(t, resp.Header.Get("X-CASSEM-Version"))
		})
	}
}
