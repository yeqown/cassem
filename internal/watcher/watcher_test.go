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

type testObserver struct {
	id     string
	topics []string
	ch     chan Changes
}

func (t *testObserver) Identity() string              { return t.id }
func (t testObserver) Topics() []string               { return t.topics }
func (t testObserver) ChangeNotifyCh() chan<- Changes { return t.ch }

// channel and topic of subscriber holds
func genTopicObserver(quit <-chan struct{}, topics ...string) *testObserver {
	ob := testObserver{
		id:     hash.RandKey(8),
		topics: topics,
		ch:     make(chan Changes, 2),
	}

	go func() {
		// how quit ?
		for {
			select {
			case n := <-ob.ch:
				log.Printf("got one notify signal of Topic=%s", n.Topic)
			case <-quit:
				return
			}
		}
	}()

	return &ob
}

func Test_Watcher(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	w := NewChannelWatcher(5)

	// count data and control flag
	counter := 20
	sent := make(map[string]int, len(topics))
	for _, topic := range topics {
		sent[topic] = 0
	}

	quit := make(chan struct{}, 1)

	// Subscribe
	ob1 := genTopicObserver(quit, "topic1")
	w.Subscribe(ob1)
	ob2 := genTopicObserver(quit, "topic1", "topic2", "topic3")
	w.Subscribe(ob2)
	ob3 := genTopicObserver(quit, "topic2", "topic3")
	w.Subscribe(ob3)
	ob4 := genTopicObserver(quit, "topic1", "topic3")
	w.Subscribe(ob4)

	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			if counter <= 0 {
				goto FINISH
			}
			// generate mock data
			topic := randChooseTopic()
			w.ChangeNotify(Changes{
				CheckSum: hash.RandKey(10),
				Topic:    topic,
				Data:     nil,
			})
			sent[topic] += 1
			counter -= 1
		}
	}

FINISH:
	quit <- struct{}{}
	// test
}
