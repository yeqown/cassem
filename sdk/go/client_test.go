package cassem_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	cassem "github.com/yeqown/cassem/sdk/go"
	clientpb "github.com/yeqown/cassem/sdk/go/protogen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestNewValidatesOptions(t *testing.T) {
	tests := []struct {
		name, address string
		options       []cassem.Option
	}{
		{"address", "", nil},
		{"client ID", "unused", []cassem.Option{cassem.WithClientId("")}},
		{"client IP", "unused", []cassem.Option{cassem.WithClientIp("")}},
		{"connect timeout", "unused", []cassem.Option{cassem.WithConnectTimeout(0)}},
		{"request timeout", "unused", []cassem.Option{cassem.WithRequestTimeout(-time.Second)}},
		{"renew interval", "unused", []cassem.Option{cassem.WithRenewInterval(0)}},
		{"nil option", "unused", []cassem.Option{nil}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := append([]cassem.Option{cassem.WithClientId("client-01"), cassem.WithClientIp("127.0.0.1")}, tt.options...)
			client, err := cassem.New(tt.address, options...)
			if err == nil || client != nil {
				t.Fatalf("New = (%v, %v), want nil client and validation error", client, err)
			}
		})
	}
}

func TestRegisterAndRead(t *testing.T) {
	f := newFixture(t)
	c := f.client(t)
	req := receive(t, f.registrations)
	if req.GetClientId() != "client-01" || req.GetClientIp() != "127.0.0.1" {
		t.Fatalf("wrong registration identity: %v", req)
	}
	elements, err := c.GetElement(context.Background(), "checkout", "production", "config", "limits")
	if err != nil {
		t.Fatal(err)
	}
	request := receive(t, f.reads)
	if request.GetApp() != "checkout" || request.GetEnv() != "production" || !reflect.DeepEqual(request.GetKeys(), []string{"config", "limits"}) {
		t.Fatalf("wrong read request: %v", request)
	}
	if len(elements) != 1 || string(elements[0].GetRaw()) != `{"enabled":true}` || elements[0].GetVersion() != 7 || elements[0].GetMetadata().GetKey() != "config" {
		t.Fatalf("wrong elements: %v", elements)
	}
}

func TestNewReturnsRegistrationFailureAndClosesConnection(t *testing.T) {
	tests := []struct {
		name     string
		register func(context.Context, *clientpb.RegisterReq) (*clientpb.EmptyResp, error)
		want     codes.Code
	}{
		{"server rejection", func(context.Context, *clientpb.RegisterReq) (*clientpb.EmptyResp, error) {
			return nil, status.Error(codes.PermissionDenied, "registration denied")
		}, codes.PermissionDenied},
		{"timeout", func(ctx context.Context, _ *clientpb.RegisterReq) (*clientpb.EmptyResp, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		}, codes.DeadlineExceeded},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			f.register = tt.register
			c, err := cassem.New("passthrough:///agent", append(f.options(), cassem.WithRequestTimeout(30*time.Millisecond))...)
			if c != nil || status.Code(err) != tt.want {
				t.Fatalf("New = (%v, %v), want %v", c, err, tt.want)
			}
			receive(t, f.connectionsClosed)
		})
	}
}

func TestNewReturnsConnectionTimeout(t *testing.T) {
	c, err := cassem.New("passthrough:///unreachable", cassem.WithClientID("client-01"), cassem.WithClientIP("127.0.0.1"), cassem.WithConnectTimeout(30*time.Millisecond),
		cassem.WithDialOptions(grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return nil, errors.New("unreachable") })))
	if c != nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("New = (%v, %v), want connection deadline error", c, err)
	}
}

