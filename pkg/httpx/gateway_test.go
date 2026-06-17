package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestNewGateway(t *testing.T) {
	httpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	grpcServer := grpc.NewServer()

	g := NewGateway(":8080", httpHandler, grpcServer)

	require.NotNil(t, g)
	assert.Equal(t, ":8080", g.Addr())
	assert.NotNil(t, g.http2Wrapper())
}

func TestGatewayServeHTTPRoutesHTTPRequests(t *testing.T) {
	tests := []struct {
		name        string
		protoMajor  int
		contentType string
	}{
		{name: "http1 grpc content type", protoMajor: 1, contentType: "application/grpc"},
		{name: "http2 json content type", protoMajor: 2, contentType: "application/json"},
		{name: "http2 empty content type", protoMajor: 2, contentType: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			g := gateway{
				http: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					called = true
					w.WriteHeader(http.StatusAccepted)
					_, err := w.Write([]byte("http"))
					require.NoError(t, err)
				}),
				grpc: grpc.NewServer(),
			}
			req := httptest.NewRequest(http.MethodGet, "/configs", nil)
			req.ProtoMajor = tt.protoMajor
			req.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()

			g.ServeHTTP(w, req)

			assert.True(t, called)
			assert.Equal(t, http.StatusAccepted, w.Code)
			assert.Equal(t, "http", w.Body.String())
		})
	}
}

func TestGatewayServerDoesNotTimeoutStreamingGRPCWrites(t *testing.T) {
	g := NewGateway(":8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), grpc.NewServer())

	srv := g.server()

	require.Equal(t, time.Duration(0), srv.WriteTimeout)
	require.Equal(t, 10*time.Second, srv.ReadTimeout)
}

func TestGatewayServeHTTPRoutesGRPCRequests(t *testing.T) {
	called := false
	g := gateway{
		http: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusTeapot)
			_, err := w.Write([]byte("http"))
			require.NoError(t, err)
		}),
		grpc: grpc.NewServer(),
	}
	req := httptest.NewRequest(http.MethodPost, "/unknown.Service/Call", nil)
	req.ProtoMajor = 2
	req.Proto = "HTTP/2.0"
	req.Header.Set("Content-Type", "application/grpc")
	w := httptest.NewRecorder()

	g.ServeHTTP(w, req)

	assert.False(t, called)
	assert.NotEqual(t, http.StatusTeapot, w.Code)
	assert.NotEqual(t, "http", w.Body.String())
}
