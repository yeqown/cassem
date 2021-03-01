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
	defer observer.Close()

	delete(t.observers, observer.Identity())
}

// distribute will not block sending to c: the caller must ensure that c has sufficient buffer space to
// keep up with the expected signal rate. For a channel used for notification of just one signal value,
// a buffer of size 1 is sufficient.
func (t *topicBucket) distribute(notify Changes) {
	t.RLock()
	observers := t.observers
	t.RUnlock()

	if len(observers) == 0 {
		log.
			WithField("count", len(observers)).
			Debug("topicBucket.distribute called")
	}

	for _, observer := range observers {
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
			ch:      make(chan Changes, bufferSize),
			_mu:     sync.RWMutex{},
			buckets: make(map[string]*topicBucket, 4),
		}

		go w.loop()

		_w = &w
	})

	return _w
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
				WithFields(log.Fields{
					"topic":    notify.Topic(),
					"checksum": notify.CheckSum,
					"data":     string(notify.Data),
				}).
				Debug("channelWatcher loop gets one signal")

			// TODO(@yeqown): optimise here to lock free?
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

			// DONE(@yeqown): use channel instead of method calling
			go bucket.distribute(notify)
		}
	}
}

// TODO(@yeqown): race detect
func (c *channelWatcher) Subscribe(obs ...IObserver) {
	for _, observer := range obs {
		log.
			WithField("observer", observer).
			Debug("channelWatcher.Subscribe called")

		if observer == nil || observer.Identity() == "" {
			log.
				WithField("observer", observer).
				Warn("channelWatcher.Subscribe would not handle with EMPTY IObserver")

			continue
		}

		// register observer into topic.
		for _, topic := range observer.Topics() {
			log.
				WithField("topic", topic).
				Debug("channelWatcher.Subscribe add one observer")

			if _, ok := c.buckets[topic]; !ok {
				c.buckets[topic] = newTopicBucket()
			}
			c.buckets[topic].add(observer)
		}
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

func (c *channelWatcher) ChangeNotify(notify Changes) {
	c.ch <- notify
}
