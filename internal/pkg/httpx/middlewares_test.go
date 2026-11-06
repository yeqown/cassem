package httpx

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatHeader(t *testing.T) {
	header := http.Header{
		"Accept":        []string{"application/json", "application/grpc"},
		"Authorization": []string{"Bearer token"},
	}

	got := formatHeader(header)

	assert.Contains(t, got, "Accept:application/json;application/grpc ")
	assert.Contains(t, got, "Authorization:Bearer token ")
}

func TestRecoveryHTTP_RecoversPanic(t *testing.T) {
	handler := RecoveryHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "boom")
}

func TestLoggerHTTP_RestoresBodyForNextHandler(t *testing.T) {
	var received string
	handler := LoggerHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		received = string(body)
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader("payload-body"))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, "payload-body", received)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestStatusRecorder_DefaultStatusIsOK(t *testing.T) {
	recorder := &statusRecorder{ResponseWriter: httptest.NewRecorder()}

	assert.Equal(t, http.StatusOK, recorder.Status())
}

func TestRecoveryHTTPRedactsCredentialsFromDiagnosticRequest(t *testing.T) {
	diagnostic, err := os.CreateTemp(t.TempDir(), "panic-diagnostic")
	require.NoError(t, err)
	originalStderr := os.Stderr
	os.Stderr = diagnostic
	t.Cleanup(func() { os.Stderr = originalStderr; require.NoError(t, diagnostic.Close()) })

	req := httptest.NewRequest(http.MethodPost, "/panic?request=useful", strings.NewReader("request-payload"))
	req.Header.Set("Authorization", "Bearer private-access-token")
	req.Header.Add("Authorization", "Bearer second-private-token")
	req.Header.Set("Proxy-Authorization", "Basic private-proxy-token")
	req.Header.Set("X-CASSEM-SESSION", "private-session-token")
	// Header maps can also arrive with noncanonical names from in-process callers.
	req.Header["authorization"] = []string{"Bearer lowercase-private-token"}
	req.Header.Set("X-Request-ID", "request-123")
	originalHeaders := req.Header.Clone()
	w := httptest.NewRecorder()
	RecoveryHTTP(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") })).ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.NoError(t, diagnostic.Sync())
	output, err := os.ReadFile(diagnostic.Name())
	require.NoError(t, err)
	for _, secret := range []string{"private-access-token", "second-private-token", "private-proxy-token", "private-session-token", "lowercase-private-token"} {
		require.NotContains(t, string(output), secret)
		require.NotContains(t, w.Body.String(), secret)
	}
	require.Contains(t, string(output), "POST /panic?request=useful HTTP/1.1")
	require.Contains(t, string(output), "X-Request-Id: request-123")
	require.Contains(t, string(output), "request-payload")
	require.Equal(t, originalHeaders, req.Header)
	body, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	require.Equal(t, "request-payload", string(body))
}
