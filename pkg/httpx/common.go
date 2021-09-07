package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yeqown/cassem/pkg/errorx"
)

type ErrorCode int

const (
	FAILED       ErrorCode = -1
	InvalidParam ErrorCode = -2
	OK           ErrorCode = 0
)

type CommonResponse struct {
	ErrCode    ErrorCode   `json:"errcode"`
	ErrMessage string      `json:"errmsg,omitempty"`
	Data       interface{} `json:"data,omitempty"`
}

func responseWithStatusAndError(c *gin.Context, status int, err error, abort bool) {
	if err == nil {
		c.JSON(http.StatusInternalServerError, CommonResponse{
			ErrCode:    FAILED,
			ErrMessage: "NIL ERROR, CHECK CODE PLZ",
		})

		return
	}

	var code = FAILED
	if e, ok := errorx.FromError(err); ok {
		code = ErrorCode(e.Code)
	}

	if status == 0 {
		status = http.StatusBadRequest
	}

	c.JSON(status, CommonResponse{
		ErrCode:    code,
		ErrMessage: err.Error(),
	})

	if abort {
		c.Abort()
	}
}

func ResponseErrorAndAbort(c *gin.Context, err error) {
	responseWithStatusAndError(c, http.StatusBadRequest, err, true)
}

func ResponseError(c *gin.Context, err error) {
	responseWithStatusAndError(c, http.StatusBadRequest, err, false)
}

func ResponseErrorStatusAndAbort(c *gin.Context, status int, err error) {
	responseWithStatusAndError(c, status, err, true)
}

func ResponseJSON(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, CommonResponse{
		ErrCode:    OK,
		ErrMessage: "success",
		Data:       data,
	})
}
