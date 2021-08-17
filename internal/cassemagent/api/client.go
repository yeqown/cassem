package api

// TODO(@yeqown): move this package outside of cassem/internal.

import (
	"context"
	"math/rand"
	"time"

	"github.com/pkg/errors"
	"github.com/yeqown/log"
	"google.golang.org/grpc"

	"github.com/yeqown/cassem/internal/concept"
)

var (
	_CLIENT_REQ_TIMEOUT  = 3 * time.Second
	_CLIENT_INIT_TIMEOUT = 10 * time.Second
)

const (
	_CLIENT_RENEW_BASE = 20
	_CLIENT_RENEW_RAND = 10
)

// Dial build a AgentClient to communicate with cassem agent server, it failed
// after 10 seconds timeout since building connection. The client has default
// interceptors, such as: recovery, errorx.
func Dial(addr string) (*clientWrapper, error) {
	timeout, cancel := context.WithTimeout(context.Background(), _CLIENT_INIT_TIMEOUT)
	defer cancel()

	cc, err := grpc.DialContext(timeout, addr,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithChainUnaryInterceptor(), // TODO(@yeqown): fill unary interceptors(recovery/errorx)
	)
	if err != nil {
		return nil, errors.Wrap(err, "cassemagent.api.Dial")
	}

	return newClient(cc), nil
}

type clientWrapper struct {
	c                  AgentClient
	app                string
	env                string
	clientId, clientIp string
	keys               []string
}

func newClient(cc *grpc.ClientConn) *clientWrapper {
	return &clientWrapper{
		c: NewAgentClient(cc),
	}
}

type NextHandlerFunc func(next *concept.Element)

// Wait of clientWrapper register itself to the agent and wait for next change of keys these
// it cares about.
//
// This is a `BLOCKING` method.
func (cw *clientWrapper) Wait(
	ctx context.Context, app, env, clientId, clientIp string, fn NextHandlerFunc, keys ...string) error {

	cw.app, cw.env, cw.keys = app, env, keys
	cw.clientId, cw.clientIp = clientId, clientIp

	stream, err := cw.c.RegisterAndWait(ctx, &RegAndWaitReq{
		App:          app,
		Env:          env,
		WatchingKeys: keys,
		ClientId:     clientId,
		ClientIp:     clientIp,
	})
	if err != nil {
		return errors.Wrap(err, "clientWrapper.Wait")
	}

	r := new(WaitResp)
	ctx2, cancel := context.WithCancel(stream.Context())
	defer cancel()

	go func(ctx context.Context) {
		// random ticker for renew client itself, random tick interval avoids
		// too many renew requests are sent to cassemdb at the same time.
		rt := time.NewTicker(time.Duration(_CLIENT_RENEW_BASE+rand.Intn(_CLIENT_RENEW_RAND)) * time.Second)

		for {
			select {
			case <-ctx.Done():
				log.Debug("clientWrapper renewSelf quit")
				return
			case <-rt.C:
				cw.renewSelf()
			}
		}
	}(ctx2)

waitLoop:
	for {
		select {
		case <-ctx2.Done():
			// connection quit?
			break waitLoop
		default:
			if err = stream.RecvMsg(r); err != nil {
				log.
					WithFields(log.Fields{
						"app":      app,
						"env":      env,
						"clientId": clientId,
						"clientIp": clientIp,
						"keys":     keys,
						"error":    err,
					}).
					Error("clientWrapper.Wait failed to receive message")
				return errors.Wrap(err, "clientWrapper.Wait.RecvMsg")
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
}

func (cw clientWrapper) renewSelf() {
	log.Debug("clientWrapper.renewSelf called")
	ctx, cancel := context.WithTimeout(context.Background(), _CLIENT_REQ_TIMEOUT)
	defer cancel()
	_, err := cw.c.Renew(ctx, &RenewReq{
		ClientId:     cw.clientId,
		Ip:           cw.clientIp,
		App:          cw.app,
		Env:          cw.env,
		WatchingKeys: cw.keys,
	})

	if err != nil {
		log.
			WithFields(log.Fields{
				"clientId": cw.clientId,
				"app":      cw.app,
				"env":      cw.env,
				"keys":     cw.keys,
				"clientIp": cw.clientIp,
			}).
			Error("clientWrapper.renewSelf failed")
	}
}

// GetConfig query those configs could be named by app, env, and keys. It means to
// get values of keys under the namespace(app.env).
func (cw clientWrapper) GetConfig(
	ctx context.Context, app, env string, keys ...string) ([]*concept.Element, error) {

	if _, ok := ctx.Deadline(); !ok {
		// hasn't set a deadline for request.
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, _CLIENT_REQ_TIMEOUT)
		defer cancel()
	}

	r, err := cw.c.GetConfig(ctx, &GetConfigReq{
		App:  app,
		Env:  env,
		Keys: keys,
	})
	if err != nil {
		return nil, errors.Wrap(err, "clientWrapper.GetConfig")
	}

	return r.GetElems(), nil
}
