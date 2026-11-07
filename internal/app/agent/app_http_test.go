package agent

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yeqown/cassem/internal/concept"
	"github.com/yeqown/cassem/internal/pkg/httpx"
	clientpb "github.com/yeqown/cassem/sdk/go/protogen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestHTTPFetchUsesPublishedBackendAndCache(t *testing.T) {
	elem := &concept.Element{Metadata: &concept.ElementMetadata{App: "demo", Env: "prod", Key: "config", ContentType: concept.ContentType_JSON}, Raw: []byte(`{"enabled":true}`), Published: true, Version: 7}
	for _, source := range []string{"backend", "cache"} {
		t.Run(source, func(t *testing.T) {
			agg := &fetchAggregate{elements: []*concept.Element{elem}}
			cache := NewCache(10)
			if source == "cache" {
				cache.Set("demo", "prod", "config", elem)
				cache.Set("demo", "prod", "config", elem)
				agg.err = errors.New("backend unavailable after cache warmup")
			}
			d := app{aggregate: agg, cache: cache, authorizer: allowTokenAuthorizer()}
			req := httptest.NewRequest(http.MethodGet, "/api/apps/demo/envs/prod/elements?key=config", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			d.httpHandler().ServeHTTP(w, req)
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			require.Equal(t, elem.Raw, w.Body.Bytes())
			require.Equal(t, "application/json", w.Header().Get("Content-Type"))
			require.Equal(t, "inline", w.Header().Get("Content-Disposition"))
			require.Equal(t, "config", w.Header().Get("X-CASSEM-Key"))
			require.Equal(t, "7", w.Header().Get("X-CASSEM-Version"))
			require.Equal(t, "true", w.Header().Get("X-CASSEM-Published"))
			require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
			if source == "backend" {
				require.Equal(t, [][]string{{"config"}}, agg.requests)
			} else {
				require.Empty(t, agg.requests)
			}
		})
	}
}

func TestHTTPFetchContentTypesAndRawBytes(t *testing.T) {
	cases := []struct {
		name        string
		contentType concept.ContentType
		raw         []byte
		want        string
	}{
		{name: "json", contentType: concept.ContentType_JSON, raw: []byte("{\n  \"enabled\": true\n}\n"), want: "application/json"},
		{name: "toml", contentType: concept.ContentType_TOML, raw: []byte("enabled = true\r\n"), want: "application/toml"},
		{name: "ini", contentType: concept.ContentType_INI, raw: []byte("[app]\nenabled=true\n"), want: "text/plain; charset=utf-8"},
		{name: "plaintext", contentType: concept.ContentType_PLAINTEXT, raw: []byte("配置内容\n"), want: "text/plain; charset=utf-8"},
		{name: "unknown binary", contentType: concept.ContentType_UNKNOWN, raw: []byte{0, 0xff, '{', '}', '\n'}, want: "application/octet-stream"},
		{name: "future type", contentType: concept.ContentType(999), raw: []byte("content"), want: "application/octet-stream"},
		{name: "empty published value", contentType: concept.ContentType_PLAINTEXT, want: "text/plain; charset=utf-8"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			elem := &concept.Element{Metadata: &concept.ElementMetadata{App: "demo", Env: "prod", Key: "config", ContentType: tc.contentType}, Raw: tc.raw, Version: 1, Published: true}
			d := app{aggregate: &fetchAggregate{elements: []*concept.Element{elem}}, cache: NewCache(10), authorizer: allowTokenAuthorizer()}
			req := httptest.NewRequest(http.MethodGet, "/api/apps/demo/envs/prod/elements?key=config", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			d.httpHandler().ServeHTTP(w, req)
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			require.Equal(t, tc.want, w.Header().Get("Content-Type"))
			require.Equal(t, string(tc.raw), w.Body.String())
		})
	}
}

