package core

import (
	"github.com/yeqown/cassem/internal/cache"

	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
	"github.com/yeqown/log"
)

type fsmSnapshot struct {
	containerCache cache.ICache
}

func (fs fsmSnapshot) Persist(sink raft.SnapshotSink) (err error) {
	log.Info("Release action in fsmSnapshot")
	defer sink.Close()

	data, err := fs.containerCache.Persist()
	if err != nil {
		return errors.Wrap(err, "fs.containerCache.Persist() failed")
	}

	if _, err = sink.Write(data); err != nil {
		return errors.Wrap(err, "sink.Write(data) failed")
	}

	return nil
}

func (fs fsmSnapshot) Release() {
	log.Info("Release action in fsmSnapshot")
}
