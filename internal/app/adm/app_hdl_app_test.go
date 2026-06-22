package adm

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

type appHandlerAggregate struct {
	concept.AdmAggregate

	created *concept.AppMetadata
}

func (f *appHandlerAggregate) CreateApp(_ context.Context, md *concept.AppMetadata) error {
	f.created = md
	return nil
}

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

func TestCreateAppUsesOperatorAsCreatorAndOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	agg := &appHandlerAggregate{}
	router := gin.New()
	d := app{aggregate: agg}
	router.POST("/api/apps/:appId", func(c *gin.Context) {
		ctx := concept.WithOperator(c.Request.Context(), "alice@example.com")
		c.Request = c.Request.WithContext(ctx)
		d.CreateApp(c)
	})
	req := httptest.NewRequest(http.MethodPost, "/api/apps/payment-service", bytes.NewBufferString(`{"name":"Payment Service","description":"release gate app"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	require.NotNil(t, agg.created)
	require.Equal(t, "payment-service", agg.created.Id)
	require.Equal(t, "release gate app", agg.created.Description)
	require.Equal(t, "alice@example.com", agg.created.Creator)
	require.Equal(t, "alice@example.com", agg.created.Owner)
	require.Greater(t, agg.created.CreatedAt, int64(0))
}
