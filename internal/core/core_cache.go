package core

import (
	"github.com/yeqown/cassem/pkg/datatypes"

	"github.com/yeqown/log"
)

func (c Core) genContainerCacheKey(ns, key string, format datatypes.ContainerFormat) string {
	return key + "#" + ns + "#" + format.String()
}

func (c Core) getContainerCache(cacheKey string) (hit bool, data []byte) {
	var err error
	data, err = c.fsm.get(cacheKey)
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
	ss := c.fsm.set(cacheKey, data)
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

	fsmLog, _ := newFsmLog(logActionSyncCache, &setCacheCommand{
		NeedSetKey:    cacheKey,
		NeedSetData:   data,
		NeedDeleteKey: ss.NeedDeleteKey,
	})

	// DONE(@yeqown): following code got error while current node is not Leader.
	// This must be run on the leader or it will fail.
	if !c.isLeader() {
		if err := c.forwardToLeaderApply(fsmLog); err != nil {
			log.
				Errorf("Core.setContainerCache forwardToLeader failed: %v", err)
		}

		return
	}

	if err := c.propagateToSlaves(fsmLog); err != nil {
		log.
			WithFields(log.Fields{
				"key":    cacheKey,
				"fsmLog": fsmLog,
			}).
			Errorf("Core.setContainerCache propagateToSlaves failed: %v", err)
	}
}

func (c Core) delContainerCache(cacheKey string) {
	ss := c.fsm.del(cacheKey)
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

	fsmLog, _ := newFsmLog(logActionSyncCache, &setCacheCommand{
		NeedSetKey:    "",
		NeedSetData:   nil,
		NeedDeleteKey: ss.NeedDeleteKey,
	})

	// DONE(@yeqown): following code got error while current node is not Leader.
	// This must be run on the leader or it will fail.
	//
	// IGNORED this part logic by @yeqown: only leader will trigger delContainerCache.
	//
	//if !c.isLeader() {
	//	if err := c.forwardToLeaderApply(fsmLog); err != nil {
	//		log.
	//			Errorf("Core.delContainerCache forwardToLeaderApply failed: %v", err)
	//	}
	//	return
	//}

	if err := c.propagateToSlaves(fsmLog); err != nil {
		log.
			WithFields(log.Fields{
				"key":    cacheKey,
				"fsmLog": fsmLog,
			}).
			Errorf("Core.delContainerCache propagateToSlaves failed: %v", err)
	}
}
