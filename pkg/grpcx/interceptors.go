package grpcx

import (
	"context"
	"fmt"

	"github.com/yeqown/cassem/pkg/errorx"
	"github.com/yeqown/cassem/pkg/runtime"

	"github.com/yeqown/log"
	"google.golang.org/grpc"
)

// ChainUnaryServer creates a single interceptor out of a chain of many interceptors.
//
// Execution is done in left-to-right order, including passing of context.
// For example chainUnaryServer(one, two, three) will execute one before two before three, and three
// will see context changes of one and two.
func ChainUnaryServer(
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

func ServerRecovery() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (resp interface{}, err error) {

		panicked := true
		defer func() {
			if v := recover(); v != nil || panicked {
				formatted := fmt.Sprintf("server panic: %v %v", req, v)
				log.Errorf(formatted)
				fmt.Println(runtime.Stack())
				err = runtime.RecoverFrom(v)
			}
		}()

		resp, err = handler(ctx, req)
		panicked = false

		return
	}
}

func ServerLogger() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{},
		info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {

		fields := log.Fields{
			"method": info.FullMethod,
			"req":    req,
		}
		log.
			WithFields(fields).
			Infof("one request coming")

		resp, err = handler(ctx, req)

		if err != nil {
			fields["error"] = err
			log.
				WithFields(fields).
				Error("request failed")
			return
		}

		log.
			WithFields(fields).
			Infof("request successful")
		return
	}
}

func SevrerErrorx() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (resp interface{}, err error) {
		resp, err = handler(ctx, req)
		if err != nil {
			err = errorx.ToStatus(err)
		}

		return
	}
}
