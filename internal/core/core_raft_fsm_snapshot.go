package core

import (
	"github.com/yeqown/cassem/internal/cache"
	"github.com/yeqown/log"

	"github.com/hashicorp/raft"
)

type fsmSnapshot struct {
	containerCache cache.ICache
}

// TODO(@yeqown): this part called persistence?
func newFSMSnapshot() raft.FSMSnapshot {
	return fsmSnapshot{}
}

func (f fsmSnapshot) Persist(sink raft.SnapshotSink) (err error) {
	log.Info("Release action in fsmSnapshot")

	// TODO(@yeqown): write all cache data into sink
	//if _, err = sink.Write(); err != nil {
	//	return err
	//}

	return nil
}

func (f fsmSnapshot) Release() {
	log.Info("Release action in fsmSnapshot")
}
