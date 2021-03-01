package watcher

import (
	"fmt"
	"log"
	"math/rand"
	"testing"
	"time"

	"github.com/yeqown/cassem/pkg/hash"
)

var keys = []string{
	"key1",
	"key2",
	"key3",
}

func randChooseTopic() string {
	n := rand.Intn(len(keys))
	return keys[n]
}

type testObserver struct {
	id        string
	keys      []string
	namespace string
	format    string
	ch        chan Changes
}

func (t *testObserver) Identity() string              { return t.id }
func (t testObserver) ChangeNotifyCh() chan<- Changes { return t.ch }
func (t testObserver) Topics() []string {
	topics := make([]string, len(t.keys))
	for idx, key := range t.keys {
		topics[idx] = fmt.Sprintf("%s#%s#%s", t.namespace, key, t.format)
	}

	return topics
}
func (t testObserver) Close() { close(t.ch) }

// channel and key of subscriber holds
func genTopicObserver(quit <-chan struct{}, ns, format string, keys ...string) *testObserver {
	ob := testObserver{
		id:        hash.RandKey(8),
		keys:      keys,
		ch:        make(chan Changes, 2),
		namespace: ns,
		format:    format,
	}

	go func() {
		// how quit ?
		for {
			select {
			case n := <-ob.ch:
				log.Printf("got one notify signal of Key=%s", n.Topic())
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
	sent := make(map[string]int, len(keys))
	for _, key := range keys {
		sent[key] = 0
	}

	quit := make(chan struct{}, 1)

	// Subscribe
	ob1 := genTopicObserver(quit, "ns", "json", "key1")
	w.Subscribe(ob1)
	ob2 := genTopicObserver(quit, "ns", "json", "key1", "key2", "key3")
	w.Subscribe(ob2)
	ob3 := genTopicObserver(quit, "ns", "json", "key2", "key3")
	w.Subscribe(ob3)
	ob4 := genTopicObserver(quit, "ns", "json", "key1", "key3")
	w.Subscribe(ob4)
	// ob5 watch other namespaces, should never be notified
	ob5 := genTopicObserver(quit, "ns222", "json", "key1", "key3")
	w.Subscribe(ob5)

	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			if counter <= 0 {
				goto FINISH
			}
			// generate mock data
			key := randChooseTopic()
			w.ChangeNotify(Changes{
				Namespace: "ns",
				Key:       key,
				Format:    "json",
				CheckSum:  hash.RandKey(10),
				Data:      nil,
			})
			sent[key] += 1
			counter -= 1
		}
	}

FINISH:
	quit <- struct{}{}
	// test
}
