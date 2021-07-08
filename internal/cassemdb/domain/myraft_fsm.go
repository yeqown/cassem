package domain

import (
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/yeqown/log"

	"github.com/yeqown/cassem/internal/cassemdb/infras/repository"
	"github.com/yeqown/cassem/pkg/runtime"
	"github.com/yeqown/cassem/pkg/types"
)

// SetKV set a KV or directory into db storage with other parameters.
// isDir parameter indicates key means a kv or directory, if it's ture val will be ignored,
// overwrite indicates the operation MUST BE failed if key exists with repository.ErrExists,
// ttl means Time To Live, which will only be stored in file and recalculated in memory to use.
func (r *myraft) SetKV(key string, val []byte, isDir, overwrite bool, ttl uint32) (err error) {
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
	last, err := r.repo.GetKV(types.StoreKey(key), isDir)
	if err != nil {
		log.
			WithFields(log.Fields{
				"key":   key,
				"error": err,
			}).
			Warn("myraft.SetKV could to load last value of key")
	}

	// remove expired value automatically.
	if r.removeExpiredValue(last) {
		last = nil
	}

	if !overwrite && last != nil {
		return repository.ErrExists
	}

	var createdAt = time.Now().Unix()
	if last != nil && !last.Expired() {
		createdAt = last.CreatedAt
	}
	k, v := types.NewKVWithCreatedAt(key, val, ttl, createdAt)
	if err := r.propagateCommand(&setKVCommand{
		SetKey: k,
		Data:   &v,
	}); err != nil {
		return errors.Wrap(err, "myraft.SetKV calling myraft.propagateCommand failed")
	}

	// touch off change signal to cassemdb cluster.
	r.triggerWatchingMechanism(types.OpSet, key, last, &v)

	return nil
}

func (r *myraft) UnsetKV(key string, isDir bool) error {
	last, err := r.repo.GetKV(types.StoreKey(key), isDir)
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
		DeleteKey: types.StoreKey(key),
		IsDir:     isDir,
		Data:      nil,
	}); err != nil {
		return errors.Wrap(err, "myraft.SetKV calling myraft.propagateCommand failed")
	}

	// touch off change signal to cassemdb cluster.
	r.triggerWatchingMechanism(types.OpUnset, key, last, nil)

	return nil
}

// triggerWatchingMechanism only trigger a change notification while:
// 1. delete a kv.
// 2. really update a existed kv.
//
func (r myraft) triggerWatchingMechanism(op types.ChangeOp, key string, last, newVal *types.StoreValue) {
	if last == nil || last.Expired() {
		// last == nil means that the key is new, there's no observer;
		return
	}

	if newVal != nil && strings.Compare(last.Fingerprint, newVal.Fingerprint) == 0 {
		// set kv but newVal is same to old value, so no need to touch off a change notification.
		return
	}

	go func() {
		log.
			WithFields(log.Fields{"key": key, "newVal": newVal}).
			Debug("myraft.triggerWatchingMechanism called")

		if err := r.propagateCommand(&changeCommand{
			Change: &types.Change{
				Op:      op,
				Key:     types.StoreKey(key),
				Last:    last,
				Current: newVal,
			}}); err != nil {

			log.
				WithFields(log.Fields{"key": key, "newVal": newVal}).
				Error("myraft.triggerWatchingMechanism called")
		}
	}()
}

func (r *myraft) GetKV(key string) (*types.StoreValue, error) {
	val, err := r.repo.GetKV(types.StoreKey(key), false)
	if err != nil {
		log.
			WithFields(log.Fields{
				"key":   key,
				"error": err,
			}).
			Error("repo.getKV failed")
		return nil, err
	}

	if r.removeExpiredValue(val) {
		return nil, repository.ErrNotFound
	}

	val.TTL = val.RecalculateTTL()

	return val, nil
}

// removeExpiredValue returns true while val.Expired() is true.
func (r *myraft) removeExpiredValue(val *types.StoreValue) bool {
	if val == nil {
		return false
	}

	if val.Expired() {
		if err := r.UnsetKV(val.Key.String(), false); err != nil {
			log.
				WithFields(log.Fields{"key": val.Key.String(), "error": err}).
				Error("repo.GetKV failed to remove expired key")
		}
		return true
	}

	return false
}
