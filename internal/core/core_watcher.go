package core

import (
	"bytes"
	"encoding/json"

	"github.com/yeqown/cassem/internal/watcher"
	"github.com/yeqown/cassem/pkg/datatypes"

	"github.com/pelletier/go-toml"
	"github.com/yeqown/log"
)

func (c Core) handleChangeHooks(container datatypes.IContainer, newCheckSum string) {
	// FIXED(@yeqown): how to notify all observers on distributed nodes.
	// SOLUTION1(used): propagate changes to cluster rather than only local node?
	//            raft.Log could be executed again while node restart.
	//
	// SOLUTION2: or let all clients connect to the leader node?

	// FIXED(@yeqwon) reset cache container cache, A brute force to delete all ns+key+formats(TOML/JSON)
	v := container.ToMarshalInterface()

	// JSON format handler
	go func() {
		buf := bytes.NewBuffer(nil)
		if err := json.NewEncoder(buf).Encode(v); err != nil {
			log.
				Errorf("Core.watchContainerChanges failed to json.Encode: %v", err)

			return
		}

		fsmLog, _ := newFsmLog(logActionChangesNotify, &changesNotifyCommand{
			Changes: watcher.Changes{
				CheckSum:  newCheckSum,
				Key:       container.Key(),
				Namespace: container.NS(),
				Format:    datatypes.JSON,
				Data:      buf.Bytes(),
			},
		})
		if err := c.propagateToSlaves(fsmLog); err != nil {
			log.
				Errorf("Core.watchContainerChanges failed to propagateToSlaves: %v", err)
		}

		// DONE(@yeqown): reset cache
		c.delContainerCache(c.genContainerCacheKey(container.NS(), container.Key(), datatypes.JSON))
	}()

	// TOML format handler
	go func() {
		buf := bytes.NewBuffer(nil)
		if err := toml.NewEncoder(buf).Encode(v); err != nil {
			log.
				Errorf("Core.watchContainerChanges failed to json.Encode: %v", err)

			return
		}

		fsmLog, _ := newFsmLog(logActionChangesNotify, &changesNotifyCommand{
			Changes: watcher.Changes{
				CheckSum:  newCheckSum,
				Key:       container.Key(),
				Namespace: container.NS(),
				Format:    datatypes.TOML,
				Data:      buf.Bytes(),
			},
		})
		if err := c.propagateToSlaves(fsmLog); err != nil {
			log.
				Errorf("Core.watchContainerChanges failed to propagateToSlaves: %v", err)
		}

		c.delContainerCache(c.genContainerCacheKey(container.NS(), container.Key(), datatypes.TOML))
	}()

}
