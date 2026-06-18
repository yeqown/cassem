package cassemadm

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
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
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Params = gin.Params{
		{Key: "appId", Value: "payment-service"},
		{Key: "env", Value: "production"},
		{Key: "key", Value: "checkout-feature-dynamic-risk-control"},
	}
	c.Request = httptest.NewRequest("POST", "/api/apps/payment-service/envs/production/elements/checkout-feature-dynamic-risk-control", bytes.NewBufferString(`{"raw":"{\"enabled\":true}","contentType":1}`))
	c.Request.Header.Set("Content-Type", "application/json")

	req := new(createAppEnvElementReq)
	require.NoError(t, bindURIParams(c, req))
	require.NoError(t, c.ShouldBind(req))
	require.Equal(t, "payment-service", req.AppId)
	require.Equal(t, "production", req.Env)
	require.Equal(t, "checkout-feature-dynamic-risk-control", req.ElementKey)
	require.Equal(t, `{"enabled":true}`, req.Raw)
	require.Equal(t, concept.ContentType_JSON, req.ContentType.concept())
}

func TestCreateAppEnvElementBindsJSONBodyAfterURI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	agg := &elementHandlerAggregate{}
	router := gin.New()
	d := app{aggregate: agg}
	router.POST("/api/apps/:appId/envs/:env/elements/:key", d.CreateAppEnvElement)
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

func TestUpdateAppEnvElementBindsJSONBodyAfterURI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	agg := &elementHandlerAggregate{}
	router := gin.New()
	d := app{aggregate: agg}
	router.PUT("/api/apps/:appId/envs/:env/elements/:key", d.UpdateAppEnvElement)
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
