package kv

import (
	"github.com/yeqown/cassem/pkg/hash"
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

func (t *builtinObserver) Identity() string                 { return t.id }
func (t *builtinObserver) Inbound() chan<- watcher.IChange  { return t.ch }
func (t *builtinObserver) Outbound() <-chan watcher.IChange { return t.ch }
func (t *builtinObserver) Close()                           { t.close() }
func (t *builtinObserver) Topics() []string {
	// deduplicate keys
	seen := make(map[string]struct{}, len(t.keys))
	result := make([]string, 0, len(t.keys))

	for _, key := range t.keys {
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}
			result = append(result, key)
		}
	}

	return result
}
