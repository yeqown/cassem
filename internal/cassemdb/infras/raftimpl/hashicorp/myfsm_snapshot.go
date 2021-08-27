package hashicorp

import (
	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
)

type fsmSnapshot struct {
	serialized []byte
}

func (fs fsmSnapshot) Persist(sink raft.SnapshotSink) (err error) {
	// log.Info("fsmSnapshot.Persist called")
	if _, err = sink.Write(fs.serialized); err != nil {
		_ = sink.Cancel()
		return errors.Wrap(err, "sink.Write(data) failed")
	}

	return sink.Close()

}

func (fs fsmSnapshot) Release() {
	// log.Info("fsmSnapshot.Release action in fsmSnapshot")
}
