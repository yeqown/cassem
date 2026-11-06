package agent

import (
	"context"
	"fmt"
	"time"

	"buf.build/go/protovalidate"
	"google.golang.org/grpc/status"

	pb "github.com/yeqown/cassem/internal/protogen"
	clientpb "github.com/yeqown/cassem/sdk/go/protogen"

	"github.com/yeqown/cassem/internal/concept"
	"github.com/yeqown/cassem/internal/pkg/runtime"
	"github.com/yeqown/log"
)

// GetElement execute query request from clients, and also at the same time, agent app is
// a cache-layer for cassemkv, so the request would be queried from cache firstly, cache is not hit
// query from cassemkv. It failed only when query from local cache and remote both failed.
//
// DONE(@yeqown): get from cache first, if not hit send request to cassemkv component, and then refresh caches.
func (d *app) GetElement(ctx context.Context, req *clientpb.GetElementReq) (*clientpb.GetElementResp, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
	}
	if req == nil {
		return nil, concept.Err_INVALID_ARGUMENT
	}
	if err := protovalidate.Validate(req); err != nil {
		return nil, concept.New(concept.Code_INVALID_ARGUMENT, err.Error())
	}
	if err := d.authorize(ctx, concept.TokenTarget{App: req.GetApp(), Env: req.GetEnv()}); err != nil {
		return nil, err
	}
	resp := new(clientpb.GetElementResp)

	cr := d.queryFromCache(req.GetApp(), req.GetEnv(), req.GetKeys()...)
	resp.Elems = append(resp.Elems, cr.elems...)

	// re-request from cassemkv missed keys from cache.
	if len(cr.miss) != 0 {
		r, err := d.aggregate.GetElementsByKeys(ctx, req.GetApp(), req.GetEnv(), cr.miss)
		if err != nil {
			return nil, err
		}
		resp.Elems = append(resp.Elems, r.Elements...)
		go d.updateCache(req.GetApp(), req.GetEnv(), r.Elements...)
	}

	return resp, nil
}

var (
	_emptyResp = new(clientpb.EmptyResp)
)

func (d *app) Register(ctx context.Context, req *clientpb.RegisterReq) (*clientpb.EmptyResp, error) {
	ins := &concept.Instance{
		ClientId:           req.GetClientId(),
		AgentId:            d.uniqueId,
		ClientIp:           req.GetClientIp(),
		Watching:           req.GetWatching(),
		LastRenewTimestamp: time.Now().Unix(),
	}

	if err := d.aggregate.RegisterInstance(ctx, ins); err != nil {
		return nil, err
	}

	return _emptyResp, nil
}

func (d *app) Renew(ctx context.Context, req *clientpb.RegisterReq) (*clientpb.EmptyResp, error) {
	ins := &concept.Instance{
		ClientId:           req.GetClientId(),
		AgentId:            d.uniqueId,
		ClientIp:           req.GetClientIp(),
		Watching:           req.GetWatching(),
		LastRenewTimestamp: time.Now().Unix(),
	}

	if err := d.aggregate.RenewInstance(ctx, ins); err != nil {
		return nil, err
	}

	return _emptyResp, nil
}

func (d *app) Unregister(ctx context.Context, req *clientpb.UnregisterReq) (*clientpb.EmptyResp, error) {
	insId := concept.InstanceID(&concept.Instance{
		ClientId: req.GetClientId(),
		ClientIp: req.GetClientIp(),
	})
	// make sure unregister instance from memory, avoid memory leaking.
	d.instancePool.Unregister(insId)

	err := d.aggregate.UnregisterInstance(ctx, insId)
	if err != nil {
		return nil, err
	}

	return _emptyResp, nil
}

