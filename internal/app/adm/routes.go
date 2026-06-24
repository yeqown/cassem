package adm

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func newAdminRouter(d app, sessionSecret string) chi.Router {
	r := chi.NewRouter()
	_ = d
	_ = sessionSecret
	return r
}

func chiPatternToAuthPattern(pattern string) string {
	parts := strings.Split(pattern, "/")
	for idx, part := range parts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			parts[idx] = ":" + strings.TrimSuffix(strings.TrimPrefix(part, "{"), "}")
		}
	}

	return strings.Join(parts, "/")
}

func (d app) mountAPIRoute(r chi.Router, method, pattern string, h http.HandlerFunc) {
	authPattern := chiPatternToAuthPattern(pattern)
	r.Method(method, pattern, withRouteAuth(d.aggregate, d.conf.Auth.SessionSecret, method, authPattern, h))
}

func (d app) mountAdminRoutes(r chi.Router) {
	r.Post("/api/account/login", d.UserLoginHTTP)

	d.mountAPIRoute(r, http.MethodGet, "/api/account/users", d.GetUsersHTTP)
	d.mountAPIRoute(r, http.MethodGet, "/api/account/users/{account}/acl", d.GetUserACLHTTP)
	d.mountAPIRoute(r, http.MethodPost, "/api/account/add", d.AddUserHTTP)
	d.mountAPIRoute(r, http.MethodGet, "/api/account/disable", d.DisableUserHTTP)
	d.mountAPIRoute(r, http.MethodGet, "/api/account/reset", d.ResetUserHTTP)
	d.mountAPIRoute(r, http.MethodPost, "/api/account/reset", d.ResetUserHTTP)
	d.mountAPIRoute(r, http.MethodGet, "/api/account/acl/domains", d.GetACLDomainOptionsHTTP)
	d.mountAPIRoute(r, http.MethodGet, "/api/account/acl/assign", d.AssignRoleHTTP)
	d.mountAPIRoute(r, http.MethodGet, "/api/account/acl/revoke", d.RevokeRoleHTTP)

	d.mountAPIRoute(r, http.MethodGet, "/api/admin/retention", d.GetRetentionPolicyHTTP)

	d.mountAPIRoute(r, http.MethodGet, "/api/apps", d.GetAppsHTTP)
	d.mountAPIRoute(r, http.MethodGet, "/api/apps/{appId}", d.GetAppHTTP)
	d.mountAPIRoute(r, http.MethodPost, "/api/apps/{appId}", d.CreateAppHTTP)
	d.mountAPIRoute(r, http.MethodDelete, "/api/apps/{appId}", d.DeleteAppHTTP)

	d.mountAPIRoute(r, http.MethodGet, "/api/apps/{appId}/envs", d.GetAppEnvironmentsHTTP)
	d.mountAPIRoute(r, http.MethodPost, "/api/apps/{appId}/envs/{env}", d.CreateAppEnvironmentHTTP)
	d.mountAPIRoute(r, http.MethodDelete, "/api/apps/{appId}/envs/{env}", d.DeleteAppEnvironmentHTTP)

	d.mountAPIRoute(r, http.MethodGet, "/api/apps/{appId}/envs/{env}/elements", d.GetAppEnvElementsHTTP)
	d.mountAPIRoute(r, http.MethodGet, "/api/apps/{appId}/envs/{env}/elements/{key}", d.GetAppEnvElementHTTP)
	d.mountAPIRoute(r, http.MethodPost, "/api/apps/{appId}/envs/{env}/elements/{key}", d.CreateAppEnvElementHTTP)
	d.mountAPIRoute(r, http.MethodPut, "/api/apps/{appId}/envs/{env}/elements/{key}", d.UpdateAppEnvElementHTTP)
	d.mountAPIRoute(r, http.MethodDelete, "/api/apps/{appId}/envs/{env}/elements/{key}", d.DeleteAppEnvElementHTTP)
	d.mountAPIRoute(r, http.MethodGet, "/api/apps/{appId}/envs/{env}/elements/{key}/versions", d.GetAppEnvElementAllVersionsHTTP)
	d.mountAPIRoute(r, http.MethodPost, "/api/apps/{appId}/envs/{env}/elements/{key}/rollback", d.RollbackAppEnvElementHTTP)
	d.mountAPIRoute(r, http.MethodPost, "/api/apps/{appId}/envs/{env}/elements/{key}/publish", d.PublishAppEnvElementHTTP)
	d.mountAPIRoute(r, http.MethodGet, "/api/apps/{appId}/envs/{env}/elements/{key}/operations", d.GetAppEnvElementOperationsHTTP)

	d.mountAPIRoute(r, http.MethodGet, "/api/cluster/topology", d.GetClusterTopologyHTTP)
	d.mountAPIRoute(r, http.MethodGet, "/api/cluster/agents", d.GetAgentsHTTP)
	d.mountAPIRoute(r, http.MethodGet, "/api/cluster/instances", d.GetInstancesHTTP)
	d.mountAPIRoute(r, http.MethodGet, "/api/cluster/instances/detail/{insId}", d.GetInstanceHTTP)
	d.mountAPIRoute(r, http.MethodGet, "/api/cluster/instances/filter", d.GetInstancesByElementHTTP)
}
