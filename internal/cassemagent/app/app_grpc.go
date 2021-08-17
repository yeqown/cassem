package app

import (
	"context"
	"time"

	"github.com/yeqown/log"

	apiagent "github.com/yeqown/cassem/internal/cassemagent/api"
	"github.com/yeqown/cassem/internal/concept"
)

// GetConfig execute query request from clients, and also at the same time, agent app is
// a cache-layer for cassemdb, so the request would be queried from cache firstly, cache is not hit
// query from cassemdb. It failed only when query from local cache and remote both failed.
//
// DONE(@yeqown): get from cache first, if not hit send request to cassemdb component, and then refresh caches.
//
func (d app) GetConfig(ctx context.Context, req *apiagent.GetConfigReq) (*apiagent.GetConfigResp, error) {
	resp := new(apiagent.GetConfigResp)
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
	_emptyResp = new(apiagent.EmptyResp)
)

func (d app) RegisterAndWait(req *apiagent.RegAndWaitReq, server apiagent.Agent_RegisterAndWaitServer) error {
	ctx, cancel := context.WithTimeout(server.Context(), 10*time.Second)
	ch, err := d.register(ctx, req)
	if err != nil {
		cancel()
		return err
	}
	cancel()

	// if connection broken, unregister the instance from app.
	defer func() {
		ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
		if _, err2 := d.Unregister(ctx2, &apiagent.UnregisterReq{
			ClientId: req.GetClientId(),
			ClientIp: req.GetClientIp(),
			App:      req.GetApp(),
			Env:      req.GetEnv(),
		}); err2 != nil {
			log.
				WithFields(log.Fields{
					"error": err2,
					"req":   req,
				}).
				Error("app.RegisterAndWait failed to unregister")
		}
		cancel2()
	}()

wait:
	for {
		select {
		case elem := <-ch:
			// DONE(@yeqown): update cache item while new changes pushed to instance.
			go d.updateCache(elem.GetMetadata().GetApp(), elem.GetMetadata().GetEnv(), elem)
			err = server.Send(&apiagent.WaitResp{
				Elem: elem,
			})
			if err != nil {
				log.
					WithFields(log.Fields{"element": elem, "err": err}).
					Error("app.RegisterAndWait could not send")
			}
		// maybe need to judge the error in case of client disconnected.
		case <-server.Context().Done():
			log.
				WithFields(log.Fields{
					"error": err,
					"req":   req,
				}).
				Warn("app.RegisterAndWait quit")
			break wait
		}
	}

	return nil
}

func (d app) register(ctx context.Context, req *apiagent.RegAndWaitReq) (<-chan *concept.Element, error) {
	ins := &concept.Instance{
		ClientId:           req.GetClientId(),
		AgentId:            d.uniqueId,
		Ip:                 req.GetClientIp(),
		App:                req.GetApp(),
		Env:                req.GetEnv(),
		WatchKeys:          req.GetWatchingKeys(),
		LastRenewTimestamp: time.Now().Unix(),
	}

	if err := d.aggregate.RegisterInstance(ctx, ins); err != nil {
		return nil, err
	}

	ch := d.instancePool.Register(ins.Id())

	return ch, nil
}

func (d app) Unregister(ctx context.Context, req *apiagent.UnregisterReq) (*apiagent.EmptyResp, error) {
	insId := (&concept.Instance{ClientId: req.GetClientId(), Ip: req.GetClientIp()}).Id()
	// make sure unregister instance from memory, avoid memory leaking.
	d.instancePool.Unregister(insId)

	err := d.aggregate.UnregisterInstance(ctx, insId)
	if err != nil {
		return nil, err
	}

	return _emptyResp, nil
}

func (d app) Renew(ctx context.Context, req *apiagent.RenewReq) (*apiagent.EmptyResp, error) {
	ins := &concept.Instance{
		ClientId:           req.GetClientId(),
		AgentId:            d.uniqueId,
		Ip:                 req.GetIp(),
		App:                req.GetApp(),
		Env:                req.GetEnv(),
		WatchKeys:          req.GetWatchingKeys(),
		LastRenewTimestamp: time.Now().Unix(),
	}
	err := d.aggregate.RenewInstance(ctx, ins)
	return _emptyResp, err
}
