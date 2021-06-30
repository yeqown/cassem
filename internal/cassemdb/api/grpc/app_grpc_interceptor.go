package grpc

import (
	"context"
	"errors"
	"fmt"

	"github.com/yeqown/cassem/pkg/runtime"

	"github.com/yeqown/log"
	"google.golang.org/grpc"
)

var (
	logger                 = log.WithField("caller", "notifier")
	errUninitializedLogger = errors.New("logger is not initialized, " +
		"you should call `grpcwrapper.SetLogger` at first")
)

// chainUnaryServer creates a single interceptor out of a chain of many interceptors.
//
// Execution is done in left-to-right order, including passing of context.
// For example chainUnaryServer(one, two, three) will execute one before two before three, and three
// will see context changes of one and two.
func chainUnaryServer(
	interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	n := len(interceptors)

	return func(ctx context.Context, req interface{},
		info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

		chainer := func(currentInter grpc.UnaryServerInterceptor, currentHandler grpc.UnaryHandler) grpc.UnaryHandler {
			return func(currentCtx context.Context, currentReq interface{}) (interface{}, error) {
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

func serverRecovery() grpc.UnaryServerInterceptor {
	if logger == nil {
		panic(errUninitializedLogger)
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (resp interface{}, err error) {

		panicked := true
		defer func() {
			if v := recover(); v != nil || panicked {
				formatted := fmt.Sprintf("server panic: %v %v", req, v)
				logger.Errorf(formatted)
				fmt.Println(runtime.Stack())
				err = runtime.RecoverFrom(v)
			}
		}()

		resp, err = handler(ctx, req)
		panicked = false

		return
	}
}

func serverLogger() grpc.UnaryServerInterceptor {
	if logger == nil {
		panic(errUninitializedLogger)
	}

	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (resp interface{}, err error) {

		fields := log.Fields{
			"method": info.FullMethod,
			"req":    req,
		}

		resp, err = handler(ctx, req)

		if err != nil {
			fields["error"] = err
			logger.WithFields(fields).Error("request failed")
			return
		}

		logger.WithFields(fields).Infof("request successful")
		return
	}
}
