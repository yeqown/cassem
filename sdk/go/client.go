// Package cassem provides a Go client for reading configuration and watching
// published changes from a Cassem agent.
package cassem

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	clientpb "github.com/yeqown/cassem/sdk/go/protogen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// Element is a published configuration value, including its raw bytes and version.
type Element = clientpb.Element

// ElementMetadata identifies and describes a configuration value.
type ElementMetadata = clientpb.ElementMetadata

// ContentType describes the encoding of an element's raw bytes.
type ContentType = clientpb.ContentType

// Supported configuration content types.
const (
	ContentTypeUnknown   = clientpb.ContentType_UNKNOWN
	ContentTypeJSON      = clientpb.ContentType_JSON
	ContentTypeTOML      = clientpb.ContentType_TOML
	ContentTypeINI       = clientpb.ContentType_INI
	ContentTypePlaintext = clientpb.ContentType_PLAINTEXT
)

var (
	// ErrClosed is returned when an operation is attempted after Close.
	ErrClosed = errors.New("cassem: client is closed")
	// ErrWatchActive is returned when a client already has a watch subscription.
	// Use separate clients with distinct IDs to watch multiple app/env namespaces.
	ErrWatchActive = errors.New("cassem: client already has a watch; use a separate client with a unique ID")
)

// BackgroundError describes a watch or renewal failure reported through Errors.
// Unwrap preserves the underlying gRPC status for status.Code and errors.Is.
type BackgroundError struct {
	Operation string
	App       string
	Env       string
	Err       error
}

// Error returns the operation and underlying error.
func (e *BackgroundError) Error() string {
	if e.Operation == "watch" {
		return fmt.Sprintf("cassem: watch %s/%s: %v", e.App, e.Env, e.Err)
	}
	return fmt.Sprintf("cassem: %s: %v", e.Operation, e.Err)
}

// Unwrap returns the underlying error.
func (e *BackgroundError) Unwrap() error { return e.Err }

// Option configures a client created by New.
type Option func(*options)

type options struct {
	clientID       string
	clientIP       string
	token          string
	connectTimeout time.Duration
	requestTimeout time.Duration
	renewInterval  time.Duration
	dialOptions    []grpc.DialOption
}

// WithClientID sets the instance ID. Use a distinct ID for each client instance.
func WithClientID(id string) Option { return func(o *options) { o.clientID = id } }

// WithClientIP sets the IP address advertised for this client instance.
func WithClientIP(ip string) Option { return func(o *options) { o.clientIP = ip } }

// WithToken sends a user-bound access token as Bearer authorization metadata on
// every RPC. An empty token leaves authorization unchanged. Nonempty tokens must
// contain only visible ASCII characters without whitespace.
func WithToken(token string) Option { return func(o *options) { o.token = token } }

// WithClientId is the compatibility spelling of WithClientID.
func WithClientId(id string) Option { return WithClientID(id) }

// WithClientIp is the compatibility spelling of WithClientIP.
func WithClientIp(ip string) Option { return WithClientIP(ip) }

// WithConnectTimeout sets the maximum connection establishment time (default 10s).
// The duration must be positive.
func WithConnectTimeout(timeout time.Duration) Option {
	return func(o *options) { o.connectTimeout = timeout }
}

// WithRequestTimeout bounds unary RPCs, including registration, reads, renewal,
// and unregister (default 3s). An earlier caller deadline takes precedence.
// The duration must be positive. It does not limit the lifetime of a watch.
func WithRequestTimeout(timeout time.Duration) Option {
	return func(o *options) { o.requestTimeout = timeout }
}

// WithRenewInterval sets the interval between registration renewals (default 20s).
// The duration must be positive.
func WithRenewInterval(interval time.Duration) Option {
	return func(o *options) { o.renewInterval = interval }
}

// WithDialOptions appends gRPC options after the default plaintext credentials.
// Use grpc.WithTransportCredentials to configure TLS, or WithContextDialer for
// a custom transport. Options that grpc.NewClient ignores are also ignored here.
func WithDialOptions(opts ...grpc.DialOption) Option {
	copied := append([]grpc.DialOption(nil), opts...)
	return func(o *options) { o.dialOptions = append(o.dialOptions, copied...) }
}

// Client owns a connection to one Cassem agent and renews its registration.
// Its methods are safe for concurrent use. Do not copy a Client; call Close to
// release its connection and background work.
type Client struct {
	conn      *grpc.ClientConn
	agent     clientpb.AgentClient
	opt       options
	ctx       context.Context
	cancel    context.CancelFunc
	renewDone chan struct{}
	errors    chan error
	mu        sync.Mutex
	watching  *watchState
	closed    bool
	closeOnce sync.Once
	closeErr  error
}

type watchState struct {
	parent           context.Context
	cancel           context.CancelFunc
	stopCancellation func() bool
	intent           *clientpb.Instance_Watching
}

