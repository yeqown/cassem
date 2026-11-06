package agent

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/yeqown/cassem/internal/concept"
	"github.com/yeqown/cassem/internal/pkg/httpx"
	clientpb "github.com/yeqown/cassem/sdk/go/protogen"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func (d *app) httpHandler() http.Handler {
	r := chi.NewRouter()
	r.Use(httpx.RecoveryHTTP)
	r.Get("/api/apps/{appId}/envs/{env}/elements", d.getElementsHTTP)
	return r
}

func (d *app) getElementsHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		httpx.WriteErrorStatus(w, http.StatusBadRequest, concept.Err_INVALID_ARGUMENT)
		return
	}
	ctx := metadata.NewIncomingContext(r.Context(), metadata.MD{"authorization": r.Header.Values("Authorization")})
	result, err := d.GetElement(ctx, &clientpb.GetElementReq{
		App: chi.URLParam(r, "appId"), Env: chi.URLParam(r, "env"), Keys: query["key"],
	})
	if err != nil {
		code := agentHTTPStatus(err)
		if code == http.StatusUnauthorized {
			w.Header().Set("WWW-Authenticate", "Bearer")
		}
		httpx.WriteErrorStatus(w, code, err)
		return
	}
	httpx.WriteJSON(w, result)
}

func agentHTTPStatus(err error) int {
	code := status.Code(err)
	if e, ok := concept.FromError(err); ok {
		code = e.Code.Code()
	} else if errors.Is(err, context.DeadlineExceeded) {
		code = codes.DeadlineExceeded
	} else if errors.Is(err, context.Canceled) {
		code = codes.Canceled
	}
	switch code {
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.InvalidArgument, codes.OutOfRange, codes.FailedPrecondition:
		return http.StatusBadRequest
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists, codes.Aborted:
		return http.StatusConflict
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.Canceled:
		return http.StatusRequestTimeout
	default:
		return http.StatusInternalServerError
	}
}
