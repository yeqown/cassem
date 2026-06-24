package httpx

import (
	"encoding/json"
	"net/http"

	errorx "github.com/yeqown/cassem/api/concept"
)

type ErrorCode int

const (
	FAILED       ErrorCode = -1
	InvalidParam ErrorCode = -2
	OK           ErrorCode = 0
)

type CommonResponse struct {
	ErrCode    ErrorCode `json:"errcode"`
	ErrMessage string    `json:"errmsg,omitempty"`
	Data       any       `json:"data,omitempty"`
}

func writeCommonJSON(w http.ResponseWriter, status int, response CommonResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func responseCodeFromError(err error) ErrorCode {
	if err == nil {
		return FAILED
	}
	if e, ok := errorx.FromError(err); ok {
		return ErrorCode(e.Code)
	}
	return FAILED
}

func WriteJSON(w http.ResponseWriter, data any) {
	writeCommonJSON(w, http.StatusOK, CommonResponse{
		ErrCode:    OK,
		ErrMessage: "success",
		Data:       data,
	})
}

func WriteError(w http.ResponseWriter, err error) {
	WriteErrorStatus(w, http.StatusBadRequest, err)
}

func WriteErrorStatus(w http.ResponseWriter, status int, err error) {
	if err == nil {
		writeCommonJSON(w, http.StatusInternalServerError, CommonResponse{
			ErrCode:    FAILED,
			ErrMessage: "NIL ERROR, CHECK CODE PLZ",
		})
		return
	}
	if status == 0 {
		status = http.StatusBadRequest
	}
	writeCommonJSON(w, status, CommonResponse{
		ErrCode:    responseCodeFromError(err),
		ErrMessage: err.Error(),
	})
}
