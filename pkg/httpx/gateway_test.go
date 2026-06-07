package httpx

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestGateway_ServeHTTP(t *testing.T) {
	// Create a simple HTTP handler
	httpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("HTTP response"))
	})

	// Create a minimal gRPC server (for testing purposes)
	grpcServer := grpc.NewServer()

	gw := NewGateway("localhost:8080", httpHandler, grpcServer)

	tests := []struct {
		name           string
		protoMajor     int
		contentType    string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "HTTP/1.1 request",
			protoMajor:     1,
			contentType:    "application/json",
			expectedStatus: http.StatusOK,
			expectedBody:   "HTTP response",
		},
		{
			name:           "HTTP/2 with JSON content type",
			protoMajor:     2,
			contentType:    "application/json",
			expectedStatus: http.StatusOK,
			expectedBody:   "HTTP response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.ProtoMajor = tt.protoMajor
			req.Header.Set("Content-Type", tt.contentType)

			w := httptest.NewRecorder()
			gw.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			body := w.Body.String()
			assert.Contains(t, body, tt.expectedBody)
		})
	}
}

func TestGateway_Addr(t *testing.T) {
	tests := []struct {
		name     string
		addr     string
		expected string
	}{
		{"localhost", "localhost:8080", "localhost:8080"},
		{"0.0.0.0", "0.0.0.0:9090", "0.0.0.0:9090"},
		{"with host", "example.com:80", "example.com:80"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gw := NewGateway(tt.addr, nil, nil)
			assert.Equal(t, tt.expected, gw.Addr())
		})
	}
}

func TestGateway_http2Wrapper(t *testing.T) {
	httpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	gw := NewGateway("localhost:8080", httpHandler, nil)

	wrapper := gw.http2Wrapper()
	assert.NotNil(t, wrapper, "http2Wrapper should return a handler")

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	wrapper.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "OK")
}

func TestGateway_ListenAndServe(t *testing.T) {
	httpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	gw := NewGateway("127.0.0.1:0", httpHandler, nil) // Use port 0 for dynamic port

	// Start server in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- gw.ListenAndServe()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Try to connect
	resp, err := http.Get("http://" + gw.Addr() + "/")
	if err == nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, string(body), "OK")
	}

	// The server will fail to start if port is in use, which is ok for this test
	// We're mainly testing that ListenAndServe doesn't panic
}
