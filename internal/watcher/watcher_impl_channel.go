package watcher

import (
	"sync"
	"time"

	"github.com/yeqown/log"
)

// topicBucket used to manage one topic and it's observers. The main purpose to design
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

	t.observers[observer.Identity()] = observer
}

// distribute will not block sending to c: the caller must ensure that c has sufficient buffer space to
// keep up with the expected signal rate. For a channel used for notification of just one signal value,
// a buffer of size 1 is sufficient.
func (t *topicBucket) distribute(notify Changes) {
	t.RLock()
	defer t.RUnlock()

	for _, observer := range t.observers {
		// NOTICE: send but do not block for it
		select {
		case observer.ChangeNotifyCh() <- notify:
		default:

		}

	}
}

// channelWatcher implement IWatcher used in core.Core.
type channelWatcher struct {
	ch chan Changes

	// buckets indicates map[topic][]IObserver
	_mu     sync.RWMutex
	buckets map[string]*topicBucket
}

func newChannelWatcher(bufferSize int) IWatcher {
	w := channelWatcher{
		ch:      make(chan Changes, bufferSize),
		_mu:     sync.RWMutex{},
		buckets: make(map[string]*topicBucket, 4),
	}

	go w.loop()

	return &w
}

func (c *channelWatcher) loop() {
	defer func() {
		if v := recover(); v != nil {
			log.
				Errorf("channelWatcher.loop panicked: %v", v)

			time.Sleep(2 * time.Second)
			go c.loop()
		}
	}()

	for {
		select {
		case notify := <-c.ch:
			log.
				WithField("notifyData", notify).
				Debug("channelWatcher loop gets one signal")

			c._mu.RLock()
			bucket, ok := c.buckets[notify.Topic]
			c._mu.RUnlock()
			if !ok {
				log.
					WithField("notify", notify).
					Warn("topic has not observer")

				return
			}

			// DONE(@yeqown): use channel instead of method calling
			go bucket.distribute(notify)
		}
	}
}

// TODO(@yeqown): race detect
func (c *channelWatcher) Subscribe(obs ...IObserver) {
	for _, observer := range obs {
		if observer == nil || observer.Identity() == "" {
			log.
				WithField("observer", observer).
				Warn("channelWatcher.Subscribe would not handle with EMPTY IObserver")

			continue
		}

		topics := observer.Topics()
		// register observer into topic.
		for _, topic := range topics {
			if _, ok := c.buckets[topic]; !ok {
				c.buckets[topic] = newTopicBucket()
			}

			c.buckets[topic].add(observer)
		}
	}
}

func (c *channelWatcher) Unsubscribe(observer IObserver) {
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

func (c *channelWatcher) ChangeNotify(notify Changes) {
	c.ch <- notify
}
