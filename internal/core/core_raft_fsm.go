package core

import (
	"io"
	"io/ioutil"
	"sync/atomic"

	"github.com/yeqown/cassem/internal/cache"

	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
	"github.com/yeqown/log"
)

// FSMWrapper contains raft.FSM and customized methods to help core.Core use raft distributed power, as for now,
// it includes, leaderAddr which represents the address of accesmd's http server.
type FSMWrapper interface {
	raft.FSM

	SetLeaderAddr(addr string)

	LeaderAddr() string

	ExecutionSinceLastSnapshot() int
}

// fsm implement raft.FSM which means the state machine in RAFT consensus algorithm.
// Here, fsm is a state machine to support distributed cache in cassem.
type fsm struct {
	containerCache cache.ICache

	// leaderAddr indicates the leader's http application address.
	leaderAddr string

	executionSinceLastSnapshot int32
}

func newFSM(c cache.ICache) FSMWrapper {
	return &fsm{
		containerCache:             c,
		executionSinceLastSnapshot: 0,
	}
}

func (f *fsm) Apply(l *raft.Log) interface{} {
	var fsmLog = new(coreFSMLog)
	if err := fsmLog.deserialize(l.Data); err != nil {
		panic("could not unmarshal: " + err.Error())
	}

	log.
		WithField("fmsLogData", string(fsmLog.Data)).
		Debug("fsm.Apply called")

	switch fsmLog.Action {
	case logActionSyncCache:
		cc := new(coreSetCache)
		if err := cc.deserialize(fsmLog.Data); err != nil {
			panic("could not unmarshal: " + err.Error())
		}

		if cc.NeedSetKey != "" {
			_ = f.containerCache.Set(cc.NeedSetKey, cc.NeedSetData)
		}
		if cc.NeedDeleteKey != "" {
			_ = f.containerCache.Del(cc.NeedDeleteKey)
		}

	case logActionSetLeaderAddr:
		cc := new(setLeaderAddr)
		if err := cc.deserialize(fsmLog.Data); err != nil {
			panic("could not unmarshal: " + err.Error())
		}

		f.SetLeaderAddr(cc.LeaderAddr)

	default:
		return errors.New("invalid action")
	}

	atomic.AddInt32(&(f.executionSinceLastSnapshot), 1)

	return nil
}

// DONE(@yeqown): figure out a way to use snapshot rather than Apply each log while node restart.
// checkout Core.doSnapshot
func (f *fsm) Snapshot() (raft.FSMSnapshot, error) {
	log.Debug("fsm.Snapshot called")
	data, err := f.containerCache.Persist()
	if err != nil {
		return nil, errors.Wrap(err, "fsm.Snapshot calling ICache.Persist() failed")
	}

	log.
		WithField("data", string(data)).
		Debug("fsm.Snapshot got persistence")

	atomic.SwapInt32(&(f.executionSinceLastSnapshot), 0)

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
		return errors.Wrapf(err, "fsm.Restore failed to read from io.ReadCloser")
	}

	log.
		WithField("data", string(buf)).
		Debug("fsm.Restore got persistence")

	if err = f.containerCache.Restore(buf); err != nil {
		return errors.Wrapf(err, "fsm.Restore could not restore into data")
	}

	return nil
}

func (f *fsm) SetLeaderAddr(addr string) {
	f.leaderAddr = addr
}

func (f *fsm) LeaderAddr() string {
	return f.leaderAddr
}

func (f fsm) ExecutionSinceLastSnapshot() int {
	//log.
	//	WithField("executionSinceLastSnapshot", f.executionSinceLastSnapshot).
	//	Debug("fsm.ExecutionSinceLastSnapshot called")

	return int(f.executionSinceLastSnapshot)
}