func TestHTTPFetchRejectsUnavailableElements(t *testing.T) {
	draft := &concept.Element{Metadata: &concept.ElementMetadata{App: "demo", Env: "prod", Key: "config"}, Raw: []byte("unpublished configuration"), Version: 1}
	published := &concept.Element{Metadata: draft.Metadata, Raw: []byte("published configuration"), Published: true, Version: 1}
	cases := []struct {
		name     string
		elements []*concept.Element
		cached   bool
		want     int
		wantCode httpx.ErrorCode
	}{
		{name: "missing", want: http.StatusNotFound, wantCode: 5},
		{name: "unpublished backend value", elements: []*concept.Element{draft}, want: http.StatusNotFound, wantCode: 5},
		{name: "unpublished cached value", elements: []*concept.Element{draft}, cached: true, want: http.StatusNotFound, wantCode: 5},
		{name: "multiple backend values", elements: []*concept.Element{published, published}, want: http.StatusInternalServerError, wantCode: 13},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			agg := &fetchAggregate{elements: tc.elements}
			cache := NewCache(10)
			if tc.cached {
				cache.Set("demo", "prod", "config", draft)
				cache.Set("demo", "prod", "config", draft)
				agg.err = errors.New("cached value must not require a backend read")
			}
			d := app{aggregate: agg, cache: cache, authorizer: allowTokenAuthorizer()}
			req := httptest.NewRequest(http.MethodGet, "/api/apps/demo/envs/prod/elements?key=config", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			d.httpHandler().ServeHTTP(w, req)
			require.Equal(t, tc.want, w.Code, w.Body.String())
			require.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
			require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
			require.Empty(t, w.Header().Get("X-CASSEM-Key"))
			require.Empty(t, w.Header().Get("X-CASSEM-Version"))
			require.Empty(t, w.Header().Get("X-CASSEM-Published"))
			require.Empty(t, w.Header().Get("Content-Disposition"))
			var body httpx.CommonResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			require.Equal(t, tc.wantCode, body.ErrCode)
			require.Nil(t, body.Data)
			require.NotContains(t, w.Body.String(), "unpublished configuration")
		})
	}
}

func TestHTTPFetchStatusAndValidation(t *testing.T) {
	cases := []struct {
		name, path          string
		headers             []string
		authErr, backendErr error
		want                int
	}{
		{name: "missing identity", path: "/api/apps/demo/envs/prod/elements?key=config", want: 401},
		{name: "duplicate identity", path: "/api/apps/demo/envs/prod/elements?key=config", headers: []string{"Bearer first", "Bearer second"}, want: 401},
		{name: "malformed identity", path: "/api/apps/demo/envs/prod/elements?key=config", headers: []string{"Basic token"}, want: 401},
		{name: "revoked identity", path: "/api/apps/demo/envs/prod/elements?key=config", headers: []string{"Bearer token"}, authErr: concept.Err_UNAUTHENTICATED, want: 401},
		{name: "denied", path: "/api/apps/demo/envs/prod/elements?key=config", headers: []string{"Bearer token"}, authErr: concept.Err_PERMISSION_DENIED, want: 403},
		{name: "missing keys", path: "/api/apps/demo/envs/prod/elements", headers: []string{"Bearer token"}, want: 400},
		{name: "empty key", path: "/api/apps/demo/envs/prod/elements?key=", headers: []string{"Bearer token"}, want: 400},
		{name: "multiple keys", path: "/api/apps/demo/envs/prod/elements?key=config&key=limits", headers: []string{"Bearer token"}, want: 400},
		{name: "duplicate keys", path: "/api/apps/demo/envs/prod/elements?key=config&key=config", headers: []string{"Bearer token"}, want: 400},
		{name: "invalid app", path: "/api/apps/dot.app/envs/prod/elements?key=config", headers: []string{"Bearer token"}, authErr: concept.Err_PERMISSION_DENIED, want: 400},
		{name: "invalid key", path: "/api/apps/demo/envs/prod/elements?key=db.url", headers: []string{"Bearer token"}, want: 400},
		{name: "malformed query", path: "/api/apps/demo/envs/prod/elements?key=config;secret", headers: []string{"Bearer token"}, want: 400},
		{name: "missing", path: "/api/apps/demo/envs/prod/elements?key=config", headers: []string{"Bearer token"}, backendErr: concept.Err_NOT_FOUND, want: 404},
		{name: "backend down", path: "/api/apps/demo/envs/prod/elements?key=config", headers: []string{"Bearer token"}, backendErr: concept.Err_UNAVAILABLE, want: 503},
		{name: "deadline", path: "/api/apps/demo/envs/prod/elements?key=config", headers: []string{"Bearer token"}, backendErr: context.DeadlineExceeded, want: 504},
		{name: "unknown backend error", path: "/api/apps/demo/envs/prod/elements?key=config", headers: []string{"Bearer token"}, backendErr: errors.New("backend failure"), want: 500},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			agg := &fetchAggregate{err: tc.backendErr}
			d := app{aggregate: agg, cache: NewCache(10), authorizer: tokenAuthorizerFunc(func(context.Context, string, ...concept.TokenTarget) error { return tc.authErr })}
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			for _, v := range tc.headers {
				req.Header.Add("Authorization", v)
			}
			w := httptest.NewRecorder()
			d.httpHandler().ServeHTTP(w, req)
			require.Equal(t, tc.want, w.Code, w.Body.String())
			require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
			if tc.want == 401 {
				require.Equal(t, "Bearer", w.Header().Get("WWW-Authenticate"))
			}
			if tc.want == 400 || tc.want == 401 || tc.want == 403 {
				require.Empty(t, agg.requests)
			}
		})
	}
}

