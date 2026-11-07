package agent

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

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
	keys := query["key"]
	if len(keys) != 1 {
		httpx.WriteErrorStatus(w, http.StatusBadRequest, concept.New(concept.Code_INVALID_ARGUMENT,
			"HTTP fetch accepts exactly one key; request each key separately"))
		return
	}
	ctx := metadata.NewIncomingContext(r.Context(), metadata.MD{"authorization": r.Header.Values("Authorization")})
	result, err := d.GetElement(ctx, &clientpb.GetElementReq{
		App: chi.URLParam(r, "appId"), Env: chi.URLParam(r, "env"), Keys: keys,
	})
	if err != nil {
		code := agentHTTPStatus(err)
		if code == http.StatusUnauthorized {
			w.Header().Set("WWW-Authenticate", "Bearer")
		}
		httpx.WriteErrorStatus(w, code, err)
		return
	}
	elems := result.GetElems()
	if len(elems) > 1 {
		httpx.WriteErrorStatus(w, http.StatusInternalServerError, fmt.Errorf("HTTP fetch returned multiple elements for one key: %w", concept.Err_INTERNAL))
		return
	}
	if len(elems) == 0 || !elems[0].GetPublished() {
		httpx.WriteErrorStatus(w, http.StatusNotFound, concept.New(concept.Code_NOT_FOUND,
			"configuration key not found or unpublished"))
		return
	}
	elem := elems[0]
	w.Header().Set("Content-Type", elementHTTPContentType(elem))
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("X-CASSEM-Key", elem.GetMetadata().GetKey())
	w.Header().Set("X-CASSEM-Version", strconv.FormatInt(int64(elem.GetVersion()), 10))
	w.Header().Set("X-CASSEM-Published", strconv.FormatBool(elem.GetPublished()))
	_, _ = w.Write(elem.GetRaw())
}

func elementHTTPContentType(elem *concept.Element) string {
	switch elem.GetMetadata().GetContentType() {
	case concept.ContentType_JSON:
		return "application/json"
	case concept.ContentType_TOML:
		return "application/toml"
	case concept.ContentType_INI, concept.ContentType_PLAINTEXT:
		return "text/plain; charset=utf-8"
	default:
		return "application/octet-stream"
	}
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
