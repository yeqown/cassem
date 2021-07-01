package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
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

func ResponseError(c *gin.Context, err error) {
	if err == nil {
		c.JSON(http.StatusInternalServerError, CommonResponse{
			ErrCode:    FAILED,
			ErrMessage: "NIL ERROR, CHECK CODE PLZ",
		})

		return
	}

	c.JSON(http.StatusBadRequest, CommonResponse{
		ErrCode:    FAILED,
		ErrMessage: err.Error(),
	})
}

func ResponseJSON(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, CommonResponse{
		ErrCode:    OK,
		ErrMessage: "success",
		Data:       data,
	})
}
