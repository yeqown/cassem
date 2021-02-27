package core

import (
	"encoding/json"
	"time"

	"github.com/yeqown/cassem/internal/cache"
	"github.com/yeqown/log"
)

func (c Core) getContainerCache(cacheKey string) (hit bool, data []byte) {
	var err error
	data, err = c._containerCache.Get(cacheKey)
	switch err {
	case nil:
		hit = true
		log.
			WithField("cacheKey", cacheKey).
			Debug("Core.getContainerCache hit")
	case cache.ErrMiss:
		log.
			WithField("cacheKey", cacheKey).
			Warn("Core.getContainerCache missed")
	default:
		log.
			WithField("cacheKey", cacheKey).
			Warnf("Core.getContainerCache failed: %v", err)
	}

	return
}

func (c Core) setContainerCache(cacheKey string, data []byte) {
	ss := c._containerCache.Set(cacheKey, data)
	if ss.Error() != nil {
		log.
			WithField("cacheKey", cacheKey).
			Error("Core.setContainerCache could not set container cache")
		return
	}

	log.WithField("setCacheResult", ss).
		Debug("Core.setContainerCache called")
	if !ss.NeedSync {
		return
	}

	// DONE(@yeqown): should call raft to synchronous other nodes' data. apply from here.
	// means cache replacing happened
	log.
		WithFields(log.Fields{
			"key": cacheKey,
		}).
		Debug("Core.setContainerCache applyTo raft")

	msg, _ := (cacheSetCommand{
		NeedSetKey:    cacheKey,
		NeedSetData:   data,
		NeedDeleteKey: ss.NeedDeleteKey,
	}).serialize()

	// FIXME(@yeqown): following code got error while current node is not Leader.
	// This must be run on the leader or it will fail.
	if f := c.raft.Apply(msg, 10*time.Second); f.Error() != nil {
		log.
			WithFields(log.Fields{
				"key": cacheKey,
				"msg": msg,
			}).
			Errorf("Core.setContainerCache applyTo raft failed: %v", f.Error())
	}

}

//type _action uint8
//
//const (
//	ccActionSet _action = iota + 1
//	ccActionDel
//)

type cacheSetCommand struct {
	NeedDeleteKey string
	NeedSetKey    string
	NeedSetData   []byte
}

func (cc cacheSetCommand) serialize() ([]byte, error) {
	return json.Marshal(cc)
}

func (cc *cacheSetCommand) deserialize(data []byte) error {
	return json.Unmarshal(data, cc)
}
