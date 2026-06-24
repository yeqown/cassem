package grpcx

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/yeqown/cassem/api/agent"
	errorx "github.com/yeqown/cassem/api/concept"
)

func assertInvalidArgumentContract(t *testing.T, err error) {
	t.Helper()

	if !assert.Error(t, err) {
		return
	}

	if xerr, ok := errorx.FromError(err); ok {
		assert.Equal(t, errorx.Code_INVALID_ARGUMENT, xerr.Code)
		return
	}

	st, ok := status.FromError(err)
	if assert.True(t, ok, "expected errorx or gRPC status error") {
		assert.Equal(t, codes.InvalidArgument, st.Code())
	}
}

func TestChainUnaryServer(t *testing.T) {
	tests := []struct {
		name         string
		interceptors []grpc.UnaryServerInterceptor
		callOrder    []string
		expectError  bool
	}{
		{
			name:         "no interceptors",
			interceptors: nil,
			callOrder:    []string{},
			expectError:  false,
		},
		{
			name: "single interceptor",
			interceptors: []grpc.UnaryServerInterceptor{
				func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
					return handler(ctx, req)
				},
			},
			callOrder:   []string{},
			expectError: false,
		},
		{
			name: "multiple interceptors executed in order",
			interceptors: []grpc.UnaryServerInterceptor{
				func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
					ctx = context.WithValue(ctx, "interceptor1", "called")
					return handler(ctx, req)
				},
				func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
					ctx = context.WithValue(ctx, "interceptor2", "called")
					return handler(ctx, req)
				},
			},
			callOrder:   []string{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chained := ChainUnaryServer(tt.interceptors...)

			handlerCalled := false
			handler := func(ctx context.Context, req any) (any, error) {
				handlerCalled = true
				return "response", nil
			}

			resp, err := chained(context.Background(), "request", &grpc.UnaryServerInfo{}, handler)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "response", resp)
				assert.True(t, handlerCalled)
			}
		})
	}
}

func TestServerRecovery(t *testing.T) {
	tests := []struct {
		name        string
		handler     grpc.UnaryHandler
		expectPanic bool
		expectError bool
	}{
		{
			name: "normal handler",
			handler: func(ctx context.Context, req any) (any, error) {
				return "ok", nil
			},
			expectPanic: false,
			expectError: false,
		},
		{
			name: "handler returns error",
			handler: func(ctx context.Context, req any) (any, error) {
				return nil, errors.New("handler error")
			},
			expectPanic: false,
			expectError: true,
		},
		{
			name: "handler panics",
			handler: func(ctx context.Context, req any) (any, error) {
				panic("test panic")
			},
			expectPanic: true,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := ServerRecovery()
			resp, err := interceptor(context.Background(), "request", &grpc.UnaryServerInfo{}, tt.handler)

			if tt.expectPanic {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "panic")
			} else if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "ok", resp)
			}
		})
	}
}

func TestServerLogger(t *testing.T) {
	tests := []struct {
		name        string
		handler     grpc.UnaryHandler
		expectError bool
	}{
		{
			name: "successful handler",
			handler: func(ctx context.Context, req any) (any, error) {
				return "ok", nil
			},
			expectError: false,
		},
		{
			name: "failing handler",
			handler: func(ctx context.Context, req any) (any, error) {
				return nil, errors.New("handler error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := ServerLogger()
			resp, err := interceptor(context.Background(), "request", &grpc.UnaryServerInfo{}, tt.handler)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "ok", resp)
			}
		})
	}
}

func TestServerErrorx(t *testing.T) {
	tests := []struct {
		name          string
		handler       grpc.UnaryHandler
		shouldConvert bool
	}{
		{
			name: "successful handler",
			handler: func(ctx context.Context, req any) (any, error) {
				return "ok", nil
			},
			shouldConvert: false,
		},
		{
			name: "errorx error",
			handler: func(ctx context.Context, req any) (any, error) {
				return nil, errorx.Err_NOT_FOUND
			},
			shouldConvert: true,
		},
		{
			name: "standard error",
			handler: func(ctx context.Context, req any) (any, error) {
				return nil, errors.New("standard error")
			},
			shouldConvert: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := SevrerErrorx()
			resp, err := interceptor(context.Background(), "request", &grpc.UnaryServerInfo{}, tt.handler)

			if tt.shouldConvert {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "ok", resp)
			}
		})
	}
}

