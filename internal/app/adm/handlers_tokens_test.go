package adm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/stretchr/testify/require"

	"github.com/yeqown/cassem/internal/concept"
	"github.com/yeqown/cassem/internal/pkg/conf"
	pb "github.com/yeqown/cassem/internal/protogen"
)

type httpTokenManager struct {
	issue  func(context.Context, string, string, time.Duration) (*concept.IssuedToken, error)
	list   func(context.Context, string, string, int) (*concept.GetTokensResult, error)
	revoke func(context.Context, string, string) error
}

func (m httpTokenManager) IssueToken(ctx context.Context, account, name string, expiry time.Duration) (*concept.IssuedToken, error) {
	return m.issue(ctx, account, name, expiry)
}

func (m httpTokenManager) GetTokens(ctx context.Context, account, seek string, limit int) (*concept.GetTokensResult, error) {
	return m.list(ctx, account, seek, limit)
}

func (m httpTokenManager) RevokeToken(ctx context.Context, account, id string) error {
	return m.revoke(ctx, account, id)
}

func tokenHTTPRouter(manager concept.TokenManager) chi.Router {
	d := app{
		aggregate: routeTestAggregate{authHTTPFakeRBAC: newAuthHTTPFakeRBAC(true)},
		conf:      &conf.CassemAdminConfig{Auth: &conf.AdminAuth{SessionSecret: "secret"}},
		tokens:    manager,
	}
	r := chi.NewRouter()
	d.mountAdminRoutes(r)
	return r
}

