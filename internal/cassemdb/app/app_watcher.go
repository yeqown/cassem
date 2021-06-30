package app

import (
	"github.com/yeqown/cassem/pkg/hash"
	"github.com/yeqown/cassem/pkg/set"
	"github.com/yeqown/cassem/pkg/watcher"
)

//
//import (
//	"bytes"
//	"encoding/json"
//
//	"github.com/yeqown/cassem/pkg/datatypes"
//
//	"github.com/yeqown/cassem/internal/myraft"
//	"github.com/yeqown/cassem/internal/watcher"
//
//	"github.com/pelletier/go-toml"
//	"github.com/yeqown/log"
//)
//
//func (c app) handleChangeHooks(container datatypes.IContainer, newCheckSum string) {
//	// FIXED(@yeqown): how to notify all observers on distributed nodes.
//	// SOLUTION1(used): propagate changes to cluster rather than only local node?
//	//            raft.Log could be executed again while node restart.
//	//
//	// SOLUTION2: or let all clients connect to the leader node?
//
//	// FIXED(@yeqwon) reset cache container cache, A brute force to delete all ns+key+formats(TOML/JSON)
//	v := container.ToMarshalInterface()
//
//	// JSON format handler
//	go func() {
//		buf := bytes.NewBuffer(nil)
//		if err := json.NewEncoder(buf).Encode(v); err != nil {
//			log.
//				Errorf("app.watchContainerChanges failed to json.Encode: %v", err)
//
//			return
//		}
//
//		fsmLog, _ := myraft.NewFsmLog(myraft.ActionChangesNotify, &myraft.ChangesNotifyCommand{
//			Changes: watcher.Changes{
//				CheckSum:  newCheckSum,
//				Key:       container.Key(),
//				Key: container.NS(),
//				Format:    datatypes.JSON,
//				Data:      buf.Bytes(),
//			},
//		})
//		if err := c.raft.applyLog(fsmLog); err != nil {
//			log.
//				Errorf("app.watchContainerChanges failed to propagateToSlaves: %v", err)
//		}
//
//		// DONE(@yeqown): reset cache
//		c.delContainerCache(c.genContainerCacheKey(container.NS(), container.Key(), datatypes.JSON))
//	}()
//
//	// TOML format handler
//	go func() {
//		buf := bytes.NewBuffer(nil)
//		if err := toml.NewEncoder(buf).Encode(v); err != nil {
//			log.
//				Errorf("app.watchContainerChanges failed to json.Encode: %v", err)
//
//			return
//		}
//
//		fsmLog, _ := myraft.NewFsmLog(myraft.ActionChangesNotify, &myraft.ChangesNotifyCommand{
//			Changes: watcher.Changes{
//				CheckSum:  newCheckSum,
//				Key:       container.Key(),
//				Key: container.NS(),
//				Format:    datatypes.TOML,
//				Data:      buf.Bytes(),
//			},
//		})
//		if err := c.raft.applyLog(fsmLog); err != nil {
//			log.
//				Errorf("app.watchContainerChanges failed to propagateToSlaves: %v", err)
//		}
//
//		c.delContainerCache(c.genContainerCacheKey(container.NS(), container.Key(), datatypes.TOML))
//	}()
//
//}

type builtinObserver struct {
	id    string
	keys  []string
	ch    chan watcher.IChange
	close func()
}

// NewTopicObserver channel and key of subscriber holds
func NewTopicObserver(changesCh chan watcher.IChange, close func(), keys []string) *builtinObserver {
	ob := builtinObserver{
		id:    hash.RandKey(8),
		keys:  keys,
		ch:    changesCh,
		close: close,
	}

	return &ob
}

func (t *builtinObserver) Identity() string                { return t.id }
func (t builtinObserver) Inbound() chan<- watcher.IChange  { return t.ch }
func (t builtinObserver) Outbound() <-chan watcher.IChange { return t.ch }
func (t builtinObserver) Close()                           { t.close() }
func (t builtinObserver) Topics() []string {
	s := set.NewStringSet(len(t.keys))

	for _, key := range t.keys {
		s.Add(key)
	}

	return s.Keys()
}
