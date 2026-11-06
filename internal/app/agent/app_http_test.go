package agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
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
	elem := &concept.Element{Metadata: &concept.ElementMetadata{App: "demo", Env: "prod", Key: "config"}, Raw: []byte{0, 0xff, '{', '}'}, Published: true, Version: 7}
	agg := &fetchAggregate{elements: []*concept.Element{elem}}
	d := app{aggregate: agg, cache: NewCache(10), authorizer: allowTokenAuthorizer()}
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/apps/demo/envs/prod/elements?key=config&key=limits", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()
		d.httpHandler().ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
		var body struct {
			ErrCode int `json:"errcode"`
			Data    struct {
				Elems []struct {
					Raw       string `json:"raw"`
					Version   int    `json:"version"`
					Published bool   `json:"published"`
				} `json:"elems"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		require.Zero(t, body.ErrCode)
		require.Len(t, body.Data.Elems, 1)
		require.Equal(t, base64.StdEncoding.EncodeToString(elem.Raw), body.Data.Elems[0].Raw)
		require.Equal(t, 7, body.Data.Elems[0].Version)
		require.True(t, body.Data.Elems[0].Published)
	}
	require.NotEmpty(t, agg.requests)
	require.Equal(t, []string{"config", "limits"}, agg.requests[0])
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
	require.NoError(t, response.Body.Close())
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
