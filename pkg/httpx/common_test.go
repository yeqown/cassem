package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/yeqown/cassem/pkg/errorx"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestResponseJSON(t *testing.T) {
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
			c, _ := gin.CreateTestContext(w)

			ResponseJSON(c, tt.data)

			assert.Equal(t, http.StatusOK, w.Code)
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

func TestResponseError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expectAbort bool
		expectCode  ErrorCode
	}{
		{
			name:        "simple error",
			err:         assert.AnError,
			expectAbort: false,
			expectCode:  FAILED,
		},
		{
			name:        "errorx error - already exists",
			err:         errorx.Err_ALREADY_EXISTS,
			expectAbort: false,
			expectCode:  ErrorCode(errorx.Code_ALREADY_EXISTS),
		},
		{
			name:        "errorx error - not found",
			err:         errorx.Err_NOT_FOUND,
			expectAbort: false,
			expectCode:  ErrorCode(errorx.Code_NOT_FOUND),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			ResponseError(c, tt.err)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			var response CommonResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectCode, response.ErrCode)
			assert.NotEmpty(t, response.ErrMessage)
		})
	}
}

func TestResponseErrorAndAbort(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expectAbort bool
	}{
		{
			name:        "error and abort",
			err:         assert.AnError,
			expectAbort: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			ResponseErrorAndAbort(c, tt.err)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.True(t, c.IsAborted(), "Context should be aborted")
		})
	}
}

func TestResponseErrorStatusAndAbort(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		err         error
		expectAbort bool
	}{
		{
			name:        "internal server error",
			status:      http.StatusInternalServerError,
			err:         assert.AnError,
			expectAbort: true,
		},
		{
			name:        "not found",
			status:      http.StatusNotFound,
			err:         errorx.Err_NOT_FOUND,
			expectAbort: true,
		},
		{
			name:        "unauthorized",
			status:      http.StatusUnauthorized,
			err:         assert.AnError,
			expectAbort: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			ResponseErrorStatusAndAbort(c, tt.status, tt.err)

			assert.Equal(t, tt.status, w.Code)
			assert.True(t, c.IsAborted(), "Context should be aborted")
		})
	}
}

func TestResponseWithStatusAndError_NilError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	responseWithStatusAndError(c, http.StatusBadRequest, nil, true)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var response CommonResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, FAILED, response.ErrCode)
	assert.Contains(t, response.ErrMessage, "NIL ERROR")
}
