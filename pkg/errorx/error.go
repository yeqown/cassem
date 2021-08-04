package errorx

import (
	"strings"

	"github.com/pkg/errors"
)

type errorx struct {
	Code    Code
	Message string
}

func (e errorx) Error() string {
	return e.Message
}

//func (e errorx) Unwrap() error {
//	return nil
//}

func (e errorx) Is(target error) bool {
	t, ok := FromError(target)
	if !ok {
		return false
	}

	return e.Code == t.Code && strings.Compare(e.Message, t.Message) == 0
}

func New(code Code, msg string) error {
	return &errorx{Code: code, Message: msg}
}

func FromError(err error) (*errorx, bool) {
	err = errors.Cause(err)
	if e, ok := err.(*errorx); ok {
		return e, ok
	}

	return nil, false
}

var (
	Err_CANCELLED           = New(Code_CANCELLED, "CANCELLED")
	Err_UNKNOWN             = New(Code_UNKNOWN, "UNKNOWN")
	Err_INVALID_ARGUMENT    = New(Code_INVALID_ARGUMENT, "INVALID_ARGUMENT")
	Err_DEADLINE_EXCEEDED   = New(Code_DEADLINE_EXCEEDED, "DEADLINE_EXCEEDED")
	Err_NOT_FOUND           = New(Code_NOT_FOUND, "NOT_FOUND")
	Err_ALREADY_EXISTS      = New(Code_ALREADY_EXISTS, "ALREADY_EXISTS")
	Err_PERMISSION_DENIED   = New(Code_PERMISSION_DENIED, "PERMISSION_DENIED")
	Err_RESOURCE_EXHAUSTED  = New(Code_RESOURCE_EXHAUSTED, "RESOURCE_EXHAUSTED")
	Err_FAILED_PRECONDITION = New(Code_FAILED_PRECONDITION, "FAILED_PRECONDITION")
	Err_ABORTED             = New(Code_ABORTED, "ABORTED")
	Err_OUT_OF_RANGE        = New(Code_OUT_OF_RANGE, "OUT_OF_RANGE")
	Err_UNIMPLEMENTED       = New(Code_UNIMPLEMENTED, "UNIMPLEMENTED")
	Err_INTERNAL            = New(Code_INTERNAL, "INTERNAL")
	Err_UNAVAILABLE         = New(Code_UNAVAILABLE, "UNAVAILABLE")
	Err_DATA_LOSS           = New(Code_DATA_LOSS, "DATA_LOSS")
	Err_UNAUTHENTICATED     = New(Code_UNAUTHENTICATED, "UNAUTHENTICATED")
)