func TestReadPropagatesStatusAndBoundsRequests(t *testing.T) {
	tests := []struct {
		name    string
		read    func(context.Context, *clientpb.GetElementReq) (*clientpb.GetElementResp, error)
		context func() (context.Context, context.CancelFunc)
		want    codes.Code
	}{
		{"server status", func(context.Context, *clientpb.GetElementReq) (*clientpb.GetElementResp, error) {
			return nil, status.Error(codes.NotFound, "missing config")
		}, func() (context.Context, context.CancelFunc) { return context.WithCancel(context.Background()) }, codes.NotFound},
		{"request timeout overrides longer deadline", func(ctx context.Context, _ *clientpb.GetElementReq) (*clientpb.GetElementResp, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		}, func() (context.Context, context.CancelFunc) {
			return context.WithTimeout(context.Background(), time.Hour)
		}, codes.DeadlineExceeded},
		{"caller cancellation", nil, func() (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return ctx, cancel
		}, codes.Canceled},
		{"caller deadline overrides request timeout", func(ctx context.Context, _ *clientpb.GetElementReq) (*clientpb.GetElementResp, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		}, func() (context.Context, context.CancelFunc) {
			return context.WithTimeout(context.Background(), 10*time.Millisecond)
		}, codes.DeadlineExceeded},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			f.read = tt.read
			c := f.client(t, cassem.WithToken("opaque-access-token"), cassem.WithRequestTimeout(40*time.Millisecond))
			ctx, cancel := tt.context()
			defer cancel()
			_, err := c.GetElement(ctx, "checkout", "production", "config")
			if status.Code(err) != tt.want {
				t.Fatalf("GetElement error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestWatchDeliversIndependentElements(t *testing.T) {
	f := newFixture(t)
	c := f.client(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	elements := make(chan *cassem.Element, 2)
	if err := c.Watch(ctx, "checkout", "production", func(element *cassem.Element) { elements <- element }, "config"); err != nil {
		t.Fatal(err)
	}
	stream := receive(t, f.watches)
	stream.elements <- nil
	stream.elements <- &cassem.Element{Version: 1, Raw: []byte("first")}
	first := receive(t, elements)
	stream.elements <- &cassem.Element{Version: 2, Raw: []byte("second")}
	second := receive(t, elements)
	if first == second || string(first.GetRaw()) != "first" || string(second.GetRaw()) != "second" || first.GetVersion() != 1 {
		t.Fatalf("previous response mutated: first=%v second=%v", first, second)
	}
	cancel()
	awaitDone(t, stream.done)
}

func TestWatchStreamFailureKeepsWatchingForRenew(t *testing.T) {
	f := newFixture(t)
	c := f.client(t, cassem.WithRenewInterval(10*time.Millisecond))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	keys := []string{"ele1", "config"}
	if err := c.Watch(ctx, "test", "default", func(*cassem.Element) {}, keys...); err != nil {
		t.Fatal(err)
	}
	stream := receive(t, f.watches)
	keys[0] = "caller-mutated"
	stream.fail <- status.Error(codes.Unavailable, "stream failed")
	err := receive(t, c.Errors())
	var background *cassem.BackgroundError
	if status.Code(err) != codes.Unavailable || !errors.As(err, &background) || background.Operation != "watch" || background.App != "test" || background.Env != "default" {
		t.Fatalf("wrong asynchronous watch error: %v", err)
	}
	req := receiveRenewal(t, f.renewals, 1)
	w := req.GetWatching()[0]
	if w.GetApp() != "test" || w.GetEnv() != "default" || !reflect.DeepEqual(w.GetWatchKeys(), []string{"ele1", "config"}) {
		t.Fatalf("watch intent lost or mutated: %v", req)
	}
	// Cancellation must remove intent even after the receive goroutine exits.
	cancel()
	receiveRenewal(t, f.renewals, 0)
}

func TestWatchContextCancellationRemovesWatchingForRenew(t *testing.T) {
	f := newFixture(t)
	c := f.client(t, cassem.WithToken("opaque-access-token"), cassem.WithRenewInterval(10*time.Millisecond))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := c.Watch(ctx, "test", "default", func(*cassem.Element) {}, "config"); err != nil {
		t.Fatal(err)
	}
	stream := receive(t, f.watches)
	receiveRenewal(t, f.renewals, 1)
	cancel()
	awaitDone(t, stream.done)
	receiveRenewal(t, f.renewals, 0)
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	for err := range c.Errors() {
		t.Errorf("normal cancellation produced background error: %v", err)
	}
}

func TestConcurrentWatchesPermitOneSubscription(t *testing.T) {
	f := newFixture(t)
	c := f.client(t, cassem.WithRenewInterval(time.Millisecond))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const count = 20
	results := make(chan error, count)
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results <- c.Watch(ctx, "test", "default", func(*cassem.Element) {}, fmt.Sprintf("key-%d", i))
		}(i)
	}
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		} else if !errors.Is(err, cassem.ErrWatchActive) {
			t.Errorf("Watch error = %v, want ErrWatchActive", err)
		}
	}
	if successes != 1 {
		t.Fatalf("started %d subscriptions, want one", successes)
	}
	stream := receive(t, f.watches)
	cancel()
	awaitDone(t, stream.done)
}

