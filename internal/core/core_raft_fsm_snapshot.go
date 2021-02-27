package core

import (
	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
	"github.com/yeqown/log"
)

type fsmSnapshot struct {
	serialized []byte
}

func (fs fsmSnapshot) Persist(sink raft.SnapshotSink) (err error) {
	log.Info("fsmSnapshot.Persist action in fsmSnapshot")
	defer sink.Close()

	if _, err = sink.Write(fs.serialized); err != nil {
		return errors.Wrap(err, "sink.Write(data) failed")
	}

	return nil
}

func (fs fsmSnapshot) Release() {
	log.Info("fsmSnapshot.Release action in fsmSnapshot")
}
