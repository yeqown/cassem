package core

import (
	"github.com/yeqown/cassem/internal/myraft"
	"github.com/yeqown/cassem/pkg/datatypes"

	"github.com/yeqown/log"
)

func (c Core) genContainerCacheKey(ns, key string, format datatypes.ContainerFormat) string {
	return key + "#" + ns + "#" + format.String()
}

func (c Core) getContainerCache(cacheKey string) (hit bool, data []byte) {
	var err error
	data, err = c.raft.Get(cacheKey)
	if err != nil {
		log.
			WithField("cacheKey", cacheKey).
			Warnf("Core.getContainerCache not hit: %v", err)
		return
	}

	hit = true
	log.
		WithField("cacheKey", cacheKey).
		Debug("Core.getContainerCache hit")

	return
}

func (c Core) setContainerCache(cacheKey string, data []byte) {
	ss := c.raft.Set(cacheKey, data)
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

	fsmLog, _ := myraft.NewFsmLog(myraft.ActionSyncCache, &myraft.SetCacheCommand{
		NeedSetKey:    cacheKey,
		NeedSetData:   data,
		NeedDeleteKey: ss.NeedDeleteKey,
	})
	if err := c.raft.ApplyLog(fsmLog); err != nil {
		log.
			WithFields(log.Fields{
				"key":    cacheKey,
				"fsmLog": fsmLog,
			}).
			Errorf("Core.setContainerCache propagateToSlaves failed: %v", err)
	}
}

func (c Core) delContainerCache(cacheKey string) {
	ss := c.raft.Del(cacheKey)
	if ss.Error() != nil {
		log.
			WithField("cacheKey", cacheKey).
			Error("Core.delContainerCache could not del container cache")
		return
	}

	log.WithField("setCacheResult", ss).
		Debug("Core.delContainerCache called")
	if !ss.NeedSync {
		return
	}

	// DONE(@yeqown): should call raft to synchronous other nodes' data. apply from here.
	// means cache replacing happened
	log.
		WithFields(log.Fields{
			"key": cacheKey,
		}).
		Debug("Core.delContainerCache applyTo raft")

	fsmLog, _ := myraft.NewFsmLog(myraft.ActionSyncCache, &myraft.SetCacheCommand{
		NeedSetKey:    "",
		NeedSetData:   nil,
		NeedDeleteKey: ss.NeedDeleteKey,
	})

	if err := c.raft.ApplyLog(fsmLog); err != nil {
		log.
			WithFields(log.Fields{
				"key":    cacheKey,
				"fsmLog": fsmLog,
			}).
			Errorf("Core.delContainerCache propagateToSlaves failed: %v", err)
	}
}
