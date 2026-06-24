package adm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/yeqown/cassem/api/concept"
)

type authHTTPFakeRBAC struct {
	user         *concept.User
	allow        bool
	enforceCalls []authHTTPEnforceCall
}

type authHTTPEnforceCall struct {
	subject string
	domain  string
	object  string
	act     string
}

func (f *authHTTPFakeRBAC) GetUser(account string) (*concept.User, error) {
	return f.user, nil
}

func (f *authHTTPFakeRBAC) GetUsers(string, int) (*concept.GetUsersResult, error) {
	return nil, nil
}

func (f *authHTTPFakeRBAC) GetUserRoles(string) ([]string, error) {
	return nil, nil
}

func (f *authHTTPFakeRBAC) GetUserRoleBindings(string) ([]concept.RoleBinding, error) {
	return nil, nil
}

func (f *authHTTPFakeRBAC) ListDomainOptions() ([]string, error) {
	return nil, nil
}

func (f *authHTTPFakeRBAC) AddUser(*concept.User) error {
	return nil
}

func (f *authHTTPFakeRBAC) DisableUser(string) error {
	return nil
}

func (f *authHTTPFakeRBAC) ResetUser(string, string) error {
	return nil
}

func (f *authHTTPFakeRBAC) AssignRole(string, string, ...string) error {
	return nil
}

func (f *authHTTPFakeRBAC) RevokeRole(string, string, ...string) error {
	return nil
}

func (f *authHTTPFakeRBAC) Enforce(subject, domain, object, act string) (bool, error) {
	f.enforceCalls = append(f.enforceCalls, authHTTPEnforceCall{
		subject: subject,
		domain:  domain,
		object:  object,
		act:     act,
	})
	return f.allow, nil
}

func (f *authHTTPFakeRBAC) AutoMigrate() error {
	return nil
}

func (f *authHTTPFakeRBAC) BootstrapAdmin(string, string, string) error {
	return nil
}

func newAuthHTTPFakeRBAC(allow bool) *authHTTPFakeRBAC {
	return &authHTTPFakeRBAC{
		allow: allow,
		user: &concept.User{
			Account: "alice",
			Salt:    "salt-1",
			Status:  concept.User_NORMAL,
		},
	}
}

func newAuthHTTPToken(t *testing.T, secret string) string {
	t.Helper()
	token, err := EncodeSession(&Session{
		Account:   "alice",
		Salt:      "salt-1",
		ExpiredAt: time.Now().Add(time.Hour).Unix(),
	}, secret)
	require.NoError(t, err)
	return token
}

func requestWithChiParams(method, target string, params map[string]string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	ctx := chi.NewRouteContext()
	for key, value := range params {
		ctx.URLParams.Add(key, value)
	}
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, ctx))
}

func TestWithRouteAuthSkipsUnmappedRoute(t *testing.T) {
	rbac := newAuthHTTPFakeRBAC(false)
	handler := withRouteAuth(rbac, "secret", http.MethodGet, "/api/debug/free", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/debug/free", nil))

	require.Equal(t, http.StatusAccepted, w.Code)
	require.Empty(t, rbac.enforceCalls)
}

func TestWithRouteAuthRejectsMissingSessionForMappedRoute(t *testing.T) {
	rbac := newAuthHTTPFakeRBAC(true)
	handler := withRouteAuth(rbac, "secret", http.MethodGet, "/api/apps/:appId", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/apps/app1", nil))

	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Empty(t, rbac.enforceCalls)
}

func TestWithRouteAuthEnforcesMappedRouteWithAppEnvDomain(t *testing.T) {
	rbac := newAuthHTTPFakeRBAC(true)
	handler := withRouteAuth(rbac, "secret", http.MethodGet, "/api/apps/:appId/envs/:env/elements", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := GetSessionFromRequest(r)
		require.True(t, ok)
		require.Equal(t, "alice", sess.Account)
		require.Equal(t, "alice", concept.OperatorFromContext(r.Context()))
		w.WriteHeader(http.StatusOK)
	}))

	req := requestWithChiParams(http.MethodGet, "/api/apps/app1/envs/prod/elements", map[string]string{
		"appId": "app1",
		"env":   "prod",
	})
	req.Header.Set("x-cassem-session", newAuthHTTPToken(t, "secret"))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, []authHTTPEnforceCall{{
		subject: "alice",
		domain:  "app1/prod",
		object:  concept.Object_APP,
		act:     concept.Action_READ,
	}}, rbac.enforceCalls)
}

func TestWithRouteAuthRejectsRBACDeny(t *testing.T) {
	rbac := newAuthHTTPFakeRBAC(false)
	handler := withRouteAuth(rbac, "secret", http.MethodGet, "/api/apps/:appId", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := requestWithChiParams(http.MethodGet, "/api/apps/app1", map[string]string{"appId": "app1"})
	req.Header.Set("x-cassem-session", newAuthHTTPToken(t, "secret"))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Equal(t, []authHTTPEnforceCall{{
		subject: "alice",
		domain:  "app1/*",
		object:  concept.Object_APP,
		act:     concept.Action_READ,
	}}, rbac.enforceCalls)
}
