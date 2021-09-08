package app

import (
	"context"
	"time"

	"github.com/yeqown/log"

	"github.com/yeqown/cassem/api/agent"
	"github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/pkg/runtime"
)

// GetElement execute query request from clients, and also at the same time, agent app is
// a cache-layer for cassemdb, so the request would be queried from cache firstly, cache is not hit
// query from cassemdb. It failed only when query from local cache and remote both failed.
//
// DONE(@yeqown): get from cache first, if not hit send request to cassemdb component, and then refresh caches.
//
func (d app) GetElement(ctx context.Context, req *agent.GetElementReq) (*agent.GetElementResp, error) {
	resp := new(agent.GetElementResp)
	if len(req.GetKeys()) == 0 {
		return resp, nil
	}

	if _, ok := ctx.Deadline(); !ok {
		// hasn't set a deadline for request.
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
	}

	cr := d.queryFromCache(req.GetApp(), req.GetEnv(), req.GetKeys()...)
	resp.Elems = append(resp.Elems, cr.elems...)

	// re-request from cassemdb missed keys from cache.
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
	_emptyResp = new(agent.EmptyResp)
)

func (d app) RegisterOrRenew(ctx context.Context, req *agent.RegisterReq) (*agent.EmptyResp, error) {
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

func (d app) Unregister(ctx context.Context, req *agent.UnregisterReq) (*agent.EmptyResp, error) {
	insId := (&concept.Instance{
		ClientId: req.GetClientId(),
		ClientIp: req.GetClientIp(),
	}).Id()
	// make sure unregister instance from memory, avoid memory leaking.
	d.instancePool.Unregister(insId)

	err := d.aggregate.UnregisterInstance(ctx, insId)
	if err != nil {
		return nil, err
	}

	return _emptyResp, nil
}

func (d app) Watch(req *agent.WatchReq, server agent.Agent_WatchServer) error {
	// if connection broken, unregister the instance from app.
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if _, err := d.Unregister(ctx, &agent.UnregisterReq{
			ClientId: req.GetClientId(),
			ClientIp: req.GetClientIp(),
		}); err != nil {
			log.
				WithFields(log.Fields{
					"error": err,
					"req":   req,
				}).
				Error("app.RegisterAndWait failed to unregister")
		}
		cancel()
	}()

	insId := (&concept.Instance{
		ClientId: req.GetClientId(),
		ClientIp: req.GetClientIp(),
	}).Id()

	// register all watching
	ch := make(<-chan *concept.Element, 10)
	watchings := req.GetWatching()
	for _, w := range watchings {
		ch = d.instancePool.Register(insId, w.GetApp(), w.GetEnv(), w.GetWatchKeys())
	}

wait:
	for {
		select {
		case elem := <-ch:
			// DONE(@yeqown): update cache item while new changes pushed to instance.
			go d.updateCache(elem.GetMetadata().GetApp(), elem.GetMetadata().GetEnv(), elem)
			if err := server.Send(&agent.WatchResp{Elem: elem}); err != nil {
				log.
					WithFields(log.Fields{"element": elem, "err": err}).
					Error("app.RegisterAndWait could not send")
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

var (
	_dispathResp = new(agent.DispatchResp)
)

// Dispatch implements apiagent.Delivery service
func (d app) Dispatch(ctx context.Context, req *agent.DispatchReq) (*agent.DispatchResp, error) {
	// DONE(@yeqown): implement dispatch rpc call to related client instances.
	log.
		WithFields(log.Fields{
			"elems": req.GetElems(),
			"count": len(req.GetElems()),
		}).
		Info("dispatch request")

	// start a routine to dispatch publish
	runtime.GoFunc("dispatchChange", func() error {
		for _, v := range req.GetElems() {
			insIds := d.instancePool.ListWatchingInstances(
				v.GetMetadata().GetApp(), v.GetMetadata().GetEnv(), v.GetMetadata().GetKey())

			log.
				WithFields(log.Fields{
					"insIds": insIds,
					"app":    v.GetMetadata().GetApp(),
					"env":    v.GetMetadata().GetEnv(),
					"key":    v.GetMetadata().GetKey(),
				}).
				Debug("app.Dispatch.dispatchChange to these instance")

			for _, insId := range insIds.Keys() {
				d.instancePool.Notify(insId, v)
			}
		}

		return nil
	})

	return _dispathResp, nil
}
