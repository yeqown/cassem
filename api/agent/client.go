package agent

import (
	"context"
	"math/rand"
	"time"

	"github.com/pkg/errors"
	"github.com/yeqown/log"
	"google.golang.org/grpc"

	"github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/pkg/grpcx"
	"github.com/yeqown/cassem/pkg/runtime"
)

var (
	_CLIENT_REQ_TIMEOUT  = 3 * time.Second
	_CLIENT_INIT_TIMEOUT = 10 * time.Second
)

const (
	_CLIENT_RENEW_BASE = 20
	_CLIENT_RENEW_RAND = 10
)

type agentInstanceClient struct {
	agentClient AgentClient
	opt         *options
	watching    []*concept.Instance_Watching
	quit        chan struct{}
	ctx         context.Context
	cancel      context.CancelFunc
}

type clientOption func(o *options)
type options struct {
	clientId string
	clientIp string

	watchingKeys []string
}

func WithClientId(clientId string) clientOption {
	return func(o *options) {
		o.clientId = clientId
	}
}

func WithClientIp(clientIp string) clientOption {
	return func(o *options) {
		o.clientIp = clientIp
	}
}

func New(agentAddress string, opts ...clientOption) (*agentInstanceClient, error) {
	dst := new(options)
	for _, apply := range opts {
		apply(dst)
	}

	if dst.clientId == "" || dst.clientIp == "" {
		return nil, errors.New("clientId and clientIp could not be empty," +
			" use WithClientId/WithClientIp to set!")
	}

	cc, err := dial(agentAddress)
	if err != nil {
		panic(err)
	}

	return newClient(cc, dst), nil
}

// dial build a AgentClient to communicate with cassem agent server, it failed
// after 10 seconds timeout since building connection. The client has default
// interceptors, such as: recovery, errorx.
func dial(addr string) (*grpc.ClientConn, error) {
	timeout, cancel := context.WithTimeout(context.Background(), _CLIENT_INIT_TIMEOUT)
	defer cancel()

	cc, err := grpc.DialContext(timeout, addr,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithChainUnaryInterceptor(grpcx.ClientRecovery(), grpcx.ClientErrorx(), grpcx.ClientValidation()),
	)
	if err != nil {
		return nil, errors.Wrap(err, "cassemagent.api.Dial")
	}

	return cc, nil
}

func newClient(cc *grpc.ClientConn, opt *options) *agentInstanceClient {
	ctx, cancel := context.WithCancel(context.Background())
	c := &agentInstanceClient{
		agentClient: NewAgentClient(cc),
		opt:         opt,
		quit:        make(chan struct{}, 1),
		ctx:         ctx,
		cancel:      cancel,
	}

	// start a renew self goroutine.
	runtime.GoFunc("renewSelf", func() error {
		// random ticker for renew client itself, random tick interval avoids
		// too many renew requests are sent to cassemdb at the same time.
		t := time.NewTicker(time.Duration(_CLIENT_RENEW_BASE+rand.Intn(_CLIENT_RENEW_RAND)) * time.Second)
		for {
			select {
			case <-c.quit:
				return nil
			case <-t.C:
				c.renewSelf()
			}
		}
	})

	return c
}

func (c *agentInstanceClient) Quit() {
	// cancel all watch goroutines
	c.cancel()

	c.quit <- struct{}{}
}

type WatchHandlerFunc func(next *concept.Element)

// Watch of agentInstanceClient register itself to the agent and wait for next change of keys these
// it cares about.
//
// This is a `BLOCKING` method.
func (c *agentInstanceClient) Watch(
	ctx context.Context, app, env string, fn WatchHandlerFunc, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	stream, err := c.agentClient.Watch(ctx, &WatchReq{
		ClientId: c.opt.clientId,
		ClientIp: c.opt.clientIp,
		Watching: []*concept.Instance_Watching{
			{
				App:       app,
				Env:       env,
				WatchKeys: keys,
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "agentInstanceClient.Watch")
	}

	// start a routine to watch
	runtime.GoFunc("watching", func() error {
		r := new(WatchResp)
	waitLoop:
		for {
			select {
			case <-ctx.Done():
				log.Debug("agentInstanceClient quit, watch Done)")
				break waitLoop
			case <-c.ctx.Done():
				log.Debug("agentInstanceClient quit, client Done)")
				break waitLoop
			case <-stream.Context().Done():
				log.Debug("agentInstanceClient quit, stream Done)")
				break waitLoop
			default:
				if err = stream.RecvMsg(r); err != nil {
					log.
						WithFields(log.Fields{
							"app":      app,
							"env":      env,
							"clientId": c.opt.clientId,
							"clientIp": c.opt.clientIp,
							"keys":     keys,
							"error":    err,
						}).
						Error("agentInstanceClient.Watch failed to receive message")
					return errors.Wrap(err, "agentInstanceClient.Watch.RecvMsg")
				}
				if r.GetElem() == nil {
					continue
				}
				// delivery element to client.
				fn(r.GetElem())
			}
			// select end
		}
		return nil
	})

	return nil
}

func (c agentInstanceClient) renewSelf() {
	log.Debug("agentInstanceClient.renewSelf called")
	ctx, cancel := context.WithTimeout(context.Background(), _CLIENT_REQ_TIMEOUT)
	defer cancel()
	_, err := c.agentClient.RegisterOrRenew(ctx, &RegisterReq{
		ClientId: c.opt.clientId,
		ClientIp: c.opt.clientIp,
		Watching: c.watching,
	})

	if err != nil {
		log.
			WithFields(log.Fields{
				"clientId": c.opt.clientId,
				"clientIp": c.opt.clientIp,
				"watching": c.watching,
			}).
			Error("agentInstanceClient.renewSelf failed")
	}
}

// GetElement query those configs could be named by app, env, and keys. It means to
// get values of keys under the namespace(app.env).
func (c agentInstanceClient) GetElement(
	ctx context.Context, app, env string, keys ...string) ([]*concept.Element, error) {

	if _, ok := ctx.Deadline(); !ok {
		// hasn't set a deadline for request.
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, _CLIENT_REQ_TIMEOUT)
		defer cancel()
	}

	r, err := c.agentClient.GetElement(ctx, &GetElementReq{
		App:  app,
		Env:  env,
		Keys: keys,
	})
	if err != nil {
		return nil, errors.Wrap(err, "agentInstanceClient.GetElement")
	}

	return r.GetElems(), nil
}
