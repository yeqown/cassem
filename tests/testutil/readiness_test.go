package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yeqown/cassem/pkg/httpx"
)

func TestCheckCassemAdmRequiresLoginSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/account/login", r.URL.Path)
		_ = json.NewEncoder(w).Encode(httpx.CommonResponse{
			ErrCode:    httpx.OK,
			ErrMessage: "success",
			Data: map[string]any{
				"session": "session-ready",
			},
		})
	}))
	defer server.Close()

	err := CheckCassemAdm(server.URL, "superadmin@example.com", "cassem", time.Second, 10*time.Millisecond)

	require.NoError(t, err)
}

func TestCheckCassemAdmHonorsTimeoutBudget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_ = json.NewEncoder(w).Encode(httpx.CommonResponse{
			ErrCode:    httpx.OK,
			ErrMessage: "success",
			Data: map[string]any{
				"session": "session-ready",
			},
		})
	}))
	defer server.Close()

	startedAt := time.Now()
	err := CheckCassemAdm(server.URL, "superadmin@example.com", "cassem", 25*time.Millisecond, 5*time.Millisecond)
	elapsed := time.Since(startedAt)

	require.Error(t, err)
	require.Contains(t, err.Error(), "did not become ready")
	require.Less(t, elapsed, 90*time.Millisecond)
}

func TestCheckCassemAdmRejectsMissingSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(httpx.CommonResponse{
			ErrCode:    httpx.OK,
			ErrMessage: "success",
			Data:       map[string]any{},
		})
	}))
	defer server.Close()

	err := CheckCassemAdm(server.URL, "superadmin@example.com", "cassem", 500*time.Millisecond, 10*time.Millisecond)

	require.Error(t, err)
	require.Contains(t, err.Error(), "session")
}

func TestCheckCassemAdmRejectsErrorEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(httpx.CommonResponse{
			ErrCode:    httpx.FAILED,
			ErrMessage: "login failed",
		})
	}))
	defer server.Close()

	err := CheckCassemAdm(server.URL, "superadmin@example.com", "wrong", 50*time.Millisecond, 10*time.Millisecond)

	require.Error(t, err)
	require.Contains(t, err.Error(), "login failed")
}

func TestCheckCassemAgentReturnsClosedEndpointError(t *testing.T) {
	err := CheckCassemAgent("127.0.0.1:1", 50*time.Millisecond, 10*time.Millisecond)

	require.Error(t, err)
	require.Contains(t, err.Error(), "127.0.0.1:1")
}