func TestServerValidation(t *testing.T) {
	invalidReq := &agent.GetElementReq{
		App:  "demo.app",
		Env:  "prod-1",
		Keys: []string{"db_url"},
	}
	tests := []struct {
		name        string
		req         any
		expectError bool
		errMsg      string
	}{
		{
			name:        "non-protobuf request",
			req:         "regular string",
			expectError: false,
		},
		{
			name: "valid protobuf request",
			req: &agent.GetElementReq{
				App:  "demo_app",
				Env:  "prod-1",
				Keys: []string{"db_url"},
			},
			expectError: false,
		},
		{
			name:        "invalid protobuf request",
			req:         invalidReq,
			expectError: true,
			errMsg:      "demo.app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := ServerValidation()
			handlerCalled := false
			handler := func(ctx context.Context, req any) (any, error) {
				handlerCalled = true
				return "ok", nil
			}

			resp, err := interceptor(context.Background(), tt.req, &grpc.UnaryServerInfo{}, handler)

			if tt.expectError {
				assertInvalidArgumentContract(t, err)
				if tt.errMsg != "" && err != nil {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.False(t, handlerCalled)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, "ok", resp)
			assert.True(t, handlerCalled)
		})
	}
}

func TestClientRecovery(t *testing.T) {
	tests := []struct {
		name        string
		invoker     grpc.UnaryInvoker
		expectPanic bool
		expectError bool
	}{
		{
			name: "normal invoker",
			invoker: func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
				return nil
			},
			expectPanic: false,
			expectError: false,
		},
		{
			name: "invoker returns error",
			invoker: func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
				return errors.New("invoker error")
			},
			expectPanic: false,
			expectError: true,
		},
		{
			name: "invoker panics",
			invoker: func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
				panic("client panic")
			},
			expectPanic: true,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := ClientRecovery()
			err := interceptor(context.Background(), "/TestMethod", "request", "reply", nil, tt.invoker)

			if tt.expectPanic {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "panic")
			} else if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClientErrorx(t *testing.T) {
	tests := []struct {
		name        string
		invoker     grpc.UnaryInvoker
		expectError bool
		checkError  func(*testing.T, error)
	}{
		{
			name: "successful invoker",
			invoker: func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
				return nil
			},
			expectError: false,
		},
		{
			name: "status error",
			invoker: func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
				return status.Error(codes.NotFound, "not found")
			},
			expectError: true,
			checkError: func(t *testing.T, err error) {
				// Should be converted to errorx
				assert.Error(t, err)
			},
		},
		{
			name: "standard error",
			invoker: func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
				return errors.New("standard error")
			},
			expectError: true,
			checkError: func(t *testing.T, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := ClientErrorx()
			err := interceptor(context.Background(), "/TestMethod", "request", "reply", nil, tt.invoker)

			if tt.expectError {
				assert.Error(t, err)
				if tt.checkError != nil {
					tt.checkError(t, err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClientValidation(t *testing.T) {
	invalidReq := &agent.GetElementReq{
		App:  "demo.app",
		Env:  "prod-1",
		Keys: []string{"db_url"},
	}
	tests := []struct {
		name          string
		req           any
		invokerCalled bool
		expectError   bool
		errMsg        string
	}{
		{
			name:          "non-protobuf request",
			req:           "regular string",
			invokerCalled: true,
			expectError:   false,
		},
		{
			name: "valid protobuf request",
			req: &agent.GetElementReq{
				App:  "demo_app",
				Env:  "prod-1",
				Keys: []string{"db_url"},
			},
			invokerCalled: true,
			expectError:   false,
		},
		{
			name:          "invalid protobuf request",
			req:           invalidReq,
			invokerCalled: false,
			expectError:   true,
			errMsg:        "demo.app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := ClientValidation()
			invokerCalled := false
			invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
				invokerCalled = true
				return nil
			}

			err := interceptor(context.Background(), "/TestMethod", tt.req, "reply", nil, invoker)

			assert.Equal(t, tt.invokerCalled, invokerCalled, "Invoker called status")

			if tt.expectError {
				assertInvalidArgumentContract(t, err)
				if tt.errMsg != "" && err != nil {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			assert.NoError(t, err)
		})
	}
}
