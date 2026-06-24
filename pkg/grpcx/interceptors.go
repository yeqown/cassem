package grpcx

import (
	"context"
	"fmt"
	"os"

	"buf.build/go/protovalidate"
	errorx "github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/pkg/runtime"

	"github.com/yeqown/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

// isDebug checks if debug mode is enabled via DEBUG environment variable.
func isDebug() bool {
	v := os.Getenv("DEBUG")
	return v == "1" || v == "TRUE" || v == "true"
}

// ChainUnaryServer creates a single interceptor out of a chain of many interceptors.
//
// Execution is done in left-to-right order, including passing of context.
// For example chainUnaryServer(one, two, three) will execute one before two before three, and three
// will see context changes of one and two.
func ChainUnaryServer(
	interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	n := len(interceptors)

	return func(ctx context.Context, req any,
		info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {

		chainer := func(currentInter grpc.UnaryServerInterceptor, currentHandler grpc.UnaryHandler) grpc.UnaryHandler {
			return func(currentCtx context.Context, currentReq any) (any, error) {
				return currentInter(currentCtx, currentReq, info, currentHandler)
			}
		}

		chainedHandler := handler
		for i := n - 1; i >= 0; i-- {
			chainedHandler = chainer(interceptors[i], chainedHandler)
		}

		return chainedHandler(ctx, req)
	}
}

func ServerRecovery() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (resp any, err error) {

		panicked := true
		defer func() {
			if v := recover(); v != nil || panicked {
				formatted := fmt.Sprintf("server panic: %v %v", req, v)
				log.Error(formatted)
				fmt.Println(string(runtime.Stack()))
				err = runtime.RecoverFrom(v)
			}
		}()

		resp, err = handler(ctx, req)
		panicked = false

		return
	}
}

func ServerLogger() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any,
		info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {

		fields := log.Fields{
			"method": info.FullMethod,
			"req":    req,
		}
		log.
			WithFields(fields).
			Infof("one request coming")

		if resp, err = handler(ctx, req); err != nil {
			fields["error"] = err
			log.
				WithFields(fields).
				Error("request failed")
			return
		}

		if isDebug() {
			fields["response"] = resp
		}
		log.
			WithFields(fields).
			Infof("request successful")
		return
	}
}

func SevrerErrorx() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (resp any, err error) {
		resp, err = handler(ctx, req)
		if err != nil {
			err = errorx.ToStatus(err)
		}

		return
	}
}

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

// ServerValidation checks protobuf requests from clients and aborts invalid requests before handlers run.
func ServerValidation() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (resp any, err error) {

		if err = validateRequest(req); err != nil {
			return nil, err
		}

		return handler(ctx, req)
	}
}

func ClientRecovery() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any,
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) (err error) {

		panicked := true
		defer func() {
			if v := recover(); v != nil || panicked {
				formatted := fmt.Sprintf("client panic: %v %v", req, v)
				log.Error(formatted)
				fmt.Println(string(runtime.Stack()))
				err = runtime.RecoverFrom(v)
			}
		}()

		err = invoker(ctx, method, req, reply, cc, opts...)
		panicked = false

		return
	}
}

func ClientErrorx() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any,
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

		err := invoker(ctx, method, req, reply, cc, opts...)
		if err == nil {
			return nil
		}

		// from status to errorx
		err = errorx.FromStatus(err)
		return err
	}
}

// ClientValidation validates protobuf requests before sending them.
func ClientValidation() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any,
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if err := validateRequest(req); err != nil {
			return err
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
