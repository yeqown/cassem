package errorx

import "google.golang.org/grpc/codes"

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

// Code of errorx.Code returns the codes.Code type in grpc.
func (c Code) Code() codes.Code {
	return codes.Code(c)
}

func (c Code) Uint32() uint32 {
	return uint32(c)
}
