package domain

import (
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/yeqown/log"

	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
	"github.com/yeqown/cassem/internal/cassemdb/infras/repository"
	"github.com/yeqown/cassem/pkg/errorx"
	"github.com/yeqown/cassem/pkg/runtime"
)

// SetKV set a KV or directory into db storage with other parameters.
// isDir parameter indicates key means a kv or directory, if it's ture val will be ignored,
// overwrite indicates the operation MUST BE failed if key exists with repository.ErrExists,
// ttl means Time To Live, which will only be stored in file and recalculated in memory to use.
func (r *myraft) SetKV(key string, val []byte, isDir, overwrite bool, ttl int32) (err error) {
	log.
		WithFields(log.Fields{
			"key":       key,
			"val":       runtime.ToString(val),
			"isDir":     isDir,
			"overwrite": overwrite,
			"ttl":       ttl,
		}).
		Debug("myraft.setKV called")

	// get preview value
	last, err := r.repo.GetKV(repository.StoreKey(key), isDir)
	if err != nil {
		log.
			WithFields(log.Fields{
				"key":   key,
				"error": err,
			}).
			Warn("myraft.SetKV could to load last value of key")
	}

	// remove expired value automatically.
	if r.probeRemoveExpired(last) {
		last = nil
	}

	if !overwrite && last != nil {
		return repository.ErrExists
	}

	var createdAt = time.Now().Unix()
	if last != nil && !last.Expired() {
		createdAt = last.CreatedAt
	}
	k, v := repository.NewKVWithCreatedAt(key, val, ttl, createdAt)
	if err = r.propagateCommand(&setKVCommand{
		SetKey: k,
		Data:   &v,
	}); err != nil {
		return errors.Wrap(err, "myraft.SetKV calling myraft.propagateCommand failed")
	}

	// touch off change signal to cassemdb cluster.
	r.triggerWatchingMechanism(apicassemdb.Change_Set, key, last, &v)

	return nil
}

func (r *myraft) UnsetKV(key string, isDir bool) error {
	last, err := r.repo.GetKV(repository.StoreKey(key), isDir)
	if err != nil {
		log.
			WithFields(log.Fields{
				"key":   key,
				"error": err,
			}).
			Warn("myraft.triggerWatchingMechanism could to load last value of key")
	}

	if err = r.propagateCommand(&setKVCommand{
		SetKey:    "",
		DeleteKey: repository.StoreKey(key),
		IsDir:     isDir,
		Data:      nil,
	}); err != nil {
		return errors.Wrap(err, "myraft.SetKV calling myraft.propagateCommand failed")
	}

	// touch off change signal to cassemdb cluster.
	r.triggerWatchingMechanism(apicassemdb.Change_Unset, key, last, nil)

	return nil
}

// triggerWatchingMechanism only trigger a change notification while:
// 1. delete a kv.
// 2. really update an existed kv.
//
func (r myraft) triggerWatchingMechanism(op apicassemdb.Change_Op, key string, last, cur *repository.StoreValue) {
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
				Last:    convertStoreValue(last),
				Current: convertStoreValue(cur),
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

func convertStoreValue(v *repository.StoreValue) *apicassemdb.Entity {
	if v == nil {
		return nil
	}

	return &apicassemdb.Entity{
		Fingerprint: v.Fingerprint,
		Key:         v.Key,
		Val:         v.Val,
		CreatedAt:   v.CreatedAt,
		UpdatedAt:   v.UpdatedAt,
		Ttl:         v.TTL,
		Typ:         v.Type(),
	}
}

func (r *myraft) GetKV(key string) (*apicassemdb.Entity, error) {
	val, err := r.repo.GetKV(repository.StoreKey(key), false)
	if err != nil {
		log.
			WithFields(log.Fields{
				"key":   key,
				"error": err,
			}).
			Error("repo.getKV failed")
		return nil, err
	}

	if r.probeRemoveExpired(val) {
		return nil, repository.ErrNotFound
	}

	return convertStoreValue(val), nil
}

// probeRemoveExpired returns true while val.Expired() is true.
func (r *myraft) probeRemoveExpired(val *repository.StoreValue) (removed bool) {
	if val == nil {
		return false
	}

	if val.Expired() {
		if err := r.UnsetKV(val.Key, false); err != nil {
			log.
				WithFields(log.Fields{"key": val.Key, "error": err}).
				Error("repo.GetKV failed to remove expired key")
		}
		return true
	}

	return false
}

func (r myraft) Range(key, seek string, limit int) (*apicassemdb.RangeResp, error) {
	// DONE(@yeqown): return expired keys and trigger probeRemoveExpired methods
	result, err := r.repo.Range(repository.StoreKey(key), seek, limit)
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
				_ = r.UnsetKV(k, false)
			}
		}()
	}

	resp := &apicassemdb.RangeResp{
		Entities:    make([]*apicassemdb.Entity, 0, len(result.Items)),
		HasMore:     result.HasMore,
		NextSeekKey: result.NextSeekKey,
	}

	for _, v := range result.Items {
		resp.Entities = append(resp.Entities, convertStoreValue(v))
	}

	return resp, err
}

func (r *myraft) Expire(key string) error {
	v, err := r.repo.GetKV(key, false)
	if err != nil {
		if errors.Is(err, errorx.Err_NOT_FOUND) {
			return nil
		}

		return errors.Wrap(err, "cassemdb.myraft.Expire")
	}

	switch v.TTL {
	case repository.NEVER_EXPIRED:
		return nil
	}

	// unset the key value directly or update it's TTL, choose update it's TTL
	// so that the expiry(expire) operation is same to method's meaning.
	return r.SetKV(key, v.Val, false, true, repository.EXPIRED)
}