// New connects to an agent and registers the client before returning. Both
// WithClientID and WithClientIP are required. Connections use plaintext unless
// transport credentials are supplied through WithDialOptions.
func New(address string, opts ...Option) (*Client, error) {
	opt := options{connectTimeout: 10 * time.Second, requestTimeout: 3 * time.Second, renewInterval: 20 * time.Second}
	for _, apply := range opts {
		if apply == nil {
			return nil, errors.New("cassem: nil option")
		}
		apply(&opt)
	}
	if strings.TrimSpace(address) == "" {
		return nil, errors.New("cassem: agent address must not be empty")
	}
	if strings.TrimSpace(opt.clientID) == "" || strings.TrimSpace(opt.clientIP) == "" {
		return nil, errors.New("cassem: client ID and IP are required; use WithClientID and WithClientIP")
	}
	if opt.connectTimeout <= 0 || opt.requestTimeout <= 0 || opt.renewInterval <= 0 {
		return nil, errors.New("cassem: timeouts and renewal interval must be positive")
	}
	for i := 0; i < len(opt.token); i++ {
		if opt.token[i] <= ' ' || opt.token[i] >= 0x7f {
			return nil, errors.New("cassem: token must contain only visible ASCII characters without whitespace")
		}
	}
	dialOptions := append([]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}, opt.dialOptions...)
	if opt.token != "" {
		dialOptions = append(dialOptions,
			grpc.WithChainUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
				return invoke(withAuthorization(ctx, opt.token), method, req, reply, cc, opts...)
			}),
			grpc.WithChainStreamInterceptor(func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, stream grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
				return stream(withAuthorization(ctx, opt.token), desc, cc, method, opts...)
			}),
		)
	}
	conn, err := grpc.NewClient(address, dialOptions...)
	if err != nil {
		return nil, fmt.Errorf("cassem: connect: %w", err)
	}
	connectCtx, cancelConnect := context.WithTimeout(context.Background(), opt.connectTimeout)
	err = waitForConnReady(connectCtx, conn)
	cancelConnect()
	if err != nil {
		return nil, closeFailedConnection(conn, fmt.Errorf("cassem: connect: %w", err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := &Client{conn: conn, agent: clientpb.NewAgentClient(conn), opt: opt, ctx: ctx, cancel: cancel, renewDone: make(chan struct{}), errors: make(chan error, 16)}
	registerCtx, cancelRegister := context.WithTimeout(ctx, opt.requestTimeout)
	_, err = c.agent.Register(registerCtx, &clientpb.RegisterReq{ClientId: opt.clientID, ClientIp: opt.clientIP})
	cancelRegister()
	if err != nil {
		cancel()
		return nil, closeFailedConnection(conn, fmt.Errorf("cassem: register: %w", err))
	}
	go c.renewLoop()
	return c, nil
}

func withAuthorization(ctx context.Context, token string) context.Context {
	md, _ := metadata.FromOutgoingContext(ctx)
	if md == nil {
		md = make(metadata.MD)
	}
	md.Set("authorization", "Bearer "+token)
	return metadata.NewOutgoingContext(ctx, md)
}

func closeFailedConnection(conn *grpc.ClientConn, cause error) error {
	if err := conn.Close(); err != nil {
		return errors.Join(cause, fmt.Errorf("cassem: close failed connection: %w", err))
	}
	return cause
}

func waitForConnReady(ctx context.Context, conn *grpc.ClientConn) error {
	conn.Connect()
	for {
		state := conn.GetState()
		if state == connectivity.Ready {
			return nil
		}
		if state == connectivity.Shutdown {
			return ErrClosed
		}
		if !conn.WaitForStateChange(ctx, state) {
			return ctx.Err()
		}
	}
}

// GetElement reads published values for keys in the app/env namespace. The
// request is canceled when its context expires or the client is closed.
func (c *Client) GetElement(ctx context.Context, app, env string, keys ...string) ([]*Element, error) {
	c.mu.Lock()
	closed := c.closed
	c.mu.Unlock()
	if closed {
		return nil, ErrClosed
	}
	if ctx == nil {
		return nil, errors.New("cassem: GetElement requires a context")
	}
	requestCtx, cancel := context.WithTimeout(ctx, c.opt.requestTimeout)
	stop := context.AfterFunc(c.ctx, cancel)
	defer func() { stop(); cancel() }()
	response, err := c.agent.GetElement(requestCtx, &clientpb.GetElementReq{App: app, Env: env, Keys: keys})
	if err != nil {
		return nil, fmt.Errorf("cassem: get element: %w", err)
	}
	return response.GetElems(), nil
}

// WatchHandlerFunc receives a published configuration change. A watch invokes
// its handler serially. Handlers should return promptly to allow further events.
type WatchHandlerFunc func(next *Element)

// Watch starts an asynchronous subscription and returns once the stream opens.
// It does not send the current values; use GetElement for the initial snapshot.
// Cancel ctx to stop the subscription. A client supports one subscription at a
// time because the server routes streams by client identity.
//
// Receive failures are reported on Errors. They stop delivery but preserve the
// subscription in renewals until ctx is canceled or Close is called. Watches
// are not automatically reconnected. Close and recreate the client to recover.
func (c *Client) Watch(ctx context.Context, app, env string, fn WatchHandlerFunc, keys ...string) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return ErrClosed
	}
	if len(keys) == 0 {
		c.mu.Unlock()
		return nil
	}
	if ctx == nil || fn == nil {
		c.mu.Unlock()
		return errors.New("cassem: Watch requires a context and handler")
	}
	if err := ctx.Err(); err != nil {
		c.mu.Unlock()
		return fmt.Errorf("cassem: watch: %w", err)
	}
	if c.watching != nil {
		c.mu.Unlock()
		return ErrWatchActive
	}
	streamCtx, cancel := context.WithCancel(ctx)
	watching := &watchState{parent: ctx, cancel: cancel, intent: &clientpb.Instance_Watching{App: app, Env: env, WatchKeys: append([]string(nil), keys...)}}
	c.watching = watching
	// AfterFunc retains cancellation cleanup without a goroutine waiting forever
	// when a stream fails while the caller's context remains alive.
	watching.stopCancellation = context.AfterFunc(ctx, func() { c.removeWatching(watching) })
	c.mu.Unlock()

	stream, err := c.agent.Watch(streamCtx, &clientpb.WatchReq{ClientId: c.opt.clientID, ClientIp: c.opt.clientIP, Watching: []*clientpb.Instance_Watching{watching.intent}})
	if err != nil {
		c.removeWatching(watching)
		return fmt.Errorf("cassem: watch: %w", err)
	}
	go c.receiveWatch(watching, stream, fn)
	return nil
}

