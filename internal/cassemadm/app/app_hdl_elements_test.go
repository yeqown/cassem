package app

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yeqown/cassem/api/concept"
)

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

	uriReq := new(commonAppEnvEltRequest)
	require.NoError(t, c.ShouldBindUri(uriReq))
	req := new(createAppEnvElementReq)
	require.NoError(t, c.ShouldBind(req))
	require.Equal(t, "payment-service", uriReq.AppId)
	require.Equal(t, "production", uriReq.Env)
	require.Equal(t, "checkout-feature-dynamic-risk-control", uriReq.ElementKey)
	require.Equal(t, `{"enabled":true}`, req.Raw)
	require.Equal(t, concept.ContentType_JSON, req.ContentType)
}

func TestDiff(t *testing.T) {
	tests := []struct {
		name    string
		base    string
		compare string
		want    string
	}{
		{name: "same text", base: "feature=true", compare: "feature=true", want: "feature=true"},
		{name: "changed text", base: "feature=false", compare: "feature=true", want: "tru"},
		{name: "added line", base: "a=1\n", compare: "a=1\nb=2\n", want: "b=2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := diff(tt.base, tt.compare)

			assert.Contains(t, got, tt.want)
		})
	}
}
