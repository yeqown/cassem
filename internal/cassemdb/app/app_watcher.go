package app

import (
	"github.com/yeqown/cassem/pkg/hash"
	"github.com/yeqown/cassem/pkg/set"
	"github.com/yeqown/cassem/pkg/watcher"
)

type builtinObserver struct {
	id    string
	keys  []string
	ch    chan watcher.IChange
	close func()
}

// newTopicObserver channel and key of subscriber holds
func newTopicObserver(changesCh chan watcher.IChange, close func(), keys []string) *builtinObserver {
	ob := builtinObserver{
		id:    hash.RandKey(8),
		keys:  keys,
		ch:    changesCh,
		close: close,
	}

	return &ob
}

func (t *builtinObserver) Identity() string                { return t.id }
func (t builtinObserver) Inbound() chan<- watcher.IChange  { return t.ch }
func (t builtinObserver) Outbound() <-chan watcher.IChange { return t.ch }
func (t builtinObserver) Close()                           { t.close() }
func (t builtinObserver) Topics() []string {
	s := set.NewStringSet(len(t.keys))

	for _, key := range t.keys {
		s.Add(key)
	}

	return s.Keys()
}
