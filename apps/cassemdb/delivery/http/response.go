package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type _errorCode int

const (
	FAILED       _errorCode = -1
	InvalidParam _errorCode = -2
	OK           _errorCode = 0
)

type commonResponse struct {
	ErrCode    _errorCode  `json:"errcode"`
	ErrMessage string      `json:"errmsg,omitempty"`
	Data       interface{} `json:"data,omitempty"`
}

func responseError(c *gin.Context, err error) {
	if err == nil {
		c.JSON(http.StatusInternalServerError, commonResponse{
			ErrCode:    FAILED,
			ErrMessage: "NIL ERROR, CHECK CODE PLZ",
		})

		return
	}

	c.JSON(http.StatusBadRequest, commonResponse{
		ErrCode:    FAILED,
		ErrMessage: err.Error(),
	})
}

func responseJSON(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, commonResponse{
		ErrCode:    OK,
		ErrMessage: "success",
		Data:       data,
	})
}
