package myraft

import (
	"io"
	"io/ioutil"
	"sync/atomic"
	"time"

	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
	"github.com/yeqown/log"

	persistence2 "github.com/yeqown/cassem/apps/cassemdb/persistence"
)

var _ myFSM = &fsm{}

type myFSM interface {
	raft.FSM

	getLeaderAddr() string
	setLeaderAddr(addr string)
}

// fsm implement raft.FSM which means the state machine in RAFT consensus algorithm.
// Here, fsm is a state machine to support distributed cache in cassem.
type fsm struct {
	repo persistence2.Repository

	// leaderAddr indicates the leader's http application address.
	leaderAddr string

	// executionSinceLastSnapshot records the count how many times has fsm.Apply been called since
	// last time fsm.Snapshot called. It helps Core.doSnapshot to judge that should Core trigger snapshot or not.
	executionSinceLastSnapshot int32
}

func newFSM(repo persistence2.Repository) myFSM {
	return &fsm{
		repo:                       repo,
		executionSinceLastSnapshot: 0,
	}
}

func (f *fsm) Apply(l *raft.Log) interface{} {
	var fsmLog = new(CoreFSMLog)
	if err := fsmLog.Deserialize(l.Data); err != nil {
		panic("could not unmarshal: " + err.Error())
	}

	log.
		WithFields(log.Fields{
			"fmsLogData": string(fsmLog.Data),
			"createdAt":  fsmLog.CreatedAt,
			"action":     fsmLog.Action,
		}).
		Debug("fsm.Apply called")

	switch fsmLog.Action {
	case ActionSet:
		cc := new(SetCommand)
		if err := cc.Deserialize(fsmLog.Data); err != nil {
			panic("could not unmarshal: " + err.Error())
		}

		if cc.SetKey != "" {
			_ = f.repo.Set(cc.SetKey, cc.NeedSetData)
		}
		if cc.DeleteKey != "" {
			_ = f.repo.Unset(cc.DeleteKey)
		}

	case ActionSetLeaderAddr:
		if time.Now().Unix()-fsmLog.CreatedAt > 10 {
			return nil
		}

		cc := new(SetLeaderAddrCommand)
		if err := cc.Deserialize(fsmLog.Data); err != nil {
			panic("could not unmarshal: " + err.Error())
		}

		f.setLeaderAddr(cc.LeaderAddr)
	//
	//case ActionUnset:
	//	if time.Now().Unix()-fsmLog.CreatedAt > 10 {
	//		return nil
	//	}
	//
	//	cc := new(ChangesNotifyCommand)
	//	if err := cc.Deserialize(fsmLog.Data); err != nil {
	//		panic("could not unmarshal: " + err.Error())
	//	}
	//
	//	// send signal with nonblocking case.
	//	select {
	//	case f.changesCh <- cc.Changes:
	//		log.Debug("send to channel")
	//	default:
	//		log.Debug("send to channel default case")
	//	}

	default:
		return errors.New("invalid action")
	}

	atomic.AddInt32(&(f.executionSinceLastSnapshot), 1)

	return nil
}

// Snapshot
// DONE(@yeqown): figure out a way to use snapshot rather than Apply each log while node restart.
// checkout Core.doSnapshot
func (f *fsm) Snapshot() (raft.FSMSnapshot, error) {
	log.Debug("fsm.Snapshot called")
	//data, err := f.containerCache.Persist()
	//if err != nil {
	//	return nil, errors.Wrap(err, "fsm.Snapshot calling ICache.Persist() failed")
	//}
	//
	//log.
	//	WithField("data", string(data)).
	//	Debug("fsm.Snapshot got persistence")

	atomic.SwapInt32(&(f.executionSinceLastSnapshot), 0)

	return fsmSnapshot{
		serialized: nil,
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

	//if err = f.containerCache.Restore(buf); err != nil {
	//	return errors.Wrapf(err, "fsm.Restore could not restore into data")
	//}

	return nil
}

func (f *fsm) setLeaderAddr(addr string) {
	f.leaderAddr = addr
}

func (f *fsm) getLeaderAddr() string {
	return f.leaderAddr
}

func (f fsm) getExecutionSinceLastSnapshot() int {
	//log.
	//	WithField("getExecutionSinceLastSnapshot", f.getExecutionSinceLastSnapshot).
	//	Debug("fsm.getExecutionSinceLastSnapshot called")

	return int(f.executionSinceLastSnapshot)
}
