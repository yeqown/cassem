package domain

import (
	"io"
	"io/ioutil"

	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
	"github.com/yeqown/log"

	"github.com/yeqown/cassem/internal/cassemdb/infras/repository"
	"github.com/yeqown/cassem/pkg/watcher"
)

var _ myFSM = &fsm{}

type myFSM interface {
	raft.FSM
}

// fsm implement raft.FSM which means the state machine in RAFT consensus algorithm.
type fsm struct {
	repo  repository.KV
	hooks map[action]actionApplyFunc
	ch    chan<- watcher.IChange
}

func newFSM(repo repository.KV, ch chan<- watcher.IChange) myFSM {
	return &fsm{
		repo: repo,
		hooks: map[action]actionApplyFunc{
			actionChange: applyActionChange,
			actionSetKV:  applyActionSetKV,
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
		log.WithFields(log.Fields{"log": fsmlog, "error": err}).
			Error("fsm.Apply failed")
	}

	return nil
}

func (f *fsm) Snapshot() (raft.FSMSnapshot, error) {
	log.Debug("fsm.Snapshot called")

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

	log.WithField("data", string(buf)).
		Debug("fsm.Restore got persistence")

	return nil
}
