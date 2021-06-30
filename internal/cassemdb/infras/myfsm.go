package infras

import (
	"io"
	"io/ioutil"
	"sync/atomic"

	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
	"github.com/yeqown/log"

	"github.com/yeqown/cassem/pkg/watcher"
)

var _ myFSM = &fsm{}

type myFSM interface {
	raft.FSM

	getLeaderAddr() string
	setLeaderAddr(addr string)
	getExecutionSinceLastSnapshot() int
}

// fsm implement raft.FSM which means the state machine in RAFT consensus algorithm.
// Here, fsm is a state machine to support distributed cache in cassem.
type fsm struct {
	repo Repository

	// leaderAddr indicates the leader's http application address.
	leaderAddr string

	// executionSinceLastSnapshot records the count how many times has fsm.Apply been called since
	// last time fsm.Snapshot called. It helps Core.doSnapshot to judge that should Core trigger snapshot or not.
	executionSinceLastSnapshot int32

	hooks map[action]actionApplyFunc

	ch chan<- watcher.IChange
}

func newFSM(repo Repository, ch chan<- watcher.IChange) myFSM {
	return &fsm{
		repo:                       repo,
		leaderAddr:                 "",
		executionSinceLastSnapshot: 0,
		hooks: map[action]actionApplyFunc{
			actionChange:    applyActionChange,
			actionSetKV:     applyActionSetKV,
			actionSetLeader: applyActionSetLeader,
		},
		ch: ch,
	}
}

func (f *fsm) Apply(l *raft.Log) interface{} {
	var fsmlog = new(fsmLog)
	if err := fsmlog.Deserialize(l.Data); err != nil {
		panic("could not unmarshal: " + err.Error())
	}

	log.
		WithFields(log.Fields{
			"log":       string(fsmlog.Data),
			"createdAt": fsmlog.CreatedAt,
			"action":    fsmlog.Action,
		}).
		Info("fsm.Apply called")

	apply, ok := f.hooks[fsmlog.Action]
	if !ok {
		return errors.New("invalid action")
	}
	if err := apply(f, fsmlog); err != nil {
		log.
			WithFields(log.Fields{
				"log":   fsmlog,
				"error": err,
			}).
			Error("fsm.Apply failed")
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
