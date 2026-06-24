package adm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/pkg/conf"
)

type routeTestAggregate struct {
	concept.AdmAggregate
	*authHTTPFakeRBAC
}

func (a routeTestAggregate) GetUser(account string) (*concept.User, error) {
	return a.authHTTPFakeRBAC.GetUser(account)
}

func (a routeTestAggregate) GetUsers(seek string, limit int) (*concept.GetUsersResult, error) {
	return a.authHTTPFakeRBAC.GetUsers(seek, limit)
}

func (a routeTestAggregate) GetUserRoles(account string) ([]string, error) {
	return a.authHTTPFakeRBAC.GetUserRoles(account)
}

func (a routeTestAggregate) GetUserRoleBindings(account string) ([]concept.RoleBinding, error) {
	return a.authHTTPFakeRBAC.GetUserRoleBindings(account)
}

func (a routeTestAggregate) ListDomainOptions() ([]string, error) {
	return a.authHTTPFakeRBAC.ListDomainOptions()
}

func (a routeTestAggregate) AddUser(u *concept.User) error {
	return a.authHTTPFakeRBAC.AddUser(u)
}

func (a routeTestAggregate) DisableUser(account string) error {
	return a.authHTTPFakeRBAC.DisableUser(account)
}

func (a routeTestAggregate) ResetUser(account, password string) error {
	return a.authHTTPFakeRBAC.ResetUser(account, password)
}

func (a routeTestAggregate) AssignRole(account, role string, domain ...string) error {
	return a.authHTTPFakeRBAC.AssignRole(account, role, domain...)
}

func (a routeTestAggregate) RevokeRole(account, role string, domain ...string) error {
	return a.authHTTPFakeRBAC.RevokeRole(account, role, domain...)
}

func (a routeTestAggregate) Enforce(subject, domain, object, act string) (bool, error) {
	return a.authHTTPFakeRBAC.Enforce(subject, domain, object, act)
}

func (a routeTestAggregate) AutoMigrate() error {
	return a.authHTTPFakeRBAC.AutoMigrate()
}

func (a routeTestAggregate) BootstrapAdmin(account, nickname, password string) error {
	return a.authHTTPFakeRBAC.BootstrapAdmin(account, nickname, password)
}

func TestChiPatternToAuthPattern(t *testing.T) {
	got := chiPatternToAuthPattern("/api/apps/{appId}/envs/{env}/elements/{key}")

	require.Equal(t, "/api/apps/:appId/envs/:env/elements/:key", got)
}

func TestMountAPIRouteSkipsAuthForUnmappedRoute(t *testing.T) {
	rbac := newAuthHTTPFakeRBAC(false)
	d := app{
		aggregate: routeTestAggregate{authHTTPFakeRBAC: rbac},
		conf:      &conf.CassemAdminConfig{Auth: &conf.AdminAuth{SessionSecret: "secret"}},
	}
	r := newAdminRouter(d, "secret")
	d.mountAPIRoute(r, http.MethodGet, "/api/debug/free", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/debug/free", nil))

	require.Equal(t, http.StatusAccepted, w.Code)
	require.Empty(t, rbac.enforceCalls)
}

func (a routeTestAggregate) GetApps(ctx context.Context, seek string, limit int, query string) (*concept.GetAppsResult, error) {
	return &concept.GetAppsResult{}, nil
}

func TestMountAPIRouteRequiresSessionForMappedRoute(t *testing.T) {
	rbac := newAuthHTTPFakeRBAC(true)
	d := app{
		aggregate: routeTestAggregate{authHTTPFakeRBAC: rbac},
		conf:      &conf.CassemAdminConfig{Auth: &conf.AdminAuth{SessionSecret: "secret"}},
	}
	r := newAdminRouter(d, "secret")
	d.mountAPIRoute(r, http.MethodGet, "/api/apps/{appId}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/apps/app1", nil))

	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Empty(t, rbac.enforceCalls)
}

func TestMountAdminRoutesRegistersProtectedAppsRoute(t *testing.T) {
	rbac := newAuthHTTPFakeRBAC(true)
	d := app{
		aggregate: routeTestAggregate{authHTTPFakeRBAC: rbac},
		conf:      &conf.CassemAdminConfig{Auth: &conf.AdminAuth{SessionSecret: "secret"}},
	}
	r := newAdminRouter(d, "secret")
	d.mountAdminRoutes(r)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/apps", nil))

	require.Equal(t, http.StatusUnauthorized, w.Code)
}
