package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	errorx "github.com/yeqown/cassem/api/concept"
)

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name     string
		data     any
		expected CommonResponse
	}{
		{
			name:     "response with data",
			data:     map[string]string{"key": "value"},
			expected: CommonResponse{ErrCode: OK, ErrMessage: "success", Data: map[string]string{"key": "value"}},
		},
		{
			name:     "response with nil data",
			data:     nil,
			expected: CommonResponse{ErrCode: OK, ErrMessage: "success"},
		},
		{
			name:     "response with string data",
			data:     "test string",
			expected: CommonResponse{ErrCode: OK, ErrMessage: "success", Data: "test string"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			WriteJSON(w, tt.data)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
			var response CommonResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected.ErrCode, response.ErrCode)
			assert.Equal(t, tt.expected.ErrMessage, response.ErrMessage)
			if tt.expected.Data != nil {
				assert.NotNil(t, response.Data)
			}
		})
	}
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		expectCode ErrorCode
	}{
		{name: "simple error", err: assert.AnError, expectCode: FAILED},
		{name: "errorx error - already exists", err: errorx.Err_ALREADY_EXISTS, expectCode: ErrorCode(errorx.Code_ALREADY_EXISTS)},
		{name: "errorx error - not found", err: errorx.Err_NOT_FOUND, expectCode: ErrorCode(errorx.Code_NOT_FOUND)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			WriteError(w, tt.err)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
			var response CommonResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectCode, response.ErrCode)
			assert.NotEmpty(t, response.ErrMessage)
		})
	}
}

func TestWriteErrorStatus(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		err        error
		expectCode ErrorCode
	}{
		{name: "internal server error", status: http.StatusInternalServerError, err: assert.AnError, expectCode: FAILED},
		{name: "not found", status: http.StatusNotFound, err: errorx.Err_NOT_FOUND, expectCode: ErrorCode(errorx.Code_NOT_FOUND)},
		{name: "zero status defaults to bad request", status: 0, err: assert.AnError, expectCode: FAILED},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			WriteErrorStatus(w, tt.status, tt.err)

			expectedStatus := tt.status
			if expectedStatus == 0 {
				expectedStatus = http.StatusBadRequest
			}
			assert.Equal(t, expectedStatus, w.Code)
			assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
			var response CommonResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectCode, response.ErrCode)
			assert.NotEmpty(t, response.ErrMessage)
		})
	}
}

func TestWriteErrorStatus_NilError(t *testing.T) {
	w := httptest.NewRecorder()

	WriteErrorStatus(w, http.StatusBadRequest, nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	var response CommonResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, FAILED, response.ErrCode)
	assert.Contains(t, response.ErrMessage, "NIL ERROR")
}
