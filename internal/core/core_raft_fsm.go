package core

import (
	"io"

	"github.com/yeqown/cassem/internal/cache"

	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
)

type fsm struct {
	containerCache cache.ICache
}

// TODO(@yeqown): this part called persistence?
func newFSM() raft.FSM {
	return fsm{}
}

func (f fsm) Apply(log *raft.Log) interface{} {
	//var c command
	//if err := json.Unmarshal(l.Data, &c); err != nil {
	//	panic("failed to unmarshal raft log")
	//}
	//
	//switch strings.ToLower(c.Op) {
	//case "set":
	//	return f.applySet(c.Key, c.Value)
	//case "delete":
	//	return f.applyDelete(c.Key)
	//default:
	//	panic("command type not support")
	//}

	return errors.New("not implement")
}

func (f fsm) Snapshot() (raft.FSMSnapshot, error) {
	return fsmSnapshot{
		containerCache: f.containerCache,
	}, nil
}

func (f fsm) Restore(closer io.ReadCloser) error {
	return errors.New("not implement")
}
