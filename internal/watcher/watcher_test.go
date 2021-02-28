package watcher

import (
	"log"
	"math/rand"
	"testing"
	"time"

	"github.com/yeqown/cassem/pkg/hash"
)

var topics = []string{
	"topic1",
	"topic2",
	"topic3",
}

func randChooseTopic() string {
	n := rand.Intn(len(topics))
	return topics[n]
}

// channel and topic of subscriber holds
func genTopicSubscriber(quit <-chan struct{}, topics ...string) (chan<- ChangeNotify, []string) {
	ch := make(chan ChangeNotify, 1)
	go func() {
		// how quit ?
		for {
			select {
			case n := <-ch:
				log.Printf("got one notify signal of Topic=%s", n.Topic)
			case <-quit:
				return
			}
		}
	}()

	return ch, topics
}

func Test_Watcher(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	w := newChannelWatcher(5)
	watcherCh := w.ChangeNotifyCh()

	// count data and control flag
	counter := 20
	sent := make(map[string]int, len(topics))
	for _, topic := range topics {
		sent[topic] = 0
	}

	quit := make(chan struct{}, 1)

	// Subscribe
	ch1, topics1 := genTopicSubscriber(quit, "topic1")
	w.Subscribe(ch1, topics1...)
	ch2, topics2 := genTopicSubscriber(quit, "topic1", "topic2", "topic3")
	w.Subscribe(ch2, topics2...)
	ch3, topics3 := genTopicSubscriber(quit, "topic2", "topic3")
	w.Subscribe(ch3, topics3...)
	ch4, topics4 := genTopicSubscriber(quit, "topic1", "topic3")
	w.Subscribe(ch4, topics4...)

	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			if counter <= 0 {
				goto FINISH
			}
			// generate mock data
			topic := randChooseTopic()
			watcherCh <- ChangeNotify{
				CheckSum: hash.RandKey(10),
				Topic:    topic,
				Data:     nil,
			}
			sent[topic] += 1
			counter -= 1
		}
	}

FINISH:
	quit <- struct{}{}
	// test
}
