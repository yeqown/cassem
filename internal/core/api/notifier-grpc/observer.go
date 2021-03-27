package notifier

import (
	"fmt"

	pb "github.com/yeqown/cassem/internal/core/api/notifier-grpc/gen"
	"github.com/yeqown/cassem/internal/watcher"
	"github.com/yeqown/cassem/pkg/datatypes"
	"github.com/yeqown/cassem/pkg/hash"
)

type builtinObserver struct {
	id        string
	keys      []string
	namespace string
	format    datatypes.ContainerFormat
	ch        chan watcher.Changes
	close     func()
}

func (t *builtinObserver) Identity() string                      { return t.id }
func (t builtinObserver) ChangeNotifyCh() chan<- watcher.Changes { return t.ch }
func (t builtinObserver) Close()                                 { t.close() }
func (t builtinObserver) Topics() []string {
	topics := make([]string, len(t.keys))
	for idx, key := range t.keys {
		topics[idx] = fmt.Sprintf("%s#%s#%s", t.namespace, key, t.format)
	}

	return topics
}

// fromPBFormat it panics while format could not be handled.
func fromPBFormat(format pb.Format) datatypes.ContainerFormat {
	switch format {
	case pb.Format_JSON:
		return datatypes.JSON
	case pb.Format_TOML:
		return datatypes.TOML
	}

	panic("unsupported pb format")
}

// toPBFormat it panics while format could not be handled.
func toPBFormat(format datatypes.ContainerFormat) pb.Format {
	switch format {
	case datatypes.JSON:
		return pb.Format_JSON
	case datatypes.TOML:
		return pb.Format_TOML
	}

	panic("unsupported datatypes format")
}

// channel and key of subscriber holds
func genTopicObserver(changesCh chan watcher.Changes, close func(), option *pb.WatchOption) *builtinObserver {
	ob := builtinObserver{
		id:        hash.RandKey(8),
		keys:      option.GetKeys(),
		ch:        changesCh,
		namespace: option.GetNamespace(),
		format:    fromPBFormat(option.GetFormat()),
		close:     close,
	}

	return &ob
}
