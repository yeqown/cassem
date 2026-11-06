package concept

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Code uint32

const (
	Code_OK                  = Code(codes.OK)
	Code_CANCELLED           = Code(codes.Canceled)
	Code_UNKNOWN             = Code(codes.Unknown)
	Code_INVALID_ARGUMENT    = Code(codes.InvalidArgument)
	Code_DEADLINE_EXCEEDED   = Code(codes.DeadlineExceeded)
	Code_NOT_FOUND           = Code(codes.NotFound)
	Code_ALREADY_EXISTS      = Code(codes.AlreadyExists)
	Code_PERMISSION_DENIED   = Code(codes.PermissionDenied)
	Code_RESOURCE_EXHAUSTED  = Code(codes.ResourceExhausted)
	Code_FAILED_PRECONDITION = Code(codes.FailedPrecondition)
	Code_ABORTED             = Code(codes.Aborted)
	Code_OUT_OF_RANGE        = Code(codes.OutOfRange)
	Code_UNIMPLEMENTED       = Code(codes.Unimplemented)
	Code_INTERNAL            = Code(codes.Internal)
	Code_UNAVAILABLE         = Code(codes.Unavailable)
	Code_DATA_LOSS           = Code(codes.DataLoss)
	Code_UNAUTHENTICATED     = Code(codes.Unauthenticated)
)

// Code returns the matching gRPC code.
func (c Code) Code() codes.Code {
	return codes.Code(c)
}

func (c Code) Uint32() uint32 {
	return uint32(c)
}

// Error carries a stable error code and message across transports.
type Error struct {
	Code    Code
	Message string
}

func (e Error) Error() string {
	return e.Message
}

func (e Error) Is(target error) bool {
	t, ok := FromError(target)
	if !ok {
		return false
	}

	return e.Code == t.Code && e.Message == t.Message
}

func New(code Code, msg string) error {
	return &Error{Code: code, Message: msg}
}

func unwrapChain(err error) error {
	for err != nil {
		if u := errors.Unwrap(err); u != nil {
			err = u
		} else {
			break
		}
	}
	return err
}

func FromError(err error) (*Error, bool) {
	err = unwrapChain(err)
	if e, ok := err.(*Error); ok {
		return e, ok
	}

	return nil, false
}

func FromStatus(err error) error {
	if err == nil {
		return nil
	}

	s, ok := status.FromError(err)
	if !ok {
		return New(Code_UNKNOWN, err.Error())
	}

	return New(Code(s.Code()), s.Message())
}

func ToStatus(err error) error {
	if err == nil {
		return nil
	}

	x, ok := FromError(err)
	if !ok {
		return status.New(Code_UNKNOWN.Code(), err.Error()).Err()
	}

	return status.New(x.Code.Code(), x.Message).Err()
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
