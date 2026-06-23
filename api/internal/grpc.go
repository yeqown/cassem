package internal

import (
	"context"
	"fmt"
	"runtime"

	"github.com/yeqown/log"
	"google.golang.org/grpc"

	errorx "github.com/yeqown/cassem/api/concept"
)

type validator interface {
	Validate() error
	ValidateAll() error
}

// ClientRecovery converts client-side panics into errors.
func ClientRecovery() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any,
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) (err error) {
		panicked := true
		defer func() {
			if v := recover(); v != nil || panicked {
				buf := make([]byte, 64<<10)
				n := runtime.Stack(buf, false)
				log.Errorf("client panic: %v %s", v, buf[:n])
				err = fmt.Errorf("panic: %v", v)
			}
		}()

		err = invoker(ctx, method, req, reply, cc, opts...)
		panicked = false
		return err
	}
}

// ClientErrorx converts gRPC status errors into errorx errors.
func ClientErrorx() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any,
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		err := invoker(ctx, method, req, reply, cc, opts...)
		if err == nil {
			return nil
		}
		return errorx.FromStatus(err)
	}
}

// ClientValidation validates requests before sending them.
func ClientValidation() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any,
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if v, ok := req.(validator); ok {
			if err := v.Validate(); err != nil {
				return errorx.New(errorx.Code_INVALID_ARGUMENT, err.Error())
			}
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
