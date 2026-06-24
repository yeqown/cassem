package internal

import (
	"context"
	"fmt"
	"runtime"

	"buf.build/go/protovalidate"
	"github.com/yeqown/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	errorx "github.com/yeqown/cassem/api/concept"
)

func validateRequest(req any) error {
	msg, ok := req.(proto.Message)
	if !ok {
		return nil
	}

	if err := protovalidate.Validate(msg); err != nil {
		return errorx.New(errorx.Code_INVALID_ARGUMENT, formatValidationError(err))
	}
	return nil
}

func formatValidationError(err error) string {
	verr, ok := err.(*protovalidate.ValidationError)
	if !ok || len(verr.Violations) == 0 {
		return err.Error()
	}

	violation := verr.Violations[0]
	if violation == nil || !violation.FieldValue.IsValid() {
		return err.Error()
	}

	return fmt.Sprintf("%s (value=%v)", err.Error(), violation.FieldValue.Interface())
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
		if err := validateRequest(req); err != nil {
			return err
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
