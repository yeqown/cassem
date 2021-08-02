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

//func Wrapf(err error, message string, args ...interface{}) error {
//	return errors.Wrapf(err, message, args...)
//}
//
//func Unwrap(err error) error {
//	return stderror.Unwrap(err)
//}
//
//func Is(err, err2 error) bool {
//	return errors.Is(err, err2)
//}