func TestAgentGatewayServesHTTPAndGRPCOnSamePort(t *testing.T) {
	d := app{aggregate: &fetchAggregate{elements: []*concept.Element{{Metadata: &concept.ElementMetadata{App: "demo", Env: "prod", Key: "config"}, Raw: []byte("published"), Published: true, Version: 1}}}, cache: NewCache(10), authorizer: allowTokenAuthorizer()}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)
	grpcServer := d.grpcServer()
	gateway := httpx.NewGateway(listener.Addr().String(), d.httpHandler(), grpcServer)
	server := &http.Server{Handler: gateway, Protocols: protocols, ReadHeaderTimeout: time.Second}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	t.Cleanup(func() {
		require.NoError(t, server.Close())
		require.ErrorIs(t, <-done, http.ErrServerClosed)
		grpcServer.Stop()
	})
	req, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/apps/demo/envs/prod/elements?key=config", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer test-token")
	response, err := (&http.Client{Timeout: time.Second}).Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, response.StatusCode)
	raw, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.Equal(t, "published", string(raw))
	require.Equal(t, "1", response.Header.Get("X-CASSEM-Version"))
	cc, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, cc.Close()) })
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	client := clientpb.NewAgentClient(cc)
	_, err = client.GetElement(ctx, &clientpb.GetElementReq{App: "demo", Env: "prod", Keys: []string{"config"}})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer test-token"))
	got, err := client.GetElement(ctx, &clientpb.GetElementReq{App: "demo", Env: "prod", Keys: []string{"config"}})
	require.NoError(t, err)
	require.Equal(t, []byte("published"), got.GetElems()[0].GetRaw())
}

type fetchAggregate struct {
	watchPersistAggregate
	elements []*concept.Element
	err      error
	requests [][]string
}

func (a *fetchAggregate) GetElementsByKeys(_ context.Context, app, env string, keys []string) (*concept.GetElementsResult, error) {
	a.requests = append(a.requests, append([]string(nil), keys...))
	if a.err != nil {
		return nil, a.err
	}
	var elements []*concept.Element
	for _, key := range keys {
		for _, elem := range a.elements {
			if elem.GetMetadata().GetKey() == key {
				elements = append(elements, elem)
			}
		}
	}
	return &concept.GetElementsResult{Elements: elements}, nil
}
