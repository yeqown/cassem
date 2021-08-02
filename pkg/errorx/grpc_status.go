package errorx

import (
	"google.golang.org/grpc/status"
)

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
