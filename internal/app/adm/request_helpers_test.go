package adm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"

	"github.com/yeqown/cassem/api/concept"
)

type bindRequestSample struct {
	AppId       string              `uri:"appId" form:"app" binding:"required,identifier"`
	Env         string              `uri:"env" form:"env" binding:"required,identifier"`
	Keys        []string            `form:"key" binding:"omitempty,dive,identifier"`
	Limit       int                 `form:"limit,default=100"`
	Raw         string              `json:"raw" binding:"required"`
	ContentType contentTypeParam    `json:"contentType" binding:"required,oneof=1 2 3 4"`
	Mode        concept.ContentType `form:"mode,default=2" binding:"required,oneof=1 2 3 4"`
}

func requestWithRouteParams(method, target, body string, params map[string]string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	ctx := chi.NewRouteContext()
	for key, value := range params {
		ctx.URLParams.Add(key, value)
	}
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, ctx))
}

func TestBindRequestBindsURIQueryDefaultsAndJSON(t *testing.T) {
	req := requestWithRouteParams(http.MethodPost, "/apps/app1/envs/prod/elements/db?key=alpha&key=beta", `{"raw":"hello","contentType":"PLAINTEXT"}`, map[string]string{
		"appId": "app1",
		"env":   "prod",
	})

	var got bindRequestSample
	err := bindRequest(req, &got)

	assert.NoError(t, err)
	assert.Equal(t, "app1", got.AppId)
	assert.Equal(t, "prod", got.Env)
	assert.Equal(t, []string{"alpha", "beta"}, got.Keys)
	assert.Equal(t, 100, got.Limit)
	assert.Equal(t, "hello", got.Raw)
	assert.Equal(t, concept.ContentType_PLAINTEXT, got.ContentType.concept())
	assert.Equal(t, concept.ContentType_TOML, got.Mode)
}

func TestBindRequestRejectsInvalidIdentifierFromQuery(t *testing.T) {
	req := requestWithRouteParams(http.MethodPost, "/apps/app1/envs/prod/elements/db?key=-bad", `{"raw":"hello","contentType":1}`, map[string]string{
		"appId": "app1",
		"env":   "prod",
	})

	var got bindRequestSample
	err := bindRequest(req, &got)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "identifier")
}

func TestBindRequestRejectsMissingRequiredJSONField(t *testing.T) {
	req := requestWithRouteParams(http.MethodPost, "/apps/app1/envs/prod/elements/db", `{"contentType":1}`, map[string]string{
		"appId": "app1",
		"env":   "prod",
	})

	var got bindRequestSample
	err := bindRequest(req, &got)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Raw")
}
