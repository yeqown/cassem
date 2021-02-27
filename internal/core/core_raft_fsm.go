package core

import (
	"io"
	"io/ioutil"

	"github.com/yeqown/cassem/internal/cache"

	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
	"github.com/yeqown/log"
)

type fsm struct {
	containerCache cache.ICache

	// leaderAddr indicates the leader's http application address.
	leaderAddr string
}

func newFSM(c cache.ICache) raft.FSM {
	return &fsm{
		containerCache: c,
	}
}

func (f *fsm) Apply(l *raft.Log) interface{} {
	var fsmLog = new(coreFSMLog)
	if err := fsmLog.deserialize(l.Data); err != nil {
		panic("could not unmarshal: " + err.Error())
	}

	switch fsmLog.Action {
	case logActionSyncCache:
		cc := new(coreSetCache)
		if err := cc.deserialize(fsmLog.Data); err != nil {
			panic("could not unmarshal: " + err.Error())
		}

		log.
			WithField("coreSetCache", cc).
			Debug("fsm.Apply called")
		if cc.NeedSetKey != "" {
			f.containerCache.Set(cc.NeedSetKey, cc.NeedSetData)
		}
		if cc.NeedDeleteKey != "" {
			_ = f.containerCache.Del(cc.NeedDeleteKey)
		}

	case logActionSetLeaderAddr:
		cc := new(setLeaderAddr)
		if err := cc.deserialize(fsmLog.Data); err != nil {
			panic("could not unmarshal: " + err.Error())
		}

		log.
			WithField("setLeaderAddr", cc).
			Debug("fsm.Apply called")
		f.leaderAddr = cc.LeaderAddr

	default:
		return errors.New("invalid action")
	}

	return nil
}

func (f fsm) Snapshot() (raft.FSMSnapshot, error) {
	log.Debug("fsm.Snapshot called")
	data, err := f.containerCache.Persist()
	if err != nil {
		return nil, errors.Wrap(err, "fs.containerCache.Persist() failed")
	}

	return fsmSnapshot{
		serialized: data,
	}, nil
}

// Restore data which is produced from fsmSnapshot.Persist.
func (f fsm) Restore(closer io.ReadCloser) error {
	log.Debug("fsm.Restore called")
	defer closer.Close()

	buf, err := ioutil.ReadAll(closer)
	if err != nil {
		return errors.Wrapf(err, "could not read from reader")
	}

	if err = f.containerCache.Restore(buf); err != nil {
		return errors.Wrapf(err, "could not restore into data")
	}

	return nil
}
