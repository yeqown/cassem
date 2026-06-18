package app

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCreateAppReqBindsURIAndJSONSeparately(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Params = gin.Params{{Key: "appId", Value: "payment-service"}}
	c.Request = httptest.NewRequest("POST", "/api/apps/payment-service", bytes.NewBufferString(`{"name":"Payment Service","description":"release gate app"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	uriReq := new(createAppUriReq)
	require.NoError(t, c.ShouldBindUri(uriReq))
	req := new(createAppReq)
	require.NoError(t, c.ShouldBind(req))
	require.Equal(t, "payment-service", uriReq.App)
	require.Equal(t, "Payment Service", req.Name)
	require.Equal(t, "release gate app", req.Description)
}
