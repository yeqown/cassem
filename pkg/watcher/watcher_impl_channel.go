package watcher

import (
	"sync"
	"time"

	"github.com/yeqown/log"

	"github.com/yeqown/cassem/pkg/runtime"
)

const slowObserverTimeout = 200 * time.Millisecond

// topicBucket used to manage one topic, and it's observers. The main purpose to design
// this is to reduce lock conflicts.
type topicBucket struct {
	sync.RWMutex

	observers map[string]IObserver
}

func newTopicBucket() *topicBucket {
	return &topicBucket{
		observers: make(map[string]IObserver, 4),
	}
}

func (t *topicBucket) add(observer IObserver) {
	t.Lock()
	defer t.Unlock()

	t.observers[observer.Identity()] = observer
}

func (t *topicBucket) remove(observer IObserver) {
	t.Lock()
	defer t.Unlock()
	defer observer.Close()

	delete(t.observers, observer.Identity())
}

func (t *topicBucket) snapshotObservers() []IObserver {
	t.RLock()
	defer t.RUnlock()

	observers := make([]IObserver, 0, len(t.observers))
	for _, observer := range t.observers {
		observers = append(observers, observer)
	}
	return observers
}

func (t *topicBucket) distribute(notify IChange) {
	observers := t.snapshotObservers()
	if len(observers) == 0 {
		log.
			WithField("count", len(observers)).
			Debug("topicBucket.distribute called with no observers")
		return
	}

	for _, observer := range observers {
		log.
			WithField("observer", observer.Identity()).
			Debug("watcher.topicBucket.distribute send to observer")
		select {
		case observer.Inbound() <- notify:
		case <-time.After(slowObserverTimeout):
			log.WithField("observer", observer.Identity()).Warn("watcher.topicBucket.distribute remove slow observer")
			t.remove(observer)
		}
	}
}

// channelWatcher implement IWatcher used in core.Core.
type channelWatcher struct {
	ch chan IChange

	_mu sync.RWMutex
	// buckets indicates map[topic][]IObserver
	buckets map[string]*topicBucket
}

var (
	_w    IWatcher
	_once sync.Once
)

// NewChannelWatcher construct a IWatcher at first call, if there is a IWatcher instance already,
// that instance would be returned at once.
func NewChannelWatcher(bufferSize int) IWatcher {
	_once.Do(func() {
		w := channelWatcher{
			ch:      make(chan IChange, bufferSize),
			_mu:     sync.RWMutex{},
			buckets: make(map[string]*topicBucket, 4),
		}

		runtime.GoFunc("watcher.loop", w.loop)

		_w = &w
	})

	return _w
}

func (c *channelWatcher) loop() (err error) {
	for notify := range c.ch {
		log.
			WithFields(log.Fields{
				"topic":  notify.Topic(),
				"change": notify,
			}).
			Debug("channelWatcher loop gets one signal")

		c._mu.RLock()
		bucket, ok := c.buckets[notify.Topic()]
		c._mu.RUnlock()
		if !ok {
			log.
				WithFields(log.Fields{
					"topic": notify.Topic(),
				}).
				Warn("topic has not observer")

			continue
		}

		bucket.distribute(notify)
	}
	return nil
}

func (c *channelWatcher) Subscribe(obs ...IObserver) {
	for _, observer := range obs {
		log.
			WithField("observer", observer).
			Debug("channelWatcher.Subscribe called")

		if observer == nil || observer.Identity() == "" {
			log.
				WithField("observer", observer).
				Warn("channelWatcher.Subscribe skipped EMPTY IObserver")

			continue
		}

		c._mu.Lock()
		for _, topic := range observer.Topics() {
			if _, ok := c.buckets[topic]; !ok {
				c.buckets[topic] = newTopicBucket()
			}
			c.buckets[topic].add(observer)
		}
		c._mu.Unlock()
	}
}

func (c *channelWatcher) Unsubscribe(observer IObserver) {
	log.
		WithField("observer", observer).
		Debug("channelWatcher.Unsubscribe called")

	if observer == nil || observer.Identity() == "" {
		log.
			WithField("observer", observer).
			Warn("channelWatcher.Unsubscribe would not handle with EMPTY IObserver")

		return
	}

	for _, topic := range observer.Topics() {
		c.buckets[topic].remove(observer)
	}
}

func (c *channelWatcher) ChangeNotify(notify IChange) {
	c.ch <- notify
}
