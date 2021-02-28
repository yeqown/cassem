package watcher

import (
	"sync"

	"github.com/yeqown/log"
)

type bucket struct {
	chs map[string][]chan<- ChangeNotify
}

type channelWatcher struct {
	ch      chan ChangeNotify
	buckets []bucket

	topicCh map[string]chan ChangeNotify
}

func newChannelWatcher(bucketSize int) IWatcher {
	buckets := make([]bucket, bucketSize)
	return &channelWatcher{
		ch:      make(chan ChangeNotify, bucketSize),
		buckets: buckets,
		topicCh: make(map[string]chan ChangeNotify, bucketSize),
	}
}

// TODO(@yeqown): how to identify which channel should be removed?
func (c channelWatcher) Unsubscribe(topics ...string) {
	panic("not implement")
}

func (c channelWatcher) Subscribe(ch chan<- ChangeNotify, topics ...string) {
	for _, topic := range topics {
		if topic == "" {
			continue
		}

		if _, ok := c.topicCh[topic]; !ok {
			ch <- ChangeNotify{}
		}

		// TODO(@yeqown): register into channelWatcher
	}

	// MAYBE ERROR got.
}

func (c channelWatcher) ChangeNotifyCh() chan<- ChangeNotify {
	return c.ch
}

func (c channelWatcher) loop() {
	for {
		select {
		case notify := <-c.ch:
			log.
				WithField("notifyData", notify).
				Debug("channelWatcher loop gets one signal")
			// TODO(@yeqown): then fan-out
		}
	}
}

func merge(cs ...<-chan ChangeNotify) <-chan ChangeNotify {
	var wg sync.WaitGroup
	out := make(chan ChangeNotify)

	// Start an output goroutine for each input channel in cs.  output
	// copies values from c to out until c is closed, then calls wg.Done.
	output := func(c <-chan ChangeNotify) {
		for n := range c {
			out <- n
		}
		wg.Done()
	}
	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	// Start a goroutine to close out once all the output goroutines are
	// done.  This must start after the wg.Add call.
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}
