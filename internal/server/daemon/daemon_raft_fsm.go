package daemon

import (
	"io"

	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
)

type fsm struct {
}

// TODO(@yeqown): this part called persistence?
func newFSM() raft.FSM {
	return fsm{}
}

func (f fsm) Apply(log *raft.Log) interface{} {
	return errors.New("not implement")
}

func (f fsm) Snapshot() (raft.FSMSnapshot, error) {
	return nil, errors.New("not implement")
}

func (f fsm) Restore(closer io.ReadCloser) error {
	return errors.New("not implement")
}