func (c *Client) receiveWatch(watching *watchState, stream clientpb.Agent_WatchClient, fn WatchHandlerFunc) {
	defer watching.cancel()
	for {
		response, err := stream.Recv()
		if err != nil {
			if watching.parent.Err() == nil && c.ctx.Err() == nil {
				c.report(&BackgroundError{Operation: "watch", App: watching.intent.GetApp(), Env: watching.intent.GetEnv(), Err: err})
			}
			return
		}
		if watching.parent.Err() != nil || c.ctx.Err() != nil {
			return
		}
		if element := response.GetElem(); element != nil {
			fn(element)
		}
	}
}

func (c *Client) removeWatching(watching *watchState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.watching == watching {
		watching.stopCancellation()
		watching.cancel()
		c.watching = nil
	}
}

func (c *Client) renewLoop() {
	defer close(c.renewDone)
	ticker := time.NewTicker(c.opt.renewInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.renewSelf()
		}
	}
}

func (c *Client) renewSelf() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	req := &clientpb.RegisterReq{ClientId: c.opt.clientID, ClientIp: c.opt.clientIP}
	if c.watching != nil && c.watching.parent.Err() == nil {
		req.Watching = []*clientpb.Instance_Watching{c.watching.intent}
	}
	c.mu.Unlock()
	ctx, cancel := context.WithTimeout(c.ctx, c.opt.requestTimeout)
	defer cancel()
	if _, err := c.agent.Renew(ctx, req); err != nil && c.ctx.Err() == nil {
		c.report(&BackgroundError{Operation: "renew", Err: err})
	}
}

// Errors reports asynchronous watch and renewal failures. The channel has a
// 16-error buffer; additional errors are dropped while full so an unread channel
// cannot block renewal or shutdown. Close closes the channel. Synchronous
// operations return their errors directly. Explicit cancellation is not an error.
func (c *Client) Errors() <-chan error { return c.errors }

func (c *Client) report(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	select {
	case c.errors <- err:
	default:
	}
}

// Close cancels ongoing requests and the watch, stops renewal, unregisters the
// instance, and closes the connection. It is safe to call concurrently or more
// than once; each call returns the same result. Unregister is bounded by the
// request timeout and its failure does not prevent local cleanup.
// A handler already executing may finish after Close returns.
func (c *Client) Close() error {
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.closed = true
		c.cancel()
		if c.watching != nil {
			c.watching.stopCancellation()
			c.watching.cancel()
			c.watching = nil
		}
		c.mu.Unlock()
		<-c.renewDone
		ctx, cancel := context.WithTimeout(context.Background(), c.opt.requestTimeout)
		_, err := c.agent.Unregister(ctx, &clientpb.UnregisterReq{ClientId: c.opt.clientID, ClientIp: c.opt.clientIP})
		cancel()
		if err != nil {
			c.closeErr = fmt.Errorf("cassem: unregister: %w", err)
		}
		if err := c.conn.Close(); err != nil {
			c.closeErr = errors.Join(c.closeErr, fmt.Errorf("cassem: close connection: %w", err))
		}
		c.mu.Lock()
		close(c.errors)
		c.mu.Unlock()
	})
	return c.closeErr
}

// Quit closes the client for compatibility with earlier releases. Use Close to
// observe an unregister or connection cleanup error.
func (c *Client) Quit() { _ = c.Close() }
