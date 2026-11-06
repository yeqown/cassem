package cassem_test

import (
	"context"
	"errors"
	"net"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	cassem "github.com/yeqown/cassem/sdk/go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestTokenReachesEveryRPCWithoutMutatingCallerMetadata(t *testing.T) {
	type call struct {
		method string
		md     metadata.MD
	}
	calls := make(chan call, 128)
	record := func(ctx context.Context, method string) {
		md, _ := metadata.FromIncomingContext(ctx)
		calls <- call{method: method, md: md}
	}
	f := newFixture(t,
		grpc.UnaryInterceptor(func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			record(ctx, info.FullMethod)
			return handler(ctx, req)
		}),
		grpc.StreamInterceptor(func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			record(stream.Context(), info.FullMethod)
			return handler(srv, stream)
		}),
	)
	c := f.client(t, cassem.WithToken("opaque-access-token"), cassem.WithRenewInterval(10*time.Millisecond))
	callerMD := metadata.Pairs("x-request-id", "request-42", "authorization", "Bearer caller-credential")
	ctx, cancel := context.WithTimeout(metadata.NewOutgoingContext(context.Background(), callerMD), time.Second)
	defer cancel()
	if _, err := c.GetElement(ctx, "checkout", "production", "config"); err != nil {
		t.Fatal(err)
	}
	if err := c.Watch(ctx, "checkout", "production", func(*cassem.Element) {}, "config"); err != nil {
		t.Fatal(err)
	}
	session := receive(t, f.watches)
	receive(t, f.renewals)
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	awaitDone(t, session.done)
	if ctx.Err() != nil {
		t.Fatalf("client shutdown canceled caller context: %v", ctx.Err())
	}
	if got, _ := metadata.FromOutgoingContext(ctx); !reflect.DeepEqual(got, callerMD) {
		t.Fatalf("caller metadata was mutated: %v", got)
	}
	seen := map[string]bool{}
	for len(calls) > 0 {
		got := <-calls
		if !reflect.DeepEqual(got.md.Get("authorization"), []string{"Bearer opaque-access-token"}) {
			t.Errorf("%s authorization = %v", got.method, got.md.Get("authorization"))
		}
		if strings.HasSuffix(got.method, "/GetElement") || strings.HasSuffix(got.method, "/Watch") {
			if !reflect.DeepEqual(got.md.Get("x-request-id"), []string{"request-42"}) {
				t.Errorf("%s lost caller metadata", got.method)
			}
		}
		seen[got.method[strings.LastIndex(got.method, "/")+1:]] = true
	}
	for _, method := range []string{"Register", "GetElement", "Watch", "Renew", "Unregister"} {
		if !seen[method] {
			t.Errorf("did not observe %s", method)
		}
	}
}

func TestMalformedTokenRejectedBeforeDialWithoutExposingToken(t *testing.T) {
	for _, token := range []string{" ", " secret", "secret ", "secret token", "secret\nvalue", "secret\rvalue", "secret\tvalue", "secret\x00value", "secret\x7fvalue", "secret\u00a0value"} {
		t.Run(strings.ReplaceAll(token, "secret", "redacted"), func(t *testing.T) {
			var dialed atomic.Bool
			c, err := cassem.New("passthrough:///agent", cassem.WithClientID("client"), cassem.WithClientIP("127.0.0.1"), cassem.WithToken(token),
				cassem.WithConnectTimeout(20*time.Millisecond),
				cassem.WithDialOptions(grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
					dialed.Store(true)
					return nil, errors.New("unexpected dial")
				})))
			if err == nil || c != nil {
				t.Fatal("malformed token should fail validation")
			}
			if dialed.Load() {
				t.Fatal("malformed token reached the transport")
			}
			if (strings.TrimSpace(token) != "" && strings.Contains(err.Error(), token)) || strings.Contains(err.Error(), "secret") {
				t.Fatal("validation error exposed token material")
			}
		})
	}
}

func TestEmptyTokenPreservesCallerAuthorization(t *testing.T) {
	seen := make(chan metadata.MD, 2)
	f := newFixture(t, grpc.UnaryInterceptor(func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, _ := metadata.FromIncomingContext(ctx)
		seen <- md
		return handler(ctx, req)
	}))
	c := f.client(t, cassem.WithToken(""))
	if got := receive(t, seen).Get("authorization"); len(got) != 0 {
		t.Fatalf("tokenless registration sent authorization: %v", got)
	}
	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer caller-credential")
	if _, err := c.GetElement(ctx, "checkout", "production", "config"); err != nil {
		t.Fatal(err)
	}
	if got := receive(t, seen).Get("authorization"); !reflect.DeepEqual(got, []string{"Bearer caller-credential"}) {
		t.Fatalf("empty token changed caller authorization: %v", got)
	}
}

func TestTokenPreservesReadAndWatchAuthorizationStatuses(t *testing.T) {
	for _, code := range []codes.Code{codes.Unauthenticated, codes.PermissionDenied} {
		t.Run(code.String(), func(t *testing.T) {
			f := newFixture(t,
				grpc.UnaryInterceptor(func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
					if strings.HasSuffix(info.FullMethod, "/GetElement") {
						return nil, status.Error(code, "access denied")
					}
					return handler(ctx, req)
				}),
				grpc.StreamInterceptor(func(any, grpc.ServerStream, *grpc.StreamServerInfo, grpc.StreamHandler) error {
					return status.Error(code, "access denied")
				}),
			)
			c := f.client(t, cassem.WithToken("opaque-access-token"))
			if _, err := c.GetElement(context.Background(), "checkout", "production", "config"); status.Code(err) != code {
				t.Fatalf("GetElement status = %v, want %v", err, code)
			}
			if err := c.Watch(context.Background(), "checkout", "production", func(*cassem.Element) {}, "config"); err != nil {
				t.Fatal(err)
			}
			if err := receive(t, c.Errors()); status.Code(err) != code {
				t.Fatalf("watch status = %v, want %v", err, code)
			}
		})
	}
}
