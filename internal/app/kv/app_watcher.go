package kv

import (
	"sync"
	"time"

	"github.com/yeqown/log"

	"github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/pkg/hash"
	"github.com/yeqown/cassem/pkg/runtime"
)

const slowObserverTimeout = 200 * time.Millisecond

type topicBucket struct {
	sync.RWMutex

	observers map[string]*builtinObserver
}

func newTopicBucket() *topicBucket {
	return &topicBucket{observers: make(map[string]*builtinObserver, 4)}
}

func (t *topicBucket) add(observer *builtinObserver) {
	t.Lock()
	defer t.Unlock()

	t.observers[observer.Identity()] = observer
}

func (t *topicBucket) remove(observer *builtinObserver) {
	t.Lock()
	defer t.Unlock()

	delete(t.observers, observer.Identity())
	observer.Close()
}

func (t *topicBucket) snapshotObservers() []*builtinObserver {
	t.RLock()
	defer t.RUnlock()

	observers := make([]*builtinObserver, 0, len(t.observers))
	for _, observer := range t.observers {
		observers = append(observers, observer)
	}
	return observers
}

func (t *topicBucket) distribute(notify concept.Change) {
	observers := t.snapshotObservers()
	if len(observers) == 0 {
		log.WithField("count", 0).Debug("topicBucket.distribute called with no observers")
		return
	}

	for _, observer := range observers {
		log.WithField("observer", observer.Identity()).Debug("watcher.topicBucket.distribute send to observer")
		select {
		case observer.Inbound() <- notify:
		case <-time.After(slowObserverTimeout):
			log.WithField("observer", observer.Identity()).Warn("watcher.topicBucket.distribute remove slow observer")
			t.remove(observer)
		}
	}
}

type channelWatcher struct {
	ch chan concept.Change

	mu      sync.RWMutex
	buckets map[string]*topicBucket
}

func newChannelWatcher(bufferSize int) *channelWatcher {
	w := &channelWatcher{
		ch:      make(chan concept.Change, bufferSize),
		buckets: make(map[string]*topicBucket, 4),
	}

	runtime.GoFunc("watcher.loop", w.loop)
	return w
}

func (c *channelWatcher) loop() error {
	for notify := range c.ch {
		log.WithFields(log.Fields{
			"topic":  notify.Topic(),
			"change": notify,
		}).Debug("channelWatcher loop gets one signal")

		c.mu.RLock()
		bucket, ok := c.buckets[notify.Topic()]
		c.mu.RUnlock()
		if !ok {
			log.WithField("topic", notify.Topic()).Warn("topic has not observer")
			continue
		}

		bucket.distribute(notify)
	}
	return nil
}

func (c *channelWatcher) Subscribe(observers ...*builtinObserver) {
	for _, observer := range observers {
		log.WithField("observer", observer).Debug("channelWatcher.Subscribe called")
		if observer == nil || observer.Identity() == "" {
			log.WithField("observer", observer).Warn("channelWatcher.Subscribe skipped EMPTY observer")
			continue
		}

		c.mu.Lock()
		for _, topic := range observer.Topics() {
			if _, ok := c.buckets[topic]; !ok {
				c.buckets[topic] = newTopicBucket()
			}
			c.buckets[topic].add(observer)
		}
		c.mu.Unlock()
	}
}

func (c *channelWatcher) Unsubscribe(observer *builtinObserver) {
	log.WithField("observer", observer).Debug("channelWatcher.Unsubscribe called")
	if observer == nil || observer.Identity() == "" {
		log.WithField("observer", observer).Warn("channelWatcher.Unsubscribe would not handle EMPTY observer")
		return
	}

	for _, topic := range observer.Topics() {
		c.mu.RLock()
		bucket := c.buckets[topic]
		c.mu.RUnlock()
		if bucket != nil {
			bucket.remove(observer)
		}
	}
}

func (c *channelWatcher) ChangeNotify(notify concept.Change) {
	c.ch <- notify
}

type builtinObserver struct {
	id      string
	keys    []string
	ch      chan concept.Change
	once    sync.Once
	closeFn func()
}

func newTopicObserver(changesCh chan concept.Change, closeFn func(), keys []string) *builtinObserver {
	return &builtinObserver{
		id:      hash.RandKey(8),
		keys:    keys,
		ch:      changesCh,
		closeFn: closeFn,
	}
}

func (t *builtinObserver) Identity() string                  { return t.id }
func (t *builtinObserver) Inbound() chan<- concept.Change    { return t.ch }
func (t *builtinObserver) Outbound() <-chan concept.Change   { return t.ch }
func (t *builtinObserver) Close() {
	t.once.Do(func() {
		if t.closeFn != nil {
			t.closeFn()
		}
	})
}

func (t *builtinObserver) Topics() []string {
	seen := make(map[string]struct{}, len(t.keys))
	result := make([]string, 0, len(t.keys))
	for _, key := range t.keys {
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, key)
	}
	return result
}
