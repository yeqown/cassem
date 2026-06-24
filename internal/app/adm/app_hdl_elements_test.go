package adm

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/yeqown/cassem/api/concept"
)

type elementHandlerAggregate struct {
	concept.AdmAggregate

	created elementHandlerCreateCall
	updated elementHandlerUpdateCall
}

type elementHandlerCreateCall struct {
	app         string
	env         string
	key         string
	raw         []byte
	contentType concept.ContentType
}

type elementHandlerUpdateCall struct {
	app string
	env string
	key string
	raw []byte
}

func (f *elementHandlerAggregate) CreateElement(_ context.Context, app, env, key string, raw []byte, contentType concept.ContentType) error {
	f.created = elementHandlerCreateCall{app: app, env: env, key: key, raw: raw, contentType: contentType}
	return nil
}

func (f *elementHandlerAggregate) UpdateElement(_ context.Context, app, env, key string, raw []byte) error {
	f.updated = elementHandlerUpdateCall{app: app, env: env, key: key, raw: raw}
	return nil
}

func TestCreateElementReqBindsURIAndJSONSeparately(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/apps/payment-service/envs/production/elements/checkout-feature-dynamic-risk-control", bytes.NewBufferString(`{"raw":"{\"enabled\":true}","contentType":1}`))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("appId", "payment-service")
	rctx.URLParams.Add("env", "production")
	rctx.URLParams.Add("key", "checkout-feature-dynamic-risk-control")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	out := new(createAppEnvElementReq)
	require.NoError(t, bindRequest(req, out))
	require.Equal(t, "payment-service", out.AppId)
	require.Equal(t, "production", out.Env)
	require.Equal(t, "checkout-feature-dynamic-risk-control", out.ElementKey)
	require.Equal(t, `{"enabled":true}`, out.Raw)
	require.Equal(t, concept.ContentType_JSON, out.ContentType.concept())
}

func TestCreateAppEnvElementHTTPBindsJSONBodyAfterURI(t *testing.T) {
	agg := &elementHandlerAggregate{}
	router := chi.NewRouter()
	d := app{aggregate: agg}
	router.Post("/api/apps/{appId}/envs/{env}/elements/{key}", d.CreateAppEnvElementHTTP)
	req := httptest.NewRequest(http.MethodPost, "/api/apps/app1/envs/prod/elements/db_url", bytes.NewBufferString(`{"raw":"copy-value","contentType":"PLAINTEXT"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	require.Equal(t, "app1", agg.created.app)
	require.Equal(t, "prod", agg.created.env)
	require.Equal(t, "db_url", agg.created.key)
	require.Equal(t, []byte("copy-value"), agg.created.raw)
	require.Equal(t, concept.ContentType_PLAINTEXT, agg.created.contentType)
}

func TestUpdateAppEnvElementHTTPBindsJSONBodyAfterURI(t *testing.T) {
	agg := &elementHandlerAggregate{}
	router := chi.NewRouter()
	d := app{aggregate: agg}
	router.Put("/api/apps/{appId}/envs/{env}/elements/{key}", d.UpdateAppEnvElementHTTP)
	req := httptest.NewRequest(http.MethodPut, "/api/apps/app1/envs/prod/elements/db_url", bytes.NewBufferString(`{"raw":"next-value"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	require.Equal(t, "app1", agg.updated.app)
	require.Equal(t, "prod", agg.updated.env)
	require.Equal(t, "db_url", agg.updated.key)
	require.Equal(t, []byte("next-value"), agg.updated.raw)
}
