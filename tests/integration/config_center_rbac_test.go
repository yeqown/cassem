//go:build integration

package integration_test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/tests/testutil"
)

func TestConfigCenterReleaseGate_HTTPRBACProtectsProductionPublish(t *testing.T) {
	cluster := testutil.RequireAdmCluster(t)
	scope := testutil.NewRunScope(t, "http rbac protects production publish")
	adm := newReleaseGateAdm(t, cluster.AdmBaseURL)
	app := scope.App("payment-service")
	env := "production"
	key := scope.Key("checkout-rbac-guard")
	domain := app + "/" + env
	createAppEnv(t, adm, app, env)
	createJSONElement(t, adm, app, env, key, `{"enabled":true}`)
	publishReleaseGateElement(t, adm, app, env, key, 1, concept.PublishMode_FULL, nil)

	anonymous := testutil.NewHTTPClient(cluster.AdmBaseURL)
	anonymous.DoExpectError(t, http.MethodGet, fmt.Sprintf("/api/apps/%s/envs/%s/elements/%s", app, env, key), nil)

	viewerAccount := scope.Account("viewer.payment")
	developerAccount := scope.Account("dev.payment")
	releaseAccount := scope.Account("release.manager")
	outsiderAccount := scope.Account("outsider")
	addAccount(t, adm, viewerAccount, "viewer")
	addAccount(t, adm, developerAccount, "developer")
	addAccount(t, adm, releaseAccount, "release manager")
	addAccount(t, adm, outsiderAccount, "outsider")
	assignRole(t, adm, viewerAccount, concept.Role_VISITOR, domain)
	assignRole(t, adm, developerAccount, concept.Role_DEVELOPER, domain)
	assignRole(t, adm, releaseAccount, concept.Role_APPOWNER, domain)
	assignRole(t, adm, outsiderAccount, concept.Role_DEVELOPER, "orders-service/production")

	viewer := loginAs(t, cluster.AdmBaseURL, viewerAccount, "cassem")
	viewer.DoJSON(t, http.MethodGet, fmt.Sprintf("/api/apps/%s/envs/%s/elements/%s", app, env, key), nil, nil)
	viewer.DoExpectError(t, http.MethodPut, fmt.Sprintf("/api/apps/%s/envs/%s/elements/%s", app, env, key), map[string]any{"raw": `{"enabled":false}`})
	viewer.DoExpectError(t, http.MethodPost, fmt.Sprintf("/api/apps/%s/envs/%s/elements/%s/publish?version=1&publishMode=%d", app, env, key, concept.PublishMode_FULL), nil)

	developer := loginAs(t, cluster.AdmBaseURL, developerAccount, "cassem")
	developer.DoJSON(t, http.MethodPut, fmt.Sprintf("/api/apps/%s/envs/%s/elements/%s", app, env, key), map[string]any{"raw": `{"enabled":false}`}, nil)
	developer.DoExpectError(t, http.MethodPost, fmt.Sprintf("/api/apps/%s/envs/%s/elements/%s/publish?version=2&publishMode=%d", app, env, key, concept.PublishMode_FULL), nil)

	outsider := loginAs(t, cluster.AdmBaseURL, outsiderAccount, "cassem")
	outsider.DoExpectError(t, http.MethodGet, fmt.Sprintf("/api/apps/%s/envs/%s/elements/%s", app, env, key), nil)

	releaseManager := loginAs(t, cluster.AdmBaseURL, releaseAccount, "cassem")
	releaseManager.DoJSON(t, http.MethodPost, fmt.Sprintf("/api/apps/%s/envs/%s/elements/%s/publish?version=2&publishMode=%d", app, env, key, concept.PublishMode_FULL), nil, nil)

	adm.DoJSON(t, http.MethodGet, "/api/account/disable?account="+url.QueryEscape(viewerAccount), nil, nil)
	viewer.DoExpectError(t, http.MethodGet, fmt.Sprintf("/api/apps/%s/envs/%s/elements/%s", app, env, key), nil)
}

func addAccount(t testing.TB, adm releaseGateAdmClient, account string, nickname string) {
	t.Helper()
	adm.DoJSON(t, http.MethodPost, "/api/account/add", map[string]any{
		"account":  account,
		"password": "cassem",
		"nickname": nickname,
	}, nil)
}

func assignRole(t testing.TB, adm releaseGateAdmClient, account string, role string, domain string) {
	t.Helper()
	q := url.Values{}
	q.Set("account", account)
	q.Set("role", role)
	q.Add("domain", domain)
	adm.DoJSON(t, http.MethodGet, "/api/account/acl/assign?"+q.Encode(), nil, nil)
}

func loginAs(t testing.TB, baseURL string, account string, password string) *testutil.HTTPClient {
	t.Helper()
	client := testutil.NewHTTPClient(baseURL)
	var resp struct {
		Session string `json:"session"`
	}
	client.DoJSON(t, http.MethodPost, "/api/account/login", map[string]any{
		"account":  account,
		"password": password,
	}, &resp)
	client.Session = resp.Session
	return client
}