func (d *app) Watch(req *clientpb.WatchReq, server clientpb.Agent_WatchServer) error {
	if req == nil {
		return concept.ToStatus(concept.Err_INVALID_ARGUMENT)
	}
	if err := protovalidate.Validate(req); err != nil {
		return status.Error(concept.Code_INVALID_ARGUMENT.Code(), err.Error())
	}
	targets := make([]concept.TokenTarget, 0, len(req.GetWatching()))
	allowed := make(map[watchElementKey]struct{})
	for _, watching := range req.GetWatching() {
		targets = append(targets, concept.TokenTarget{App: watching.GetApp(), Env: watching.GetEnv()})
		for _, key := range watching.GetWatchKeys() {
			allowed[watchElementKey{watching.GetApp(), watching.GetEnv(), key}] = struct{}{}
		}
	}
	if err := d.authorize(server.Context(), targets...); err != nil {
		return concept.ToStatus(err)
	}
	// if connection broken, unregister the instance from app.
	defer func() {
		ctx, cancel := context.WithTimeout(d.lifecycleContext(), 10*time.Second)
		if _, err := d.Unregister(ctx, &clientpb.UnregisterReq{
			ClientId: req.GetClientId(),
			ClientIp: req.GetClientIp(),
		}); err != nil {
			log.
				WithFields(log.Fields{
					"error": err,
					"req":   req,
				}).
				Error("app.Watch failed to unregister")
		}
		cancel()
	}()

	insId := concept.InstanceID(&concept.Instance{
		ClientId: req.GetClientId(),
		ClientIp: req.GetClientIp(),
	})

	// register all watching
	ch := make(<-chan *concept.Element, 10)
	watchings := req.GetWatching()
	for _, w := range watchings {
		ch = d.instancePool.Register(insId, w.GetApp(), w.GetEnv(), w.GetWatchKeys())
	}

	if err := d.aggregate.RenewInstance(server.Context(), &concept.Instance{
		ClientId:           req.GetClientId(),
		AgentId:            d.uniqueId,
		ClientIp:           req.GetClientIp(),
		Watching:           watchings,
		LastRenewTimestamp: time.Now().Unix(),
	}); err != nil {
		return err
	}

	log.
		WithFields(log.Fields{"req": req}).
		Debug("app.Watch called")

wait:
	for {
		select {
		case elem, ok := <-ch:
			if !ok {
				return nil
			}
			md := elem.GetMetadata()
			if _, ok := allowed[watchElementKey{md.GetApp(), md.GetEnv(), md.GetKey()}]; !ok {
				continue
			}
			if err := d.authorize(server.Context(), targets...); err != nil {
				return concept.ToStatus(err)
			}
			// DONE(@yeqown): update cache item while new changes pushed to instance.
			go d.updateCache(elem.GetMetadata().GetApp(), elem.GetMetadata().GetEnv(), elem)
			if err := server.Send(&clientpb.WatchResp{Elem: elem}); err != nil {
				log.
					WithFields(log.Fields{"element": elem, "err": err}).
					Error("app.Watch could not send")
				return fmt.Errorf("app.Watch.send: %w", err)
			}
		// maybe need to judge the error in case of client disconnected.
		case <-server.Context().Done():
			log.
				WithFields(log.Fields{
					"error": server.Context().Err(),
					"req":   req,
				}).
				Warn("app.Watch quit")
			break wait
		}
	}

	return nil
}

type watchElementKey struct{ app, env, key string }

var (
	_dispathResp = new(pb.DispatchResp)
)

// Dispatch implements the internal Delivery service.
func (d *app) Dispatch(ctx context.Context, req *pb.DispatchReq) (*pb.DispatchResp, error) {
	// DONE(@yeqown): implement dispatch rpc call to related client instances.
	log.
		WithFields(log.Fields{
			"elems":       req.GetElems(),
			"count":       len(req.GetElems()),
			"instanceIds": req.GetInstanceIds(),
		}).
		Info("cassemagent.app.Dispatch called")

	// start a routine to dispatch publish
	runtime.GoFunc("dispatchChange", func() error {
		targets := make(map[string]struct{}, len(req.GetInstanceIds()))
		for _, insId := range req.GetInstanceIds() {
			targets[insId] = struct{}{}
		}

		for _, v := range req.GetElems() {
			insIds := d.instancePool.ListWatchingInstances(
				v.GetMetadata().GetApp(), v.GetMetadata().GetEnv(), v.GetMetadata().GetKey())
			if len(targets) != 0 {
				filtered := make([]string, 0, len(insIds))
				for _, insId := range insIds {
					if _, ok := targets[insId]; ok {
						filtered = append(filtered, insId)
					}
				}
				insIds = filtered
			}

			log.
				WithFields(log.Fields{
					"insIds": insIds,
					"app":    v.GetMetadata().GetApp(),
					"env":    v.GetMetadata().GetEnv(),
					"key":    v.GetMetadata().GetKey(),
				}).
				Debug("app.Dispatch.dispatchChange to these instance")

			for _, insId := range insIds {
				d.instancePool.Notify(insId, v)
			}
		}

		return nil
	})

	return _dispathResp, nil
}
