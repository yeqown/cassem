package coordinator

import (
	"context"
	"errors"
	"fmt"
	apikv "github.com/yeqown/cassem/api/kv"
	"time"

	"github.com/yeqown/log"

	"github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/pkg/errorx"
	"github.com/yeqown/cassem/pkg/runtime"
)

var (
	_ concept.InstanceHybrid = instanceHybrid{}
)

type instanceHybrid struct {
	cassemdb apikv.KVClient
}

// func NewInstanceHybrid(endpoints []string) (InstanceHybrid, error) {
//	cc, err := apikv.DialWithMode(endpoints, apikv.Mode_X)
//	if err != nil {
//		return nil, errors.Wrap(err, "NewInstanceHybrid")
//	}
//
//	return instanceHybrid{
//		cassemdb: apikv.NewKVClient(cc),
//	}, nil
// }

func (i instanceHybrid) GetInstances(
	ctx context.Context, seek string, limit int) (*concept.GetInstancesResult, error) {
	k := concept.GenInstanceNormalDirKey()
	log.
		WithFields(log.Fields{
			"seek":  seek,
			"limit": limit,
			"k":     k,
		}).
		Debug("instanceHybrid.GetInstances")

	r, err := i.cassemdb.Range(ctx, &apikv.RangeReq{
		Key:   k,
		Seek:  seek,
		Limit: int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("instanceHybrid.GetInstances: %w", err)
	}

	// insIds := make([]string, 0, len(r.GetEntities()))
	result := &concept.GetInstancesResult{
		CommonPager: concept.CommonPager{
			HasMore:  r.GetHasMore(),
			NextSeek: r.GetNextSeekKey(),
		},
		Instances: make([]*concept.Instance, 0, len(r.GetEntities())),
	}
	for _, v := range r.GetEntities() {
		// insId := concept.GenInstanceNormalKey(runtime.ToString(v.GetVal()))
		// insIds = append(insIds, insId)
		ins := new(concept.Instance)
		_ = concept.UnmarshalProto(v.GetVal(), ins)
		result.Instances = append(result.Instances, ins)
	}

	// // get all instance detail information.
	// r2, err2 := i.cassemdb.GetKVs(ctx, &apikv.GetKVsReq{
	//	Keys: insIds,
	// })
	// if err2 != nil {
	//	return nil, fmt.Errorf("instanceHybrid.GetInstances: %w", err)
	// }
	//
	// for _, v := range r2.GetEntities() {
	//	ins := new(concept.Instance)
	//	_ = concept.UnmarshalProto(v.GetVal(), ins)
	//	instances = append(instances, ins)
	// }

	return result, nil
}

func (i instanceHybrid) GetInstancesByElement(
	ctx context.Context, app, env, key string) (*concept.GetInstancesResult, error) {
	k := concept.GenInstanceReversedKey(app, env, key)
	log.
		WithFields(log.Fields{
			"app": app,
			"env": env,
			"key": key,
			"k":   k,
		}).
		Debug("instanceHybrid.GetInstances")

	r, err := i.cassemdb.Range(ctx, &apikv.RangeReq{
		Key:   k,
		Seek:  "",
		Limit: 100,
	})
	if err != nil {
		return nil, fmt.Errorf("instanceHybrid.GetInstances: %w", err)
	}

	result := &concept.GetInstancesResult{
		CommonPager: concept.CommonPager{
			HasMore:  r.GetHasMore(),
			NextSeek: r.GetNextSeekKey(),
		},
		Instances: make([]*concept.Instance, 0, len(r.GetEntities())),
	}
	if len(r.GetEntities()) == 0 {
		return result, nil
	}

	insIds := make([]string, 0, len(r.GetEntities()))
	for _, v := range r.GetEntities() {
		insId := concept.GenInstanceNormalKey(runtime.ToString(v.GetVal()))
		insIds = append(insIds, insId)
	}
	// get all instance detail information.
	r2, err2 := i.cassemdb.GetKVs(ctx, &apikv.GetKVsReq{
		Keys: insIds,
	})
	if err2 != nil {
		return nil, fmt.Errorf("instanceHybrid.GetInstances: %w", err2)
	}
	if err2 = getKVsNonNotFoundErrors(r2); err2 != nil {
		return nil, fmt.Errorf("instanceHybrid.GetInstances.details: %w", err2)
	}

	for _, v := range r2.GetEntities() {
		ins := new(concept.Instance)
		_ = concept.UnmarshalProto(v.GetVal(), ins)
		result.Instances = append(result.Instances, ins)
	}

	return result, nil
}

func (i instanceHybrid) GetInstance(ctx context.Context, insId string) (*concept.Instance, error) {
	k := concept.GenInstanceNormalKey(insId)
	r, err := i.cassemdb.GetKV(ctx, &apikv.GetKVReq{
		Key: k,
	})
	if err != nil {
		return nil, err
	}

	ins := new(concept.Instance)
	err = concept.UnmarshalProto(r.GetEntity().GetVal(), ins)
	return ins, err
}

// RegisterInstance registers a new instance.
// DONE(@yeqown): keep insId unique in cluster, if register duplicated just return
// duplicated error to client.
func (i instanceHybrid) RegisterInstance(ctx context.Context, ins *concept.Instance) (err error) {
	// check duplicate instance
	insId := ins.Id()
	k := concept.GenInstanceNormalKey(insId)

	r, err := i.cassemdb.GetKV(ctx, &apikv.GetKVReq{
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

func (i instanceHybrid) setInstanceInfo(ctx context.Context, ins *concept.Instance) (err error) {
	if ins == nil {
		log.
			Warn("InstanceHybrid.RegisterInstance get nil instance, skipped")
		return
	}
	insId := ins.Id()

	// save normalized kv
	k := concept.GenInstanceNormalKey(insId)
	log.
		WithFields(log.Fields{
			"insId":         insId,
			"normalizedKey": k,
		}).
		Debug("instanceHybrid.UnregisterInstance")

	bytes, err := concept.MarshalProto(ins)
	if err != nil {
		return fmt.Errorf("instanceHybrid.setInstanceInfo.marshal: %w", err)
	}
	_, err = i.cassemdb.SetKV(ctx, &apikv.SetKVReq{
		Key:       k,
		IsDir:     false,
		Ttl:       120,
		Val:       bytes,
		Overwrite: true,
	})
	if err != nil {
		return fmt.Errorf("instanceHybrid.setInstanceInfo.normalized: %w", err)
	}

	var reversedErrs []error
	for _, w := range ins.GetWatching() {
		for _, key := range w.GetWatchKeys() {
			k2 := concept.GenInstanceReversedKeyWithInsId(w.GetApp(), w.GetEnv(), key, insId)
			_, err = i.cassemdb.SetKV(ctx, &apikv.SetKVReq{
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
					Error("instanceHybrid.setInstanceInfo failed to update reversed")
				reversedErrs = append(reversedErrs, fmt.Errorf("set reversed key %s: %w", k2, err))
			}
		}
	}
	if len(reversedErrs) > 0 {
		return fmt.Errorf("instanceHybrid.setInstanceInfo reversed index errors: %w", errors.Join(reversedErrs...))
	}

	return nil
}

func (i instanceHybrid) RenewInstance(ctx context.Context, ins *concept.Instance) error {
	// check duplicate instance
	// insId := ins.Id()
	// k := concept.GenInstanceNormalKey(insId)
	// r, _ := i.cassemdb.GetKV(ctx, &apikv.GetKVReq{
	//	Key: k,
	// })
	// if r.GetEntity() != nil {
	//	if ins.LastRenewTimestamp.IsZero() {
	//		ins.LastRenewTimestamp = r.GetEntity().Get
	//	}
	// }

	return i.setInstanceInfo(ctx, ins)
}

func (i instanceHybrid) UnregisterInstance(ctx context.Context, insId string) error {
	k := concept.GenInstanceNormalKey(insId)
	log.
		WithFields(log.Fields{
			"insId":         insId,
			"normalizedKey": k,
		}).
		Debug("instanceHybrid.UnregisterInstance")

	// try to get instance detail
	r, err := i.cassemdb.GetKV(ctx, &apikv.GetKVReq{
		Key: k,
	})
	if err != nil {
		if errors.Is(err, errorx.Err_NOT_FOUND) {
			return nil
		}

		return fmt.Errorf("instanceHybrid.UnregisterInstance: %w", err)
	}

	ins := new(concept.Instance)
	if err = concept.UnmarshalProto(r.GetEntity().GetVal(), ins); err != nil {
		return fmt.Errorf("instanceHybrid.UnregisterInstance: %w", err)
	}

	// unset normalized kv
	_, err = i.cassemdb.UnsetKV(ctx, &apikv.UnsetKVReq{
		Key:   k,
		IsDir: false,
	})
	if err != nil {
		return fmt.Errorf("instanceHybrid.UnregisterInstance.normalized: %w", err)
	}

	var reversedErrs []error
	for _, w := range ins.GetWatching() {
		for _, key := range w.GetWatchKeys() {
			k2 := concept.GenInstanceReversedKeyWithInsId(w.GetApp(), w.GetEnv(), key, insId)
			_, err = i.cassemdb.UnsetKV(ctx, &apikv.UnsetKVReq{
				Key: k2,
			})
			if err != nil {
				log.
					WithFields(log.Fields{
						"key":   k2,
						"error": err,
					}).
					Error("instanceHybrid.UnregisterInstance failed to delete reversed")
				reversedErrs = append(reversedErrs, fmt.Errorf("delete reversed key %s: %w", k2, err))
			}
		}
	}
	if len(reversedErrs) > 0 {
		return fmt.Errorf("instanceHybrid.UnregisterInstance reversed index errors: %w", errors.Join(reversedErrs...))
	}

	return nil
}
