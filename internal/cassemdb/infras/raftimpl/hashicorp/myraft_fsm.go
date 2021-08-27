package hashicorp

import (
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/yeqown/log"

	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
	"github.com/yeqown/cassem/internal/cassemdb/infras/storage"
	"github.com/yeqown/cassem/pkg/errorx"
	"github.com/yeqown/cassem/pkg/runtime"
)

// SetKV set a KV or directory into db storage with other parameters.
// isDir parameter indicates key means a kv or directory, if it's ture val will be ignored,
// overwrite indicates the operation MUST BE failed if key exists with storage.ErrExists,
// ttl means Time To Live, which will only be stored in file and recalculated in memory to use.
func (r *myraft) SetKV(req *apicassemdb.SetKVReq) (err error) {
	log.
		WithFields(log.Fields{
			"key":       req.GetKey(),
			"val":       runtime.ToString(req.GetVal()),
			"isDir":     req.GetIsDir(),
			"overwrite": req.GetOverwrite(),
			"ttl":       req.GetTtl(),
		}).
		Debug("myraft.setKV called")

	// get preview value
	last, err := r.repo.GetKV(req.GetKey(), req.GetIsDir())
	if err != nil {
		log.
			WithFields(log.Fields{
				"key":   req.GetKey(),
				"error": err,
			}).
			Warn("myraft.SetKV could to load last value of key")
	}

	// remove expired value automatically.
	if r.probeRemoveExpired(last) {
		last = nil
	}

	if !req.GetOverwrite() && last != nil {
		return storage.ErrExists
	}

	var createdAt = time.Now().Unix()
	if last != nil && !last.Expired() {
		createdAt = last.CreatedAt
	}

	v := apicassemdb.NewEntityWithCreated(req.GetKey(), req.GetVal(), req.GetTtl(), createdAt)
	if err = r.propagateCommand(&setKVCommand{
		SetKey: req.GetKey(),
		Data:   v,
	}); err != nil {
		return errors.Wrap(err, "myraft.SetKV calling myraft.propagateCommand failed")
	}

	// touch off change signal to cassemdb cluster.
	r.triggerWatchingMechanism(apicassemdb.Change_Set, req.GetKey(), last, v)

	return nil
}

func (r *myraft) UnsetKV(req *apicassemdb.UnsetKVReq) error {
	last, err := r.repo.GetKV(req.GetKey(), req.GetIsDir())
	if err != nil {
		log.
			WithFields(log.Fields{
				"key":   req.GetKey(),
				"error": err,
			}).
			Warn("myraft.triggerWatchingMechanism could to load last value of key")
	}

	if err = r.propagateCommand(&setKVCommand{
		SetKey:    "",
		DeleteKey: req.GetKey(),
		IsDir:     req.GetIsDir(),
		Data:      nil,
	}); err != nil {
		return errors.Wrap(err, "myraft.SetKV calling myraft.propagateCommand failed")
	}

	// touch off change signal to cassemdb cluster.
	r.triggerWatchingMechanism(apicassemdb.Change_Unset, req.GetKey(), last, nil)

	return nil
}

// triggerWatchingMechanism only trigger a change notification while:
// 1. delete a kv.
// 2. really update an existed kv.
//
func (r myraft) triggerWatchingMechanism(op apicassemdb.Change_Op, key string, last, cur *apicassemdb.Entity) {
	log.
		WithFields(log.Fields{
			"key": key,
			"op":  op,
		}).
		Debug("myraft.triggerWatchingMechanism called")

	// FIXED(@yeqown): new value should also notify watchers.
	//if last == nil || last.Expired() {
	//	// last == nil means that the key is new, there's no observer;
	//	return
	//}

	if last != nil && cur != nil && strings.Compare(last.Fingerprint, cur.Fingerprint) == 0 {
		// set kv but cur is same to old value, so no need to touch off a change notification.
		return
	}

	go func() {
		log.
			WithFields(log.Fields{"key": key, "cur": cur}).
			Debug("myraft.triggerWatchingMechanism called")

		if err := r.propagateCommand(&changeCommand{
			Change: &apicassemdb.Change{
				Op:      op,
				Key:     key,
				Last:    last,
				Current: cur,
			}}); err != nil {

			log.
				WithFields(log.Fields{
					"key": key,
					"cur": cur,
				}).
				Error("myraft.triggerWatchingMechanism called")
		}
	}()
}

//func convertStoreValue(v *apicassemdb.Entity) *apicassemdb.Entity {
//	if v == nil {
//		return nil
//	}
//
//	return &apicassemdb.Entity{
//		Fingerprint: v.Fingerprint,
//		Key:         v.Key,
//		Val:         v.Val,
//		CreatedAt:   v.CreatedAt,
//		UpdatedAt:   v.UpdatedAt,
//		Ttl:         v.Ttl,
//		Typ:         v.Type(),
//	}
//}

func (r *myraft) GetKV(req *apicassemdb.GetKVReq) (*apicassemdb.Entity, error) {
	val, err := r.repo.GetKV(req.GetKey(), false)
	if err != nil {
		log.
			WithFields(log.Fields{
				"key":   req.GetKey(),
				"error": err,
			}).
			Error("repo.getKV failed")
		return nil, err
	}

	if r.probeRemoveExpired(val) {
		return nil, storage.ErrNotFound
	}

	return val, nil
}

// probeRemoveExpired returns true while val.Expired() is true.
func (r *myraft) probeRemoveExpired(val *apicassemdb.Entity) (removed bool) {
	if val == nil {
		return false
	}

	if val.Expired() {
		if err := r.UnsetKV(&apicassemdb.UnsetKVReq{Key: val.Key}); err != nil {
			log.
				WithFields(log.Fields{"key": val.Key, "error": err}).
				Error("repo.GetKV failed to remove expired key")
		}
		return true
	}

	return false
}

func (r myraft) Range(req *apicassemdb.RangeReq) (*apicassemdb.RangeResp, error) {
	// DONE(@yeqown): return expired keys and trigger probeRemoveExpired methods
	result, err := r.repo.Range(req.GetKey(), req.GetSeek(), int(req.GetLimit()))
	if err != nil {
		return nil, errors.Wrap(err, "myraft.Range")
	}

	if len(result.ExpiredKeys) != 0 {
		// DONE(@yeqown): delete the expired keys while got expired keys.
		go func() {
			log.
				WithFields(log.Fields{
					"keys": result.ExpiredKeys,
				}).
				Debug("myraft.Range trigger remove expired keys")

			for _, k := range result.ExpiredKeys {
				_ = r.UnsetKV(&apicassemdb.UnsetKVReq{Key: k})
			}
		}()
	}

	resp := &apicassemdb.RangeResp{
		Entities:    result.Items,
		HasMore:     result.HasMore,
		NextSeekKey: result.NextSeekKey,
	}

	return resp, err
}

func (r *myraft) Expire(req *apicassemdb.ExpireReq) error {
	v, err := r.repo.GetKV(req.GetKey(), false)
	if err != nil {
		if errors.Is(err, errorx.Err_NOT_FOUND) {
			return nil
		}

		return errors.Wrap(err, "cassemdb.myraft.Expire")
	}

	switch v.GetTtl() {
	case apicassemdb.NEVER_EXPIRED:
		return nil
	}

	// unset the key value directly or update it's TTL, choose update it's TTL
	// so that the expiry(expire) operation is same to method's meaning.
	return r.SetKV(&apicassemdb.SetKVReq{
		Key:       req.GetKey(),
		IsDir:     false,
		Ttl:       apicassemdb.EXPIRED,
		Val:       v.Val,
		Overwrite: true,
	})
}