func TestRenewalFailureIsObservable(t *testing.T) {
	f := newFixture(t)
	f.renew = func(context.Context, *clientpb.RegisterReq) (*clientpb.EmptyResp, error) {
		return nil, status.Error(codes.Unavailable, "renew unavailable")
	}
	c := f.client(t, cassem.WithRenewInterval(time.Millisecond))
	err := receive(t, c.Errors())
	var background *cassem.BackgroundError
	if !errors.As(err, &background) || background.Operation != "renew" || status.Code(err) != codes.Unavailable {
		t.Fatalf("wrong renewal error: %v", err)
	}
}

func TestCloseCancelsRequestsUnregistersAndIsIdempotent(t *testing.T) {
	f := newFixture(t)
	f.read = func(ctx context.Context, _ *clientpb.GetElementReq) (*clientpb.GetElementResp, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	c := f.client(t, cassem.WithToken("opaque-access-token"), cassem.WithRequestTimeout(time.Second))
	if err := c.Watch(context.Background(), "test", "default", func(*cassem.Element) {}, "config"); err != nil {
		t.Fatal(err)
	}
	stream := receive(t, f.watches)
	readDone := make(chan error, 1)
	go func() { _, err := c.GetElement(context.Background(), "test", "default", "config"); readDone <- err }()
	receive(t, f.reads)
	closed := make(chan error, 10)
	for i := 0; i < cap(closed); i++ {
		go func() { closed <- c.Close() }()
	}
	for i := 0; i < cap(closed); i++ {
		if err := receive(t, closed); err != nil {
			t.Fatal(err)
		}
	}
	if status.Code(receive(t, readDone)) != codes.Canceled {
		t.Error("Close did not cancel the active read")
	}
	awaitDone(t, stream.done)
	receive(t, f.connectionsClosed)
	if got := f.unregisterCount.Load(); got != 1 {
		t.Errorf("Unregister called %d times, want once", got)
	}
	req := receive(t, f.unregistrations)
	if req.GetClientId() != "client-01" || req.GetClientIp() != "127.0.0.1" {
		t.Errorf("wrong unregister identity: %v", req)
	}
	if _, err := c.GetElement(context.Background(), "test", "default", "config"); !errors.Is(err, cassem.ErrClosed) {
		t.Errorf("GetElement after Close = %v, want ErrClosed", err)
	}
	if err := c.Watch(context.Background(), "test", "default", func(*cassem.Element) {}, "config"); !errors.Is(err, cassem.ErrClosed) {
		t.Errorf("Watch after Close = %v, want ErrClosed", err)
	}
	for err := range c.Errors() {
		t.Errorf("Close produced background error: %v", err)
	}
	c.Quit()
}

func TestClosePropagatesUnregisterFailureAndStillClosesConnection(t *testing.T) {
	f := newFixture(t)
	f.unregister = func(context.Context, *clientpb.UnregisterReq) (*clientpb.EmptyResp, error) {
		return nil, status.Error(codes.Unavailable, "unregister unavailable")
	}
	c, err := cassem.New("passthrough:///agent", f.options()...)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := c.Close(); status.Code(err) != codes.Unavailable {
			t.Fatalf("Close = %v, want Unavailable", err)
		}
	}
	receive(t, f.connectionsClosed)
	if f.unregisterCount.Load() != 1 {
		t.Fatal("failed unregister was repeated by Close")
	}
}

type fixture struct {
	clientpb.UnimplementedAgentServer
	listener          *bufconn.Listener
	registrations     chan *clientpb.RegisterReq
	reads             chan *clientpb.GetElementReq
	renewals          chan *clientpb.RegisterReq
	unregistrations   chan *clientpb.UnregisterReq
	watches           chan *watchSession
	connectionsClosed chan struct{}
	unregisterCount   atomic.Int32
	register          func(context.Context, *clientpb.RegisterReq) (*clientpb.EmptyResp, error)
	read              func(context.Context, *clientpb.GetElementReq) (*clientpb.GetElementResp, error)
	renew             func(context.Context, *clientpb.RegisterReq) (*clientpb.EmptyResp, error)
	unregister        func(context.Context, *clientpb.UnregisterReq) (*clientpb.EmptyResp, error)
}

