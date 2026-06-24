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

type appHandlerAggregate struct {
	concept.AdmAggregate

	created *concept.AppMetadata
}

func (f *appHandlerAggregate) CreateApp(_ context.Context, md *concept.AppMetadata) error {
	f.created = md
	return nil
}

func TestCreateAppReqBindsURIAndJSONSeparately(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/apps/payment-service", bytes.NewBufferString(`{"name":"Payment Service","description":"release gate app"}`))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("appId", "payment-service")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	out := struct {
		createAppUriReq
		createAppReq
	}{}
	require.NoError(t, bindRequest(req, &out))
	require.Equal(t, "payment-service", out.App)
	require.Equal(t, "Payment Service", out.Name)
	require.Equal(t, "release gate app", out.Description)
}

func TestCreateAppHTTPUsesOperatorAsCreatorAndOwner(t *testing.T) {
	agg := &appHandlerAggregate{}
	router := chi.NewRouter()
	d := app{aggregate: agg}
	router.Post("/api/apps/{appId}", func(w http.ResponseWriter, r *http.Request) {
		ctx := concept.WithOperator(r.Context(), "alice@example.com")
		d.CreateAppHTTP(w, r.WithContext(ctx))
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
