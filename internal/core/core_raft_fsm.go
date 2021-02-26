package core

import (
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/yeqown/cassem/internal/cache"

	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
)

type fsm struct {
	containerCache cache.ICache
}

func newFSM(c cache.ICache) raft.FSM {
	return fsm{
		containerCache: c,
	}
}

func (f fsm) Apply(log *raft.Log) interface{} {
	var cc cacheSetCommand

	if err := json.Unmarshal(log.Data, &cc); err != nil {
		panic("could not unmarshal: " + err.Error())
	}

	if cc.NeedSetKey != "" {
		f.containerCache.Set(cc.NeedSetKey, cc.NeedSetData)
	}
	if cc.NeedDeleteKey != "" {
		_ = f.containerCache.Del(cc.NeedDeleteKey)
	}

	return nil
}

func (f fsm) Snapshot() (raft.FSMSnapshot, error) {
	return fsmSnapshot{
		containerCache: f.containerCache,
	}, nil
}

// Restore data which is produced from fsmSnapshot.Persist.
func (f fsm) Restore(closer io.ReadCloser) error {
	// "github.com/golang/protobuf/proto"
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