type watchSession struct {
	elements chan *cassem.Element
	fail     chan error
	done     chan struct{}
}

func newFixture(t *testing.T, options ...grpc.ServerOption) *fixture {
	t.Helper()
	f := &fixture{listener: bufconn.Listen(1024 * 1024), registrations: make(chan *clientpb.RegisterReq, 16), reads: make(chan *clientpb.GetElementReq, 16), renewals: make(chan *clientpb.RegisterReq, 128), unregistrations: make(chan *clientpb.UnregisterReq, 16), watches: make(chan *watchSession, 16), connectionsClosed: make(chan struct{}, 16)}
	server := grpc.NewServer(options...)
	clientpb.RegisterAgentServer(server, f)
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(f.listener) }()
	t.Cleanup(func() {
		server.Stop()
		if err := receive(t, serveDone); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			t.Errorf("serve fixture: %v", err)
		}
	})
	return f
}

func (f *fixture) options() []cassem.Option {
	return []cassem.Option{cassem.WithClientId("client-01"), cassem.WithClientIp("127.0.0.1"), cassem.WithConnectTimeout(time.Second), cassem.WithRequestTimeout(200 * time.Millisecond), cassem.WithRenewInterval(time.Hour),
		cassem.WithDialOptions(grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			conn, err := f.listener.DialContext(ctx)
			if err != nil {
				return nil, err
			}
			return &trackedConn{Conn: conn, closed: f.connectionsClosed}, nil
		}))}
}

func (f *fixture) client(t *testing.T, options ...cassem.Option) *cassem.Client {
	t.Helper()
	c, err := cassem.New("passthrough:///agent", append(f.options(), options...)...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Errorf("close client: %v", err)
		}
	})
	return c
}

func (f *fixture) Register(ctx context.Context, req *clientpb.RegisterReq) (*clientpb.EmptyResp, error) {
	f.registrations <- req
	if f.register != nil {
		return f.register(ctx, req)
	}
	return &clientpb.EmptyResp{}, nil
}

func (f *fixture) GetElement(ctx context.Context, req *clientpb.GetElementReq) (*clientpb.GetElementResp, error) {
	f.reads <- req
	if f.read != nil {
		return f.read(ctx, req)
	}
	return &clientpb.GetElementResp{Elems: []*cassem.Element{{Metadata: &cassem.ElementMetadata{Key: "config"}, Version: 7, Raw: []byte(`{"enabled":true}`)}}}, nil
}

func (f *fixture) Renew(ctx context.Context, req *clientpb.RegisterReq) (*clientpb.EmptyResp, error) {
	select {
	case f.renewals <- req:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if f.renew != nil {
		return f.renew(ctx, req)
	}
	return &clientpb.EmptyResp{}, nil
}

func (f *fixture) Unregister(ctx context.Context, req *clientpb.UnregisterReq) (*clientpb.EmptyResp, error) {
	f.unregisterCount.Add(1)
	f.unregistrations <- req
	if f.unregister != nil {
		return f.unregister(ctx, req)
	}
	return &clientpb.EmptyResp{}, nil
}

func (f *fixture) Watch(_ *clientpb.WatchReq, stream clientpb.Agent_WatchServer) error {
	session := &watchSession{elements: make(chan *cassem.Element, 4), fail: make(chan error, 1), done: make(chan struct{})}
	defer close(session.done)
	f.watches <- session
	for {
		select {
		case element := <-session.elements:
			if err := stream.Send(&clientpb.WatchResp{Elem: element}); err != nil {
				return err
			}
		case err := <-session.fail:
			return err
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

type trackedConn struct {
	net.Conn
	once   sync.Once
	closed chan<- struct{}
}

func (c *trackedConn) Close() error {
	err := c.Conn.Close()
	c.once.Do(func() { c.closed <- struct{}{} })
	return err
}

func receive[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case value, ok := <-ch:
		if !ok {
			t.Fatal("channel closed before expected event")
		}
		return value
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event")
		var zero T
		return zero
	}
}

func receiveRenewal(t *testing.T, ch <-chan *clientpb.RegisterReq, watchingCount int) *clientpb.RegisterReq {
	t.Helper()
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	for {
		select {
		case req := <-ch:
			if len(req.GetWatching()) == watchingCount {
				return req
			}
		case <-timer.C:
			t.Fatal("timed out waiting for matching renewal")
			return nil
		}
	}
}

func awaitDone(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for stream cancellation")
	}
}
