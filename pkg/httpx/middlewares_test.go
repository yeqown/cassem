package httpx

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