func tokenHTTPRequest(t *testing.T, r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("X-Cassem-Session", newAuthHTTPToken(t, "secret"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	return w
}

func TestIssueTokenHTTPReturnsSecretOnceAndBoundsLifetime(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		duration   time.Duration
	}{
		{"default", `{"name":"deployment"}`, 30 * 24 * time.Hour},
		{"account comes from path", `{"name":"deployment","account":"other@example.com"}`, 30 * 24 * time.Hour},
		{"minimum", `{"name":"deployment","expiresInSeconds":1}`, time.Second},
		{"maximum", `{"name":"deployment","expiresInSeconds":31536000}`, 365 * 24 * time.Hour},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := tokenHTTPRouter(httpTokenManager{issue: func(ctx context.Context, account, name string, expiry time.Duration) (*concept.IssuedToken, error) {
				require.Equal(t, "alice", concept.OperatorFromContext(ctx))
				require.Equal(t, "user@example.com", account)
				require.Equal(t, "deployment", name)
				require.Equal(t, tc.duration, expiry)
				return &concept.IssuedToken{AccessToken: concept.AccessToken{ID: strings.Repeat("a", 64), Account: account, Name: name, CreatedAt: 100, ExpiresAt: 200}, Token: "secret-once"}, nil
			}})
			w := tokenHTTPRequest(t, r, http.MethodPost, "/api/account/users/user@example.com/tokens", tc.body)
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			var out struct {
				Errcode int                 `json:"errcode"`
				Data    concept.IssuedToken `json:"data"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
			require.Zero(t, out.Errcode)
			require.Equal(t, "secret-once", out.Data.Token)
			require.Equal(t, int64(200), out.Data.ExpiresAt)
		})
	}
}

func TestListTokensHTTPReturnsMetadataAndPagination(t *testing.T) {
	for _, query := range []string{"", "?limit=1&seek=" + strings.Repeat("b", 64)} {
		r := tokenHTTPRouter(httpTokenManager{list: func(ctx context.Context, account, seek string, limit int) (*concept.GetTokensResult, error) {
			require.Equal(t, "alice", concept.OperatorFromContext(ctx))
			require.Equal(t, "user@example.com", account)
			if query == "" {
				require.Empty(t, seek)
				require.Equal(t, 100, limit)
			} else {
				require.Equal(t, strings.Repeat("b", 64), seek)
				require.Equal(t, 1, limit)
			}
			return &concept.GetTokensResult{Tokens: []*concept.AccessToken{{ID: strings.Repeat("a", 64), Account: account, Name: "deployment", CreatedAt: 100, ExpiresAt: 200}}}, nil
		}})
		w := tokenHTTPRequest(t, r, http.MethodGet, "/api/account/users/user@example.com/tokens"+query, "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var out struct {
			Data struct {
				Tokens []map[string]any `json:"tokens"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
		require.Len(t, out.Data.Tokens, 1)
		require.Equal(t, "deployment", out.Data.Tokens[0]["name"])
		require.NotContains(t, out.Data.Tokens[0], "token")
	}
}

func TestRevokeTokenHTTPScopesToPathAccount(t *testing.T) {
	r := tokenHTTPRouter(httpTokenManager{revoke: func(ctx context.Context, account, id string) error {
		require.Equal(t, "alice", concept.OperatorFromContext(ctx))
		require.Equal(t, "user@example.com", account)
		require.Equal(t, strings.Repeat("a", 64), id)
		return nil
	}})
	w := tokenHTTPRequest(t, r, http.MethodDelete, "/api/account/users/user@example.com/tokens/"+strings.Repeat("a", 64), "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.JSONEq(t, `{"errcode":0,"errmsg":"success"}`, w.Body.String())
}

func TestTokenHTTPRejectsInvalidRequests(t *testing.T) {
	const base = "/api/account/users/user@example.com/tokens"
	for _, tc := range []struct{ name, method, path, body string }{
		{"account", http.MethodPost, "/api/account/users/invalid/tokens", `{"name":"ok"}`},
		{"empty name", http.MethodPost, base, `{}`},
		{"whitespace name", http.MethodPost, base, `{"name":"   "}`},
		{"long name", http.MethodPost, base, `{"name":"` + strings.Repeat("a", 101) + `"}`},
		{"zero expiry", http.MethodPost, base, `{"name":"ok","expiresInSeconds":0}`},
		{"null expiry", http.MethodPost, base, `{"name":"ok","expiresInSeconds":null}`},
		{"negative expiry", http.MethodPost, base, `{"name":"ok","expiresInSeconds":-1}`},
		{"large expiry", http.MethodPost, base, `{"name":"ok","expiresInSeconds":31536001}`},
		{"fraction expiry", http.MethodPost, base, `{"name":"ok","expiresInSeconds":1.2}`},
		{"overflow expiry", http.MethodPost, base, `{"name":"ok","expiresInSeconds":9223372036854775808}`},
		{"malformed JSON", http.MethodPost, base, `{"name":`},
		{"zero limit", http.MethodGet, base + "?limit=0", ""},
		{"negative limit", http.MethodGet, base + "?limit=-1", ""},
		{"large limit", http.MethodGet, base + "?limit=101", ""},
		{"bad limit", http.MethodGet, base + "?limit=abc", ""},
		{"bad seek", http.MethodGet, base + "?seek=not-a-digest", ""},
		{"upper seek", http.MethodGet, base + "?seek=" + strings.Repeat("A", 64), ""},
		{"bad list account", http.MethodGet, "/api/account/users/invalid/tokens", ""},
		{"bad revoke account", http.MethodDelete, "/api/account/users/invalid/tokens/" + strings.Repeat("a", 64), ""},
		{"short id", http.MethodDelete, base + "/abc", ""},
		{"upper id", http.MethodDelete, base + "/" + strings.Repeat("A", 64), ""},
		{"nonhex id", http.MethodDelete, base + "/" + strings.Repeat("g", 64), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manager := httpTokenManager{
				issue: func(context.Context, string, string, time.Duration) (*concept.IssuedToken, error) {
					t.Fatal("invalid request reached token issuance")
					return nil, nil
				},
				list: func(context.Context, string, string, int) (*concept.GetTokensResult, error) {
					t.Fatal("invalid request reached token listing")
					return nil, nil
				},
				revoke: func(context.Context, string, string) error {
					t.Fatal("invalid request reached token revocation")
					return nil
				},
			}
			w := tokenHTTPRequest(t, tokenHTTPRouter(manager), tc.method, tc.path, tc.body)
			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			var out struct {
				Errcode concept.Code `json:"errcode"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
			require.Equal(t, concept.Code_INVALID_ARGUMENT, out.Errcode)
		})
	}
}

func TestTokenHTTPPropagatesServiceErrors(t *testing.T) {
	for _, err := range []error{concept.Err_NOT_FOUND, concept.Err_PERMISSION_DENIED, errors.New("storage unavailable")} {
		manager := httpTokenManager{
			issue:  func(context.Context, string, string, time.Duration) (*concept.IssuedToken, error) { return nil, err },
			list:   func(context.Context, string, string, int) (*concept.GetTokensResult, error) { return nil, err },
			revoke: func(context.Context, string, string) error { return err },
		}
		for _, method := range []string{http.MethodPost, http.MethodGet, http.MethodDelete} {
			path := "/api/account/users/user@example.com/tokens"
			if method == http.MethodDelete {
				path += "/" + strings.Repeat("a", 64)
			}
			w := tokenHTTPRequest(t, tokenHTTPRouter(manager), method, path, `{"name":"deployment"}`)
			require.Equal(t, http.StatusBadRequest, w.Code)
			require.Contains(t, w.Body.String(), err.Error())
			require.NotContains(t, w.Body.String(), `"data"`)
			var out struct {
				Errcode int `json:"errcode"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
			if ex, ok := concept.FromError(err); ok {
				require.Equal(t, int(ex.Code), out.Errcode)
			} else {
				require.Equal(t, -1, out.Errcode)
			}
		}
	}
}

func TestTokenRoutesRequireSessionAndClusterACL(t *testing.T) {
	for _, route := range []struct{ method, path, action string }{
		{http.MethodPost, "/api/account/users/user@example.com/tokens", concept.Action_WRITE},
		{http.MethodGet, "/api/account/users/user@example.com/tokens", concept.Action_READ},
		{http.MethodDelete, "/api/account/users/user@example.com/tokens/" + strings.Repeat("a", 64), concept.Action_WRITE},
	} {
		t.Run(route.method, func(t *testing.T) {
			for _, auth := range []struct {
				name     string
				header   string
				disabled bool
				status   int
			}{
				{name: "missing", status: http.StatusUnauthorized},
				{name: "invalid", header: "invalid-session", status: http.StatusUnauthorized},
				{name: "disabled", header: newAuthHTTPToken(t, "secret"), disabled: true, status: http.StatusUnauthorized},
				{name: "denied", header: newAuthHTTPToken(t, "secret"), status: http.StatusBadRequest},
			} {
				t.Run(auth.name, func(t *testing.T) {
					rbac := newAuthHTTPFakeRBAC(false)
					if auth.disabled {
						rbac.user.Status = pb.User_FORBIDDEN
					}
					d := app{
						aggregate: routeTestAggregate{authHTTPFakeRBAC: rbac},
						conf:      &conf.CassemAdminConfig{Auth: &conf.AdminAuth{SessionSecret: "secret"}},
					}
					r := newAdminRouter(d, "secret")
					d.mountAdminRoutes(r)
					req := httptest.NewRequest(route.method, route.path, strings.NewReader(`{"name":"deployment"}`))
					req.Header.Set("X-Cassem-Session", auth.header)
					w := httptest.NewRecorder()
					r.ServeHTTP(w, req)
					require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
					require.Equal(t, auth.status, w.Code)
					if auth.name == "denied" {
						require.Equal(t, []authHTTPEnforceCall{{subject: "alice", domain: concept.Domain_CLUSTER, object: concept.Object_ACL, act: route.action}}, rbac.enforceCalls)
					} else {
						require.Empty(t, rbac.enforceCalls)
					}
				})
			}
		})
	}
}
