package concept

import (
	"context"

	"github.com/pkg/errors"
	"github.com/yeqown/log"

	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
	pbcassemdb "github.com/yeqown/cassem/internal/cassemdb/api/gen"
	"github.com/yeqown/cassem/pkg/runtime"
)

var (
	_ InstanceHybrid = instanceHybrid{}
)

type instanceHybrid struct {
	cassemdb pbcassemdb.KVClient
}

func NewInstanceHybrid(endpoints []string) (InstanceHybrid, error) {
	cc, err := apicassemdb.DialWithMode(endpoints, apicassemdb.Mode_X)
	if err != nil {
		return nil, errors.Wrap(err, "NewInstanceHybrid")
	}

	return instanceHybrid{
		cassemdb: pbcassemdb.NewKVClient(cc),
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

	r, err := i.cassemdb.Range(ctx, &pbcassemdb.RangeReq{
		Key:   k,
		Seek:  "",
		Limit: 100, // TODO(@yeqown): allow limit variable
	})
	if err != nil {
		return nil, errors.Wrap(err, "instanceHybrid.GetElementInstances")
	}

	// TODO(@yeqown): Get instance detail information
	instances := make([]*Instance, 0, len(r.GetEntities()))
	for _, v := range r.GetEntities() {
		ins := new(Instance)
		_ = ins.Unmarshal(v.GetVal())
		ins.ClientID = runtime.ToString(v.GetVal())
		instances = append(instances, ins)
	}

	return instances, nil
}

func (i instanceHybrid) GetInstance(ctx context.Context, insId string) (*Instance, error) {
	k := genInstanceNormalKey(insId)
	r, err := i.cassemdb.GetKV(ctx, &pbcassemdb.GetKVReq{
		Key: k,
	})
	if err != nil {
		return nil, err
	}

	ins := new(Instance)
	err = ins.Unmarshal(r.GetEntity().GetVal())
	return ins, err
}

// RegisterInstance
// TODO(@yeqown): retry strategy
func (i instanceHybrid) RegisterInstance(ctx context.Context, ins *Instance) (err error) {
	insId := ins.Id()
	//ins, err := i.GetInstance(ctx, insId)
	//if err != nil {
	//	return errors.Wrap(err, "instanceHybrid.GetInstance")
	//}
	//if ins != nil {
	//	return errors.New("instance exists: " + ins.Id())
	//}

	// TODO(@yeqown): if instance is not nil, return error
	// save normalized kv
	k := genInstanceNormalKey(insId)
	log.
		WithFields(log.Fields{
			"insId":         insId,
			"normalizedKey": k,
		}).
		Debug("instanceHybrid.UnregisterInstance")

	bytes, _ := ins.Marshal()
	_, err = i.cassemdb.SetKV(ctx, &pbcassemdb.SetKVReq{
		Key:       k,
		IsDir:     false,
		Ttl:       120, // TODO(@yeqown): set a TTL
		Val:       bytes,
		Overwrite: true,
	})
	if err != nil {
		return errors.Wrap(err, "instanceHybrid.GetInstance")
	}

	// save reversed kv
	for _, key := range ins.WatchKeys {
		k2 := genInstanceReversedKeyWithInsid(ins.AppId, ins.Env, key, insId)
		_, err = i.cassemdb.SetKV(ctx, &pbcassemdb.SetKVReq{
			Key:       k2,
			IsDir:     false,
			Ttl:       120, // TODO(@yeqown) set a TTL
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
	return i.RegisterInstance(ctx, ins)
}

func (i instanceHybrid) UnregisterInstance(ctx context.Context, insId string) error {
	k := genInstanceNormalKey(insId)
	log.
		WithFields(log.Fields{
			"insId":         insId,
			"normalizedKey": k,
		}).
		Debug("instanceHybrid.UnregisterInstance")

	r, err := i.cassemdb.GetKV(ctx, &pbcassemdb.GetKVReq{
		Key: k,
	})
	// FIXME(@yeqown): if not found, just return
	if err != nil {
		return errors.Wrap(err, "instanceHybrid.UnregisterInstance")
	}

	ins := new(Instance)
	if err = ins.Unmarshal(r.GetEntity().GetVal()); err != nil {
		return errors.Wrap(err, "instanceHybrid.UnregisterInstance")
	}

	// unset normalized kv
	_, err = i.cassemdb.UnsetKV(ctx, &pbcassemdb.UnsetKVReq{
		Key:   k,
		IsDir: false,
	})

	// unset reversed kv
	for _, key := range ins.WatchKeys {
		k2 := genInstanceReversedKeyWithInsid(ins.AppId, ins.Env, key, insId)
		_, err = i.cassemdb.UnsetKV(ctx, &pbcassemdb.UnsetKVReq{
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
