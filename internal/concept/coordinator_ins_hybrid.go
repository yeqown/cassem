package concept

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/yeqown/log"

	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
	"github.com/yeqown/cassem/pkg/errorx"
	"github.com/yeqown/cassem/pkg/runtime"
)

var (
	_ InstanceHybrid = instanceHybrid{}
)

type instanceHybrid struct {
	cassemdb apicassemdb.KVClient
}

func NewInstanceHybrid(endpoints []string) (InstanceHybrid, error) {
	cc, err := apicassemdb.DialWithMode(endpoints, apicassemdb.Mode_X)
	if err != nil {
		return nil, errors.Wrap(err, "NewInstanceHybrid")
	}

	return instanceHybrid{
		cassemdb: apicassemdb.NewKVClient(cc),
	}, nil
}

func (i instanceHybrid) GetElementInstances(ctx context.Context, app, env, key string) ([]*Instance, error) {
	k := genInstanceReversedKey(app, env, key)
	log.
		WithFields(log.Fields{
			"app":         app,
			"env":         env,
			"key":         key,
			"reversedKey": k,
		}).
		Debug("instanceHybrid.GetElementInstances")

	r, err := i.cassemdb.Range(ctx, &apicassemdb.RangeReq{
		Key:   k,
		Seek:  "",
		Limit: 100, // TODO(@yeqown): allow limit variable
	})
	if err != nil {
		return nil, errors.Wrap(err, "instanceHybrid.GetElementInstances")
	}

	insIds := make([]string, 0, len(r.GetEntities()))
	for _, v := range r.GetEntities() {
		insId := genInstanceNormalKey(runtime.ToString(v.GetVal()))
		insIds = append(insIds, insId)
	}

	// get all instance detail information.
	r2, err2 := i.cassemdb.GetKVs(ctx, &apicassemdb.GetKVsReq{
		Keys: insIds,
	})
	if err2 != nil {
		return nil, errors.Wrap(err, "instanceHybrid.GetElementInstances")
	}
	instances := make([]*Instance, 0, len(r2.GetEntities()))
	for _, v := range r2.GetEntities() {
		ins := new(Instance)
		_ = UnmarshalProto(v.GetVal(), ins)
		instances = append(instances, ins)
	}

	return instances, nil
}

func (i instanceHybrid) GetInstance(ctx context.Context, insId string) (*Instance, error) {
	k := genInstanceNormalKey(insId)
	r, err := i.cassemdb.GetKV(ctx, &apicassemdb.GetKVReq{
		Key: k,
	})
	if err != nil {
		return nil, err
	}

	ins := new(Instance)
	err = UnmarshalProto(r.GetEntity().GetVal(), ins)
	return ins, err
}

// RegisterInstance registers a new instance.
func (i instanceHybrid) RegisterInstance(ctx context.Context, ins *Instance) (err error) {
	// check duplicate instance
	insId := ins.Id()
	k := genInstanceNormalKey(insId)

	r, err := i.cassemdb.GetKV(ctx, &apicassemdb.GetKVReq{
		Key: k,
	})
	if err != nil && !errors.Is(err, errorx.Err_NOT_FOUND) {
		return err
	}
	if r.GetEntity() != nil {
		return errorx.New(errorx.Code_ALREADY_EXISTS, "instance has already been registered")
	}

	if t := time.Unix(ins.LastRenewTimestamp, 0); t.IsZero() {
		ins.LastRenewTimestamp = time.Now().Unix()
	}

	return i.setInstanceInfo(ctx, ins)
}

func (i instanceHybrid) setInstanceInfo(ctx context.Context, ins *Instance) (err error) {
	if ins == nil {
		log.
			Warn("InstanceHybrid.RegisterInstance get nil instance, skipped")
		return
	}
	insId := ins.Id()
	// TODO(@yeqown): should keep insId be unique in cluster?
	// save normalized kv
	k := genInstanceNormalKey(insId)
	log.
		WithFields(log.Fields{
			"insId":         insId,
			"normalizedKey": k,
		}).
		Debug("instanceHybrid.UnregisterInstance")

	bytes, _ := MarshalProto(ins)
	_, err = i.cassemdb.SetKV(ctx, &apicassemdb.SetKVReq{
		Key:       k,
		IsDir:     false,
		Ttl:       120,
		Val:       bytes,
		Overwrite: true,
	})
	if err != nil {
		return errors.Wrap(err, "instanceHybrid.GetInstance")
	}

	// save reversed kv
	for _, key := range ins.WatchKeys {
		k2 := genInstanceReversedKeyWithInsid(ins.App, ins.Env, key, insId)
		_, err = i.cassemdb.SetKV(ctx, &apicassemdb.SetKVReq{
			Key:       k2,
			IsDir:     false,
			Ttl:       120,
			Val:       runtime.ToBytes(insId),
			Overwrite: true,
		})
		if err != nil {
			log.
				WithFields(log.Fields{
					"key":   k2,
					"error": err,
				}).
				Error("instanceHybrid.GetInstance failed to update reversed")
		}
	}

	return nil
}

func (i instanceHybrid) RenewInstance(ctx context.Context, ins *Instance) error {
	// check duplicate instance
	//insId := ins.Id()
	//k := genInstanceNormalKey(insId)
	//r, _ := i.cassemdb.GetKV(ctx, &apicassemdb.GetKVReq{
	//	Key: k,
	//})
	//if r.GetEntity() != nil {
	//	if ins.LastRenewTimestamp.IsZero() {
	//		ins.LastRenewTimestamp = r.GetEntity().Get
	//	}
	//}

	return i.setInstanceInfo(ctx, ins)
}

func (i instanceHybrid) UnregisterInstance(ctx context.Context, insId string) error {
	k := genInstanceNormalKey(insId)
	log.
		WithFields(log.Fields{
			"insId":         insId,
			"normalizedKey": k,
		}).
		Debug("instanceHybrid.UnregisterInstance")

	// try to get instance detail
	r, err := i.cassemdb.GetKV(ctx, &apicassemdb.GetKVReq{
		Key: k,
	})
	if err != nil {
		if errors.Is(err, errorx.Err_NOT_FOUND) {
			return nil
		}

		return errors.Wrap(err, "instanceHybrid.UnregisterInstance")
	}

	ins := new(Instance)
	if err = UnmarshalProto(r.GetEntity().GetVal(), ins); err != nil {
		return errors.Wrap(err, "instanceHybrid.UnregisterInstance")
	}

	// unset normalized kv
	_, err = i.cassemdb.UnsetKV(ctx, &apicassemdb.UnsetKVReq{
		Key:   k,
		IsDir: false,
	})

	// unset reversed kv
	for _, key := range ins.WatchKeys {
		k2 := genInstanceReversedKeyWithInsid(ins.App, ins.Env, key, insId)
		_, err = i.cassemdb.UnsetKV(ctx, &apicassemdb.UnsetKVReq{
			Key: k2,
		})
		if err != nil {
			log.
				WithFields(log.Fields{
					"key":   k2,
					"error": err,
				}).
				Error("instanceHybrid.GetInstance failed to update reversed")
		}
	}

	return err
}
